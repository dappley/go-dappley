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

package logic

import (
	"errors"
	"os"
	"testing"

	"fmt"
	"time"

	"github.com/dappley/go-dappley/client"
	"github.com/dappley/go-dappley/consensus"
	"github.com/dappley/go-dappley/core"
	"github.com/dappley/go-dappley/storage"
	"github.com/stretchr/testify/assert"
)

const invalidAddress = "Invalid Address"

var databaseInstance *storage.LevelDB

func TestMain(m *testing.M) {
	setup()
	databaseInstance = storage.OpenDatabase(core.BlockchainDbFile)
	defer databaseInstance.Close()
	retCode := m.Run()
	os.Exit(retCode)
}

func TestCreateWallet(t *testing.T) {
	wallet, err := CreateWallet()
	assert.Nil(t, err)
	assert.NotEmpty(t, wallet)
}

func TestCreateBlockchain(t *testing.T) {
	//create a wallet address
	wallet, err := CreateWallet()
	assert.NotEmpty(t, wallet)
	addr := wallet.GetAddress()

	//create a blockchain
	b, err := CreateBlockchain(addr, databaseInstance)
	assert.Nil(t, err)
	assert.NotNil(t, b)
}

//create a blockchain with invalid address
func TestCreateBlockchainWithInvalidAddress(t *testing.T) {
	//create a blockchain with an invalid address
	b, err := CreateBlockchain(core.NewAddress(invalidAddress), databaseInstance)
	assert.Equal(t, err, ErrInvalidAddress)
	assert.Nil(t, b)
}

func TestGetBalance(t *testing.T) {
	//create a wallet address
	wallet, err := CreateWallet()
	assert.NotEmpty(t, wallet)
	addr := wallet.GetAddress()

	//create a blockchain
	b, err := CreateBlockchain(addr, databaseInstance)
	assert.Nil(t, err)
	assert.NotNil(t, b)

	//The balance should be 10 after creating a blockchain
	balance, err := GetBalance(addr, databaseInstance)
	assert.Nil(t, err)
	assert.Equal(t, balance, 10)
}

func TestGetBalanceWithInvalidAddress(t *testing.T) {
	//create a wallet address
	wallet, err := CreateWallet()
	assert.NotEmpty(t, wallet)
	addr := wallet.GetAddress()

	//create a blockchain
	b, err := CreateBlockchain(addr, databaseInstance)
	assert.Nil(t, err)
	assert.NotNil(t, b)

	//The balance should be 10 after creating a blockchain
	balance1, err := GetBalance(core.NewAddress("1AUrNJCRM5X5fDdmm3E3yjCrXQMLvDj9tb"), databaseInstance)
	assert.Nil(t, err)
	assert.Equal(t, balance1, 0)

	balance2, err := GetBalance(core.NewAddress("1AUrNJCRM5X5fDdmm3E3yjCrXQMLwfwfww"), databaseInstance)
	assert.Equal(t, errors.New("ERROR: Address is invalid"), err)
	assert.Equal(t, balance2, 0)
}

func TestGetAllAddresses(t *testing.T) {
	setup()
	expected_res := []core.Address{}
	//create a wallet address
	wallet, err := CreateWallet()
	assert.NotEmpty(t, wallet)
	addr := wallet.GetAddress()

	expected_res = append(expected_res, addr)

	//create a blockchain
	b, err := CreateBlockchain(addr, databaseInstance)
	assert.Nil(t, err)
	assert.NotNil(t, b)

	//create 10 more addresses
	for i := 0; i < 10; i++ {
		//create a wallet address
		wallet, err = CreateWallet()
		addr = wallet.GetAddress()
		assert.NotEmpty(t, addr)
		assert.Nil(t, err)
		expected_res = append(expected_res, addr)
	}

	//get all addresses
	addrs, err := GetAllAddresses()
	assert.Nil(t, err)
	assert.NotNil(t, addrs)

	//the length should be equal
	assert.Equal(t, len(expected_res), len(addrs))
	assert.ElementsMatch(t, expected_res, addrs)
	teardown()
}

//test send
func TestSend(t *testing.T) {
	//setup: clean up database and files
	setup()
	mineReward := int(10)
	transferAmount := int(5)
	tip := int64(5)
	//create a wallet address
	wallet1, err := CreateWallet()
	assert.NotEmpty(t, wallet1)

	//create a blockchain
	b, err := CreateBlockchain(wallet1.GetAddress(), databaseInstance)
	assert.Nil(t, err)
	assert.NotNil(t, b)

	//The balance1 should be 10 after creating a blockchain
	balance1, err := GetBalance(wallet1.GetAddress(), databaseInstance)
	assert.Nil(t, err)
	assert.Equal(t, mineReward, balance1)
	fmt.Println(balance1)
	//Create a second wallet
	wallet2, err := CreateWallet()
	assert.NotEmpty(t, wallet2)
	assert.Nil(t, err)
	addr1 := wallet1.GetAddress()
	addr2 := wallet2.GetAddress()

	//The balance2 should be 0
	balance2, err := GetBalance(addr2, databaseInstance)
	assert.Nil(t, err)
	assert.Equal(t, balance2, 0)
	fmt.Println(balance2)

	//Send 5 coins from wallet1 to wallet2
	err = Send(addr1, addr2, transferAmount, tip, databaseInstance)
	assert.Nil(t, err)
	miner := consensus.NewMiner(b, addr1.Address, consensus.NewProofOfWork(b))
	go miner.Start()
	for i := 0; i < 3; i++ {
		miner.Feed(time.Now().String())
		time.Sleep(1 * time.Second)
	}
	assert.Nil(t, err)
	//send function creates utxo results in 1 mineReward, adding unto the blockchain creation is 3*mineReward
	balance1, err = GetBalance(wallet1.GetAddress(), databaseInstance)

	assert.Nil(t, err)
	assert.Equal(t, 2*mineReward-transferAmount, balance1)

	//the balance1 of the second wallet should be 5
	balance2, err = GetBalance(wallet2.GetAddress(), databaseInstance)
	assert.Nil(t, err)
	assert.Equal(t, transferAmount, balance2)

	miner.Stop()
	//teardown :clean up database amd files
	teardown()
}

