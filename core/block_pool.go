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
	"github.com/dappley/go-dappley/common"
	"github.com/hashicorp/golang-lru"
	"github.com/libp2p/go-libp2p-peer"
	logger "github.com/sirupsen/logrus"
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
	blockchain     *Blockchain
	blkCache       *lru.Cache //cache of full blks
	nodeCache      *lru.Cache //cache of tree nodes that contain blk header as value

}


func NewBlockPool(size int) *BlockPool {
	pool := &BlockPool{
		size:           size,
		blockRequestCh: make(chan BlockRequestPars, size),
		blockchain:     nil,
	}
	pool.blkCache,_ = lru.New(BlockPoolLRUCacheLimit)
	pool.nodeCache,_ = lru.New(BlockPoolLRUCacheLimit)

	return pool
}

func (pool *BlockPool) SetBlockchain(bc *Blockchain) {
	pool.blockchain = bc
}

func (pool *BlockPool) BlockRequestCh() chan BlockRequestPars {
	return pool.blockRequestCh
}

func (pool *BlockPool) GetBlockchain() *Blockchain {
	return pool.blockchain
}

//Verify all transactions in a fork
func (pool *BlockPool) VerifyTransactions(utxo UTXOIndex, forkBlks []*Block) bool {
	for i := len(forkBlks) - 1; i >= 0; i-- {
		logger.Info("Start Verify")
		if !forkBlks[i].VerifyTransactions(utxo) {
			return false
		}
		logger.Info("Verifyed a block. Height: ", forkBlks[i].GetHeight(), "Have ", i, "block left")
		utxoIndex := LoadUTXOIndex(pool.blockchain.GetDb())
		utxoIndex.BuildForkUtxoIndex(forkBlks[i], pool.blockchain.GetDb())
	}
	return true
}


func (pool *BlockPool) Push(block *Block, pid peer.ID) {
	logger.Debug("BlockPool: Has received a new block")

	if !block.VerifyHash() {
		logger.Debug("BlockPool: Verify Hash failed!")
		return
	}

	if !(pool.blockchain.GetConsensus().VerifyBlock(block)) {
		logger.Debug("BlockPool: Verify Signature failed!")
		return
	}
	//TODO: Verify double spending transactions in the same block

	logger.Debug("BlockPool: Block has been verified")
	pool.handleRecvdBlock(block, pid)
}

func (pool *BlockPool) handleRecvdBlock(blk *Block, sender peer.ID)  {
	logger.Debug("BlockPool: Received a new block: ", blk.GetHash(), " From Sender: ", sender.String())
	node,_ := pool.blockchain.forkTree.NewNode(blk.hashString(), blk.header, blk.header.height)

	blkCache := pool.blkCache
	nodeCache := pool.nodeCache

	if   pool.blockchain.consensus.Validate(blk) {
		if blkCache.Contains(blk.hashString()){
			logger.Debug("BlockPool: BlockPool blkCache already contains blk: ", blk.GetHash(), " returning")
			return
		}else{
			logger.Debug("BlockPool: Adding blk key to blockcache: ", blk.GetHash())
			blkCache.Add(blk.hashString(), blk)
		}

		if pool.blockchain.IsInBlockchain(blk.GetHash()){
			logger.Debug("BlockPool: Blockchain already contains blk: ", blk.GetHash(), " returning")
			return
		}else{
			logger.Debug("BlockPool: Adding node key to nodecache: ", blk.GetHash())
			nodeCache.Add(node.GetKey(), node)
		}
	}else{
		logger.Debug("BlockPool: Block: ", blk.GetHash(), " did not pass consensus validation, discarding block")
		return
	}

	bcTailBlk , err := pool.GetBlockchain().GetTailBlock()
	if err != nil{
		nodeCache.Remove(node.GetKey())
		blkCache.Remove(blk.hashString())
		return
	}

	if bcTailBlk.IsParentBlock(blk){
		pool.blockchain.AddBlockToBlockchainTail(blk)
		return
	}
	//build partial tree in bpcache
	forkParent := pool.updatePoolNodeCache(node)
	//attach above partial tree to forktree
	if ok := pool.updateForkTree(forkParent, sender); ok == true {
		//build forkchain based on highest leaf in tree
		forkBlks := pool.getForkBlks()
		//merge forkchain into blockchain
		pool.blockchain.MergeFork(forkBlks)
	}

}


