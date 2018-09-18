// Copyright (C) 2018 go-dappley authors
//
// This file is part of the go-dappley library.
//
// the go-dappley library is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
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
	"github.com/hashicorp/golang-lru"
	"github.com/libp2p/go-libp2p-peer"
	logger "github.com/sirupsen/logrus"
	"github.com/dappley/go-dappley/common"
)
const BlockPoolLRUCacheLimit = 128

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
	size           int
	bc             *Blockchain
	forkPool       []*Block
	blkCache       *lru.Cache //cache of full blks
	nodeCache       *lru.Cache //cache of tree nodes that contain blk header as value

}


func NewBlockPool(size int) *BlockPool {
	pool := &BlockPool{
		size:           size,
		blockRequestCh: make(chan BlockRequestPars, size),
		bc:             nil,
		forkPool:       []*Block{},
	}
	pool.blkCache,_ = lru.New(BlockPoolLRUCacheLimit)
	pool.nodeCache,_ = lru.New(BlockPoolLRUCacheLimit)

	return pool
}

func (pool *BlockPool) SetBlockchain(bc *Blockchain) {
	pool.bc = bc
}

func (pool *BlockPool) BlockRequestCh() chan BlockRequestPars {
	return pool.blockRequestCh
}

func (pool *BlockPool) GetForkPool() []*Block { return pool.forkPool }

func (pool *BlockPool) ForkPoolLen() int {
	return len(pool.forkPool)
}

func (pool *BlockPool) GetForkPoolHeadBlk() *Block {
	if len(pool.forkPool) > 0 {
		return pool.forkPool[len(pool.forkPool)-1]
	}
	return nil
}

func (pool *BlockPool) GetForkPoolTailBlk() *Block {
	if len(pool.forkPool) > 0 {
		return pool.forkPool[0]
	}
	return nil
}

func (pool *BlockPool) ResetForkPool() {
	pool.forkPool = []*Block{}
}

func (pool *BlockPool) ReInitializeForkPool(blk *Block) {
	logger.Debug("Fork: Re-initilaize fork with the new block")
	pool.ResetForkPool()
	pool.forkPool = append(pool.forkPool, blk)
}

func (pool *BlockPool) IsParentOfFork(blk *Block) bool {
	if blk == nil || pool.ForkPoolLen() == 0 {
		return false
	}

	return blk.IsParentBlock(pool.GetForkPoolHeadBlk())
}

func (pool *BlockPool) IsTailOfFork(blk *Block) bool {
	if blk == nil || pool.ForkPoolLen() == 0 {
		return false
	}

	return pool.GetForkPoolTailBlk().IsParentBlock(blk)
}

func (pool *BlockPool) GetBlockchain() *Blockchain {
	return pool.bc
}

//Verify all transactions in a fork
func (pool *BlockPool) VerifyTransactions(utxo UTXOIndex) bool {
	for i := pool.ForkPoolLen() - 1; i >= 0; i-- {
		logger.Info("Start Verify")
		if !pool.forkPool[i].VerifyTransactions(utxo) {
			return false
		}
		logger.Info("Verifyed a block. Height: ", pool.forkPool[i].GetHeight(), "Have ", i, "block left")
		utxoIndex := LoadUTXOIndex(pool.bc.GetDb())
		utxoIndex.Update(pool.forkPool[i], pool.bc.GetDb())
	}
	return true
}

func (pool *BlockPool) updateForkFromTail(blk *Block) bool {

	isTail := pool.IsTailOfFork(blk)
	if isTail {
		//only update if the block is higher than the current blockchain
		if pool.bc.IsHigherThanBlockchain(blk) {
			logger.Debug("BlockPool: Add block to tail")
			pool.addTailToForkPool(blk)
		} else {
			//if the fork's max height is less than the blockchain, delete the fork
			logger.Debug("BlockPool: Fork height too low. Dump the fork...")
			pool.ResetForkPool()
		}
	}
	return isTail
}

//returns if the operation is successful
func (pool *BlockPool) addParentToFork(blk *Block) bool {

	isParent := pool.IsParentOfFork(blk)
	if isParent {
		//check if fork's max height is still higher than the blockchain
		if pool.GetForkPoolTailBlk().GetHeight() > pool.bc.GetMaxHeight() {
			logger.Debug("BlockPool: Add block to head")
			pool.addParentToForkPool(blk)
		} else {
			//if the fork's max height is less than the blockchain, delete the fork
			logger.Debug("BlockPool: Fork height too low. Dump the fork...")
			pool.ResetForkPool()
		}
	}
	return isParent
}

func (pool *BlockPool) IsHigherThanFork(block *Block) bool {
	if block == nil {
		return false
	}
	tailBlk := pool.GetForkPoolTailBlk()
	if tailBlk == nil {
		return true
	}
	return block.GetHeight() > tailBlk.GetHeight()
}

