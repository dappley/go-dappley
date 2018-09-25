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

package logic

import (
	"testing"

	"github.com/dappley/go-dappley/common"

	"time"

	"github.com/dappley/go-dappley/consensus"
	"github.com/dappley/go-dappley/core"
	"github.com/dappley/go-dappley/network"
	"github.com/dappley/go-dappley/storage"
	"github.com/stretchr/testify/assert"
)

const testport_msg_relay = 19999
const testport_msg_relay_port = 21202
const testport_fork = 10200

//test send
func TestSend(t *testing.T) {
	var mineReward = common.NewAmount(10)
	testCases := []struct {
		name             string
		transferAmount   *common.Amount
		tipAmount        uint64
		expectedTransfer *common.Amount
		expectedTip      uint64
		expectedErr      error
	}{
		{"Send with no tip", common.NewAmount(7), 0, common.NewAmount(7), 0, nil},
		{"Send with tips", common.NewAmount(6), 2, common.NewAmount(6), 2, nil},
		{"Send zero with no tip", common.NewAmount(0), 0, common.NewAmount(0), 0, ErrInvalidAmount},
		{"Send zero with tips", common.NewAmount(0), 2, common.NewAmount(0), 0, ErrInvalidAmount},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {

			store := storage.NewRamStorage()
			defer store.Close()

			// Create a wallet address
			senderWallet, err := CreateWallet()
			if err != nil {
				panic(err)
			}

			// Create a PoW blockchain with the sender wallet's address as the coinbase address
			// i.e. sender's wallet would have mineReward amount after blockchain created
			bc, pow := createBlockchain(senderWallet.GetAddress(), store)
			node := network.FakeNodeWithPidAndAddr(bc, "test", "test")

			// Create a receiver wallet; Balance is 0 initially
			receiverWallet, err := CreateWallet()
			if err != nil {
				panic(err)
			}

			// Send coins from senderWallet to receiverWallet
			err = Send(senderWallet, receiverWallet.GetAddress(), tc.transferAmount, uint64(tc.tipAmount), bc, node)
			assert.Equal(t, tc.expectedErr, err)

			// Create a miner wallet; Balance is 0 initially
			minerWallet, err := CreateWallet()
			if err != nil {
				panic(err)
			}

			// Make sender the miner and mine for 1 block (which should include the transaction)
			pow.Setup(node, minerWallet.GetAddress().Address)
			pow.Start()
			for bc.GetMaxHeight() < 1 {
			}
			pow.Stop()
			currentTime := time.Now().UTC().Unix()
			for !pow.FullyStop() && time.Now().UTC().Unix()-currentTime < 20 {
			}

			// Verify balance of sender's wallet (genesis "mineReward" - transferred amount)
			senderBalance, err := GetBalance(senderWallet.GetAddress(), store)
			if err != nil {
				panic(err)
			}
			expectedBalance, _ := mineReward.Sub(tc.expectedTransfer)
			assert.Equal(t, expectedBalance, senderBalance)

			// Balance of the receiver's wallet should be the amount transferred
			receiverBalance, err := GetBalance(receiverWallet.GetAddress(), store)
			if err != nil {
				panic(err)
			}
			assert.Equal(t, tc.expectedTransfer, receiverBalance)

			// Balance of the miner's wallet should be the amount tipped + mineReward
			minerBalance, err := GetBalance(minerWallet.GetAddress(), store)
			if err != nil {
				panic(err)
			}
			assert.Equal(t, mineReward.Times(bc.GetMaxHeight()), minerBalance)

		})
	}
}

