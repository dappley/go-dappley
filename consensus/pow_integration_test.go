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
	"testing"

	"github.com/dappley/go-dappley/common"

	"os"

	"time"

	"github.com/dappley/go-dappley/client"
	"github.com/dappley/go-dappley/core"
	"github.com/dappley/go-dappley/network"
	"github.com/dappley/go-dappley/storage"
	logger "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

var sendAmount = common.NewAmount(7)
var sendAmount2 = common.NewAmount(6)
var mineReward = common.NewAmount(10)

func TestMain(m *testing.M) {

	logger.SetLevel(logger.WarnLevel)
	retCode := m.Run()
	os.Exit(retCode)
}

//mine multiple transactions
func TestMiner_SingleValidTx(t *testing.T) {

	//create new wallet
	wallets := &client.WalletManager{}

	wallet1 := client.NewWallet()
	wallet2 := client.NewWallet()
	wallets.AddWallet(wallet1)
	wallets.AddWallet(wallet2)

	keyPair := wallets.GetKeyPairByAddress(wallet1.GetAddress())

	//create a blockchain
	db := storage.NewRamStorage()
	defer db.Close()

	pow := NewProofOfWork()
	bc := core.CreateBlockchain(wallet1.GetAddress(), db, pow, 128)
	assert.NotNil(t, bc)

	//create a transaction
	tx, err := core.NewUTXOTransaction(core.LoadUTXOIndex(bc.GetDb()), wallet1.GetAddress(), wallet2.GetAddress(), sendAmount, *keyPair, 0)
	assert.Nil(t, err)

	//push the transaction to transaction pool
	bc.GetTxPool().Push(tx)

	//start a miner
	n := network.FakeNodeWithPidAndAddr(bc, "asd", "test")
	pow.Setup(n, wallet1.GetAddress().Address)

	pow.Start()

	//Make sure there are blocks have been mined
	count := GetNumberOfBlocks(t, bc.Iterator())
	for count < 2 {
		count = GetNumberOfBlocks(t, bc.Iterator())
	}
	pow.Stop()
	core.WaitFullyStop(pow, 20)
	time.Sleep(time.Second)

	//get the number of blocks
	count = GetNumberOfBlocks(t, bc.Iterator())
	//set the expected wallet value for all wallets
	remaining, err := mineReward.Times(uint64(count)).Sub(sendAmount)
	if err != nil {
		panic(err)
	}
	var expectedVal = map[core.Address]*common.Amount{
		wallet1.GetAddress(): remaining,  //balance should be all mining rewards minus sendAmount
		wallet2.GetAddress(): sendAmount, //balance should be the amount rcved from wallet1
	}

	//check balance
	checkBalance(t, bc, expectedVal)
}

//mine empty blocks
func TestMiner_MineEmptyBlock(t *testing.T) {

	//create new wallet
	walletManager := &client.WalletManager{}

	wallet := client.NewWallet()
	walletManager.AddWallet(wallet)
	assert.NotNil(t, wallet)

	//Create Blockchain
	db := storage.NewRamStorage()
	defer db.Close()

	pow := NewProofOfWork()
	bc := core.CreateBlockchain(wallet.GetAddress(), db, pow, 128)
	assert.NotNil(t, bc)

	//start a miner

	n := network.FakeNodeWithPidAndAddr(bc, "asd", "asd")
	pow.Setup(n, wallet.GetAddress().Address)
	pow.Start()

	//Make sure at least 5 blocks mined
	count := GetNumberOfBlocks(t, bc.Iterator())
	for count < 5 {
		count = GetNumberOfBlocks(t, bc.Iterator())
	}
	pow.Stop()
	core.WaitFullyStop(pow, 20)
	time.Sleep(time.Second)

	count = GetNumberOfBlocks(t, bc.Iterator())

	//set expected mining rewarded
	var expectedVal = map[core.Address]*common.Amount{
		wallet.GetAddress(): mineReward.Times(uint64(count)),
	}

	//check balance
	checkBalance(t, bc, expectedVal)
}

