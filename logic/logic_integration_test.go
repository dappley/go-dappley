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
package logic

import (
	"github.com/dappley/go-dappley/core/block"
	"github.com/dappley/go-dappley/logic/block_logic"
	"testing"
	"time"

	"github.com/dappley/go-dappley/util"

	"github.com/dappley/go-dappley/common"
	"github.com/dappley/go-dappley/consensus"
	"github.com/dappley/go-dappley/core"
	"github.com/dappley/go-dappley/core/account"
	"github.com/dappley/go-dappley/network"
	"github.com/dappley/go-dappley/storage"
	"github.com/stretchr/testify/assert"
)

const testport_msg_relay_port1 = 31202
const testport_msg_relay_port2 = 31212
const testport_msg_relay_port3 = 31222
const testport_fork = 10500
const testport_fork_segment = 10511
const testport_fork_syncing = 10531
const testport_fork_download = 10600

//test send
func TestSend(t *testing.T) {
	var mineReward = common.NewAmount(10000000)
	testCases := []struct {
		name             string
		transferAmount   *common.Amount
		tipAmount        *common.Amount
		contract         string
		gasLimit         *common.Amount
		gasPrice         *common.Amount
		expectedTransfer *common.Amount
		expectedTip      *common.Amount
		expectedErr      error
	}{
		{"Deploy contract", common.NewAmount(7), common.NewAmount(0), "dapp_schedule!", common.NewAmount(30000), common.NewAmount(1), common.NewAmount(7), common.NewAmount(0), nil},
		{"Send with no tip", common.NewAmount(7), common.NewAmount(0), "", common.NewAmount(0), common.NewAmount(0), common.NewAmount(7), common.NewAmount(0), nil},
		{"Send with tips", common.NewAmount(6), common.NewAmount(2), "", common.NewAmount(0), common.NewAmount(0), common.NewAmount(6), common.NewAmount(2), nil},
		{"Send zero with no tip", common.NewAmount(0), common.NewAmount(0), "", common.NewAmount(0), common.NewAmount(0), common.NewAmount(0), common.NewAmount(0), ErrInvalidAmount},
		{"Send zero with tips", common.NewAmount(0), common.NewAmount(2), "", common.NewAmount(0), common.NewAmount(0), common.NewAmount(0), common.NewAmount(0), ErrInvalidAmount},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {

			store := storage.NewRamStorage()
			defer store.Close()

			// Create a account address
			senderAccount, err := CreateAccount(GetTestAccountPath(), "test")
			if err != nil {
				panic(err)
			}
			node := network.FakeNodeWithPidAndAddr(store, "test", "test")
			// Create a PoW blockchain with the sender wallet's address as the coinbase address
			// i.e. sender's wallet would have mineReward amount after blockchain created
			bc, pow := createBlockchain(senderAccount.GetKeyPair().GenerateAddress(), store, core.NewTransactionPool(node, 128))
			pool := core.NewBlockPool()

			bm := core.NewBlockChainManager(bc, pool, node)

			// Create a receiver account; Balance is 0 initially
			receiverAccount, err := CreateAccount(GetTestAccountPath(), "test")
			if err != nil {
				panic(err)
			}

			// Send coins from senderAccount to receiverAccount
			var rcvAddr account.Address
			isContract := (tc.contract != "")
			if isContract {
				rcvAddr = account.NewAddress("")
			} else {
				rcvAddr = receiverAccount.GetKeyPair().GenerateAddress()
			}

			_, _, err = Send(senderAccount, rcvAddr, tc.transferAmount, tc.tipAmount, tc.gasLimit, tc.gasPrice, tc.contract, bc)

			assert.Equal(t, tc.expectedErr, err)

			// Create a miner account; Balance is 0 initially
			minerAccount, err := CreateAccount(GetTestAccountPath(), "test")
			if err != nil {
				panic(err)
			}

			//a short delay before mining starts
			time.Sleep(time.Millisecond * 500)

			// Make sender the miner and mine for 1 block (which should include the transaction)
			pow.Setup(node, minerAccount.GetKeyPair().GenerateAddress().String(), bm)
			pow.Start()
			for bc.GetMaxHeight() < 1 {
			}
			pow.Stop()
			util.WaitDoneOrTimeout(func() bool {
				return !pow.IsProducingBlock()
			}, 20)
			// Verify balance of sender's account (genesis "mineReward" - transferred amount)
			senderBalance, err := GetBalance(senderAccount.GetKeyPair().GenerateAddress(), bc)
			if err != nil {
				panic(err)
			}
			expectedBalance, _ := mineReward.Sub(tc.expectedTransfer)
			expectedBalance, _ = expectedBalance.Sub(tc.expectedTip)
			assert.Equal(t, expectedBalance, senderBalance)

			// Balance of the miner's account should be the amount tipped + mineReward
			minerBalance, err := GetBalance(minerAccount.GetKeyPair().GenerateAddress(), bc)
			if err != nil {
				panic(err)
			}
			assert.Equal(t, mineReward.Times(bc.GetMaxHeight()).Add(tc.expectedTip), minerBalance)

			//check smart contract deployment
			res := string("")
			contractAddr := account.NewAddress("")
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

			// Balance of the receiver's account should be the amount transferred
			var receiverBalance *common.Amount
			if isContract {
				receiverBalance, err = GetBalance(contractAddr, bc)
			} else {
				receiverBalance, err = GetBalance(receiverAccount.GetKeyPair().GenerateAddress(), bc)
			}
			assert.Equal(t, tc.expectedTransfer, receiverBalance)
		})
	}
}