//test send to invalid address
func TestSendToInvalidAddress(t *testing.T) {
	//setup: clean up database and files
	setup()

	store := storage.NewRamStorage()
	defer store.Close()

	//this is internally set. Dont modify
	mineReward := common.NewAmount(10)
	//Transfer ammount
	transferAmount := common.NewAmount(25)
	tip := uint64(5)
	//create a wallet address
	wallet1, err := CreateWallet()
	assert.NotEmpty(t, wallet1)
	addr1 := wallet1.GetAddress()

	//create a blockchain
	bc, err := CreateBlockchain(addr1, store, nil)
	assert.Nil(t, err)
	assert.NotNil(t, bc)

	//The balance should be 10 after creating a blockchain
	balance1, err := GetBalance(addr1, store)
	assert.Nil(t, err)
	assert.Equal(t, mineReward, balance1)
	node := network.FakeNodeWithPidAndAddr(bc, "test", "test")

	//Send 5 coins from addr1 to an invalid address
	err = Send(wallet1, core.NewAddress(InvalidAddress), transferAmount, tip, bc, node)
	assert.NotNil(t, err)

	//the balance of the first wallet should be still be 10
	balance1, err = GetBalance(addr1, store)
	assert.Nil(t, err)
	assert.Equal(t, mineReward, balance1)
	//teardown :clean up database amd files
	teardown()
}

//insufficient fund
func TestSendInsufficientBalance(t *testing.T) {
	//setup: clean up database and files
	setup()

	store := storage.NewRamStorage()
	defer store.Close()

	tip := uint64(5)

	//this is internally set. Dont modify
	mineReward := common.NewAmount(10)
	//Transfer ammount is larger than the balance
	transferAmount := common.NewAmount(25)

	//create a wallet address
	wallet1, err := CreateWallet()
	assert.NotEmpty(t, wallet1)
	addr1 := wallet1.GetAddress()

	//create a blockchain
	bc, err := CreateBlockchain(addr1, store, nil)
	assert.Nil(t, err)
	assert.NotNil(t, bc)

	//The balance should be 10 after creating a blockchain
	balance1, err := GetBalance(addr1, store)
	assert.Nil(t, err)
	assert.Equal(t, mineReward, balance1)

	//Create a second wallet
	wallet2, err := CreateWallet()
	assert.NotEmpty(t, wallet2)
	assert.Nil(t, err)
	addr2 := wallet2.GetAddress()

	//The balance should be 0
	balance2, err := GetBalance(addr2, store)
	assert.Nil(t, err)
	assert.Equal(t, common.NewAmount(0), balance2)
	node := network.FakeNodeWithPidAndAddr(bc, "test", "test")

	//Send 5 coins from addr1 to addr2
	err = Send(wallet1, addr2, transferAmount, tip, bc, node)
	assert.NotNil(t, err)

	//the balance of the first wallet should be still be 10
	balance1, err = GetBalance(addr1, store)
	assert.Nil(t, err)
	assert.Equal(t, mineReward, balance1)

	//the balance of the second wallet should be 0
	balance2, err = GetBalance(addr2, store)
	assert.Nil(t, err)
	assert.Equal(t, common.NewAmount(0), balance2)

	//teardown :clean up database amd files
	teardown()
}

func TestBlockMsgRelay(t *testing.T) {
	setup()
	var pows []*consensus.ProofOfWork
	var bcs []*core.Blockchain
	var nodes []*network.Node
	addr := core.Address{"17DgRtQVvaytkiKAfXx9XbV23MESASSwUz"}

	numOfNodes := 4
	for i := 0; i < numOfNodes; i++ {

		db := storage.NewRamStorage()
		defer db.Close()

		//create blockchain instance
		bc, pow := createBlockchain(addr, db)
		bcs = append(bcs, bc)

		node := setupNode(addr, pow, bc, testport_msg_relay+100*i)

		nodes = append(nodes, node)
		pows = append(pows, pow)
	}
	//each node connects to the subsequent node only
	for i := 0; i < len(nodes)-1; i++ {
		connectNodes(nodes[i], nodes[i+1])
	}

	//firstNode Starts Mining

	pows[0].Start()
	for bcs[0].GetMaxHeight() < 6 {
	}
	//expect every node should have # of entries in dapmsg cache equal to their blockchain height
	heights := []int{0, 0, 0, 0} //keep track of each node's blockchain height
	for i := 0; i < len(nodes); i++ {
		nodes[i].GetRecentlyRcvedDapMsgs().Range(func(k, v interface{}) bool {
			heights[i]++
			return true
		})
		assert.Equal(t, heights[i], int(bcs[i].GetMaxHeight()))

	}
}

