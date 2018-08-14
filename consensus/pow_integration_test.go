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

	"time"

	"github.com/dappley/go-dappley/client"
	"github.com/dappley/go-dappley/core"
	"github.com/dappley/go-dappley/storage"
	"github.com/dappley/go-dappley/util"
	"github.com/stretchr/testify/assert"
	logger "github.com/sirupsen/logrus"
	"fmt"
	"os"
	"github.com/dappley/go-dappley/network"
)

var sendAmount = int(7)
var sendAmount2 = int(6)
var mineReward = int(10)
var tip = int64(5)


func TestMain(m *testing.M){

	logger.SetLevel(logger.WarnLevel)
	retCode := m.Run()
	os.Exit(retCode)
}


//mine multiple transactions
func TestMiner_SingleValidTx(t *testing.T) {

	//create new wallet
	wallets, err := client.NewWallets()
	assert.Nil(t, err)
	assert.NotNil(t, wallets)

	wallet1 := wallets.CreateWallet()
	assert.NotNil(t, wallet1)

	wallet2 := wallets.CreateWallet()
	assert.NotNil(t, wallet2)

	wallet := wallets.GetKeyPairByAddress(wallet1.GetAddress())

	//create a blockchain
	db := storage.NewRamStorage()
	defer db.Close()

	pow:= NewProofOfWork()
	bc:= core.CreateBlockchain(wallet1.GetAddress(), db, pow)
	assert.NotNil(t, bc)

	//create a transaction
	tx, err := core.NewUTXOTransaction(db,wallet1.GetAddress(), wallet2.GetAddress(), sendAmount, wallet, bc, 0)
	assert.Nil(t, err)

	//push the transaction to transaction pool
	core.GetTxnPoolInstance().Push(tx)

	//start a miner
	pow.Setup(network.NewNode(bc), wallet1.GetAddress().Address)
	pow.Start()
	
	//Make sure there are blocks have been mined
	count := GetNumberOfBlocks(t, bc.Iterator())
	for count < 2 {
		time.Sleep(time.Millisecond*500)
		count = GetNumberOfBlocks(t, bc.Iterator())
	}
	pow.Stop()

	//get the number of blocks
	count = GetNumberOfBlocks(t, bc.Iterator())
	//set the expected wallet value for all wallets
	var expectedVal = map[core.Address]int{
		wallet1.GetAddress()	:mineReward*count-sendAmount,  	//balance should be all mining rewards minus sendAmount
		wallet2.GetAddress()	:sendAmount,					//balance should be the amount rcved from wallet1
	}

	//check balance
	checkBalance(t,bc, expectedVal)
}

//mine empty blocks
func TestMiner_MineEmptyBlock(t *testing.T) {

	//create new wallet
	wallets, _ := client.NewWallets()
	assert.NotNil(t, wallets)

	cbWallet := wallets.CreateWallet()
	assert.NotNil(t, cbWallet)

	//Create Blockchain
	db := storage.NewRamStorage()
	defer db.Close()

	pow := NewProofOfWork()
	bc := core.CreateBlockchain(cbWallet.GetAddress(), db, pow)
	assert.NotNil(t, bc)

	//start a miner
	pow.Setup(network.NewNode(bc), cbWallet.GetAddress().Address)
	pow.Start()

	//Make sure at least 5 blocks mined
	count := GetNumberOfBlocks(t, bc.Iterator())
	for count < 5 {
		count = GetNumberOfBlocks(t, bc.Iterator())
		time.Sleep(time.Second)
	}
	pow.Stop()

	count = GetNumberOfBlocks(t, bc.Iterator())

	//set expected mining rewarded
	var expectedVal = map[core.Address]int{
		cbWallet.GetAddress()	: count * mineReward,
	}

	//check balance
	checkBalance(t,bc, expectedVal)

}