//test send to invalid address
func TestSendToInvalidAddress(t *testing.T) {
	cleanUpDatabase()

	store := storage.NewRamStorage()
	defer store.Close()

	//this is internally set. Dont modify
	mineReward := common.NewAmount(10000000)
	//Transfer ammount
	transferAmount := common.NewAmount(25)
	tip := common.NewAmount(5)
	//create a account address
	account1, err := CreateAccount(GetTestAccountPath(), "test")
	assert.NotEmpty(t, account1)
	addr1 := account1.GetKeyPair().GenerateAddress()

	node := network.FakeNodeWithPidAndAddr(store, "test", "test")
	//create a blockchain
	bc, err := CreateBlockchain(addr1, store, nil, core.NewTransactionPool(node, 128), nil, 1000000)
	assert.Nil(t, err)
	assert.NotNil(t, bc)

	//The balance should be 10 after creating a blockchain
	balance1, err := GetBalance(addr1, bc)
	assert.Nil(t, err)
	assert.Equal(t, mineReward, balance1)
	//pool := core.NewBlockPool()

	//bm := core.NewBlockChainManager(bc, pool, node)

	//Send 5 coins from addr1 to an invalid address
	_, _, err = Send(account1, account.NewAddress(InvalidAddress), transferAmount, tip, common.NewAmount(0), common.NewAmount(0), "", bc)

	assert.NotNil(t, err)

	//the balance of the first account should be still be 10
	balance1, err = GetBalance(addr1, bc)
	assert.Nil(t, err)
	assert.Equal(t, mineReward, balance1)

	cleanUpDatabase()
}

