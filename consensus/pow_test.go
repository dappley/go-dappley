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
	"github.com/dappley/go-dappley/client"
	"github.com/libp2p/go-libp2p-peer"
	"github.com/sirupsen/logrus"
)

func TestProofOfWork_NewPoW(t *testing.T){
	pow := NewProofOfWork()
	assert.Equal(t,"",pow.cbAddr)
	assert.Equal(t,false,pow.newBlkRcvd)
	assert.Equal(t,prepareBlockState, pow.nextState)
}

func TestProofOfWork_ValidateDifficulty(t *testing.T) {
	cbAddr := core.Address{"121yKAXeG4cw6uaGCBYjWk9yTWmMkhcoDD"}
	bc := core.CreateBlockchain(
		cbAddr,
		storage.NewRamStorage(),
	)
	defer bc.DB.Close()

	pow := NewProofOfWork()
	pow.Setup(bc,cbAddr.Address)

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
	bc := core.CreateBlockchain(
		cbAddr,
		storage.NewRamStorage(),
	)
	defer bc.DB.Close()
	pow := NewProofOfWork()
	pow.Setup(bc,cbAddr.Address)

	//start the pow process and wait for at least 1 block produced
	pow.Start()
	blkHeight := uint64(0)
	loop:
		for{
			blk,err := bc.GetTailBlock()
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
	blk,err := bc.GetTailBlock()
	assert.Nil(t,err)
	assert.Equal(t,blkHeight,blk.GetHeight())

	//it should be able to start again
	pow.Start()
	time.Sleep(time.Second)
	pow.Stop()
}

func TestProofOfWork_ReceiveBlockFromPeers(t *testing.T) {
	logrus.SetLevel(logrus.WarnLevel)
	cbAddr := core.Address{"121yKAXeG4cw6uaGCBYjWk9yTWmMkhcoDD"}
	bc := core.CreateBlockchain(
		cbAddr,
		storage.NewRamStorage(),
	)
	defer bc.DB.Close()
	pow := NewProofOfWork()
	pow.Setup(bc,cbAddr.Address)

	//start the pow process and wait for at least 1 block produced
	pow.Start()
	blkHeight := uint64(0)
	loop:
	for{
		blk,err := bc.GetTailBlock()
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


	//push the prepared block to block pool
	bc.BlockPool().Push(newBlock,peer.ID("1"))

	//start mining
	pow.Start()
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
	tailBlock,err := bc.GetTailBlock()
	assert.Nil(t,err)

	for tailBlock.GetHeight() > 3 {
		tailBlock,err = bc.GetBlockByHash(tailBlock.GetPrevHash())
		assert.Nil(t,err)
	}

	assert.Equal(t, newBlock,tailBlock)

}

func TestProofOfWork_verifyNonce(t *testing.T){
	cbAddr := core.Address{"121yKAXeG4cw6uaGCBYjWk9yTWmMkhcoDD"}
	bc := core.CreateBlockchain(
		cbAddr,
		storage.NewRamStorage(),
	)
	defer bc.DB.Close()
	pow := NewProofOfWork()
	pow.Setup(bc,cbAddr.Address)

	//prepare a block with correct nonce value
	newBlock := pow.prepareBlock()
	nonce := int64(0)
	mineloop2:
	for{
		if hash, ok := pow.verifyNonce(nonce, newBlock); ok {
			newBlock.SetHash(hash)
			newBlock.SetNonce(nonce)
			break mineloop2
		}else{
			nonce++
		}
	}

	//check if the verifyNonce function returns true
	_, ok := pow.verifyNonce(nonce, newBlock)
	assert.True(t, ok)

	//input a wrong nonce value, check if it returns false
	_, ok = pow.verifyNonce(nonce-1, newBlock)
	assert.False(t, ok)
}

func TestProofOfWork_verifyTransactions(t *testing.T){
	wallets,err := client.NewWallets()
	assert.Nil(t, err)
	wallet1 := wallets.CreateWallet()
	wallet2 := wallets.CreateWallet()

	bc := core.CreateBlockchain(
		wallet1.GetAddress(),
		storage.NewRamStorage(),
	)
	defer bc.DB.Close()

	pow := NewProofOfWork()
	pow.Setup(bc,wallet1.GetAddress().Address)

	//mock two transactions and push them to transaction pool
	//the first transaction is a valid transaction
	tx1, err := core.NewUTXOTransaction(
		bc.DB,
		wallet1.GetAddress(),
		wallet2.GetAddress(),
		5,
		wallets.GetKeyPairByAddress(wallet1.GetAddress()),
		bc,
		0)

	//the second transaction is not a valid transaction
	tx2 := *core.MockTransaction()
	//push the transactions to the transaction pool
	txPool := core.GetTxnPoolInstance()
	txPool.Push(tx1)
	txPool.Push(tx2)

	//verify the transactions
	pow.verifyTransactions()

	//the second transaction should be removed
	assert.Equal(t, 1, txPool.Len())
	//the remaining transaction should be the first one (the valid transaction)
	assert.Equal(t, tx1, txPool.Pop())
}


/*func TestProofOfWork_testMiningSpeed(t *testing.T){
	cbAddr := core.Address{"121yKAXeG4cw6uaGCBYjWk9yTWmMkhcoDD"}
	bc,err := core.CreateBlockchain(
		cbAddr,
		storage.NewRamStorage(),
	)
	defer bc.DB.Close()
	assert.Nil(t,err)
	pow := NewProofOfWork(bc,cbAddr.Address)

	//mine 10 blocks and calculate average time
	for i:=14;i < 20;i++ {
		pow.SetTargetBit(i)
		pow.Start()
		startTime := time.Now()
		targetHeight := uint64((i-14)*10 +9)
	loop:
		for {
			blk, err := bc.GetTailBlock()
			assert.Nil(t, err)
			if blk.GetHeight() > targetHeight {
				break loop
			}
		}
		fmt.Println("The average time for difficulty level",
			i,
			"is",
			time.Now().Sub(startTime).Seconds(),
			"seconds")
		pow.Stop()
	}
}*/