func (pool *BlockPool) updatePoolNodeCache(node *common.Node) *common.Node {
	// try to link children
	nodeCache:= pool.nodeCache
	for _,key := range nodeCache.Keys() {
		if possibleChild,ok:= nodeCache.Get(key); ok == true {
			if possibleChild.(*common.Node).Parent == node{
				logger.Debug("BlockPool: Block: ", node.GetValue().(*BlockHeader).hash, " found child Block: ", possibleChild.(*common.Node).GetValue().(*BlockHeader).hash, " in BlockPool blkCache, adding child")
				node.AddChild(possibleChild.(*common.Node))
			}
		}
	}
	//link parent
	if parent,ok:= nodeCache.Get(string(node.GetValue().(*BlockHeader).prevHash)); ok == true {
		logger.Debug("BlockPool: Block: ", node.GetValue().(*BlockHeader).hash, " found parent: ", node.GetValue().(*BlockHeader).prevHash, " in BlockPool blkCache, adding parent")
		//parent found in blkCache
		node.AddParent(parent.(*common.Node))
		node = parent.(*common.Node)
	}else{
		logger.Debug("BlockPool: Block: ",  node.GetValue().(*BlockHeader).hash, " no more parent found in BlockPool blkCache")
	}


	logger.Debug("BlockPool: Block: ", node.GetValue().(*BlockHeader).hash, " finished updating BlockPoolCache")
	return node
}

func (pool *BlockPool) updateForkTree(node *common.Node, sender peer.ID) bool {

	bc := pool.blockchain
	tree := bc.forkTree

	// parent exists on tree, add node to tree
	prevhash := node.GetValue().(*BlockHeader).prevHash

	if tree.Get(tree.Root, string(prevhash)); tree.Found != nil {
		logger.Debug("BlockPool: Block: ", node.GetValue().(*BlockHeader).hash, " being added as child to parent ", node.Parent.GetValue().(*BlockHeader).hash)
		tree.Found.AddChild(node)
		pool.nodeCache.Remove(node)
		return true
	}else{
		// parent doesnt exist on tree, download parent from sender
		logger.Debug("BlockPool: Block: ", node.GetValue().(*BlockHeader).hash, " parent not found, proceeding to download parent: ", node.GetValue().(*BlockHeader).prevHash, " from ", sender)
		pool.requestBlock(node.GetValue().(*BlockHeader).prevHash, sender)
		return false
	}
}

func (pool *BlockPool) getForkBlks() []*Block {
	var forkblks []*Block
	tree := pool.blockchain.forkTree
	forkTailNode := tree.HighestLeaf
	bc := pool.blockchain
	tree.Get(tree.Root, string(bc.GetTailBlockHash()))
	bcTailInTree := tree.Found
	tree.FindCommonParent(forkTailNode, bcTailInTree)
	forkParentHash :=tree.Found.GetValue().(*BlockHeader).hash

	if tree.Found != nil {
		logger.Debug("Blockpool: no common parent found between fork tail and blockchain tail")
		for  {
			if IsHashEqual(forkTailNode.GetValue().(*BlockHeader).hash , forkParentHash ){
				break
			}
			blk := pool.getBlkFromBlkCache(string(forkTailNode.GetValue().(BlockHeader).hash))
			forkblks = append(forkblks, blk)
			forkTailNode = forkTailNode.Parent
		}
	}
	return forkblks
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