//insufficient fund
func TestSendInsufficientBalance(t *testing.T) {
	cleanUpDatabase()

	store := storage.NewRamStorage()
	defer store.Close()

	tip := common.NewAmount(5)

	//this is internally set. Dont modify
	mineReward := common.NewAmount(10000000)
	//Transfer ammount is larger than the balance
	transferAmount := common.NewAmount(250000000)

	//create a account address
	account1, err := CreateAccount(GetTestAccountPath(), "test")
	assert.NotEmpty(t, account1)
	addr1 := account1.GetKeyPair().GenerateAddress()

	node := network.FakeNodeWithPidAndAddr(store, "test", "test")

	//create a blockchain
	bc, err := CreateBlockchain(addr1, store, nil, core.NewTransactionPool(node, 128), nil, 1000000)
	assert.Nil(t, err)
	assert.NotNil(t, bc)

	//The balance should be 10 after creating a blockchain
	balance1, err := GetBalance(addr1, bc)
	assert.Nil(t, err)
	assert.Equal(t, mineReward, balance1)

	//Create a second account
	account2, err := CreateAccount(GetTestAccountPath(), "test")
	assert.NotEmpty(t, account2)
	assert.Nil(t, err)
	addr2 := account2.GetKeyPair().GenerateAddress()

	//The balance should be 0
	balance2, err := GetBalance(addr2, bc)
	assert.Nil(t, err)
	assert.Equal(t, common.NewAmount(0), balance2)
	//pool := core.NewBlockPool()

	//bm := core.NewBlockChainManager(bc, pool, node)

	//Send 5 coins from addr1 to addr2
	_, _, err = Send(account1, addr2, transferAmount, tip, common.NewAmount(0), common.NewAmount(0), "", bc)

	assert.NotNil(t, err)

	//the balance of the first account should be still be 10
	balance1, err = GetBalance(addr1, bc)
	assert.Nil(t, err)
	assert.Equal(t, mineReward, balance1)

	//the balance of the second account should be 0
	balance2, err = GetBalance(addr2, bc)
	assert.Nil(t, err)
	assert.Equal(t, common.NewAmount(0), balance2)

	cleanUpDatabase()
}

func TestBlockMsgRelaySingleMiner(t *testing.T) {
	const (
		timeBetweenBlock = 1
	)
	cleanUpDatabase()
	var dposArray []*consensus.DPOS
	var bcs []*core.Blockchain
	var nodes []*network.Node

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

		db := storage.NewRamStorage()
		node := network.NewNode(db, nil)
		node.Start(testport_msg_relay_port1+i, "")

		bc := core.CreateBlockchain(account.NewAddress(producerAddrs[0]), db, dpos, core.NewTransactionPool(node, 128), nil, 100000)
		bcs = append(bcs, bc)
		pool := core.NewBlockPool()

		nodes = append(nodes, node)
		bm := core.NewBlockChainManager(bc, pool, node)

		dpos.Setup(node, producerAddrs[0], bm)
		dpos.SetKey(producerKey[0])
		dposArray = append(dposArray, dpos)
	}
	//each node connects to the subsequent node only
	for i := 0; i < len(nodes)-1; i++ {
		connectNodes(nodes[i], nodes[i+1])
	}

	//firstNode Starts Mining
	dposArray[0].Start()
	util.WaitDoneOrTimeout(func() bool {
		return bcs[0].GetMaxHeight() >= 5
	}, 8)
	dposArray[0].Stop()

	//expect every node should have # of entries in dapmsg cache equal to their blockchain height
	for i := 0; i < len(nodes)-1; i++ {
		assert.Equal(t, bcs[i].GetMaxHeight(), bcs[i+1].GetMaxHeight())
	}

	for _, node := range nodes {
		node.Stop()
	}
	cleanUpDatabase()
}

// Test if network radiation bounces forever
func TestBlockMsgRelayMeshNetworkMultipleMiners(t *testing.T) {
	const (
		timeBetweenBlock = 1
		dposRounds       = 2
		bufferTime       = 0
	)
	cleanUpDatabase()
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

		db := storage.NewRamStorage()
		node := network.NewNode(db, nil)
		node.Start(testport_msg_relay_port2+i, "")
		bc := core.CreateBlockchain(account.NewAddress(producerAddrs[0]), storage.NewRamStorage(), dpos, core.NewTransactionPool(node, 128), nil, 100000)
		bcs = append(bcs, bc)
		pool := core.NewBlockPool()

		if i == 0 {
			firstNode = node
		} else {
			node.GetNetwork().ConnectToSeed(firstNode.GetHostPeerInfo())
		}
		nodes = append(nodes, node)

		bm := core.NewBlockChainManager(bc, pool, node)

		dpos.Setup(node, producerAddrs[0], bm)
		dpos.SetKey(producerKey[0])
		dposArray = append(dposArray, dpos)
	}

	//each node connects to every other node
	for i := range nodes {
		for j := range nodes {
			if i != j {
				connectNodes(nodes[i], nodes[j])
			}
		}
	}

	//firstNode Starts Mining
	for _, dpos := range dposArray {
		dpos.Start()
	}

	time.Sleep(time.Second * time.Duration(dynasty.GetDynastyTime()*dposRounds+bufferTime))

	for _, dpos := range dposArray {
		dpos.Stop()
	}
	//expect every node should have # of entries in dapmsg cache equal to their blockchain height
	for i := 0; i < len(nodes)-1; i++ {
		assert.Equal(t, bcs[i].GetMaxHeight(), bcs[i+1].GetMaxHeight())
	}

	for _, node := range nodes {
		node.Stop()
	}

	cleanUpDatabase()
}