// Test if network radiation bounces forever
func TestBlockMsgRelayMeshNetwork(t *testing.T) {
	setup()
	var pows []*consensus.ProofOfWork
	var bcs []*core.Blockchain
	var nodes []*network.Node
	addr := core.Address{"17DgRtQVvaytkiKAfXx9XbV23MESASSwUz"}

	numOfNodes := 4
	for i := 0; i < numOfNodes; i++ {
		//init db for each node we want
		db := storage.NewRamStorage()
		defer db.Close()
		bc, pow := createBlockchain(addr, db)
		//init node based on blockchain
		node := setupNode(addr, pow, bc, testport_msg_relay+100*i)

		//index each node and service
		bcs = append(bcs, bc)
		nodes = append(nodes, node)
		pows = append(pows, pow)
	}

	//each node connects to every other node
	for i := 0; i < len(nodes); i++ {
		for j := 0; j < len(nodes); j++ {
			if i != j {
				connectNodes(nodes[i], nodes[j])
			}
		}
	}

	//firstNode Starts Mining

	pows[0].Start()
	for bcs[0].GetMaxHeight() < 5 {
	}

	//expect every node should have # of entries in dapmsg cache equal to their blockchain height
	heights := []int{0, 0, 0, 0} //keep track of each node's blockchain height
	for i := 0; i < len(nodes); i++ {
		nodes[i].GetRecentlyRcvedDapMsgs().Range(func(k, v interface{}) bool {
			heights[i]++
			return true
		})
		assert.Equal(t, heights[i], int(bcs[i].GetMaxHeight()))

	}
}

func TestBlockMsgWithDpos(t *testing.T) {
	const (
		timeBetweenBlock = 2
		dposRounds       = 3
		bufferTime       = 1
	)

	miners := []string{
		"1ArH9WoB9F7i6qoJiAi7McZMFVQSsBKXZR",
		"1BpXBb3uunLa9PL8MmkMtKNd3jzb5DHFkG",
	}
	keystrs := []string{
		"5a66b0fdb69c99935783059bb200e86e97b506ae443a62febd7d0750cd7fac55",
		"bb23d2ff19f5b16955e8a24dca34dd520980fe3bddca2b3e1b56663f0ec1aa7e",
	}
	dynasty := consensus.NewDynastyWithProducers(miners)
	dynasty.SetTimeBetweenBlk(timeBetweenBlock)
	dynasty.SetMaxProducers(len(miners))
	dposArray := []*consensus.Dpos{}
	var firstNode *network.Node
	for i := 0; i < len(miners); i++ {
		dpos := consensus.NewDpos()
		dpos.SetDynasty(dynasty)
		dpos.SetTargetBit(0) //gennerate a block every round
		bc := core.CreateBlockchain(core.Address{miners[0]}, storage.NewRamStorage(), dpos)
		node := network.NewNode(bc)
		node.Start(testport_msg_relay_port + i)
		if i == 0 {
			firstNode = node
		} else {
			node.AddStream(firstNode.GetPeerID(), firstNode.GetPeerMultiaddr())
		}
		dpos.Setup(node, miners[i])
		dpos.SetKey(keystrs[i])
		dposArray = append(dposArray, dpos)
	}

	firstNode.SyncPeersBroadcast()

	for i := 0; i < len(miners); i++ {
		dposArray[i].Start()
	}

	time.Sleep(time.Second * time.Duration(dynasty.GetDynastyTime()*dposRounds+bufferTime))

	for i := 0; i < len(miners); i++ {
		dposArray[i].Stop()
	}

	for i := 0; i < len(miners); i++ {
		v := dposArray[i].FullyStop()
		currentTime := time.Now().UTC().Unix()
		for !v && time.Now().UTC().Unix()-currentTime < 20 {
		}
	}

	for i := 0; i < len(miners); i++ {
		assert.Equal(t, uint64(dynasty.GetDynastyTime()*dposRounds/timeBetweenBlock), dposArray[i].GetBlockChain().GetMaxHeight())
	}
}

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
		db := storage.NewRamStorage()
		defer db.Close()

		bc, pow := createBlockchain(addr, db)
		bcs = append(bcs, bc)

		node := network.NewNode(bcs[i])
		pow.Setup(node, addr.Address)
		pow.SetTargetBit(16)
		node.Start(testport_fork + i)
		pows = append(pows, pow)
		nodes = append(nodes, node)
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
		nodes[0].SyncPeersBroadcast()
	}

	currentTime := time.Now().UTC().Unix()

	for !compareTwoBlockchains(bcs[0], bcs[numOfNodes-1]) && time.Now().UTC().Unix()-currentTime <= 5 {
	}

	//Check if all nodes have the same tail block
	for i := 0; i < numOfNodes-1; i++ {
		assert.True(t, compareTwoBlockchains(bcs[0], bcs[i]))
	}
}

