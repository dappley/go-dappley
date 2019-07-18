// +build integration

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
	"os"
	"testing"
	"time"

	"github.com/dappley/go-dappley/client"
	"github.com/dappley/go-dappley/common"
	"github.com/dappley/go-dappley/core"
	"github.com/dappley/go-dappley/network"
	"github.com/dappley/go-dappley/storage"
	logger "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

var sendAmount = common.NewAmount(7)
var sendAmount2 = common.NewAmount(6)
var mineReward = common.NewAmount(10000000)

func TestMain(m *testing.M) {

	logger.SetLevel(logger.WarnLevel)
	retCode := m.Run()
	os.Exit(retCode)
}

//mine multiple transactions
func TestBlockProducer_SingleValidTx(t *testing.T) {

	//create new account
	accounts := &client.AccountManager{}

	account1 := client.NewAccount()
	account2 := client.NewAccount()
	accounts.AddAccount(account1)
	accounts.AddAccount(account2)

	keyPair := accounts.GetKeyPairByAddress(account1.GetAddress())

	//create a blockchain
	db := storage.NewRamStorage()
	defer db.Close()

	pow := NewProofOfWork()
	bc := core.CreateBlockchain(account1.GetAddress(), db, pow, 128, nil, 100000)
	assert.NotNil(t, bc)

	pubKeyHash, _ := account1.GetAddress().GetPubKeyHash()
	utxos, err := core.NewUTXOIndex(bc.GetUtxoCache()).GetUTXOsByAmount(pubKeyHash, sendAmount)
	assert.Nil(t, err)

	//create a transaction
	sendTxParam := core.NewSendTxParam(account1.GetAddress(), keyPair, account2.GetAddress(), sendAmount, common.NewAmount(0), common.NewAmount(0), common.NewAmount(0), "")
	tx, err := core.NewUTXOTransaction(utxos, sendTxParam)
	assert.Nil(t, err)

	//push the transaction to transaction pool
	bc.GetTxPool().Push(tx)

	//start a miner
	pool := core.NewBlockPool(0)
	n := network.FakeNodeWithPidAndAddr(pool, bc, "asd", "test")
	pow.Setup(n, account1.GetAddress().String())

	pow.Start()

	//Make sure there are blocks have been mined
	count := GetNumberOfBlocks(t, bc.Iterator())
	for count < 2 {
		count = GetNumberOfBlocks(t, bc.Iterator())
	}
	pow.Stop()
	core.WaitDoneOrTimeout(func() bool {
		return !pow.IsProducingBlock()
	}, 20)

	//get the number of blocks
	count = GetNumberOfBlocks(t, bc.Iterator())

	//set the expected account value for all accounts
	remaining, err := mineReward.Times(uint64(count)).Sub(sendAmount)

	if err != nil {
		panic(err)
	}
	var expectedVal = map[client.Address]*common.Amount{
		account1.GetAddress(): remaining,  //balance should be all mining rewards minus sendAmount
		account2.GetAddress(): sendAmount, //balance should be the amount rcved from account1
	}

	//check balance
	checkBalance(t, bc, expectedVal)
}

//mine empty blocks
func TestBlockProducer_MineEmptyBlock(t *testing.T) {

	//create new account
	accountManager := &client.AccountManager{}

	account := client.NewAccount()
	accountManager.AddAccount(account)
	assert.NotNil(t, account)

	//Create Blockchain
	db := storage.NewRamStorage()
	defer db.Close()

	pow := NewProofOfWork()
	bc := core.CreateBlockchain(account.GetAddress(), db, pow, 128, nil, 100000)
	assert.NotNil(t, bc)

	//start a miner
	pool := core.NewBlockPool(0)
	n := network.FakeNodeWithPidAndAddr(pool, bc, "asd", "asd")
	pow.Setup(n, account.GetAddress().String())
	pow.Start()

	//Make sure at least 5 blocks mined
	count := GetNumberOfBlocks(t, bc.Iterator())
	for count < 5 {
		count = GetNumberOfBlocks(t, bc.Iterator())
	}
	pow.Stop()
	core.WaitDoneOrTimeout(func() bool {
		return !pow.IsProducingBlock()
	}, 20)
	time.Sleep(time.Second)

	count = GetNumberOfBlocks(t, bc.Iterator())

	//set expected mining rewarded
	var expectedVal = map[client.Address]*common.Amount{
		account.GetAddress(): mineReward.Times(uint64(count)),
	}

	//check balance
	checkBalance(t, bc, expectedVal)
}

