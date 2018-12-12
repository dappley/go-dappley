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
	"time"

	"github.com/dappley/go-dappley/client"
	"github.com/dappley/go-dappley/common"
	"github.com/dappley/go-dappley/consensus"
	"github.com/dappley/go-dappley/core"
	"github.com/dappley/go-dappley/network"
	"github.com/dappley/go-dappley/storage"
	"github.com/stretchr/testify/assert"
)

const testport_msg_relay = 19999
const testport_msg_relay_port = 21202
const testport_fork = 10200
const testport_fork_segment = 10201

//test send
func TestSend(t *testing.T) {
	var mineReward = common.NewAmount(10)
	testCases := []struct {
		name             string
		transferAmount   *common.Amount
		tipAmount        uint64
		contract         string
		expectedTransfer *common.Amount
		expectedTip      uint64
		expectedErr      error
	}{
		{"Deploy contract", common.NewAmount(7), 0, "helloworld!", common.NewAmount(7), 0, nil},
		{"Send with no tip", common.NewAmount(7), 0, "", common.NewAmount(7), 0, nil},
		{"Send with tips", common.NewAmount(6), 2, "", common.NewAmount(6), 2, nil},
		{"Send zero with no tip", common.NewAmount(0), 0, "", common.NewAmount(0), 0, ErrInvalidAmount},
		{"Send zero with tips", common.NewAmount(0), 2, "", common.NewAmount(0), 0, ErrInvalidAmount},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {

			store := storage.NewRamStorage()
			defer store.Close()

			// Create a wallet address
			senderWallet, err := CreateWallet(GetTestWalletPath(), "test")
			if err != nil {
				panic(err)
			}

			// Create a PoW blockchain with the sender wallet's address as the coinbase address
			// i.e. sender's wallet would have mineReward amount after blockchain created
			bc, pow := createBlockchain(senderWallet.GetAddress(), store)
			node := network.FakeNodeWithPidAndAddr(bc, "test", "test")

			// Create a receiver wallet; Balance is 0 initially
			receiverWallet, err := CreateWallet(GetTestWalletPath(), "test")
			if err != nil {
				panic(err)
			}

			// Send coins from senderWallet to receiverWallet
			var rcvAddr core.Address
			isContract := (tc.contract != "")
			if isContract {
				rcvAddr = core.NewAddress("")
			} else {
				rcvAddr = receiverWallet.GetAddress()
			}

			_, err = Send(senderWallet, rcvAddr, tc.transferAmount, uint64(tc.tipAmount), tc.contract, bc, node)
			assert.Equal(t, tc.expectedErr, err)

			// Create a miner wallet; Balance is 0 initially
			minerWallet, err := CreateWallet(GetTestWalletPath(), "test")
			if err != nil {
				panic(err)
			}

			//a short delay before mining starts
			time.Sleep(time.Millisecond * 500)

			// Make sender the miner and mine for 1 block (which should include the transaction)
			pow.Setup(node, minerWallet.GetAddress().String())
			pow.Start()
			for bc.GetMaxHeight() < 1 {
			}
			pow.Stop()
			core.WaitDoneOrTimeout(func() bool {
				return !pow.IsProducingBlock()
			}, 20)
			// Verify balance of sender's wallet (genesis "mineReward" - transferred amount)
			senderBalance, err := GetBalance(senderWallet.GetAddress(), store)
			if err != nil {
				panic(err)
			}
			expectedBalance, _ := mineReward.Sub(tc.expectedTransfer)
			expectedBalance, _ = expectedBalance.Sub(common.NewAmount(tc.expectedTip))
			assert.Equal(t, expectedBalance, senderBalance)

			// Balance of the miner's wallet should be the amount tipped + mineReward
			minerBalance, err := GetBalance(minerWallet.GetAddress(), store)
			if err != nil {
				panic(err)
			}
			assert.Equal(t, mineReward.Times(bc.GetMaxHeight()).Add(common.NewAmount(tc.expectedTip)), minerBalance)

			//check smart contract deployment
			res := string("")
			contractAddr := core.NewAddress("")
		loop:
			for i := bc.GetMaxHeight(); i > 0; i-- {
				blk, err := bc.GetBlockByHeight(i)
				assert.Nil(t, err)
				for _, tx := range blk.GetTransactions() {
					contractAddr = tx.GetContractAddress()
					if contractAddr.String() != "" {
						res = tx.Vout[core.ContractTxouputIndex].Contract
						break loop
					}
				}
			}
			assert.Equal(t, tc.contract, res)

			// Balance of the receiver's wallet should be the amount transferred
			var receiverBalance *common.Amount
			if isContract {
				receiverBalance, err = GetBalance(contractAddr, store)
			} else {
				receiverBalance, err = GetBalance(receiverWallet.GetAddress(), store)
			}
			assert.Equal(t, tc.expectedTransfer, receiverBalance)
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
	wallet1, err := CreateWallet(GetTestWalletPath(), "test")
	assert.NotEmpty(t, wallet1)
	addr1 := wallet1.GetAddress()

	//create a blockchain
	bc, err := CreateBlockchain(addr1, store, nil, 128, nil)
	assert.Nil(t, err)
	assert.NotNil(t, bc)

	//The balance should be 10 after creating a blockchain
	balance1, err := GetBalance(addr1, store)
	assert.Nil(t, err)
	assert.Equal(t, mineReward, balance1)
	node := network.FakeNodeWithPidAndAddr(bc, "test", "test")

	//Send 5 coins from addr1 to an invalid address
	_, err = Send(wallet1, core.NewAddress(InvalidAddress), transferAmount, tip, "", bc, node)
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
	wallet1, err := CreateWallet(GetTestWalletPath(), "test")
	assert.NotEmpty(t, wallet1)
	addr1 := wallet1.GetAddress()

	//create a blockchain
	bc, err := CreateBlockchain(addr1, store, nil, 128, nil)
	assert.Nil(t, err)
	assert.NotNil(t, bc)

	//The balance should be 10 after creating a blockchain
	balance1, err := GetBalance(addr1, store)
	assert.Nil(t, err)
	assert.Equal(t, mineReward, balance1)

	//Create a second wallet
	wallet2, err := CreateWallet(GetTestWalletPath(), "test")
	assert.NotEmpty(t, wallet2)
	assert.Nil(t, err)
	addr2 := wallet2.GetAddress()

	//The balance should be 0
	balance2, err := GetBalance(addr2, store)
	assert.Nil(t, err)
	assert.Equal(t, common.NewAmount(0), balance2)
	node := network.FakeNodeWithPidAndAddr(bc, "test", "test")

	//Send 5 coins from addr1 to addr2
	_, err = Send(wallet1, addr2, transferAmount, tip, "", bc, node)
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

func TestBlockMsgRelaySingleMiner(t *testing.T) {
	const (
		timeBetweenBlock = 1
		dposRounds       = 2
		bufferTime       = 0
	)
	setup()
	var dposArray []*consensus.DPOS
	var bcs []*core.Blockchain
	var nodes []*network.Node
	var firstNode *network.Node

	validProducerAddr := "dastXXWLe5pxbRYFhcyUq8T3wb5srWkHKa"
	validProducerKey := "300c0338c4b0d49edc66113e3584e04c6b907f9ded711d396d522aae6a79be1a"

	producerAddrs := []string{}
	producerKey := []string{}
	numOfNodes := 4
	for i := 0; i < numOfNodes; i++ {
		producerAddrs = append(producerAddrs, validProducerAddr)
		producerKey = append(producerKey, validProducerKey)
	}

	dynasty := consensus.NewDynasty(producerAddrs, numOfNodes, timeBetweenBlock)

	for i := 0; i < numOfNodes; i++ {
		dpos := consensus.NewDPOS()
		dpos.SetDynasty(dynasty)
		bc := core.CreateBlockchain(core.Address{producerAddrs[0]}, storage.NewRamStorage(), dpos, 128, nil)
		bcs = append(bcs, bc)
		node := network.NewNode(bc)
		node.Start(testport_msg_relay_port + i)
		if i == 0 {
			firstNode = node
		} else {
			node.AddStream(firstNode.GetPeerID(), firstNode.GetPeerMultiaddr())
		}
		dpos.Setup(node, producerAddrs[0])
		dpos.SetKey(producerKey[0])
		dposArray = append(dposArray, dpos)
	}
	//each node connects to the subsequent node only
	for i := 0; i < len(nodes)-1; i++ {
		connectNodes(nodes[i], nodes[i+1])
	}

	//firstNode Starts Mining
	dposArray[0].Start()
	core.WaitDoneOrTimeout(func() bool {
		return bcs[0].GetMaxHeight() >= 5
	}, 8)
	dposArray[0].Stop()
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
func TestBlockMsgRelayMeshNetworkMultipleMiners(t *testing.T) {
	const (
		timeBetweenBlock = 1
		dposRounds       = 2
		bufferTime       = 0
	)
	setup()
	var dposArray []*consensus.DPOS
	var bcs []*core.Blockchain
	var nodes []*network.Node

	var firstNode *network.Node

	validProducerAddr := "dastXXWLe5pxbRYFhcyUq8T3wb5srWkHKa"
	validProducerKey := "300c0338c4b0d49edc66113e3584e04c6b907f9ded711d396d522aae6a79be1a"

	producerAddrs := []string{}
	producerKey := []string{}
	numOfNodes := 4
	for i := 0; i < numOfNodes; i++ {
		producerAddrs = append(producerAddrs, validProducerAddr)
		producerKey = append(producerKey, validProducerKey)
	}

	dynasty := consensus.NewDynasty(producerAddrs, numOfNodes, timeBetweenBlock)

	for i := 0; i < numOfNodes; i++ {
		dpos := consensus.NewDPOS()
		dpos.SetDynasty(dynasty)

		bc := core.CreateBlockchain(core.Address{producerAddrs[0]}, storage.NewRamStorage(), dpos, 128, nil)
		bcs = append(bcs, bc)

		node := network.NewNode(bc)
		node.Start(testport_msg_relay_port + i)
		if i == 0 {
			firstNode = node
		} else {
			node.AddStream(firstNode.GetPeerID(), firstNode.GetPeerMultiaddr())
		}
		dpos.Setup(node, producerAddrs[0])
		dpos.SetKey(producerKey[0])
		dposArray = append(dposArray, dpos)
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
	for i := 0; i < len(dposArray); i++ {
		dposArray[i].Start()
	}

	time.Sleep(time.Second * time.Duration(dynasty.GetDynastyTime()*dposRounds+bufferTime))

	for i := 0; i < len(dposArray); i++ {
		dposArray[i].Stop()
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

func TestForkChoice(t *testing.T) {
	var pows []*consensus.ProofOfWork
	var bcs []*core.Blockchain
	var dbs []storage.Storage
	// Remember to close all opened databases after test
	defer func() {
		for _, db := range dbs {
			db.Close()
		}
	}()

	addr := core.Address{"17DgRtQVvaytkiKAfXx9XbV23MESASSwUz"}
	//wait for mining for at least "targetHeight" blocks
	//targetHeight := uint64(4)
	//num of nodes to be created in the test
	numOfNodes := 2
	var nodes []*network.Node
	for i := 0; i < numOfNodes; i++ {
		db := storage.NewRamStorage()
		dbs = append(dbs, db)

		bc, pow := createBlockchain(addr, db)
		bcs = append(bcs, bc)

		node := network.NewNode(bcs[i])
		pow.Setup(node, addr.String())
		pow.SetTargetBit(10)
		node.Start(testport_fork + i)
		pows = append(pows, pow)
		nodes = append(nodes, node)
	}

	// Mine more blocks on node[0] than on node[1]
	pows[1].Start()
	core.WaitDoneOrTimeout(func() bool {
		return bcs[1].GetMaxHeight() > 4
	}, 10)
	pows[1].Stop()
	desiredHeight := uint64(10)
	if bcs[1].GetMaxHeight() > 10 {
		desiredHeight = bcs[1].GetMaxHeight() + 1
	}
	pows[0].Start()
	core.WaitDoneOrTimeout(func() bool {
		return bcs[0].GetMaxHeight() > desiredHeight
	}, 20)
	pows[0].Stop()

	core.WaitDoneOrTimeout(func() bool {
		return !pows[0].IsProducingBlock()
	}, 5)

	// Trigger fork choice in node[1] by broadcasting tail block of node[0]
	tailBlk, _ := bcs[0].GetTailBlock()
	connectNodes(nodes[0], nodes[1])
	nodes[0].BroadcastBlock(tailBlk)
	// Make sure syncing starts on node[1]
	core.WaitDoneOrTimeout(func() bool {
		return bcs[1].GetBlockPool().GetSyncState()
	}, 10)
	// Make sure syncing ends on node[1]
	core.WaitDoneOrTimeout(func() bool {
		return !bcs[1].GetBlockPool().GetSyncState()
	}, 20)

	assert.True(t, isSameBlockChain(bcs[0], bcs[1]))
}

func TestForkSegmentHandling(t *testing.T) {
	var pows []*consensus.ProofOfWork
	var bcs []*core.Blockchain
	var dbs []storage.Storage
	// Remember to close all opened databases after test
	defer func() {
		for _, db := range dbs {
			db.Close()
		}
	}()
	addr := core.Address{"17DgRtQVvaytkiKAfXx9XbV23MESASSwUz"}

	numOfNodes := 2
	var nodes []*network.Node
	for i := 0; i < numOfNodes; i++ {
		db := storage.NewRamStorage()
		dbs = append(dbs, db)

		bc, pow := createBlockchain(addr, db)
		bcs = append(bcs, bc)

		node := network.NewNode(bcs[i])
		pow.Setup(node, addr.String())
		pow.SetTargetBit(10)
		node.Start(testport_fork_segment + i)
		pows = append(pows, pow)
		nodes = append(nodes, node)
	}

	blk1 := &core.Block{}
	blk2 := &core.Block{}

	// Ensure node[1] mined some blocks
	pows[1].Start()
	core.WaitDoneOrTimeout(func() bool {
		return bcs[1].GetMaxHeight() > 3
	}, 10)
	pows[1].Stop()

	// Ensure node[0] mines more blocks than node[1]
	pows[0].Start()
	core.WaitDoneOrTimeout(func() bool {
		return bcs[0].GetMaxHeight() > 12
	}, 30)
	pows[0].Stop()

	core.WaitDoneOrTimeout(func() bool {
		return !pows[0].IsProducingBlock()
	}, 5)

	// Pick 2 blocks from blockchain[0] which can trigger syncing on node[1]
	mid := uint64(7)
	if bcs[0].GetMaxHeight() < 7 {
		mid = bcs[0].GetMaxHeight() - 1
	}
	blk1, _ = bcs[0].GetBlockByHeight(mid)
	blk2, _ = bcs[0].GetTailBlock()

	connectNodes(nodes[0], nodes[1])
	nodes[0].BroadcastBlock(blk1)
	// Wait for node[1] to start syncing
	core.WaitDoneOrTimeout(func() bool {
		return bcs[1].GetBlockPool().GetSyncState()
	}, 10)

	// node[0] broadcast higher block on the same fork and should trigger another sync on node[1]
	nodes[0].BroadcastBlock(blk2)

	// Make sure previous syncing ends
	core.WaitDoneOrTimeout(func() bool {
		return !bcs[1].GetBlockPool().GetSyncState()
	}, 10)
	// Make sure node[1] is syncing again
	core.WaitDoneOrTimeout(func() bool {
		return bcs[1].GetBlockPool().GetSyncState()
	}, 10)
	// Make sure syncing ends
	core.WaitDoneOrTimeout(func() bool {
		return !bcs[1].GetBlockPool().GetSyncState()
	}, 10)

	assert.True(t, isSameBlockChain(bcs[0], bcs[1]))
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
			key := "bb23d2ff19f5b16955e8a24dca34dd520980fe3bddca2b3e1b56663f0ec1aa7e"
			minerKeyPair := core.GetKeyPairByString(key)
			minerWallet := &client.Wallet{}
			minerWallet.Key = minerKeyPair

			addr := minerWallet.Key.GenerateAddress(false)

			bc, pow := createBlockchain(addr, store)

			// Create a new wallet address for testing
			testAddr := core.Address{"dGDrVKjCG3sdXtDUgWZ7Fp3Q97tLhqWivf"}

			// Start mining to approve the transaction
			node := network.FakeNodeWithPidAndAddr(bc, "a", "b")
			SetMinerKeyPair(key)
			pow.Setup(node, addr.String())
			pow.SetTargetBit(0)
			pow.Start()

			for bc.GetMaxHeight() <= 1 {
			}

			// Add `addAmount` to the balance of the new wallet
			_, err := SendFromMiner(testAddr, tc.addAmount, bc, node)
			height := bc.GetMaxHeight()
			assert.Equal(t, err, tc.expectedErr)
			for bc.GetMaxHeight()-height <= 1 {
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
			addr := core.Address{"dG6HhzSdA5m7KqvJNszVSf8i5f4neAteSs"}
			// Create a blockchain
			bc, err := CreateBlockchain(addr, store, nil, 128, nil)
			assert.Nil(t, err)
			node := network.FakeNodeWithPidAndAddr(bc, "a", "b")
			_, err = SendFromMiner(core.Address{tc.address}, common.NewAmount(8), bc, node)
			assert.Equal(t, ErrInvalidRcverAddress, err)
		})
	}
}

func TestSmartContractLocalStorage(t *testing.T) {
	store := storage.NewRamStorage()
	defer store.Close()

	contract := `'use strict';

	var StorageTest = function(){

	};

	StorageTest.prototype = {
	set:function(key,value){
			return LocalStorage.set(key,value);
		},
	get:function(key){
			return LocalStorage.get(key);
		}
	};
	var storageTest = new StorageTest;
	`

	// Create a wallet address
	senderWallet, err := CreateWallet(GetTestWalletPath(), "test")
	assert.Nil(t, err)

	bc, pow := createBlockchain(senderWallet.GetAddress(), store)
	node := network.FakeNodeWithPidAndAddr(bc, "test", "test")

	//deploy smart contract
	_, err = Send(senderWallet, core.Address{""}, common.NewAmount(1), uint64(0), contract, bc, node)
	assert.Nil(t, err)

	txp := bc.GetTxPool().GetTransactions()[0]
	contractAddr := txp.GetContractAddress()

	// Create a miner wallet; Balance is 0 initially
	minerWallet, err := CreateWallet(GetTestWalletPath(), "test")
	if err != nil {
		panic(err)
	}

	//a short delay before mining starts
	time.Sleep(time.Millisecond * 500)

	// Make sender the miner and mine for 1 block (which should include the transaction)
	pow.Setup(node, minerWallet.GetAddress().String())
	pow.Start()
	for bc.GetMaxHeight() < 1 {
	}
	pow.Stop()

	//a short delay before mining starts
	time.Sleep(time.Millisecond * 500)

	//store data
	functionCall := `{"function":"set","args":["testKey","222"]}`
	_, err = Send(senderWallet, contractAddr, common.NewAmount(1), uint64(0), functionCall, bc, node)
	assert.Nil(t, err)
	pow.Start()
	for bc.GetMaxHeight() < 1 {
	}
	pow.Stop()

	//get data
	functionCall = `{"function":"get","args":["testKey"]}`
	_, err = Send(senderWallet, contractAddr, common.NewAmount(1), uint64(0), functionCall, bc, node)
	assert.Nil(t, err)
	pow.Start()
	for bc.GetMaxHeight() < 1 {
	}
	pow.Stop()
}

func connectNodes(node1 *network.Node, node2 *network.Node) {
	node1.AddStream(
		node2.GetPeerID(),
		node2.GetPeerMultiaddr(),
	)
}

func setupNode(addr core.Address, pow *consensus.ProofOfWork, bc *core.Blockchain, port int) *network.Node {
	node := network.NewNode(bc)
	pow.Setup(node, addr.String())
	pow.SetTargetBit(12)
	node.Start(port)
	return node
}

func createBlockchain(addr core.Address, db *storage.RamStorage) (*core.Blockchain, *consensus.ProofOfWork) {
	pow := consensus.NewProofOfWork()
	return core.CreateBlockchain(addr, db, pow, 128, nil), pow
}
func TestDoubleMint(t *testing.T) {
	var sendNode *network.Node
	var recvNode *network.Node
	var blks []*core.Block
	var parent *core.Block

	validProducerAddr := "dPGZmHd73UpZhrM6uvgnzu49ttbLp4AzU8"
	validProducerKey := "5a66b0fdb69c99935783059bb200e86e97b506ae443a62febd7d0750cd7fac55"

	dynasty := consensus.NewDynasty([]string{validProducerAddr}, len([]string{validProducerAddr}), 15)
	producerHash, _ := core.NewAddress(validProducerAddr).GetPubKeyHash()
	tx := &core.Transaction{nil, []core.TXInput{{[]byte{}, -1, nil, nil}}, []core.TXOutput{{common.NewAmount(0), core.PubKeyHash{producerHash}, ""}}, 0}

	for i := 0; i < 3; i++ {
		blk := createValidBlock(producerHash, []*core.Transaction{tx}, validProducerKey, parent)
		blks = append(blks, blk)
		parent = blk
	}
	//check all timestamps are equal
	for i := 0; i < len(blks)-1; i++ {
		assert.True(t, blks[i].GetTimestamp() == blks[i+1].GetTimestamp())
	}
	for i := 0; i < 2; i++ {

		dpos := consensus.NewDPOS()
		dpos.SetDynasty(dynasty)
		bc := core.CreateBlockchain(core.Address{validProducerAddr}, storage.NewRamStorage(), dpos, 128, nil)
		node := network.NewNode(bc)
		node.Start(testport_msg_relay_port + i)
		if i == 0 {
			sendNode = node
		} else {
			recvNode = node
			recvNode.AddStream(sendNode.GetPeerID(), sendNode.GetPeerMultiaddr())
		}
	}

	for i := 0; i < len(blks); i++ {
		sendNode.BroadcastBlock(blks[i])
	}

	time.Sleep(time.Second * 2)
	assert.True(t, recvNode.GetBlockchain().GetMaxHeight() < 2)
}

func createValidBlock(hash core.Hash, tx []*core.Transaction, validProducerKey string, parent *core.Block) *core.Block {
	blk := core.NewBlock(tx, parent)
	blk.SetHash(blk.CalculateHashWithoutNonce())
	blk.SignBlock(validProducerKey, blk.CalculateHashWithoutNonce())
	return blk
}
