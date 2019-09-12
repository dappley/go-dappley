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

	"github.com/dappley/go-dappley/common/hash"
	"github.com/dappley/go-dappley/core/block"

	lru "github.com/hashicorp/golang-lru"
	logger "github.com/sirupsen/logrus"

	"github.com/dappley/go-dappley/common"
)

const BlockCacheLRUCacheLimit = 1024
const ForkCacheLRUCacheLimit = 128

type BlockPool struct {
	blkCache       *lru.Cache //cache of full blks
	forkHeads      map[string]*common.TreeNode
	forkHeadsMutex *sync.RWMutex
}

func NewBlockPool() *BlockPool {

	pool := &BlockPool{
		forkHeads:      make(map[string]*common.TreeNode),
		forkHeadsMutex: &sync.RWMutex{},
	}
	pool.blkCache, _ = lru.New(BlockCacheLRUCacheLimit)
	return pool
}

//AddBlock adds the block to the forks and return the parent hash of the fork that contains the input block
func (pool *BlockPool) AddBlock(blk *block.Block) {

	if pool.blkCache.Contains(blk.GetHash().String()) {
		return
	}

	node, _ := common.NewTreeNode(blk)
	pool.blkCache.Add(blk.GetHash().String(), node)
	pool.updateForkHead(node)
}

//GetForkHead returns the head of the fork that contains the input block
func (pool *BlockPool) GetForkHead(blk *block.Block) *block.Block {
	if pool.blkCache == nil {
		return nil
	}

	node, _ := pool.blkCache.Get(blk.GetHash().String())

	if node == nil {
		return nil
	}

	return node.(*common.TreeNode).GetRoot().GetValue().(*block.Block)
}

func (pool *BlockPool) GetFork(parentHash hash.Hash) []*block.Block {

	root := pool.findLongestChain(parentHash)

	return getBlocksFromTrees(root.GetLongestPath())
}

func (pool *BlockPool) findLongestChain(parentHash hash.Hash) *common.TreeNode {

	longest := int64(0)
	var longestForkHead *common.TreeNode

	for _, blkHash := range pool.blkCache.Keys() {
		if cachedBlk, ok := pool.blkCache.Get(blkHash); ok {
			root := cachedBlk.(*common.TreeNode)
			if root.GetValue().(*block.Block).GetPrevHash().String() == parentHash.String() {
				if root.Height() > longest {
					longestForkHead = root
					longest = root.Height()
				}
			}
		}
	}
	return longestForkHead
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

func getBlocksFromTrees(trees []*common.TreeNode) []*block.Block {
	var blocks []*block.Block
	for _, tree := range trees {
		blocks = append(blocks, tree.GetValue().(*block.Block))
	}
	return blocks
}

// updateForkHead updates parent and Children of the tree
func (pool *BlockPool) updateForkHead(forkHead *common.TreeNode) {
	pool.linkChildren(forkHead)
	pool.linkParent(forkHead)
}

func (pool *BlockPool) linkChildren(forkHead *common.TreeNode) {
	pool.forkHeadsMutex.Lock()
	defer pool.forkHeadsMutex.Unlock()
	for _, blkHash := range pool.blkCache.Keys() {
		if cachedBlk, ok := pool.blkCache.Get(blkHash); ok {
			if cachedBlk.(*common.TreeNode).GetValue().(*block.Block).GetPrevHash().String() == forkHead.GetValue().(*block.Block).GetHash().String() {
				logger.WithFields(logger.Fields{
					"tree_height":  forkHead.GetValue().(*block.Block).GetHeight(),
					"child_height": cachedBlk.(*common.TreeNode).GetValue().(*block.Block).GetHeight(),
				}).Debug("BlockPool: added a child block to the forkHead.")
				forkHead.AddChild(cachedBlk.(*common.TreeNode))
				delete(pool.forkHeads, blkHash.(string))
			}
		}
	}
}

func (pool *BlockPool) linkParent(forkHead *common.TreeNode) {

	pool.forkHeadsMutex.Lock()
	defer pool.forkHeadsMutex.Unlock()

	parentBlkKey := forkHead.GetValue().(*block.Block).GetPrevHash().String()
	forkHeadKey := forkHead.GetValue().(*block.Block).GetHash().String()

	if parent, ok := pool.blkCache.Get(parentBlkKey); ok {
		forkHead.SetParent(parent.(*common.TreeNode))
		logger.WithFields(logger.Fields{
			"tree_height":   forkHead.GetValue().(*block.Block).GetHeight(),
			"parent_height": parent.(*common.TreeNode).GetValue().(*block.Block).GetHeight(),
		}).Debug("BlockPool: added a parent block to the forkHead.")

	} else {
		pool.forkHeads[forkHeadKey] = forkHead
	}
}

func (pool *BlockPool) ForkHeadRange(fn func(blkHash string, tree *common.TreeNode)) {
	pool.forkHeadsMutex.RLock()
	defer pool.forkHeadsMutex.RUnlock()
	for k, v := range pool.forkHeads {
		fn(k, v)
	}
}