//mine multiple transactions
func TestBlockProducer_MultipleValidTx(t *testing.T) {

	//create new account
	accounts := &client.AccountManager{}

	account1 := client.NewAccount()
	account2 := client.NewAccount()
	accounts.AddAccount(account1)
	accounts.AddAccount(account2)

	keyPair := accounts.GetKeyPairByAddress(account1.GetAddress())

	//create a blockchain
	db := storage.NewRamStorage()
	defer db.Close()

	pow := NewProofOfWork()
	bc := core.CreateBlockchain(account1.GetAddress(), db, pow, 128, nil, 100000)
	assert.NotNil(t, bc)

	pubKeyHash, _ := account1.GetAddress().GetPubKeyHash()
	utxos, err := core.NewUTXOIndex(bc.GetUtxoCache()).GetUTXOsByAmount(pubKeyHash, sendAmount)
	assert.Nil(t, err)

	//create a transaction
	sendTxParam := core.NewSendTxParam(account1.GetAddress(), keyPair, account2.GetAddress(), sendAmount, common.NewAmount(0), common.NewAmount(0), common.NewAmount(0), "")
	tx, err := core.NewUTXOTransaction(utxos, sendTxParam)
	assert.Nil(t, err)

	//push the transaction to transaction pool
	bc.GetTxPool().Push(tx)

	//start a producer
	pool := core.NewBlockPool(0)
	n := network.FakeNodeWithPidAndAddr(pool, bc, "asd", "asd")
	pow.Setup(n, account1.GetAddress().String())
	pow.Start()

	//Make sure there are blocks have been mined
	count := GetNumberOfBlocks(t, bc.Iterator())
	for count < 5 {
		count = GetNumberOfBlocks(t, bc.Iterator())
	}

	utxos2, err := core.NewUTXOIndex(bc.GetUtxoCache()).GetUTXOsByAmount(pubKeyHash, sendAmount)
	assert.Nil(t, err)

	//add second transaction
	sendTxParam2 := core.NewSendTxParam(account1.GetAddress(), keyPair, account2.GetAddress(), sendAmount2, common.NewAmount(0), common.NewAmount(0), common.NewAmount(0), "")
	tx2, err := core.NewUTXOTransaction(utxos2, sendTxParam2)
	assert.Nil(t, err)

	bc.GetTxPool().Push(tx2)

	//Make sure there are blocks have been mined
	currCount := GetNumberOfBlocks(t, bc.Iterator())

	for count < currCount+2 {
		count = GetNumberOfBlocks(t, bc.Iterator())
	}

	//stop mining
	pow.Stop()
	core.WaitDoneOrTimeout(func() bool {
		return !pow.IsProducingBlock()
	}, 20)
	time.Sleep(time.Second)

	//get the number of blocks
	count = GetNumberOfBlocks(t, bc.Iterator())
	//set the expected account value for all accounts
	remaining, err := mineReward.Times(uint64(count)).Sub(sendAmount.Add(sendAmount2))
	var expectedVal = map[client.Address]*common.Amount{
		account1.GetAddress(): remaining,                   //balance should be all mining rewards minus sendAmount
		account2.GetAddress(): sendAmount.Add(sendAmount2), //balance should be the amount rcved from account1
	}

	//check balance
	checkBalance(t, bc, expectedVal)
}

func TestProofOfWork_StartAndStop(t *testing.T) {

	pow := NewProofOfWork()
	cbAddr := client.Address{"121yKAXeG4cw6uaGCBYjWk9yTWmMkhcoDD"}
	bc := core.CreateBlockchain(
		cbAddr,
		storage.NewRamStorage(),
		pow,
		128,
		nil,
		100000,
	)
	defer bc.GetDb().Close()
	pool := core.NewBlockPool(0)
	n := network.FakeNodeWithPidAndAddr(pool, bc, "asd", "asd")
	pow.Setup(n, cbAddr.String())
	pow.SetTargetBit(10)
	//start the pow process and wait for at least 1 block produced
	pow.Start()
	blkHeight := uint64(0)
loop:
	for {
		blk, err := bc.GetTailBlock()
		assert.Nil(t, err)
		blkHeight = blk.GetHeight()
		if blkHeight > 1 {
			break loop
		}
	}

	//stop pow process and wait
	pow.Stop()
	core.WaitDoneOrTimeout(func() bool {
		return !pow.IsProducingBlock()
	}, 20)
	//there should be not block produced anymore
	blk, err := bc.GetTailBlock()
	assert.Nil(t, err)
	assert.True(t, blkHeight <= blk.GetHeight())

	//it should be able to start again
	pow.Start()
	pow.Stop()
}

