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

func (pool *BlockPool) Add(blk *block.Block) {

	if pool.blkCache.Contains(blk.GetHash().String()) {
		return
	}

	//TODO: inject consensus to check if the block should be disgarded
	forkhead, _ := common.NewTree(blk.GetHash().String(), blk)
	pool.blkCache.Add(blk.GetHash().String(), forkhead)
}

func (pool *BlockPool) GetFork(blk *block.Block) []*block.Block {

	if !pool.blkCache.Contains(blk.GetHash().String()) {
		return nil
	}

	forkheadTree, _ := pool.blkCache.Get(blk.GetHash().String())
	forkHead := forkheadTree.(*common.Tree)
	pool.updateForkHead(forkHead)

	_, forkTailTree := forkHead.FindHeightestChild(&common.Tree{}, 0, 0)
	forkTrees := forkTailTree.GetParentTreesRange(forkHead)
	return getBlocksFromTrees(forkTrees)
}

func (pool *BlockPool) RemoveFork(fork []*block.Block) {
	pool.forkHeadsMutex.Lock()
	defer pool.forkHeadsMutex.Unlock()

	for _, forkBlk := range fork {
		pool.blkCache.Remove(forkBlk.GetHash().String())
	}

	delete(pool.forkHeads, fork[0].GetHash().String())
	logger.Debug("BlockPool: merge finished or exited, setting syncstate to false.")
}

func getBlocksFromTrees(trees []*common.Tree) []*block.Block {
	var blocks []*block.Block
	for _, tree := range trees {
		blocks = append(blocks, tree.GetValue().(*block.Block))
	}
	return blocks
}

// updateForkHead updates parent and Children of the tree
func (pool *BlockPool) updateForkHead(forkHead *common.Tree) {
	pool.linkChildren(forkHead)
	pool.linkParent(forkHead)
}

func (pool *BlockPool) linkChildren(forkHead *common.Tree) {
	pool.forkHeadsMutex.Lock()
	defer pool.forkHeadsMutex.Unlock()
	for _, blkHash := range pool.blkCache.Keys() {
		if cachedBlk, ok := pool.blkCache.Get(blkHash); ok {
			if cachedBlk.(*common.Tree).GetValue().(*block.Block).GetPrevHash().String() == forkHead.GetValue().(*block.Block).GetHash().String() {
				logger.WithFields(logger.Fields{
					"tree_height":  forkHead.GetValue().(*block.Block).GetHeight(),
					"child_height": cachedBlk.(*common.Tree).GetValue().(*block.Block).GetHeight(),
				}).Debug("BlockPool: added a child block to the forkHead.")
				forkHead.AddChild(cachedBlk.(*common.Tree))
				delete(pool.forkHeads, cachedBlk.(*common.Tree).GetValue().(*block.Block).GetHash().String())
			}
		}
	}
}

func (pool *BlockPool) linkParent(forkHead *common.Tree) {

	pool.forkHeadsMutex.Lock()
	defer pool.forkHeadsMutex.Unlock()

	if parent, ok := pool.blkCache.Get(forkHead.GetValue().(*block.Block).GetPrevHash().String()); ok {
		err := forkHead.AddParent(parent.(*common.Tree))
		if err != nil {
			logger.WithError(err).WithFields(logger.Fields{
				"tree_height":   forkHead.GetValue().(*block.Block).GetHeight(),
				"parent_height": parent.(*common.Tree).GetValue().(*block.Block).GetHeight(),
				"parent_hash":   parent.(*common.Tree).GetValue().(*block.Block).GetHash(),
			}).Error("BlockPool: failed to add a parent block to the forkHead.")
			return
		}
		logger.WithFields(logger.Fields{
			"tree_height":   forkHead.GetValue().(*block.Block).GetHeight(),
			"parent_height": parent.(*common.Tree).GetValue().(*block.Block).GetHeight(),
		}).Debug("BlockPool: added a parent block to the forkHead.")
	} else {
		pool.forkHeads[forkHead.GetValue().(*block.Block).GetHash().String()] = forkHead
	}

	forkHead = forkHead.GetRoot()
}

func (pool *BlockPool) ForkHeadRange(fn func(blkHash string, tree *common.Tree)) {
	pool.forkHeadsMutex.RLock()
	defer pool.forkHeadsMutex.RUnlock()
	for k, v := range pool.forkHeads {
		fn(k, v)
	}
}