func TestDeleteWallets(t *testing.T) {
	//create wallets address
	addr1, err := CreateWallet()
	assert.NotEmpty(t, addr1)

	addr2, err := CreateWallet()
	assert.NotEmpty(t, addr2)

	addr3, err := CreateWallet()
	assert.NotEmpty(t, addr3)

	err = DeleteWallets()
	assert.Nil(t, err)

	list, err := GetAllAddresses()
	assert.Nil(t, err)
	assert.Empty(t, list)
}

//test send to invalid address
func TestSendToInvalidAddress(t *testing.T) {
	//setup: clean up database and files
	setup()
	//this is internally set. Dont modify
	mineReward := int(10)
	//Transfer ammount
	transferAmount := int(25)
	tip := int64(5)
	//create a wallet address
	wallet1, err := CreateWallet()
	assert.NotEmpty(t, wallet1)
	addr1 := wallet1.GetAddress()

	//create a blockchain
	b, err := CreateBlockchain(addr1, databaseInstance)
	assert.Nil(t, err)
	assert.NotNil(t, b)

	//The balance should be 10 after creating a blockchain
	balance1, err := GetBalance(addr1, databaseInstance)
	assert.Nil(t, err)
	assert.Equal(t, balance1, mineReward)

	//Send 5 coins from addr1 to an invalid address
	err = Send(addr1, core.NewAddress(invalidAddress), transferAmount, tip, databaseInstance)
	assert.NotNil(t, err)

	//the balance of the first wallet should be still be 10
	balance1, err = GetBalance(addr1, databaseInstance)
	assert.Nil(t, err)
	assert.Equal(t, balance1, mineReward)
	//teardown :clean up database amd files
	teardown()
}

func TestDeleteInvalidWallet(t *testing.T) {
	//setup: clean up database and files
	setup()
	//create wallets address
	wallet1, err := CreateWallet()
	assert.NotEmpty(t, wallet1)
	addr1 := wallet1.GetAddress()

	addressList := []core.Address{addr1}

	println(addr1.Address)

	list, err := GetAllAddresses()
	assert.Nil(t, err)
	assert.ElementsMatch(t, list, addressList)

	//teardown :clean up database amd files
	teardown()
}

//insufficient fund
func TestSendInsufficientBalance(t *testing.T) {
	//setup: clean up database and files
	setup()
	tip := int64(5)

	//this is internally set. Dont modify
	mineReward := int(10)
	//Transfer ammount is larger than the balance
	transferAmount := int(25)

	//create a wallet address
	wallet1, err := CreateWallet()
	assert.NotEmpty(t, wallet1)
	addr1 := wallet1.GetAddress()

	//create a blockchain
	b, err := CreateBlockchain(addr1, databaseInstance)
	assert.Nil(t, err)
	assert.NotNil(t, b)

	//The balance should be 10 after creating a blockchain
	balance1, err := GetBalance(addr1, databaseInstance)
	assert.Nil(t, err)
	assert.Equal(t, balance1, mineReward)

	//Create a second wallet
	wallet2, err := CreateWallet()
	assert.NotEmpty(t, wallet2)
	assert.Nil(t, err)
	addr2 := wallet2.GetAddress()

	//The balance should be 0
	balance2, err := GetBalance(addr2, databaseInstance)
	assert.Nil(t, err)
	assert.Equal(t, balance2, 0)

	//Send 5 coins from addr1 to addr2
	err = Send(addr1, addr2, transferAmount, tip, databaseInstance)
	assert.NotNil(t, err)

	//the balance of the first wallet should be still be 10
	balance1, err = GetBalance(addr1, databaseInstance)
	assert.Nil(t, err)
	assert.Equal(t, balance1, mineReward)

	//the balance of the second wallet should be 0
	balance2, err = GetBalance(addr2, databaseInstance)
	assert.Nil(t, err)
	assert.Equal(t, balance2, 0)

	//teardown :clean up database amd files
	teardown()
}

func TestProofOfWork_Start(t *testing.T) {
	//setup: clean up database and files
	setup()

	//create a wallet address
	wallet, err := CreateWallet()
	assert.NotEmpty(t, wallet)
	addr := wallet.GetAddress()

	//create a blockchain
	b, err := CreateBlockchain(addr, databaseInstance)
	assert.Nil(t, err)
	assert.NotNil(t, b)

	pow := consensus.NewProofOfWork(b)

	go pow.Start()
	for i := 0; i < 3; i++ {
		pow.Feed(time.Now().String())
		pow.Feed("test test")

		bk := core.NewBlock(core.GetTxnPoolInstance().GetSortedTransactions(), []byte{})
		bk.SetHash([]byte{123})
		b.BlockPool().Push(bk)
		time.Sleep(1 * time.Second)
	}
	pow.Stop()
}

func setup() {
	cleanUpDatabase()
}

func teardown() {
	cleanUpDatabase()
}

func cleanUpDatabase() {
	os.RemoveAll("../bin/blockchain.DB")
	os.RemoveAll(client.WalletFile)
}
