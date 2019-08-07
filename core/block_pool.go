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
	"github.com/dappley/go-dappley/core/block"
	"sync"

	"github.com/hashicorp/golang-lru"
	logger "github.com/sirupsen/logrus"

	"github.com/dappley/go-dappley/common"
)

const BlockCacheLRUCacheLimit = 1024
const ForkCacheLRUCacheLimit = 128

type BlockPool struct {
	blkCache       *lru.Cache //cache of full blks
	forkHeads      map[string]*common.Tree
	forkHeadsMutex *sync.RWMutex
}

func NewBlockPool() *BlockPool {

	pool := &BlockPool{
		forkHeads:      make(map[string]*common.Tree),
		forkHeadsMutex: &sync.RWMutex{},
	}
	pool.blkCache, _ = lru.New(BlockCacheLRUCacheLimit)
	return pool
}

// CacheBlock caches the provided block if it is not a duplicate and it's height is within the upper bound of maxHeight,
// returning the head of it's fork
func (pool *BlockPool) CacheBlock(blk *block.Block, maxHeight uint64) *common.Tree {

	tree, _ := common.NewTree(blk.GetHash().String(), blk)

	if pool.blkCache.Contains(blk.GetHash().String()) {
		return tree.GetRoot()
	}
	if !pool.isChildBlockInCache(blk.GetHash().String()) && blk.GetHeight() <= maxHeight {
		return tree.GetRoot()
	}

	pool.blkCache.Add(blk.GetHash().String(), tree)
	pool.updateBlkCache(tree)
	return tree.GetRoot()
}

func (pool *BlockPool) GenerateForkBlocks(tree *common.Tree, maxHeight uint64) []*block.Block {
	_, forkTailTree := tree.FindHeightestChild(&common.Tree{}, 0, 0)
	if forkTailTree.GetValue().(*block.Block).GetHeight() > maxHeight {
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
	delete(pool.forkHeads, tree.GetValue().(*block.Block).GetHash().String())
	pool.forkHeadsMutex.Unlock()
	tree.Delete()
	logger.Debug("BlockPool: merge finished or exited, setting syncstate to false.")
}

func getBlocksFromTrees(trees []*common.Tree) []*block.Block {
	var blocks []*block.Block
	for _, tree := range trees {
		blocks = append(blocks, tree.GetValue().(*block.Block))
	}
	return blocks
}

// updateBlkCache updates parent and Children of the tree
func (pool *BlockPool) updateBlkCache(tree *common.Tree) {
	blkCache := pool.blkCache
	// try to link child
	for _, key := range blkCache.Keys() {
		if cachedBlk, ok := blkCache.Get(key); ok {
			if cachedBlk.(*common.Tree).GetValue().(*block.Block).GetPrevHash().String() == tree.GetValue().(*block.Block).GetHash().String() {
				logger.WithFields(logger.Fields{
					"tree_height":  tree.GetValue().(*block.Block).GetHeight(),
					"child_height": cachedBlk.(*common.Tree).GetValue().(*block.Block).GetHeight(),
				}).Info("BlockPool: added a child block to the tree.")
				tree.AddChild(cachedBlk.(*common.Tree))
				pool.forkHeadsMutex.Lock()
				delete(pool.forkHeads, cachedBlk.(*common.Tree).GetValue().(*block.Block).GetHash().String())
				pool.forkHeadsMutex.Unlock()
			}
		}
	}

	//try to link parent
	if parent, ok := blkCache.Get(tree.GetValue().(*block.Block).GetPrevHash().String()); ok {
		err := tree.AddParent(parent.(*common.Tree))
		if err != nil {
			logger.WithError(err).WithFields(logger.Fields{
				"tree_height":   tree.GetValue().(*block.Block).GetHeight(),
				"parent_height": parent.(*common.Tree).GetValue().(*block.Block).GetHeight(),
				"parent_hash":   parent.(*common.Tree).GetValue().(*block.Block).GetHash(),
			}).Error("BlockPool: failed to add a parent block to the tree.")
			return
		}
		logger.WithFields(logger.Fields{
			"tree_height":   tree.GetValue().(*block.Block).GetHeight(),
			"parent_height": parent.(*common.Tree).GetValue().(*block.Block).GetHeight(),
		}).Info("BlockPool: added a parent block to the tree.")
	} else {
		pool.forkHeadsMutex.Lock()
		pool.forkHeads[tree.GetValue().(*block.Block).GetHash().String()] = tree
		pool.forkHeadsMutex.Unlock()
	}

	logger.WithFields(logger.Fields{
		"height": tree.GetValue().(*block.Block).GetHeight(),
		"hash":   tree.GetValue().(*block.Block).GetHash().String(),
	}).Debug("BlockPool: finished updating BlockPoolCache.")
}

func (pool *BlockPool) getBlkFromBlkCache(hashString string) *block.Block {
	if val, ok := pool.blkCache.Get(hashString); ok == true {
		return val.(*block.Block)
	}
	return nil
}

func (pool BlockPool) isChildBlockInCache(hashString string) bool {
	blkCache := pool.blkCache
	for _, key := range blkCache.Keys() {
		if cachedBlk, ok := blkCache.Get(key); ok {
			if cachedBlk.(*common.Tree).GetValue().(*block.Block).GetPrevHash().String() == hashString {
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
