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

	"github.com/dappley/go-dappley/common"
	"github.com/hashicorp/golang-lru"
	"github.com/libp2p/go-libp2p-peer"
	logger "github.com/sirupsen/logrus"
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
	blockRequestCh chan BlockRequestPars
	syncState      bool
	size           int
	blkCache       *lru.Cache //cache of full blks
}

func (pool *BlockPool) GetSyncState() bool {
	return pool.syncState
}
func (pool *BlockPool) SetSyncState(sync bool) {
	pool.syncState = sync
}
func NewBlockPool(size int) *BlockPool {
	if size <= 0 {
		size = BlockPoolMaxSize
	}
	pool := &BlockPool{
		size:           size,
		blockRequestCh: make(chan BlockRequestPars, size),
		syncState:      false,
	}
	pool.blkCache, _ = lru.New(BlockCacheLRUCacheLimit)

	return pool
}

func (pool *BlockPool) BlockRequestCh() chan BlockRequestPars {
	return pool.blockRequestCh
}

func (pool *BlockPool) Verify(block *Block) bool {
	logger.Info("BlockPool: Has received a new block")
	if pool.syncState {
		logger.Debug("BlockPool: is syncing already, tossing block ")
		return false
	}
	if !block.VerifyHash() {
		logger.Warn("BlockPool: The received block cannot pass hash verification!")
		return false
	}
	//TODO: Verify double spending transactions in the same block

	return true
}

func (pool *BlockPool) CacheRecvdBlock(tree *common.Tree, maxHeight uint64) Hash {
	blkCache := pool.blkCache

	if blkCache.Contains(tree.GetValue().(*Block).GetHash().String()) {
		return nil
	}
	if !pool.isChildBlockInCache(tree.GetValue().(*Block).GetHash().String()) && tree.GetValue().(*Block).GetHeight() <= maxHeight {
		return nil
	}
	blkCache.Add(tree.GetValue().(*Block).GetHash().String(), tree)
	pool.updateBlkCache(tree)

	forkheadParentHash := tree.GetValue().(*Block).GetPrevHash()
	return forkheadParentHash

}
func (pool *BlockPool) GenerateForkBlocks(tree *common.Tree, maxHeight uint64) []*Block {
	_, forkTailTree := tree.FindHeightestChild(&common.Tree{}, 0, 0)
	if forkTailTree.GetValue().(*Block).GetHeight() > maxHeight {
		pool.SetSyncState(true)
	} else {
		return nil
	}

	trees := forkTailTree.GetParentTreesRange(tree)
	forkBlks := getBlocksFromTrees(trees)
	return forkBlks
}

func (pool *BlockPool) CleanCache(tree *common.Tree) {
	tree.Delete()
	logger.WithFields(logger.Fields{
		"syncstate": 0,
	}).Debug("Merge finished or exited, setting syncstate to false")
	pool.SetSyncState(false)
}

func getBlocksFromTrees(trees []*common.Tree) []*Block {
	var blocks []*Block
	for _, tree := range trees {
		blocks = append(blocks, tree.GetValue().(*Block))
	}
	return blocks
}

func (pool *BlockPool) updateBlkCache(tree *common.Tree) {
	blkCache := pool.blkCache
	// try to link child
	for _, key := range blkCache.Keys() {
		if cachedBlk, ok := blkCache.Get(key); ok {
			if hex.EncodeToString(cachedBlk.(*common.Tree).GetValue().(*Block).GetPrevHash()) == tree.GetValue().(*Block).GetHash().String() {
				logger.WithFields(logger.Fields{
					"treeheight":     tree.GetValue().(*Block).GetHeight(),
					"cacheblkHeight": cachedBlk.(*common.Tree).GetValue().(*Block).GetHeight(),
				}).Info("child added")
				tree.AddChild(cachedBlk.(*common.Tree))
			}
		}
	}

	//try to link parent
	if parent, ok := blkCache.Get(string(tree.GetValue().(*Block).GetPrevHash())); ok == true {
		tree.AddParent(parent.(*common.Tree))
	}
	logger.WithFields(logger.Fields{
		"height": tree.GetValue().(*Block).GetHeight(),
		"hash":   hex.EncodeToString(tree.GetValue().(*Block).GetHash()),
	}).Debug("BlockPool: Finished updating BlockPoolCache")
}

func (pool *BlockPool) requestPrevBlock(tree *common.Tree, sender peer.ID) {
	logger.WithFields(logger.Fields{
		"requestedHash": hex.EncodeToString(tree.GetValue().(*Block).GetPrevHash()),
		"height":        tree.GetValue().(*Block).GetHeight() - 1,
		"from":          sender,
	}).Info("BlockPool: Parent not found, requesting block")
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