func TestPreventDoubleSpend(t *testing.T) {
	//create new account
	accounts := &client.AccountManager{}

	account1 := client.NewAccount()
	account2 := client.NewAccount()
	account3 := client.NewAccount()

	accounts.AddAccount(account1)
	accounts.AddAccount(account2)
	accounts.AddAccount(account3)

	sendAmount := common.NewAmount(10)
	keyPair := accounts.GetKeyPairByAddress(account1.GetAddress())

	//create a blockchain
	db := storage.NewRamStorage()
	defer db.Close()

	pow := NewProofOfWork()
	bc := core.CreateBlockchain(account1.GetAddress(), db, pow, 128, nil, 100000)
	assert.NotNil(t, bc)

	pubKeyHash, _ := account1.GetAddress().GetPubKeyHash()
	utxos, err := core.NewUTXOIndex(bc.GetUtxoCache()).GetUTXOsByAmount(pubKeyHash, sendAmount)
	assert.Nil(t, err)

	//create a transaction
	sendTxParam1 := core.NewSendTxParam(account1.GetAddress(), keyPair, account2.GetAddress(), sendAmount, common.NewAmount(0), common.NewAmount(0), common.NewAmount(0), "")
	sendTxParam2 := core.NewSendTxParam(account1.GetAddress(), keyPair, account3.GetAddress(), sendAmount, common.NewAmount(0), common.NewAmount(0), common.NewAmount(0), "")
	tx1, err := core.NewUTXOTransaction(utxos, sendTxParam1)
	tx2, err := core.NewUTXOTransaction(utxos, sendTxParam2)

	assert.Nil(t, err)

	//push the transaction to transaction pool
	bc.GetTxPool().Push(tx1)
	bc.GetTxPool().Push(tx2)

	//start a miner
	pool := core.NewBlockPool(0)
	n := network.FakeNodeWithPidAndAddr(pool, bc, "asd", "test")
	pow.Setup(n, account1.GetAddress().Address)

	pow.Start()

	//Make sure there are blocks have been mined
	count := GetNumberOfBlocks(t, bc.Iterator())
	for count < 2 {
		count = GetNumberOfBlocks(t, bc.Iterator())
	}
	pow.Stop()

	block, _ := bc.GetBlockByHeight(1)
	// Only one transaction packaged(1 coinbase + 1 transaction)
	assert.Equal(t, 2, len(block.GetTransactions()))
}

func GetNumberOfBlocks(t *testing.T, i *core.Blockchain) int {
	//find how many blocks have been mined
	numOfBlocksMined := 0
	blk, err := i.Next()
	assert.Nil(t, err)
	for blk != nil {
		numOfBlocksMined++
		blk, err = i.Next()
	}
	return numOfBlocksMined
}

//TODO: test mining with invalid transactions
func TestBlockProducer_InvalidTransactions(t *testing.T) {

}

func printBalances(bc *core.Blockchain, addrs []client.Address) {
	for _, addr := range addrs {
		b, _ := getBalance(bc, addr.String())
		logger.WithFields(logger.Fields{
			"address": addr,
			"balance": b,
		}).Debug("Printing balance...")
	}
}

//balance
func getBalance(bc *core.Blockchain, addr string) (*common.Amount, error) {

	balance := common.NewAmount(0)
	pubKeyHash, _ := client.NewAddress(addr).GetPubKeyHash()
	utxoIndex := core.NewUTXOIndex(bc.GetUtxoCache())
	utxos := utxoIndex.GetAllUTXOsByPubKeyHash(pubKeyHash)
	//_, utxo, nextUtxos := utxos.Iterator()
	for _, utxo := range utxos.Indices {
		balance = balance.Add(utxo.Value)
		//_, utxo, nextUtxos = nextUtxos.Iterator()
	}
	return balance, nil
}

func checkBalance(t *testing.T, bc *core.Blockchain, addrBals map[client.Address]*common.Amount) {
	for addr, bal := range addrBals {
		bc, err := getBalance(bc, addr.String())
		assert.Nil(t, err)
		assert.Equal(t, bal, bc)
	}
}
