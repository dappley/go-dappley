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

	"time"

	"reflect"

	"github.com/dappley/go-dappley/client"
	"github.com/dappley/go-dappley/consensus"
	"github.com/dappley/go-dappley/core"
	"github.com/dappley/go-dappley/network"
	"github.com/dappley/go-dappley/storage"
	logger "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

const invalidAddress = "Invalid Address"
const BlockchainDbFile = "../bin/blockchain.DB"

var databaseInstance *storage.LevelDB

func TestMain(m *testing.M) {
	setup()
	databaseInstance = storage.OpenDatabase(BlockchainDbFile)
	defer databaseInstance.Close()
	logger.SetLevel(logger.WarnLevel)
	retCode := m.Run()
	os.Exit(retCode)
}

func TestCreateWallet(t *testing.T) {
	wallet, err := CreateWallet()
	assert.Nil(t, err)
	assert.Equal(t, len(wallet.Addresses[0].Address), 34)
}

func TestCreateBlockchain(t *testing.T) {
	//create a wallet address
	addr := core.Address{"1G4r54VdJsotfCukXUWmg1ZRnhjUs6TvbV"}

	//create a blockchain
	_, err := CreateBlockchain(addr, databaseInstance, nil)
	assert.Nil(t, err)
}

//create a blockchain with invalid address
func TestCreateBlockchainWithInvalidAddress(t *testing.T) {
	//create a blockchain with an invalid address
	b, err := CreateBlockchain(core.NewAddress(invalidAddress), databaseInstance, nil)
	assert.Equal(t, err, ErrInvalidAddress)
	assert.Nil(t, b)
}

func TestGetBalance(t *testing.T) {
	//create a wallet address
	addr := core.Address{"1G4r54VdJsotfCukXUWmg1ZRnhjUs6TvbV"}
	//create a blockchain
	b, err := CreateBlockchain(addr, databaseInstance, nil)
	assert.Nil(t, err)
	assert.NotNil(t, b)

	//The balance should be 10 after creating a blockchain
	balance, err := GetBalance(addr, databaseInstance)
	assert.Nil(t, err)
	assert.Equal(t, balance, 10)
}

