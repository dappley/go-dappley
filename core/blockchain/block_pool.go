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

package blockchain

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
	root           *common.TreeNode
	orphans        map[string]*common.TreeNode
	forkHeadsMutex *sync.RWMutex
}

func NewBlockPool(rootBlk *block.Block) *BlockPool {
	var node *common.TreeNode

	if rootBlk != nil {
		node, _ = common.NewTreeNode(rootBlk)
	}

	pool := &BlockPool{
		root:           node,
		orphans:        make(map[string]*common.TreeNode),
		forkHeadsMutex: &sync.RWMutex{},
	}
	pool.blkCache, _ = lru.New(BlockCacheLRUCacheLimit)

	if rootBlk != nil {
		pool.blkCache.Add(getKey(node), node)
	}

	return pool
}

//AddBlock adds the block to the forks and return the parent hash of the fork that contains the input block
func (pool *BlockPool) AddBlock(blk *block.Block) {

	if blk == nil {
		return
	}

	if pool.blkCache.Contains(blk.GetHash().String()) {
		return
	}

	if !pool.isBlockValid(blk) {
		return
	}

	node, _ := common.NewTreeNode(blk)
	pool.blkCache.Add(getKey(node), node)
	pool.link(node)
}

//GetForkHead returns the head of the fork that contains the input block
func (pool *BlockPool) GetForkHead(blk *block.Block) *block.Block {

	node, _ := pool.blkCache.Get(blk.GetHash().String())

	if node == nil {
		return nil
	}

	return node.(*common.TreeNode).GetRoot().GetValue().(*block.Block)
}

//SetRootBlock updates the last irreversible block
func (pool *BlockPool) SetRootBlock(rootBlk *block.Block) {

	value, _ := pool.blkCache.Get(rootBlk.GetHash().String())

	if value == nil {
		if pool.root != nil {
			pool.removeTree(pool.root)
		}
		pool.AddBlock(rootBlk)
		pool.root, _ = common.NewTreeNode(rootBlk)
		return
	}

	pool.forkHeadsMutex.Lock()
	defer pool.forkHeadsMutex.Unlock()

	//update orphans
	newRoot := value.(*common.TreeNode)
	rootKey := getKey(newRoot.GetRoot())
	if _, isOrphan := pool.orphans[rootKey]; isOrphan {
		delete(pool.orphans, rootKey)
		pool.removeTree(pool.root)
	}

	//update tree
	newRoot.Prune(pool.removeNode)

	//update root
	pool.root = newRoot

	//update orphan forks
	pool.pruneOrphans()
}

//pruneOrphans remove invalid orphans according to the root block height
func (pool *BlockPool) pruneOrphans() {
	rootBlkHash := pool.root.GetValue().(*block.Block).GetHash()
	for key, orphanTreeHead := range pool.orphans {

		orphanBlk := orphanTreeHead.GetValue().(*block.Block)

		if orphanBlk.GetPrevHash().Equals(rootBlkHash) {
			pool.root.AddChild(orphanTreeHead)
			delete(pool.orphans, key)
			continue
		}

		if !pool.isBlockValid(orphanBlk) {
			pool.removeTree(orphanTreeHead)
			delete(pool.orphans, key)
		}
	}
}

//isBlockValid returns if the block pool will accept the block
func (pool *BlockPool) isBlockValid(blk *block.Block) bool {
	if blk == nil {
		return false
	}

	if pool.root == nil {
		return true
	}

	rootBlk := pool.getRootBlk()

	if blk.GetPrevHash().Equals(rootBlk.GetHash()) {
		return true
	}

	return blk.GetHeight() > rootBlk.GetHeight()+1
}

//removeTree removes all nodes under root
func (pool *BlockPool) removeTree(root *common.TreeNode) {
	root.RemoveAllDescendants(pool.removeNode)
	pool.removeNode(root)
}

