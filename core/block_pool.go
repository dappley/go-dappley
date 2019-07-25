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
	"sync"

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
	blkCache         *lru.Cache //cache of full blks
	forkHeads        map[string]*common.Tree
	forkHeadsMutex   *sync.RWMutex
}

func NewBlockPool(size int) *BlockPool {
	if size <= 0 {
		size = BlockPoolMaxSize
	}
	pool := &BlockPool{
		size:             size,
		blockRequestCh:   make(chan BlockRequestPars, size),
		downloadBlocksCh: make(chan bool, 1),
		forkHeads:        make(map[string]*common.Tree),
		forkHeadsMutex:   &sync.RWMutex{},
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

// CacheBlock caches the provided block if it is not a duplicate and it's height is within the upper bound of maxHeight,
// returning the head of it's fork
func (pool *BlockPool) CacheBlock(block *Block, maxHeight uint64) *common.Tree {
	blkCache := pool.blkCache
	tree, _ := common.NewTree(block.GetHash().String(), block)

	if blkCache.Contains(block.GetHash().String()) {
		return tree.GetRoot()
	}
	if !pool.isChildBlockInCache(block.GetHash().String()) && block.GetHeight() <= maxHeight {
		return tree.GetRoot()
	}

	blkCache.Add(block.GetHash().String(), tree)
	pool.updateBlkCache(tree)
	return tree.GetRoot()
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

	pool.forkHeadsMutex.Lock()
	delete(pool.forkHeads, tree.GetValue().(*Block).GetHash().String())
	pool.forkHeadsMutex.Unlock()
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
			if cachedBlk.(*common.Tree).GetValue().(*Block).GetPrevHash().String() == tree.GetValue().(*Block).GetHash().String() {
				logger.WithFields(logger.Fields{
					"tree_height":  tree.GetValue().(*Block).GetHeight(),
					"child_height": cachedBlk.(*common.Tree).GetValue().(*Block).GetHeight(),
				}).Info("BlockPool: added a child block to the tree.")
				tree.AddChild(cachedBlk.(*common.Tree))
				pool.forkHeadsMutex.Lock()
				delete(pool.forkHeads, cachedBlk.(*common.Tree).GetValue().(*Block).GetHash().String())
				pool.forkHeadsMutex.Unlock()
			}
		}
	}

	//try to link parent
	if parent, ok := blkCache.Get(tree.GetValue().(*Block).GetPrevHash().String()); ok {
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
	} else {
		pool.forkHeadsMutex.Lock()
		pool.forkHeads[tree.GetValue().(*Block).GetHash().String()] = tree
		pool.forkHeadsMutex.Unlock()
	}

	logger.WithFields(logger.Fields{
		"height": tree.GetValue().(*Block).GetHeight(),
		"hash":   tree.GetValue().(*Block).GetHash().String(),
	}).Debug("BlockPool: finished updating BlockPoolCache.")
}

func (pool *BlockPool) requestPrevBlock(tree *common.Tree, sender peer.ID) {
	logger.WithFields(logger.Fields{
		"hash":   tree.GetValue().(*Block).GetPrevHash().String(),
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
			if cachedBlk.(*common.Tree).GetValue().(*Block).GetPrevHash().String() == hashString {
				return true
			}
		}
	}
	return false
}

func (pool *BlockPool) ForkHeadRange(fn func(blkHash string, tree *common.Tree)) {
	pool.forkHeadsMutex.RLock()
	defer pool.forkHeadsMutex.RUnlock()
	for k, v := range pool.forkHeads {
		fn(k, v)
	}
}