func TestGetBalanceWithInvalidAddress(t *testing.T) {
	//create a wallet address
	addr := core.Address{"1G4r54VdJsotfCukXUWmg1ZRnhjUs6TvbV"}
	//create a blockchain
	b, err := CreateBlockchain(addr, databaseInstance, nil)
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
	b, err := CreateBlockchain(addr, databaseInstance, nil)
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
	tip := uint64(5)
	//create a wallet address
	wallet1, err := CreateWallet()
	assert.NotEmpty(t, wallet1)

	pow := consensus.NewProofOfWork()
	//create a blockchain
	b, err := CreateBlockchain(wallet1.GetAddress(), databaseInstance, pow)
	assert.Nil(t, err)
	assert.NotNil(t, b)
	node := network.NewNode(b)

	//The balance1 should be 10 after creating a blockchain
	balance1, err := GetBalance(wallet1.GetAddress(), databaseInstance)
	assert.Nil(t, err)
	assert.Equal(t, mineReward, balance1)

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

	//Send 5 coins from wallet1 to wallet2
	err = Send(wallet1, addr2, transferAmount, tip, b)
	assert.Nil(t, err)
	pow.Setup(node, addr1.Address)

	pow.Start()
	time.Sleep(3 * time.Second)

	//send function creates utxo results in 1 mineReward, adding unto the blockchain creation is 3*mineReward
	balance1, err = GetBalance(wallet1.GetAddress(), databaseInstance)

	assert.Nil(t, err)
	assert.True(t, 2*mineReward-transferAmount < balance1)

	//the balance1 of the second wallet should be 5
	balance2, err = GetBalance(wallet2.GetAddress(), databaseInstance)
	assert.Nil(t, err)
	assert.Equal(t, transferAmount, balance2)

	pow.Stop()

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
	tip := uint64(5)
	//create a wallet address
	wallet1, err := CreateWallet()
	assert.NotEmpty(t, wallet1)
	addr1 := wallet1.GetAddress()

	//create a blockchain
	b, err := CreateBlockchain(addr1, databaseInstance, nil)
	assert.Nil(t, err)
	assert.NotNil(t, b)

	//The balance should be 10 after creating a blockchain
	balance1, err := GetBalance(addr1, databaseInstance)
	assert.Nil(t, err)
	assert.Equal(t, balance1, mineReward)

	//Send 5 coins from addr1 to an invalid address
	err = Send(wallet1, core.NewAddress(invalidAddress), transferAmount, tip, b)
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
	tip := uint64(5)

	//this is internally set. Dont modify
	mineReward := int(10)
	//Transfer ammount is larger than the balance
	transferAmount := int(25)

	//create a wallet address
	wallet1, err := CreateWallet()
	assert.NotEmpty(t, wallet1)
	addr1 := wallet1.GetAddress()

	//create a blockchain
	b, err := CreateBlockchain(addr1, databaseInstance, nil)
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
	err = Send(wallet1, addr2, transferAmount, tip, b)
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

func TestSyncBlocks(t *testing.T) {

	var pows []*consensus.ProofOfWork
	var bcs []*core.Blockchain
	addr := core.Address{"17DgRtQVvaytkiKAfXx9XbV23MESASSwUz"}
	//wait for mining for at least "targetHeight" blocks
	targetHeight := uint64(4)
	//num of nodes to be created in the test
	numOfNodes := 4
	var firstNode *network.Node
	for i := 0; i < numOfNodes; i++ {
		//create storage instance
		db := storage.NewRamStorage()
		defer db.Close()

		//create blockchain instance
		pow := consensus.NewProofOfWork()
		bc := core.CreateBlockchain(addr, db, pow)
		bcs = append(bcs, bc)

		n := network.NewNode(bcs[i])
		pow.Setup(n, addr.Address)
		pow.SetTargetBit(16)
		n.Start(testport + i)

		if i == 0{
			firstNode = n
		}else {
			n.AddStream(
				firstNode.GetPeerID(),
				firstNode.GetPeerMultiaddr(),
			)
		}

		pows = append(pows, pow)
	}

	//seed node broadcasts syncpeers
	firstNode.SyncPeers()

	//wait for 2 seconds for syncing
	time.Sleep(time.Second * 2)

	//count and is Stopped tracks the num of nodes that have been stopped
	count := 0
	isStopped := []bool{}
	blkHeight := []uint64{}
	//Start Mining and set average block time to 5 seconds (difficulty = 16)
	for i := 0; i < numOfNodes; i++ {
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
			}
			if blk.GetHeight() >= targetHeight {
				//count the number of nodes that have already stopped mining
				if isStopped[i] == false {
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

	time.Sleep(time.Second*2)

	//Check if all nodes have the same tail block
	for i := 0; i < numOfNodes-1; i++ {
		blk0, err := bcs[i].GetTailBlock()
		assert.Equal(t,targetHeight,blk0.GetHeight())
		assert.Nil(t, err)
		blk1, err := bcs[i+1].GetTailBlock()
		assert.Equal(t,targetHeight,blk1.GetHeight())
		assert.Nil(t, err)
		assert.Equal(t, blk0.GetHash(), blk1.GetHash())
	}

}

const testport_fork = 10200

func TestForkChoice(t *testing.T) {
	var pows []*consensus.ProofOfWork
	var bcs []*core.Blockchain
	addr := core.Address{"17DgRtQVvaytkiKAfXx9XbV23MESASSwUz"}
	//wait for mining for at least "targetHeight" blocks
	//targetHeight := uint64(4)
	//num of nodes to be created in the test
	numOfNodes := 2
	nodes := []*network.Node{}
	for i := 0; i < numOfNodes; i++ {
		//create storage instance
		db := storage.NewRamStorage()
		defer db.Close()

		pow := consensus.NewProofOfWork()
		//create blockchain instance
		bc := core.CreateBlockchain(addr, db, pow)
		bcs = append(bcs, bc)

		n := network.NewNode(bcs[i])
		pow.Setup(n, addr.Address)
		pow.SetTargetBit(16)
		n.Start(testport_fork + i)
		pows = append(pows, pow)
		nodes = append(nodes,n)
	}

	//start node0 first. the next node starts mining after the previous node is at least at height 5
	for i := 0; i < numOfNodes; i++ {
		pows[i].Start()
		//seed node broadcasts syncpeers
		for bcs[i].GetMaxHeight() < 5 {
		}
	}

	for i := 0; i < numOfNodes; i++ {
		if i != 0 {
			nodes[i].AddStream(
				nodes[0].GetPeerID(),
				nodes[0].GetPeerMultiaddr(),
			)
		}
		nodes[0].SyncPeers()
	}

	time.Sleep(time.Second * 5)

	//Check if all nodes have the same tail block
	for i := 0; i < numOfNodes-1; i++ {
		assert.True(t, compareTwoBlockchains(bcs[0], bcs[i]))
	}
}

func TestCompare(t *testing.T) {
	bc1 := core.GenerateMockBlockchain(5)
	bc2 := bc1
	assert.True(t, compareTwoBlockchains(bc1, bc2))
	bc3 := core.GenerateMockBlockchain(5)
	assert.False(t, compareTwoBlockchains(bc1, bc3))
}

func compareTwoBlockchains(bc1, bc2 *core.Blockchain) bool {
	if bc1 == nil || bc2 == nil {
		return false
	}

	bci1 := bc1.Iterator()
	bci2 := bc2.Iterator()
	if bc1.GetMaxHeight() != bc2.GetMaxHeight() {
		return false
	}

loop:
	for {
		blk1, _ := bci1.Next()
		blk2, _ := bci2.Next()
		if blk1 == nil || blk2 == nil {
			break loop
		}
		if !reflect.DeepEqual(blk1.GetHash(), blk2.GetHash()) {
			return false
		}
	}
	return true
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
