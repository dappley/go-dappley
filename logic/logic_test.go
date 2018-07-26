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
	"github.com/sirupsen/logrus"
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
	pow := consensus.NewProofOfWork()
	pow.Setup(b,addr1.Address)
	miner := consensus.NewMiner(pow)

	go miner.Start()
	time.Sleep(3 * time.Second)

	//send function creates utxo results in 1 mineReward, adding unto the blockchain creation is 3*mineReward
	balance1, err = GetBalance(wallet1.GetAddress(), databaseInstance)

	assert.Nil(t, err)
	assert.True(t, 2*mineReward-transferAmount < balance1)

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
const testport = 10100

func TestSyncBlocks(t *testing.T){
	logrus.SetLevel(logrus.WarnLevel)
	var pows []*consensus.ProofOfWork
	var bcs []*core.Blockchain
	addr := core.Address{"17DgRtQVvaytkiKAfXx9XbV23MESASSwUz"}
	//wait for mining for at least "targetHeight" blocks
	targetHeight := uint64(4)
	//num of nodes to be created in the test
	numOfNodes := 4
	for i := 0; i < numOfNodes; i++{
		//create storage instance
		db := storage.NewRamStorage()
		defer db.Close()

		//create blockchain instance
		bc,err := core.CreateBlockchain(addr,db)
		assert.Nil(t, err)
		bcs = append(bcs, bc)

		pow := consensus.NewProofOfWork()
		pow.Setup(bcs[i],addr.Address)
		pow.SetTargetBit(16)
		pow.GetNode().Start(testport+i)

		if i != 0 {
			pow.GetNode().AddStream(
				pows[0].GetNode().GetPeerID(),
				pows[0].GetNode().GetPeerMultiaddr(),
				)
		}

		pows = append(pows, pow)
	}

	//seed node broadcasts syncpeers
	pows[0].GetNode().SyncPeers()

	//wait for 2 seconds for syncing
	time.Sleep(time.Second*2)

	//count and is Stopped tracks the num of nodes that have been stopped
	count := 0
	isStopped := []bool{}
	blkHeight := []uint64{}
	//Start Mining and set average block time to 15 seconds (difficulty = 16)
	for i := 0; i < numOfNodes; i++{
		pows[i].Start()
		isStopped = append(isStopped, false)
		blkHeight = append(blkHeight, 0)
	}

	loop:
		for {
			for i := 0; i < numOfNodes; i++ {
				blk, err := bcs[i].GetTailBlock()
				assert.Nil(t, err)
				if blk.GetHeight() > blkHeight[i] {
					blkHeight[i]++
					logrus.Info("BlkHeight:",blkHeight[i], " Node:", pows[i].GetNode().GetPeerMultiaddr())
				}
				if blk.GetHeight() > targetHeight {
					//count the number of nodes that have already stopped mining
					if isStopped[i]==false{
						//stop the first miner that reaches the target height
						pows[i].Stop()
						isStopped[i] = true
						count++
					}
				}
				//break the loop if all miners stop
				if count >= numOfNodes {
					break loop
				}
			}
		}

	//Check if all nodes have the same tail block
	for i := 0; i < numOfNodes-1; i++{
		blk0, err := bcs[i].GetTailBlock()
		assert.Nil(t,err)
		blk1, err := bcs[i+1].GetTailBlock()
		assert.Nil(t,err)
		assert.True(t, pows[i].ValidateDifficulty(blk0))
		assert.True(t, pows[i+1].ValidateDifficulty(blk1))
		assert.Equal(t,blk0.GetHash(),blk1.GetHash())
	}

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