func TestForkChoice(t *testing.T) {
	var pows []*consensus.ProofOfWork
	var bcs []*core.Blockchain
	var bms []*core.BlockChainManager
	var dbs []storage.Storage
	var pools []*core.BlockPool
	// Remember to close all opened databases after test
	defer func() {
		for _, db := range dbs {
			db.Close()
		}
	}()

	addr := account.NewAddress("17DgRtQVvaytkiKAfXx9XbV23MESASSwUz")
	//wait for mining for at least "targetHeight" blocks
	//targetHeight := uint64(4)
	//num of nodes to be created in the test
	numOfNodes := 2
	var nodes []*network.Node
	for i := 0; i < numOfNodes; i++ {
		db := storage.NewRamStorage()
		dbs = append(dbs, db)

		node := network.NewNode(db, nil)
		bc, pow := createBlockchain(addr, db, core.NewTransactionPool(node, 128))
		bcs = append(bcs, bc)
		pool := core.NewBlockPool()
		pools = append(pools, pool)

		bm := core.NewBlockChainManager(bc, pool, node)

		pow.Setup(node, addr.String(), bm)
		pow.SetTargetBit(10)
		node.Start(testport_fork+i, "")
		pows = append(pows, pow)
		nodes = append(nodes, node)
		bms = append(bms, bm)
	}
	defer nodes[0].Stop()
	defer nodes[1].Stop()

	// Mine more blocks on node[0] than on node[1]
	pows[1].Start()
	util.WaitDoneOrTimeout(func() bool {
		return bcs[1].GetMaxHeight() > 4
	}, 10)
	pows[1].Stop()
	desiredHeight := uint64(10)
	if bcs[1].GetMaxHeight() > 10 {
		desiredHeight = bcs[1].GetMaxHeight() + 1
	}
	pows[0].Start()
	util.WaitDoneOrTimeout(func() bool {
		return bcs[0].GetMaxHeight() > desiredHeight
	}, 20)
	pows[0].Stop()

	util.WaitDoneOrTimeout(func() bool {
		return !pows[0].IsProducingBlock()
	}, 5)

	// Trigger fork choice in node[1] by broadcasting tail block of node[0]
	tailBlk, _ := bcs[0].GetTailBlock()
	connectNodes(nodes[0], nodes[1])
	bms[0].BroadcastBlock(tailBlk)
	// Make sure syncing starts on node[1]
	util.WaitDoneOrTimeout(func() bool {
		return bcs[1].GetState() == core.BlockchainSync
	}, 10)
	// Make sure syncing ends on node[1]
	util.WaitDoneOrTimeout(func() bool {
		return bcs[1].GetState() != core.BlockchainSync
	}, 20)

	assert.Equal(t, bcs[0].GetMaxHeight(), bcs[1].GetMaxHeight())
	assert.True(t, isSameBlockChain(bcs[0], bcs[1]))
}