//removeNode removes the node from the cache
func (pool *BlockPool) removeNode(node *common.TreeNode) {
	pool.blkCache.Remove(getKey(node))
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
			if root.GetValue().(*block.Block).GetHash().String() == parentHash.String() {
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

	delete(pool.orphans, fork[0].GetHash().String())
}

func getBlocksFromTrees(trees []*common.TreeNode) []*block.Block {
	var blocks []*block.Block
	for _, tree := range trees {
		blocks = append(blocks, tree.GetValue().(*block.Block))
	}
	return blocks
}

// link updates parent and Children of the tree
func (pool *BlockPool) link(node *common.TreeNode) {
	pool.linkOrphan(node)
	pool.linkParent(node)
}

func (pool *BlockPool) linkOrphan(node *common.TreeNode) {
	pool.forkHeadsMutex.Lock()
	defer pool.forkHeadsMutex.Unlock()
	for _, root := range pool.orphans {
		if root.GetValue().(*block.Block).GetPrevHash().String() == node.GetValue().(*block.Block).GetHash().String() {
			logger.WithFields(logger.Fields{
				"tree_height":  node.GetValue().(*block.Block).GetHeight(),
				"child_height": root.GetValue().(*block.Block).GetHeight(),
			}).Debug("BlockPool: added an orphan to the node.")
			node.AddChild(root)
			delete(pool.orphans, getKey(root))
		}
	}
}

func (pool *BlockPool) linkParent(node *common.TreeNode) {

	pool.forkHeadsMutex.Lock()
	defer pool.forkHeadsMutex.Unlock()

	if parent, ok := pool.blkCache.Get(node.GetValue().(*block.Block).GetPrevHash().String()); ok {
		node.SetParent(parent.(*common.TreeNode))
		logger.WithFields(logger.Fields{
			"tree_height":   node.GetValue().(*block.Block).GetHeight(),
			"parent_height": parent.(*common.TreeNode).GetValue().(*block.Block).GetHeight(),
		}).Debug("BlockPool: added a parent block to the node.")

	} else {
		pool.orphans[getKey(node)] = node
	}
}

func (pool *BlockPool) ForkHeadRange(fn func(blkHash string, tree *common.TreeNode)) {
	pool.forkHeadsMutex.RLock()
	defer pool.forkHeadsMutex.RUnlock()
	for k, v := range pool.orphans {
		fn(k, v)
	}
}

func (pool *BlockPool) getRootBlk() *block.Block {
	if pool.root == nil {
		return nil
	}

	return pool.root.GetValue().(*block.Block)
}

//getKey gets the key that is used to store in orphans map
func getKey(node *common.TreeNode) string {
	if node == nil {
		return ""
	}

	if node.GetValue() == nil {
		return ""
	}

	return node.GetValue().(*block.Block).GetHash().String()
}

func (pool *BlockPool) printInfo() {
	logger.Info("********Block Pool Summary**********")
	logger.WithFields(logger.Fields{
		"num_of_nodes": pool.blkCache.Len(),
	}).Info("Basic Info: BlockPool")
	logger.WithFields(logger.Fields{
		"root_blk_hash":            pool.root.GetValue().(*block.Block).GetHash(),
		"root_blk_height":          pool.root.GetValue().(*block.Block).GetHeight(),
		"is_root":                  pool.root.Parent == nil,
		"num_of_root_blk_children": len(pool.root.Children),
		"height":                   pool.root.Height(),
		"num_of_leaves":            pool.root.NumLeaves(),
		"num_of_nodes":             pool.root.Size(),
	}).Info("Basic Info: Main Fork")
	logger.WithFields(logger.Fields{
		"num_of_orphan_forks": len(pool.orphans),
	}).Info("Basic Info: Orphans")
	for key, orphan := range pool.orphans {
		logger.WithFields(logger.Fields{
			"orphan_root_hash":   orphan.GetValue().(*block.Block).GetHash(),
			"orphan_fork_height": orphan.GetValue().(*block.Block).GetHeight(),
			"is_orphan":          orphan.Parent == nil,
			"height":             orphan.Height(),
			"num_of_leaves":      orphan.NumLeaves(),
			"num_of_nodes":       orphan.Size(),
		}).Info("Basic Info: Orphan ", key)
	}

	logger.Info("************************************")
}
