// Copyright (C) 2018 go-dappley authors
//
// This file is part of the go-dappley library.
//
// the go-dappley library is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either pubKeyHash 3 of the License, or
// (at your option) any later pubKeyHash.
//
// the go-dappley library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with the go-dappley library.  If not, see <http://www.gnu.org/licenses/>.
//

package core

import (
	"encoding/hex"

	lru "github.com/hashicorp/golang-lru"
	peer "github.com/libp2p/go-libp2p-peer"
	logger "github.com/sirupsen/logrus"

	"github.com/dappley/go-dappley/common"
)

const BlockPoolMaxSize = 100
const BlockCacheLRUCacheLimit = 1024
const ForkCacheLRUCacheLimit = 128

type BlockRequestPars struct {
	BlockHash Hash
	Pid       peer.ID
}

type RcvedBlock struct {
	Block *Block
	Pid   peer.ID
}

type BlockPool struct {
	blockRequestCh   chan BlockRequestPars
	downloadBlocksCh chan bool
	size             int
	blkCache         *lru.Cache   //cache of full blks
}

func NewBlockPool(size int) *BlockPool {
	if size <= 0 {
		size = BlockPoolMaxSize
	}
	pool := &BlockPool{
		size:             size,
		blockRequestCh:   make(chan BlockRequestPars, size),
		downloadBlocksCh: make(chan bool, 1),
	}
	pool.blkCache, _ = lru.New(BlockCacheLRUCacheLimit)

	return pool
}

func (pool *BlockPool) BlockRequestCh() chan BlockRequestPars {
	return pool.blockRequestCh
}

func (pool *BlockPool) DownloadBlocksCh() chan bool {
	return pool.downloadBlocksCh
}

func (pool *BlockPool) Verify(block *Block) bool {
	logger.Info("BlockPool: Has received a new block")
	if !block.VerifyHash() {
		logger.Warn("BlockPool: received block cannot pass hash verification!")
		return false
	}
	//TODO: Verify double spending transactions in the same block

	return true
}

// CacheBlock cache the tree, update the cache and return the fork head
func (pool *BlockPool) CacheBlock(tree *common.Tree, maxHeight uint64) {
	blkCache := pool.blkCache

	if blkCache.Contains(tree.GetValue().(*Block).GetHash().String()) {
		return
	}
	if !pool.isChildBlockInCache(tree.GetValue().(*Block).GetHash().String()) && tree.GetValue().(*Block).GetHeight() <= maxHeight {
		return
	}
	blkCache.Add(tree.GetValue().(*Block).GetHash().String(), tree)
	pool.updateBlkCache(tree)
}

func (pool *BlockPool) GenerateForkBlocks(tree *common.Tree, maxHeight uint64) []*Block {
	_, forkTailTree := tree.FindHeightestChild(&common.Tree{}, 0, 0)
	if forkTailTree.GetValue().(*Block).GetHeight() > maxHeight {
	} else {
		return nil
	}

	trees := forkTailTree.GetParentTreesRange(tree)
	forkBlks := getBlocksFromTrees(trees)
	return forkBlks
}

func (pool *BlockPool) CleanCache(tree *common.Tree) {
	_, forkTailTree := tree.FindHeightestChild(&common.Tree{}, 0, 0)
	trees := forkTailTree.GetParentTreesRange(tree)
	forkBlks := getBlocksFromTrees(trees)
	for _, forkBlk := range forkBlks {
		pool.blkCache.Remove(forkBlk.GetHash().String())
	}

	tree.Delete()
	logger.Debug("BlockPool: merge finished or exited, setting syncstate to false.")
}

func getBlocksFromTrees(trees []*common.Tree) []*Block {
	var blocks []*Block
	for _, tree := range trees {
		blocks = append(blocks, tree.GetValue().(*Block))
	}
	return blocks
}

// updateBlkCache updates parent and Children of the tree
func (pool *BlockPool) updateBlkCache(tree *common.Tree) {
	blkCache := pool.blkCache
	// try to link child
	for _, key := range blkCache.Keys() {
		if cachedBlk, ok := blkCache.Get(key); ok {
			if hex.EncodeToString(cachedBlk.(*common.Tree).GetValue().(*Block).GetPrevHash()) == tree.GetValue().(*Block).GetHash().String() {
				logger.WithFields(logger.Fields{
					"tree_height":  tree.GetValue().(*Block).GetHeight(),
					"child_height": cachedBlk.(*common.Tree).GetValue().(*Block).GetHeight(),
				}).Info("BlockPool: added a child block to the tree.")
				tree.AddChild(cachedBlk.(*common.Tree))
			}
		}
	}

	//try to link parent
	if parent, ok := blkCache.Get(tree.GetValue().(*Block).GetPrevHash().String()); ok == true {
		err := tree.AddParent(parent.(*common.Tree))
		if err != nil {
			logger.WithError(err).WithFields(logger.Fields{
				"tree_height":   tree.GetValue().(*Block).GetHeight(),
				"parent_height": parent.(*common.Tree).GetValue().(*Block).GetHeight(),
				"parent_hash":   parent.(*common.Tree).GetValue().(*Block).GetHash(),
			}).Error("BlockPool: failed to add a parent block to the tree.")
			return
		}
		logger.WithFields(logger.Fields{
			"tree_height":   tree.GetValue().(*Block).GetHeight(),
			"parent_height": parent.(*common.Tree).GetValue().(*Block).GetHeight(),
		}).Info("BlockPool: added a parent block to the tree.")
	}

	logger.WithFields(logger.Fields{
		"height": tree.GetValue().(*Block).GetHeight(),
		"hash":   hex.EncodeToString(tree.GetValue().(*Block).GetHash()),
	}).Debug("BlockPool: finished updating BlockPoolCache.")
}

func (pool *BlockPool) requestPrevBlock(tree *common.Tree, sender peer.ID) {
	logger.WithFields(logger.Fields{
		"hash":   hex.EncodeToString(tree.GetValue().(*Block).GetPrevHash()),
		"height": tree.GetValue().(*Block).GetHeight() - 1,
		"from":   sender,
	}).Info("BlockPool: is requesting a block.")
	pool.blockRequestCh <- BlockRequestPars{tree.GetValue().(*Block).GetPrevHash(), sender}
}

func (pool *BlockPool) getBlkFromBlkCache(hashString string) *Block {
	if val, ok := pool.blkCache.Get(hashString); ok == true {
		return val.(*Block)
	}
	return nil
}

func (pool BlockPool) isChildBlockInCache(hashString string) bool {
	blkCache := pool.blkCache
	for _, key := range blkCache.Keys() {
		if cachedBlk, ok := blkCache.Get(key); ok {
			if hex.EncodeToString(cachedBlk.(*common.Tree).GetValue().(*Block).GetPrevHash()) == hashString {
				return true
			}
		}
	}
	return false
}
