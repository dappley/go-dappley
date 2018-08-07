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
	logger "github.com/sirupsen/logrus"
	"github.com/libp2p/go-libp2p-peer"
)

type BlockRequestPars struct{
	BlockHash 	Hash
	Pid 		peer.ID
}

type RcvedBlock struct{
	Block 	*Block
	Pid 	peer.ID
}

type BlockPool struct {
	blockReceivedCh chan *RcvedBlock
	blockRequestCh  chan BlockRequestPars
	size            int
	bc              *Blockchain
	forkPool        []*Block
}

func NewBlockPool(size int) (*BlockPool) {
	pool := &BlockPool{
		size:            size,
		blockReceivedCh: make(chan *RcvedBlock, size),
		blockRequestCh:  make(chan BlockRequestPars, size),
		bc:              nil,
		forkPool:        []*Block{},
	}
	return pool
}

func (pool *BlockPool) SetBlockchain(bc *Blockchain){
	pool.bc = bc
}

func (pool *BlockPool) BlockReceivedCh() chan *RcvedBlock {
	return pool.blockReceivedCh
}

func (pool *BlockPool) BlockRequestCh() chan BlockRequestPars {
	return pool.blockRequestCh
}

func (pool *BlockPool) ForkPoolLen() int{
	return len(pool.forkPool)
}

func (pool *BlockPool) GetForkPoolHeadBlk() *Block{
	if len(pool.forkPool) > 0 {
		return pool.forkPool[len(pool.forkPool)-1]
	}
	return nil
}

func (pool *BlockPool) GetForkPoolTailBlk() *Block{
	if len(pool.forkPool) > 0 {
		return pool.forkPool[0]
	}
	return nil
}

func (pool *BlockPool) ResetForkPool() {
	pool.forkPool = []*Block{}
}

func (pool *BlockPool) ReInitializeForkPool(blk *Block){
	logger.Debug("Fork: Re-initilaize fork with the new block")
	pool.ResetForkPool()
	pool.forkPool = append(pool.forkPool, blk)
}

func (pool *BlockPool) IsParentOfFork(blk *Block) bool{
	if blk == nil || pool.ForkPoolLen() == 0 {
		return false
	}

	return IsParentBlock(blk, pool.GetForkPoolHeadBlk())
}

func (pool *BlockPool) IsTailOfFork(blk *Block) bool{
	if blk == nil || pool.ForkPoolLen() == 0 {
		return false
	}

	return IsParentBlock(pool.GetForkPoolTailBlk(), blk)
}

func (pool *BlockPool) UpdateForkFromTail(blk *Block) bool{

	isTail := pool.IsTailOfFork(blk)
	if isTail{
		//only update if the block is higher than the current blockchain
		if pool.bc.HigherThanBlockchain(blk) {
			logger.Debug("Fork: Add block to tail")
			pool.addTailToForkPool(blk)
		}else{
			//if the fork's max height is less than the blockchain, delete the fork
			logger.Debug("Fork: Fork height too low. Dump the fork...")
			pool.ResetForkPool()
		}
	}
	return isTail

}

//returns if the operation is successful
func (pool *BlockPool) AddParentToFork(blk *Block) bool{

	isParent := pool.IsParentOfFork(blk)
	if isParent{
		//check if fork's max height is still higher than the blockchain
		if pool.GetForkPoolTailBlk().GetHeight() > pool.bc.GetMaxHeight() {
			logger.Debug("Fork: Add block to head")
			pool.addParentToForkPool(blk)
		}else{
			//if the fork's max height is less than the blockchain, delete the fork
			logger.Debug("Fork: Fork height too low. Dump the fork...")
			pool.ResetForkPool()
		}
	}
	return isParent
}


func (pool *BlockPool) IsHigherThanFork(block *Block) bool{
	if block == nil {
		return false
	}
	tailBlk := pool.GetForkPoolTailBlk()
	if tailBlk == nil {
		return true
	}
	return 	block.GetHeight() > tailBlk.GetHeight()
}

func (pool *BlockPool) Push(block *Block, pid peer.ID) {
	logger.Debug("BlockPool: Has received a new block")

	if !block.VerifyHash() {
		logger.Info("BlockPool: Verify Hash failed!")
		return
	}

	//TODO: Temporarily disable verify transaction since it only verifies transactions against it own transaction pool
	utxoPool := GetStoredUtxoMap(pool.bc.DB, UtxoMapKey)
	if !block.VerifyTransactions(utxoPool){
		logger.Info("BlockPool: Verify Transactions failed!")
		return
	}

	logger.Debug("BlockPool: Block has been verified")
	pool.blockReceivedCh <- &RcvedBlock{block,pid}
}

//TODO: RequestChannel should be in PoW.go
func (pool *BlockPool) RequestBlock(hash Hash, pid peer.ID){
	pool.blockRequestCh <- BlockRequestPars{hash, pid}
}

func (pool *BlockPool) GetBlockchain() *Blockchain{
	return pool.bc
}

func (pool *BlockPool) addTailToForkPool(blk *Block){
	pool.forkPool = append([]*Block{blk}, pool.forkPool...)
}

func (pool *BlockPool) addParentToForkPool(blk *Block)  {
	pool.forkPool = append(pool.forkPool, blk)
}

//Verify all transactions in a fork
func (pool *BlockPool) VerifyTransactions(utxo utxoIndex) bool{
	for i := pool.ForkPoolLen()-1; i>=0; i--{
		if !pool.forkPool[i].VerifyTransactions(utxo){
			return false
		}
		pool.forkPool[i].UpdateUtxoIndexAfterNewBlock(UtxoMapKey, pool.bc.DB)
	}
	return true
}