func TestForkSegmentHandling(t *testing.T) {
	var pows []*consensus.ProofOfWork
	var bcs []*core.Blockchain
	var dbs []storage.Storage
	var pools []*core.BlockPool
	var bms []*core.BlockChainManager
	// Remember to close all opened databases after test
	defer func() {
		for _, db := range dbs {
			db.Close()
		}
	}()
	addr := account.NewAddress("17DgRtQVvaytkiKAfXx9XbV23MESASSwUz")

	numOfNodes := 2
	var nodes []*network.Node
	for i := 0; i < numOfNodes; i++ {
		db := storage.NewRamStorage()
		dbs = append(dbs, db)
		node := network.NewNode(db, nil)

		bc, pow := createBlockchain(addr, db, core.NewTransactionPool(node, 128))
		bcs = append(bcs, bc)
		pool := core.NewBlockPool()
		pools = append(pools, pool)
		bm := core.NewBlockChainManager(bc, pool, node)

		pow.Setup(node, addr.String(), bm)
		pow.SetTargetBit(10)
		node.Start(testport_fork_segment+i, "")
		pows = append(pows, pow)
		nodes = append(nodes, node)
		bms = append(bms, bm)
	}
	defer nodes[0].Stop()
	defer nodes[1].Stop()

	blk1 := &block.Block{}
	blk2 := &block.Block{}

	// Ensure node[1] mined some blocks
	pows[1].Start()
	util.WaitDoneOrTimeout(func() bool {
		return bcs[1].GetMaxHeight() > 3
	}, 10)
	pows[1].Stop()

	// Ensure node[0] mines more blocks than node[1]
	pows[0].Start()
	util.WaitDoneOrTimeout(func() bool {
		return bcs[0].GetMaxHeight() > 12
	}, 30)
	pows[0].Stop()

	util.WaitDoneOrTimeout(func() bool {
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
	bms[0].BroadcastBlock(blk1)
	// Wait for node[1] to start syncing
	util.WaitDoneOrTimeout(func() bool {
		return bcs[1].GetState() == core.BlockchainSync
	}, 10)

	// node[0] broadcast higher block on the same fork and should trigger another sync on node[1]
	bms[0].BroadcastBlock(blk2)

	// Make sure previous syncing ends
	util.WaitDoneOrTimeout(func() bool {
		return bcs[1].GetState() != core.BlockchainSync
	}, 10)
	// Make sure node[1] is syncing again
	util.WaitDoneOrTimeout(func() bool {
		return bcs[1].GetState() == core.BlockchainSync
	}, 10)
	// Make sure syncing ends
	util.WaitDoneOrTimeout(func() bool {
		return bcs[1].GetState() != core.BlockchainSync
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
			minerKeyPair := account.GenerateKeyPairByPrivateKey(key)
			minerAccount := account.NewAccountByKey(minerKeyPair)

			addr := minerAccount.GetKeyPair().GenerateAddress()
			node := network.FakeNodeWithPidAndAddr(store, "a", "b")

			bc, pow := createBlockchain(addr, store, core.NewTransactionPool(node, 128))

			// Create a new account address for testing
			testAddr := account.NewAddress("dGDrVKjCG3sdXtDUgWZ7Fp3Q97tLhqWivf")

			// Start mining to approve the transaction
			pool := core.NewBlockPool()

			bm := core.NewBlockChainManager(bc, pool, node)

			SetMinerKeyPair(key)
			pow.Setup(node, addr.String(), bm)
			pow.SetTargetBit(0)
			pow.Start()

			for bc.GetMaxHeight() <= 1 {
			}

			// Add `addAmount` to the balance of the new account
			_, _, err := SendFromMiner(testAddr, tc.addAmount, bc)
			height := bc.GetMaxHeight()
			assert.Equal(t, err, tc.expectedErr)
			for bc.GetMaxHeight()-height <= 1 {
			}

			pow.Stop()

			// The account balance should be the expected difference
			balance, err := GetBalance(testAddr, bc)
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
			db := storage.NewRamStorage()
			defer db.Close()

			// Create a coinbase wallet address
			addr := account.NewAddress("dG6HhzSdA5m7KqvJNszVSf8i5f4neAteSs")
			node := network.FakeNodeWithPidAndAddr(db, "a", "b")
			// Create a blockchain
			bc, err := CreateBlockchain(addr, db, nil, core.NewTransactionPool(node, 128), nil, 1000000)
			assert.Nil(t, err)

			_, _, err = SendFromMiner(account.NewAddress(tc.address), common.NewAmount(8), bc)
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

	// Create a account address
	senderAccount, err := CreateAccount(GetTestAccountPath(), "test")
	assert.Nil(t, err)
	node := network.FakeNodeWithPidAndAddr(store, "test", "test")
	bc, pow := createBlockchain(senderAccount.GetKeyPair().GenerateAddress(), store, core.NewTransactionPool(node, 128))
	pool := core.NewBlockPool()
	bm := core.NewBlockChainManager(bc, pool, node)

	//deploy smart contract
	_, _, err = Send(senderAccount, account.NewAddress(""), common.NewAmount(1), common.NewAmount(0), common.NewAmount(10000), common.NewAmount(1), contract, bc)

	assert.Nil(t, err)

	txp := bc.GetTxPool().GetTransactions()[0]
	contractAddr := txp.GetContractAddress()

	// Create a miner account; Balance is 0 initially
	minerAccount, err := CreateAccount(GetTestAccountPath(), "test")
	if err != nil {
		panic(err)
	}

	//a short delay before mining starts
	time.Sleep(time.Millisecond * 500)

	// Make sender the miner and mine for 1 block (which should include the transaction)
	pow.Setup(node, minerAccount.GetKeyPair().GenerateAddress().String(), bm)
	pow.Start()
	for bc.GetMaxHeight() < 1 {
	}
	pow.Stop()

	//a short delay before mining starts
	time.Sleep(time.Millisecond * 500)

	//store data
	functionCall := `{"function":"set","args":["testKey","222"]}`
	_, _, err = Send(senderAccount, contractAddr, common.NewAmount(1), common.NewAmount(0), common.NewAmount(100), common.NewAmount(1), functionCall, bc)

	assert.Nil(t, err)
	pow.Start()
	for bc.GetMaxHeight() < 1 {
	}
	pow.Stop()

	//get data
	functionCall = `{"function":"get","args":["testKey"]}`
	_, _, err = Send(senderAccount, contractAddr, common.NewAmount(1), common.NewAmount(0), common.NewAmount(100), common.NewAmount(1), functionCall, bc)

	assert.Nil(t, err)
	pow.Start()
	for bc.GetMaxHeight() < 1 {
	}
	pow.Stop()
}

func connectNodes(node1 *network.Node, node2 *network.Node) {
	node1.GetNetwork().ConnectToSeed(node2.GetHostPeerInfo())
}

func setupNode(addr account.Address, pow *consensus.ProofOfWork, bc *core.Blockchain, port int) *network.Node {
	pool := core.NewBlockPool()

	node := network.NewNode(bc.GetDb(), nil)
	bm := core.NewBlockChainManager(bc, pool, node)

	pow.Setup(node, addr.String(), bm)
	pow.SetTargetBit(12)
	node.Start(port, "")
	defer node.Stop()
	return node
}

func createBlockchain(addr account.Address, db *storage.RamStorage, txPool *core.TransactionPool) (*core.Blockchain, *consensus.ProofOfWork) {
	pow := consensus.NewProofOfWork()
	return core.CreateBlockchain(addr, db, pow, txPool, nil, 100000), pow
}

func TestDoubleMint(t *testing.T) {
	var sendNode *network.Node
	var recvNode *network.Node
	var recvNodeBc *core.Blockchain
	var blks []*block.Block
	var parent *block.Block
	var dposArray []*consensus.DPOS
	var sendBm *core.BlockChainManager

	validProducerAddr := "dPGZmHd73UpZhrM6uvgnzu49ttbLp4AzU8"
	validProducerKey := "5a66b0fdb69c99935783059bb200e86e97b506ae443a62febd7d0750cd7fac55"

	dynasty := consensus.NewDynasty([]string{validProducerAddr}, len([]string{validProducerAddr}), 15)
	producerHash, _ := account.GeneratePubKeyHashByAddress(account.NewAddress(validProducerAddr))
	tx := &core.Transaction{nil, []core.TXInput{{[]byte{}, -1, nil, nil}}, []core.TXOutput{{common.NewAmount(0), account.PubKeyHash(producerHash), ""}}, common.NewAmount(0), common.NewAmount(0), common.NewAmount(0)}

	for i := 0; i < 3; i++ {
		blk := createValidBlock([]*core.Transaction{tx}, validProducerKey, validProducerAddr, parent)
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

		db := storage.NewRamStorage()
		node := network.NewNode(db, nil)
		node.Start(testport_msg_relay_port3+i, "")

		bc := core.CreateBlockchain(account.NewAddress(validProducerAddr), db, dpos, core.NewTransactionPool(node, 128), nil, 100000)
		pool := core.NewBlockPool()

		bm := core.NewBlockChainManager(bc, pool, node)

		dpos.Setup(node, validProducerAddr, bm)
		dpos.SetKey(validProducerKey)
		if i == 0 {
			sendNode = node
			sendBm = bm
		} else {
			recvNode = node
			recvNode.GetNetwork().ConnectToSeed(sendNode.GetHostPeerInfo())
			recvNodeBc = bc
		}
		dposArray = append(dposArray, dpos)
	}

	defer recvNode.Stop()
	defer sendNode.Stop()

	for _, blk := range blks {
		sendBm.BroadcastBlock(blk)
	}

	time.Sleep(time.Second * 2)
	assert.True(t, recvNodeBc.GetMaxHeight() < 2)
	assert.False(t, dposArray[1].Validate(blks[1]))
}

func createValidBlock(tx []*core.Transaction, validProducerKey, validProducerAddr string, parent *block.Block) *block.Block {
	blk := block.NewBlock(tx, parent, validProducerAddr)
	blk.SetHash(block_logic.CalculateHashWithNonce(blk))
	block_logic.SignBlock(blk, validProducerKey)
	return blk
}

func TestSimultaneousSyncingAndBlockProducing(t *testing.T) {
	const genesisAddr = "121yKAXeG4cw6uaGCBYjWk9yTWmMkhcoDD"

	validProducerAddress := "dPGZmHd73UpZhrM6uvgnzu49ttbLp4AzU8"
	validProducerKey := "5a66b0fdb69c99935783059bb200e86e97b506ae443a62febd7d0750cd7fac55"
	//fmt.Println(validProducerAddress, validProducerKey)
	conss := consensus.NewDPOS()
	dynasty := consensus.NewDynasty([]string{validProducerAddress}, 1, 1)
	conss.SetDynasty(dynasty)

	db := storage.NewRamStorage()
	seedNode := network.NewNode(db, nil)
	seedNode.Start(testport_fork_syncing, "")
	defer seedNode.Stop()

	bc := core.CreateBlockchain(account.NewAddress(genesisAddr), storage.NewRamStorage(), conss, core.NewTransactionPool(seedNode, 128), nil, 100000)

	//create and start seed node
	pool := core.NewBlockPool()
	bm := core.NewBlockChainManager(bc, pool, seedNode)

	conss.Setup(seedNode, validProducerAddress, bm)
	conss.SetKey(validProducerKey)

	// seed node start mining
	conss.Start()
	util.WaitDoneOrTimeout(func() bool {
		return bc.GetMaxHeight() > 8
	}, 10)

	// set up another node for syncing
	dpos := consensus.NewDPOS()
	dpos.SetDynasty(dynasty)

	db1 := storage.NewRamStorage()
	node := network.NewNode(db1, nil)
	node.Start(testport_fork_syncing+1, "")
	defer node.Stop()

	bc1 := core.CreateBlockchain(account.NewAddress(genesisAddr), db1, dpos, core.NewTransactionPool(node, 128), nil, 100000)

	pool1 := core.NewBlockPool()

	bm1 := core.NewBlockChainManager(bc1, pool1, node)

	dpos.Setup(node, validProducerAddress, bm1)
	dpos.SetKey(validProducerKey)

	// Trigger fork choice in node by broadcasting tail block of node[0]
	tailBlk, _ := bc.GetTailBlock()

	connectNodes(seedNode, node)
	bm.BroadcastBlock(tailBlk)

	time.Sleep(time.Second * 5)
	conss.Stop()
	assert.True(t, bc.GetMaxHeight()-bc1.GetMaxHeight() <= 1)
}
