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
	"github.com/dappley/go-dappley/common"
	"os"
	"testing"

	"time"

	"reflect"

	"github.com/dappley/go-dappley/client"
	"github.com/dappley/go-dappley/consensus"
	"github.com/dappley/go-dappley/core"
	"github.com/dappley/go-dappley/network"
	"github.com/dappley/go-dappley/storage"
	"github.com/stretchr/testify/assert"
	"github.com/sirupsen/logrus"
)

const InvalidAddress = "Invalid Address"

func TestMain(m *testing.M) {
	setup()
	logrus.SetLevel(logrus.WarnLevel)
	retCode := m.Run()
	teardown()
	os.Exit(retCode)
}

func TestCreateWallet(t *testing.T) {
	wallet, err := CreateWallet()
	assert.Nil(t, err)
	assert.Equal(t, 34, len(wallet.Addresses[0].Address))
}

func TestCreateBlockchain(t *testing.T) {
	// Create storage
	store := storage.NewRamStorage()
	defer store.Close()

	//create a wallet address
	addr := core.Address{"1G4r54VdJsotfCukXUWmg1ZRnhjUs6TvbV"}

	//create a blockchain
	_, err := CreateBlockchain(addr, store, nil)
	assert.Nil(t, err)
}

//create a blockchain with invalid address
func TestCreateBlockchainWithInvalidAddress(t *testing.T) {
	store := storage.NewRamStorage()
	// Create storage
	defer store.Close()

	//create a blockchain with an invalid address
	bc, err := CreateBlockchain(core.NewAddress(InvalidAddress), store, nil)
	assert.Equal(t, ErrInvalidAddress, err)
	assert.Nil(t, bc)
}

func TestGetBalance(t *testing.T) {
	// Create storage
	store := storage.NewRamStorage()
	defer store.Close()

	//create a wallet address
	addr := core.Address{"1G4r54VdJsotfCukXUWmg1ZRnhjUs6TvbV"}
	//create a blockchain
	bc, err := CreateBlockchain(addr, store, nil)
	assert.Nil(t, err)
	assert.NotNil(t, bc)

	//The balance should be 10 after creating a blockchain
	balance, err := GetBalance(addr, store)
	assert.Nil(t, err)
	assert.Equal(t, common.NewAmount(10), balance)
}

func TestGetBalanceWithInvalidAddress(t *testing.T) {
	// Create storage
	store := storage.NewRamStorage()
	defer store.Close()

	//create a wallet address
	addr := core.Address{"1G4r54VdJsotfCukXUWmg1ZRnhjUs6TvbV"}
	//create a blockchain
	bc, err := CreateBlockchain(addr, store, nil)
	assert.Nil(t, err)
	assert.NotNil(t, bc)

	//The balance should be 10 after creating a blockchain
	balance1, err := GetBalance(core.NewAddress("1AUrNJCRM5X5fDdmm3E3yjCrXQMLvDj9tb"), store)
	assert.Nil(t, err)
	assert.Equal(t, common.NewAmount(0), balance1)

	balance2, err := GetBalance(core.NewAddress("1AUrNJCRM5X5fDdmm3E3yjCrXQMLwfwfww"), store)
	assert.Equal(t, errors.New("ERROR: Address is invalid"), err)
	assert.Equal(t, common.NewAmount(0), balance2)
}

func TestGetAllAddresses(t *testing.T) {
	setup()

	// Create storage
	store := storage.NewRamStorage()
	defer store.Close()

	expected_res := []core.Address{}
	//create a wallet address
	wallet, err := CreateWallet()
	assert.NotEmpty(t, wallet)
	addr := wallet.GetAddress()

	expected_res = append(expected_res, addr)

	//create a blockchain
	bc, err := CreateBlockchain(addr, store, nil)
	assert.Nil(t, err)
	assert.NotNil(t, bc)

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

	//the length should be equal
	assert.Equal(t, len(expected_res), len(addrs))
	assert.ElementsMatch(t, expected_res, addrs)
	teardown()
}

