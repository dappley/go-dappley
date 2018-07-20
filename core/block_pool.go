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

import "fmt"
import (
	"reflect"
	logger "github.com/sirupsen/logrus"
)


type BlockPool struct {
	blockReceivedCh chan *Block
	size            int
	exitCh          chan bool
	bc 				*Blockchain
}

func NewBlockPool(size int, bc *Blockchain) (*BlockPool) {
	pool := &BlockPool{
		size:            size,
		blockReceivedCh: make(chan *Block, size),
		bc:				 bc,
	}
	return pool
}

func (pool *BlockPool) BlockReceivedCh() chan *Block {
	return pool.blockReceivedCh
}

func (pool *BlockPool) Push(block *Block) {
	lastBlk,err := pool.bc.GetLastBlock()
	if err!=nil {
		logger.Warn(err)
	}
	if verifyBlock(lastBlk, block){
		pool.blockReceivedCh <- block
	}
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
			fmt.Println("quit block pool")
			return
		case blk := <-pool.blockReceivedCh:
			pool.handleBlock(blk)
		}
	}
}

func (pool *BlockPool) handleBlock(blk *Block) {
	pool.Push(blk)
}

func verifyHeight(lastBlk, newblk *Block) bool{
	return lastBlk.height + 1 == newblk.height
}

func verifyLastBlockHash(lastBlk, newblk *Block) bool{
	return reflect.DeepEqual(lastBlk.GetHash(), newblk.GetPrevHash())
}

func verifyBlock(lastBlk, newblk *Block) bool{
	if newblk.VerifyHash()==false{
		return false
	}

	if verifyHeight(lastBlk, newblk)==false{
		return false
	}

	if verifyLastBlockHash(lastBlk, newblk)==false{
		return false
	}

	return true
}

func (pool *BlockPool) GetBlockchain() *Blockchain{
	return pool.bc
}