// Integration test for adding balance
func TestAddBalance(t *testing.T) {
	testCases := []struct {
		name         string
		addAmount    *common.Amount
		expectedDiff *common.Amount
		expectedErr  error
	}{
		{"Add 5", common.NewAmount(5), common.NewAmount(5), nil},
		{"Add zero", common.NewAmount(0), common.NewAmount(0), ErrInvalidAmount},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {

			store := storage.NewRamStorage()
			defer store.Close()

			// Create a coinbase address
			addr := core.Address{"1G4r54VdJsotfCukXUWmg1ZRnhjUs6TvbV"}

			bc, pow := createBlockchain(addr, store)

			// Create a new wallet address for testing
			testAddr := core.Address{"1AUrNJCRM5X5fDdmm3E3yjCrXQMLvDj9tb"}

			// Add `addAmount` to the balance of the new wallet
			err := AddBalance(testAddr, tc.addAmount, bc)
			assert.Equal(t, err, tc.expectedErr)

			// Start mining to approve the transaction
			node := network.FakeNodeWithPidAndAddr(bc, "a", "b")
			pow.Setup(node, addr.Address)
			pow.SetTargetBit(0)
			pow.Start()

			for bc.GetMaxHeight() <= 1 {
			}
			pow.Stop()

			// The wallet balance should be the expected difference
			balance, err := GetBalance(testAddr, store)
			assert.Nil(t, err)
			assert.Equal(t, tc.expectedDiff, balance)
		})
	}
}

// Integration test for adding balance to invalid address
func TestAddBalanceWithInvalidAddress(t *testing.T) {
	testCases := []struct {
		name    string
		address string
	}{
		{"Invalid char in address", InvalidAddress},
		{"Invalid checksum address", "1AUrNJCRM5X5fDdmm3E3yjCrXQMLwfwfww"},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			store := storage.NewRamStorage()
			defer store.Close()

			// Create a coinbase wallet address
			addr := core.Address{"1G4r54VdJsotfCukXUWmg1ZRnhjUs6TvbV"}
			// Create a blockchain
			bc, err := CreateBlockchain(addr, store, nil)
			assert.Nil(t, err)
			err = AddBalance(core.Address{tc.address}, common.NewAmount(8), bc)
			assert.Equal(t, ErrInvalidAddress, err)
		})
	}
}

func connectNodes(node1 *network.Node, node2 *network.Node) {
	node1.AddStream(
		node2.GetPeerID(),
		node2.GetPeerMultiaddr(),
	)
}

func setupNode(addr core.Address, pow *consensus.ProofOfWork, bc *core.Blockchain, port int) *network.Node {
	node := network.NewNode(bc)
	pow.Setup(node, addr.Address)
	pow.SetTargetBit(16)
	node.Start(port)
	return node
}

func createBlockchain(addr core.Address, db *storage.RamStorage) (*core.Blockchain, *consensus.ProofOfWork) {
	pow := consensus.NewProofOfWork()
	return core.CreateBlockchain(addr, db, pow), pow
}