//test send
func TestSend(t *testing.T) {
	var mineReward = common.NewAmount(10)
	testCases := []struct {
		name  string
		transferAmount  *common.Amount
		tipAmount  uint64
		expectedTransfer  *common.Amount
		expectedTip  uint64
		expectedErr  error
	}{
		{"Send with no tip", common.NewAmount(7), 0, common.NewAmount(7), 0, nil},
		{"Send with tips", common.NewAmount(6), 2, common.NewAmount(6), 2, nil},
		{"Send zero with no tip", common.NewAmount(0), 0, common.NewAmount(0), 0, ErrInvalidAmount},
		{"Send zero with tips", common.NewAmount(0), 2, common.NewAmount(0), 0, ErrInvalidAmount},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {

			// Create storage
			store := storage.NewRamStorage()
			defer store.Close()

			// Create a wallet address
			senderWallet, err := CreateWallet()
			if err != nil {
				panic(err)
			}

			// Create a PoW blockchain with the sender wallet's address as the coinbase address
			// i.e. sender's wallet would have mineReward amount after blockchain created
			pow := consensus.NewProofOfWork()
			bc, err := CreateBlockchain(senderWallet.GetAddress(), store, pow)
			if err != nil {
				panic(err)
			}

			node := network.FakeNodeWithPidAndAddr(bc, "test", "test")

			// Create a receiver wallet; Balance is 0 initially
			receiverWallet, err := CreateWallet()
			if err != nil {
				panic(err)
			}

			// Send coins from senderWallet to receiverWallet
			err = Send(senderWallet, receiverWallet.GetAddress(), tc.transferAmount, uint64(tc.tipAmount), bc)
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
			assert.Equal(t, mineReward, minerBalance)

		})
	}
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

	// Create storage
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

	//Send 5 coins from addr1 to an invalid address
	err = Send(wallet1, core.NewAddress(InvalidAddress), transferAmount, tip, bc)
	assert.NotNil(t, err)

	//the balance of the first wallet should be still be 10
	balance1, err = GetBalance(addr1, store)
	assert.Nil(t, err)
	assert.Equal(t, mineReward, balance1)
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

	// Create storage
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
	b, err := CreateBlockchain(addr1, store, nil)
	assert.Nil(t, err)
	assert.NotNil(t, b)

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

	//Send 5 coins from addr1 to addr2
	err = Send(wallet1, addr2, transferAmount, tip, b)
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

const testport_msg_relay = 19999


func TestBlockMsgRelay(t *testing.T) {
	setup()
	var pows []*consensus.ProofOfWork
	var bcs []*core.Blockchain
	var nodes []*network.Node
	addr := core.Address{"17DgRtQVvaytkiKAfXx9XbV23MESASSwUz"}

	numOfNodes := 4
	for i := 0; i < numOfNodes; i++ {
		//create storage instance
		db := storage.NewRamStorage()
		defer db.Close()

		//create blockchain instance
		pow := consensus.NewProofOfWork()
		bc := core.CreateBlockchain(addr, db, pow)
		bcs = append(bcs, bc)

		n := network.NewNode(bcs[i])

		if(i == 0){
			pow.Setup(n, addr.Address)
			pow.SetTargetBit(16)
		}

		n.Start(testport_msg_relay + 100*i)

		nodes = append(nodes, n)
		pows = append(pows, pow)
	}

	for i := 0; i < len(nodes); i++ {
		nodes[i].AddStream(
			nodes[i+1].GetPeerID(),
			nodes[i+1].GetPeerMultiaddr(),
		)
		if i == (len(nodes) - 2) {
			break
		}
	}

	//firstNode Starts Mining

	pows[0].Start()
	time.Sleep(time.Second*3)

	//expect every node should have # of entries in dapmsg cache equal to their blockchain height
	heights := []int{0,0,0,0} //keep track of each node's blockchain height
	for i := 0; i < len(nodes); i++ {
		for _,_ = range *nodes[i].GetRecentlyRcvedDapMsgs() {
			heights[i]++
		}
		assert.Equal(t, heights[i], int(bcs[i].GetMaxHeight()))

		}
}