func (pool *BlockPool) Push(block *Block, pid peer.ID) {
	logger.Debug("BlockPool: Has received a new block")

	if !block.VerifyHash() {
		logger.Info("BlockPool: Verify Hash failed!")
		return
	}

	if !(pool.bc.GetConsensus().VerifyBlock(block)) {
		logger.Warn("GetBlockPool: Verify Signature failed!")
		return
	}
	//TODO: Verify double spending transactions in the same block

	logger.Debug("BlockPool: Block has been verified")
	pool.handleRecvdBlock(block, pid)
}
func (pool *BlockPool) handleRecvdBlock(blk *Block, sender peer.ID)  {
	logger.Debug("BlockPool: Received a new block: ", blk.hashString(), " From Sender: ", sender.String())
	node,_ := pool.bc.forkTree.NewNode(blk.hashString(), blk.header, blk.header.height)

	blkCache := pool.blkCache
	nodeCache := pool.nodeCache
	if blkCache.Contains(blk.hashString()){
		logger.Debug("BlockPool: BlockPool blkCache already contains blk: ", blk.hashString(), " returning")
		return
	}
	if pool.bc.IsInBlockchain(blk.GetHash()){
		logger.Debug("BlockPool: Blockchain already contains blk: ", blk.hashString(), " returning")
		return
	}

	//TODO: verify
	if   true {
		logger.Debug("BlockPool: Adding node key to bpcache: ", blk.hashString())
		nodeCache.Add(node.GetKey(), node)
		blkCache.Add(blk.hashString(), blk)
	}else{
		logger.Debug("BlockPool: Block: ", blk.hashString(), " did not pass verification process, discarding block")
		return
	}

	//build partial tree in bpcache
	forkParent := pool.updatePoolNodeCache(node)
	//attach above partial tree to forktree
	if ok := pool.updateForkTree(forkParent, sender); ok == true {
		//build forkchain based on highest leaf in tree
		pool.updateForkPool()
		//merge forkchain into blockchain
		pool.bc.MergeFork()
	}

}


func (pool *BlockPool) updatePoolNodeCache(node *common.Node) *common.Node {
	// try to link children
	nodeCache:= pool.nodeCache
	for _,key := range nodeCache.Keys() {
		if possibleChild,ok:= nodeCache.Get(key); ok == true {
			if possibleChild.(*common.Node).Parent == node{
				logger.Debug("BlockPool: Block: ", node.GetKey(), " found child Block: ", key, " in BlockPool blkCache, adding child")
				node.AddChild(possibleChild.(*common.Node))
			}
		}
	}
	//link parents and ancestors
	for {
		if parent,ok:= nodeCache.Get(string(node.GetValue().(*BlockHeader).prevHash)); ok == true {
			logger.Debug("BlockPool: Block: ", node.GetKey(), " found parent: ", node.GetValue().(*BlockHeader).prevHash, " in BlockPool blkCache, adding parent")
			//parent found in blkCache
			node.AddParent(parent.(*common.Node))
			node = parent.(*common.Node)
		}else{
			logger.Debug("BlockPool: Block: ", node.GetKey(), " no more parent found in BlockPool blkCache")
			break
		}
	}

	logger.Debug("BlockPool: Block: ", node.GetKey(), " finished updating BlockPoolCache")
	return node
}

func (pool *BlockPool) updateForkTree(node *common.Node, sender peer.ID) bool {

	bc := pool.bc
	tree := bc.forkTree

	// parent exists on tree, add node to tree
	prevhash := node.GetValue().(*BlockHeader).prevHash
	if tree.Get(tree.Root, string(prevhash)); tree.Found != nil {
		logger.Debug("BlockPool: Block: ", node.GetKey(), " being added as child to parent ", node.Parent.GetKey())
		tree.Found.AddChild(node)
		pool.nodeCache.Remove(node)
		return true
	}else{
		// parent doesnt exist on tree, download parent from sender
		logger.Debug("BlockPool: Block: ", node.GetKey(), " parent not found, proceeding to download parent: ", node.GetValue().(*BlockHeader).prevHash, " from ", sender)
		pool.requestBlock(node.GetValue().(*BlockHeader).prevHash, sender)
		return false
	}
}

func (pool *BlockPool) updateForkPool() {
	tree := pool.bc.forkTree
	forkTailNode := tree.HighestLeaf
	bc := pool.bc
	tree.Get(tree.Root, string(bc.GetTailBlockHash()))
	bcTailInTree := tree.Found
	tree.FindCommonParent(forkTailNode, bcTailInTree)
	forkParentHash :=tree.Found.GetValue().(*BlockHeader).hash

	if tree.Found != nil {
		pool.ResetForkPool()
		for  {
			if IsHashEqual(forkTailNode.GetValue().(*BlockHeader).hash , forkParentHash ){
				break
			}
			blk := pool.getBlkFromBlkCache(string(forkTailNode.GetValue().(BlockHeader).hash))
			pool.forkPool = append(pool.forkPool, blk)
			forkTailNode = forkTailNode.Parent
		}
	}
}

func (pool *BlockPool) getBlkFromBlkCache(hashString string) *Block {
	if val,ok:= pool.blkCache.Get(hashString); ok == true {
		return val.(*Block)
	}
	return nil

}


//TODO: RequestChannel should be in PoW.go
func (pool *BlockPool) requestBlock(hash Hash, pid peer.ID) {
	pool.blockRequestCh <- BlockRequestPars{hash, pid}
}

func (pool *BlockPool) addTailToForkPool(blk *Block) {
	pool.forkPool = append([]*Block{blk}, pool.forkPool...)
}

func (pool *BlockPool) addParentToForkPool(blk *Block) {
	pool.forkPool = append(pool.forkPool, blk)
}