//mine multiple transactions
func TestMiner_MultipleValidTx(t *testing.T) {

	//create new wallet
	wallets := &client.WalletManager{}

	wallet1 := client.NewWallet()
	wallet2 := client.NewWallet()
	wallets.AddWallet(wallet1)
	wallets.AddWallet(wallet2)

	keyPair := wallets.GetKeyPairByAddress(wallet1.GetAddress())

	//create a blockchain
	db := storage.NewRamStorage()
	defer db.Close()

	pow := NewProofOfWork()
	bc := core.CreateBlockchain(wallet1.GetAddress(), db, pow, 128)
	assert.NotNil(t, bc)
	//create a transaction
	tx, err := core.NewUTXOTransaction(core.LoadUTXOIndex(bc.GetDb()), wallet1.GetAddress(), wallet2.GetAddress(), sendAmount, *keyPair, 0)
	assert.Nil(t, err)

	//push the transaction to transaction pool
	bc.GetTxPool().Push(tx)

	//start a producer
	n := network.FakeNodeWithPidAndAddr(bc, "asd", "asd")
	pow.Setup(n, wallet1.GetAddress().Address)
	pow.Start()

	//Make sure there are blocks have been mined
	count := GetNumberOfBlocks(t, bc.Iterator())
	for count < 5 {
		count = GetNumberOfBlocks(t, bc.Iterator())
	}

	//add second transaction
	tx2, err := core.NewUTXOTransaction(core.LoadUTXOIndex(bc.GetDb()), wallet1.GetAddress(), wallet2.GetAddress(), sendAmount2, *keyPair, 0)
	assert.Nil(t, err)

	bc.GetTxPool().Push(tx2)

	//Make sure there are blocks have been mined
	currCount := GetNumberOfBlocks(t, bc.Iterator())

	for count < currCount+2 {
		count = GetNumberOfBlocks(t, bc.Iterator())
	}

	//stop mining
	pow.Stop()
	core.WaitFullyStop(pow, 20)
	time.Sleep(time.Second)

	//get the number of blocks
	count = GetNumberOfBlocks(t, bc.Iterator())
	//set the expected wallet value for all wallets
	remaining, err := mineReward.Times(uint64(count)).Sub(sendAmount.Add(sendAmount2))
	var expectedVal = map[core.Address]*common.Amount{
		wallet1.GetAddress(): remaining,                   //balance should be all mining rewards minus sendAmount
		wallet2.GetAddress(): sendAmount.Add(sendAmount2), //balance should be the amount rcved from wallet1
	}

	//check balance
	checkBalance(t, bc, expectedVal)
}

func TestProofOfWork_StartAndStop(t *testing.T) {

	pow := NewProofOfWork()
	cbAddr := core.Address{"121yKAXeG4cw6uaGCBYjWk9yTWmMkhcoDD"}
	bc := core.CreateBlockchain(
		cbAddr,
		storage.NewRamStorage(),
		pow,
		128,
	)
	defer bc.GetDb().Close()
	n := network.FakeNodeWithPidAndAddr(bc, "asd", "asd")
	pow.Setup(n, cbAddr.Address)
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
	core.WaitFullyStop(pow, 20)
	//there should be not block produced anymore
	blk, err := bc.GetTailBlock()
	assert.Nil(t, err)
	assert.Equal(t, blkHeight, blk.GetHeight())

	//it should be able to start again
	pow.Start()
	pow.Stop()
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
func TestMiner_InvalidTransactions(t *testing.T) {

}

func printBalances(bc *core.Blockchain, addrs []core.Address) {
	for _, addr := range addrs {
		b, _ := getBalance(bc, addr.Address)
		logger.Debug("addr", addr, ":", b)
	}
}

//balance
func getBalance(bc *core.Blockchain, addr string) (*common.Amount, error) {

	balance := common.NewAmount(0)
	pubKeyHash := core.HashAddress(addr)
	utxoIndex := core.LoadUTXOIndex(bc.GetDb())
	utxos := utxoIndex.GetUTXOsByPubKeyHash(pubKeyHash)
	for _, out := range utxos {
		balance = balance.Add(out.Value)
	}
	return balance, nil
}

func checkBalance(t *testing.T, bc *core.Blockchain, addrBals map[core.Address]*common.Amount) {
	for addr, bal := range addrBals {
		bc, err := getBalance(bc, addr.Address)
		assert.Nil(t, err)
		assert.Equal(t, bal, bc)
	}
}