func TestBlockMsgMeshRelay(t *testing.T) {
	setup()
	var pows []*consensus.ProofOfWork
	var bcs []*core.Blockchain
	var nodes []*network.Node
	addr := core.Address{"17DgRtQVvaytkiKAfXx9XbV23MESASSwUz"}

	numOfNodes := 4
	for i := 0; i < numOfNodes; i++ {
		//create storage instance
		db := storage.NewRamStorage()
		defer db.Close()

		//create blockchain instance
		pow := consensus.NewProofOfWork()
		bc := core.CreateBlockchain(addr, db, pow)
		bcs = append(bcs, bc)

		n := network.NewNode(bcs[i])

		if(i == 0){
			pow.Setup(n, addr.Address)
			pow.SetTargetBit(16)
		}

		n.Start(testport_msg_relay + 100*i)

		nodes = append(nodes, n)
		pows = append(pows, pow)
	}

	for i := 0; i < len(nodes); i++ {
		for j := 0; j < len(nodes); j++{
			if i != j {
				nodes[i].AddStream(
					nodes[j].GetPeerID(),
					nodes[j].GetPeerMultiaddr(),
				)
			}
		}
	}

	//firstNode Starts Mining

	pows[0].Start()
	time.Sleep(time.Second*3)

	//expect every node should have # of entries in dapmsg cache equal to their blockchain height
	heights := []int{0,0,0,0} //keep track of each node's blockchain height
	for i := 0; i < len(nodes); i++ {
		for _,_ = range *nodes[i].GetRecentlyRcvedDapMsgs() {
			heights[i]++
		}
		assert.Equal(t, heights[i], int(bcs[i].GetMaxHeight()))

	}
}

const testport_msg_relay_port = 21202
func TestBlockMsgWithDpos(t *testing.T) {

	miners := []string{
		"1ArH9WoB9F7i6qoJiAi7McZMFVQSsBKXZR",
		"1BpXBb3uunLa9PL8MmkMtKNd3jzb5DHFkG",
	}
	keystrs := []string{
		"5a66b0fdb69c99935783059bb200e86e97b506ae443a62febd7d0750cd7fac55",
		"bb23d2ff19f5b16955e8a24dca34dd520980fe3bddca2b3e1b56663f0ec1aa7e",
	}
	dynasty := consensus.NewDynastyWithProducers(miners)
	dynasty.SetTimeBetweenBlk(5)
	dynasty.SetMaxProducers(len(miners))
	dposArray := []*consensus.Dpos{}
	var firstNode *network.Node
	for i := 0; i < len(miners); i++ {
		dpos := consensus.NewDpos()
		dpos.SetDynasty(dynasty)
		dpos.SetTargetBit(0)        //gennerate a block every round
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


	time.Sleep(time.Second * time.Duration(dynasty.GetDynastyTime()*2+1))

	for i := 0; i < len(miners); i++ {
		dposArray[i].Stop()
	}

	time.Sleep(time.Second)

	for i := 0; i < len(miners); i++ {
		assert.True(t, dposArray[i].GetBlockChain().GetMaxHeight() >= 3)
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
		nodes = append(nodes, n)
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


// Integration test for adding balance
func TestAddBalance(t *testing.T) {
	testCases := []struct {
		name  string
		addAmount  *common.Amount
		expectedDiff  *common.Amount
		expectedErr  error
	}{
		{"Add 5", common.NewAmount(5), common.NewAmount(5), nil},
		{"Add zero", common.NewAmount(0), common.NewAmount(0), ErrInvalidAmount},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create storage
			store := storage.NewRamStorage()
			defer store.Close()

			// Create a coinbase address
			addr := core.Address{"1G4r54VdJsotfCukXUWmg1ZRnhjUs6TvbV"}

			// Create a pow consensus
			pow := consensus.NewProofOfWork()

			// Create a blockchain
			bc, err := CreateBlockchain(addr, store, pow)

			// Create a new wallet address for testing
			testAddr := core.Address{"1AUrNJCRM5X5fDdmm3E3yjCrXQMLvDj9tb"}

			// Add `addAmount` to the balance of the new wallet
			err = AddBalance(testAddr, tc.addAmount, bc)
			assert.Equal(t, err, tc.expectedErr)

			// Start mining to approve the transaction
			node := network.FakeNodeWithPidAndAddr(bc, "a", "b")
			pow.Setup(node, addr.Address)
			pow.SetTargetBit(0)
			pow.Start()

			for bc.GetMaxHeight()<=1{}
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
		name  string
		address  string
	}{
		{"Invalid char in address", InvalidAddress},
		{"Invalid checksum address", "1AUrNJCRM5X5fDdmm3E3yjCrXQMLwfwfww"},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create storage
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
	os.RemoveAll(client.WalletFile)
}