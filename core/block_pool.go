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

type BlockPool struct {
	blockReceivedCh chan *Block
	size            int
	exitCh          chan bool
}

func NewBlockPool(size int) (*BlockPool) {
	pool := &BlockPool{
		size:            size,
		blockReceivedCh: make(chan *Block, size),
	}
	return pool
}

func (pool *BlockPool) BlockReceivedCh() chan *Block {
	return pool.blockReceivedCh
}

func (pool *BlockPool) Push(block *Block) {
	pool.blockReceivedCh <- block
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