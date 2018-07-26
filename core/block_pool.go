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
	exitCh          chan bool
	bc              *Blockchain
	forkPool        []*Block
}

func NewBlockPool(size int, bc *Blockchain) (*BlockPool) {
	pool := &BlockPool{
		size:            size,
		blockReceivedCh: make(chan *RcvedBlock, size),
		blockRequestCh:  make(chan BlockRequestPars, 1),
		exitCh:          make(chan bool, 1),
		bc:              bc,
		forkPool:        []*Block{},
	}
	bc.blockPool = pool
	return pool
}

func (pool *BlockPool) BlockReceivedCh() chan *RcvedBlock {
	return pool.blockReceivedCh
}

func (pool *BlockPool) BlockRequestCh() chan BlockRequestPars {
	return pool.blockRequestCh
}

func (pool *BlockPool) AddParentToForkPool(blk *Block)  {
	pool.forkPool = append(pool.forkPool, blk)
}

func (pool *BlockPool) ForkPoolLen() int{
	return len(pool.forkPool)
}

func (pool *BlockPool) GetForkPoolHeadBlk() *Block{
	if len(pool.forkPool) > 0 {
		return pool.forkPool[len(pool.forkPool)-1]
	}else{
		return nil
	}
}

func (pool *BlockPool) GetForkPoolTailBlk() *Block{
	if len(pool.forkPool) > 0 {
		return pool.forkPool[0]
	}else{
		return nil
	}
}

func (pool *BlockPool) ResetForkPool() {
	pool.forkPool = []*Block{}
}

func (pool *BlockPool) IsParentOfFork(blk *Block) bool{
	if blk == nil {
		return false
	}

	if pool.ForkPoolLen() == 0 {
		return true
	}

	return VerifyParentBlock(blk, pool.GetForkPoolHeadBlk())
}

func (pool *BlockPool) IsTailOfFork(blk *Block) bool{
	if blk == nil {
		return false
	}

	if pool.ForkPoolLen() == 0 {
		return true
	}

	return VerifyParentBlock(pool.GetForkPoolTailBlk(), blk)
}

func (pool *BlockPool) AddTailToForkPool(blk *Block){
	pool.forkPool = append([]*Block{blk}, pool.forkPool...)
}

func (pool *BlockPool) Push(block *Block, pid peer.ID) {
	logger.Debug("BlockPool: Has received a new block")

	if !block.VerifyHash() {
		logger.Info("BlockPool: Verify Hash failed!")
		return
	}

	//TODO: Temporarily disable verify transaction since it only verifies transactions against it own transaction pool
/*	if !block.VerifyTransactions(pool.bc){
		logger.Info("BlockPool: Verify Transactions failed!")
		return
	}*/

	logger.Debug("BlockPool: Block has been verified")
	pool.blockReceivedCh <- &RcvedBlock{block,pid}
}

func (pool *BlockPool) RequestBlock(hash Hash, pid peer.ID){
	pool.blockRequestCh <- BlockRequestPars{hash, pid}
}

func (pool *BlockPool) Start() {
	go pool.messageLoop()
}

func (pool *BlockPool) Stop() {
	pool.exitCh <- true
}

func (pool *BlockPool) messageLoop() {
	for {
		select {
		case <-pool.exitCh:
			logger.Info("BlockPool Exited")
			return
		}
	}
}

func (pool *BlockPool) GetBlockchain() *Blockchain{
	return pool.bc
}