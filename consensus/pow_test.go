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

package consensus

import (
	"testing"
	"github.com/dappley/go-dappley/core"
	"math/big"
	"github.com/stretchr/testify/assert"
	"github.com/dappley/go-dappley/storage"
	"time"
)

func TestProofOfWork_ValidateDifficulty(t *testing.T) {
	cbAddr := core.Address{"121yKAXeG4cw6uaGCBYjWk9yTWmMkhcoDD"}
	bc,err := core.CreateBlockchain(
		cbAddr,
		storage.NewRamStorage(),
	)
	defer bc.DB.Close()
	assert.Nil(t,err)
	pow := NewProofOfWork(bc,cbAddr.Address)

	//create a block that has a hash value larger than the target
	blk := core.GenerateMockBlock()
	target := big.NewInt(1)
	target.Lsh(target, uint(256-targetBits+1))

	blk.SetHash(target.Bytes())

	assert.False(t,pow.ValidateDifficulty(blk))

	//create a block that has a hash value smaller than the target
	target = big.NewInt(1)
	target.Lsh(target, uint(256-targetBits-1))
	blk.SetHash(target.Bytes())

	assert.True(t,pow.ValidateDifficulty(blk))
}

func TestProofOfWork_StartAndStop(t *testing.T) {
	cbAddr := core.Address{"121yKAXeG4cw6uaGCBYjWk9yTWmMkhcoDD"}
	bc,err := core.CreateBlockchain(
		cbAddr,
		storage.NewRamStorage(),
	)
	defer bc.DB.Close()
	assert.Nil(t,err)
	pow := NewProofOfWork(bc,cbAddr.Address)

	//start the pow process and wait for at least 1 block produced
	pow.Start()
	blkHeight := uint64(0)
	loop:
		for{
			blk,err := bc.GetLastBlock()
			assert.Nil(t,err)
			blkHeight = blk.GetHeight()
			if blkHeight > 1 {
				break loop
			}
		}

	//stop pow process and wait
	pow.Stop()
	time.Sleep(time.Second*2)

	//there should be not block produced anymore
	blk,err := bc.GetLastBlock()
	assert.Nil(t,err)
	assert.Equal(t,blkHeight,blk.GetHeight())

	//it should be able to start again
	pow.Start()
	time.Sleep(time.Second)
	pow.Stop()
}

func TestProofOfWork_ReceiveBlockFromPeers(t *testing.T) {
	cbAddr := core.Address{"121yKAXeG4cw6uaGCBYjWk9yTWmMkhcoDD"}
	bc,err := core.CreateBlockchain(
		cbAddr,
		storage.NewRamStorage(),
	)
	defer bc.DB.Close()
	assert.Nil(t,err)
	pow := NewProofOfWork(bc,cbAddr.Address)

	//start the pow process and wait for at least 1 block produced
	pow.Start()
	blkHeight := uint64(0)
	loop:
	for{
		blk,err := bc.GetLastBlock()
		assert.Nil(t,err)
		blkHeight = blk.GetHeight()
		if blkHeight > 1 {
			break loop
		}
	}
	//stop pow process
	pow.Stop()

	//prepare a new block
	newBlock := pow.prepareBlock()
	nonce := int64(0)
	mineloop:
		for{
			if hash, ok := pow.verifyNonce(nonce, newBlock); ok {
				newBlock.SetHash(hash)
				newBlock.SetNonce(nonce)
				break mineloop
			}else{
				nonce++
			}
		}

	//start mining
	pow.Start()
	//push the prepared block to block pool
	bc.BlockPool().Push(newBlock)
	//the pow loop should stop current mining and go to updateNewBlockState. Wait until that happens
	loop1:
	for {
		time.Sleep(time.Microsecond)
		if pow.nextState == updateNewBlockState {
			break loop1
		}
	}
	//Wait until the loop updates the new block to the blockchain. Stop the loop after that happens
	loop2:
	for {
		time.Sleep(time.Microsecond)
		if pow.nextState != updateNewBlockState {
			pow.Stop()
			break loop2
		}
	}

	//the tail block should be the block that we have pushed into blockpool
	tailBlock,err := bc.GetLastBlock()

	assert.Nil(t,err)
	assert.Equal(t, newBlock,tailBlock)

}