//mine multiple transactions
func TestMiner_MultipleValidTx(t *testing.T) {

	//create new wallet
	wallets, err := client.NewWallets()
	assert.Nil(t, err)
	assert.NotNil(t, wallets)

	wallet1 := wallets.CreateWallet()
	assert.NotNil(t, wallet1)

	wallet2 := wallets.CreateWallet()
	assert.NotNil(t, wallet2)

	wallet := wallets.GetKeyPairByAddress(wallet1.GetAddress())

	//create a blockchain
	db := storage.NewRamStorage()
	defer db.Close()

	pow := NewProofOfWork()
	bc := core.CreateBlockchain(wallet1.GetAddress(), db, pow)
	assert.NotNil(t, bc)

	//create a transaction
	tx, err := core.NewUTXOTransaction(db, wallet1.GetAddress(), wallet2.GetAddress(), sendAmount, wallet, bc, 0)
	assert.Nil(t, err)

	//push the transaction to transaction pool
	core.GetTxnPoolInstance().Push(tx)

	//start a miner
	pow.Setup(network.NewNode(bc), wallet1.GetAddress().Address)
	pow.Start()

	//Make sure there are blocks have been mined
	count := GetNumberOfBlocks(t, bc.Iterator())
	for count < 5 {
		time.Sleep(time.Millisecond*500)
		count = GetNumberOfBlocks(t, bc.Iterator())
	}

	//add second transation
	tx2, err := core.NewUTXOTransaction(db, wallet1.GetAddress(), wallet2.GetAddress(), sendAmount2, wallet, bc, 0)
	assert.Nil(t, err)

	core.GetTxnPoolInstance().Push(tx2)

	//Make sure there are blocks have been mined
	currCount := GetNumberOfBlocks(t, bc.Iterator())

	for count < currCount + 2 {
		time.Sleep(time.Millisecond*500)
		count = GetNumberOfBlocks(t, bc.Iterator())
	}

	//stop mining
	pow.Stop()

	//get the number of blocks
	count = GetNumberOfBlocks(t, bc.Iterator())
	//set the expected wallet value for all wallets
	var expectedVal = map[core.Address]int{
		wallet1.GetAddress()	:mineReward*(count+1)-sendAmount-sendAmount2,  	//balance should be all mining rewards minus sendAmount
		wallet2.GetAddress()	:sendAmount+sendAmount2,					//balance should be the amount rcved from wallet1
	}

	//check balance
	checkBalance(t,bc, expectedVal)

}

func TestProofOfWork_StartAndStop(t *testing.T) {

	pow := NewProofOfWork()
	cbAddr := core.Address{"121yKAXeG4cw6uaGCBYjWk9yTWmMkhcoDD"}
	bc := core.CreateBlockchain(
		cbAddr,
		storage.NewRamStorage(),
		pow,
	)
	defer bc.DB.Close()

	pow.Setup(network.NewNode(bc),cbAddr.Address)

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

func GetNumberOfBlocks(t *testing.T, i *core.Blockchain) int{
	//find how many blocks have been mined
	numOfBlocksMined := 0
	blk, err := i.Next()
	assert.Nil(t, err)
	for blk!=nil {
		numOfBlocksMined++
		blk, err = i.Next()
	}
	return numOfBlocksMined
}

//TODO: test mining with invalid transactions
func TestMiner_InvalidTransactions(t *testing.T) {

}

func printBalances(bc *core.Blockchain, addrs []core.Address) {
	for _, addr := range addrs{
		b, _ := getBalance(bc, addr.Address)
		fmt.Println("addr", addr, ":", b)
	}
}

//balance
func getBalance(bc *core.Blockchain, addr string) (int, error) {

	balance := 0
	pubKeyHash := util.Base58Decode([]byte(addr))
	pubKeyHash = pubKeyHash[1 : len(pubKeyHash)-4]
	UTXOs, err := bc.FindUTXO(pubKeyHash)
	if err != nil {
		return 0, err
	}

	for _, out := range UTXOs {
		balance += out.Value
	}
	return balance, nil
}

//balance
func getBalancePrint(bc *core.Blockchain, addr string) (int, error) {

	balance := 0
	pubKeyHash := util.Base58Decode([]byte(addr))
	pubKeyHash = pubKeyHash[1 : len(pubKeyHash)-4]
	UTXOs, err := bc.FindUTXO(pubKeyHash)
	if err != nil {
		return 0, err
	}
	fmt.Println(UTXOs)

	for _, out := range UTXOs {
		balance += out.Value
	}
	return balance, nil
}

func checkBalance(t *testing.T, bc *core.Blockchain, addrBals map[core.Address]int) {
	for addr, bal := range addrBals{
		b, err := getBalance(bc, addr.Address)
		assert.Nil(t, err)
		assert.Equal(t, bal, b)
	}
}