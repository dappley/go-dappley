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
package intergationTest

import (
	"fmt"
	"github.com/dappley/go-dappley/logic/downloadmanager"
	"reflect"
	"testing"
	"time"

	"github.com/dappley/go-dappley/common/deadline"
	"github.com/dappley/go-dappley/logic/blockproducer/mocks"
	blockchainMock "github.com/dappley/go-dappley/logic/lblockchain/mocks"
	"github.com/dappley/go-dappley/logic/ltransaction"
	"github.com/stretchr/testify/mock"

	"github.com/dappley/go-dappley/logic/blockproducer"

	"github.com/dappley/go-dappley/core/block"
	"github.com/dappley/go-dappley/core/blockchain"
	"github.com/dappley/go-dappley/core/blockproducerinfo"
	"github.com/dappley/go-dappley/core/transaction"
	"github.com/dappley/go-dappley/core/transactionbase"
	"github.com/dappley/go-dappley/core/utxo"
	"github.com/dappley/go-dappley/logic"
	"github.com/dappley/go-dappley/logic/lblock"
	"github.com/dappley/go-dappley/logic/lblockchain"
	"github.com/dappley/go-dappley/logic/lutxo"
	"github.com/dappley/go-dappley/logic/transactionpool"
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
const InvalidAddress = "Invalid Address"

//test logic.Send
func TestSend(t *testing.T) {
	var mineReward = transaction.Subsidy
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
		{"Send zero with no tip", common.NewAmount(0), common.NewAmount(0), "", common.NewAmount(0), common.NewAmount(0), common.NewAmount(0), common.NewAmount(0), logic.ErrInvalidAmount},
		{"Send zero with tips", common.NewAmount(0), common.NewAmount(2), "", common.NewAmount(0), common.NewAmount(0), common.NewAmount(0), common.NewAmount(0), logic.ErrInvalidAmount},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {

			store := storage.NewRamStorage()
			defer store.Close()

			// Create a account address
			senderAccount, err := logic.CreateAccountWithPassphrase("test", logic.GetTestAccountPath())
			if err != nil {
				panic(err)
			}
			// Create a miner account; Balance is 0 initially
			minerAccount, err := logic.CreateAccountWithPassphrase("test", logic.GetTestAccountPath())
			if err != nil {
				panic(err)
			}

			node := network.FakeNodeWithPidAndAddr(store, "test", "test")
			// Create a PoW blockchain with the logic.Sender wallet's address as the coinbase address
			// i.e. logic.Sender's wallet would have mineReward amount after blockchain created
			bm, bp := CreateProducer(minerAccount.GetAddress(), senderAccount.GetAddress(), store, transactionpool.NewTransactionPool(node, 128), node)

			// Create a receiver account; Balance is 0 initially
			receiverAccount, err := logic.CreateAccountWithPassphrase("test", logic.GetTestAccountPath())
			if err != nil {
				panic(err)
			}

			// logic.Send coins from logic.SenderAccount to receiverAccount
			var rcvAddr account.Address
			isContract := tc.contract != ""
			if isContract {
				rcvAddr = account.NewAddress("")
			} else {
				rcvAddr = receiverAccount.GetAddress()
			}

			_, _, err = logic.Send(senderAccount, rcvAddr, tc.transferAmount, tc.tipAmount, tc.gasLimit, tc.gasPrice, tc.contract, bm.Getblockchain())

			assert.Equal(t, tc.expectedErr, err)

			//a short delay before mining starts
			time.Sleep(time.Millisecond * 500)

			// Make logic.Sender the miner and mine for 1 block (which should include the transaction)
			bp.Start()
			for bm.Getblockchain().GetMaxHeight() < 1 {
			}

			bp.Stop()
			util.WaitDoneOrTimeout(func() bool {
				return !bp.IsProducingBlock()
			}, 20)

			// Verify balance of logic.Sender's account (genesis "mineReward" - transferred amount)
			senderBalance, err := logic.GetBalance(senderAccount.GetAddress(), bm.Getblockchain())

			if err != nil {
				panic(err)
			}
			expectedBalance, _ := mineReward.Sub(tc.expectedTransfer)
			expectedBalance, _ = expectedBalance.Sub(tc.expectedTip)
			assert.Equal(t, expectedBalance, senderBalance)

			// Balance of the miner's account should be the amount tipped + mineReward
			minerBalance, err := logic.GetBalance(minerAccount.GetAddress(), bm.Getblockchain())
			if err != nil {
				panic(err)
			}
			assert.Equal(t, mineReward.Times(bm.Getblockchain().GetMaxHeight()).Add(tc.expectedTip), minerBalance)

			//check smart contract deployment
			res := string("")
			contractAddr := account.NewAddress("")
		loop:
			for i := bm.Getblockchain().GetMaxHeight(); i > 0; i-- {
				blk, err := bm.Getblockchain().GetBlockByHeight(i)
				assert.Nil(t, err)
				for _, tx := range blk.GetTransactions() {
					ctx := ltransaction.NewTxContract(tx)
					if ctx != nil {
						contractAddr = ctx.GetContractAddress()
						res = ctx.GetContract()
						break loop
					}
				}
			}
			assert.Equal(t, tc.contract, res)

			// Balance of the receiver's account should be the amount transferred
			var receiverBalance *common.Amount
			if isContract {
				receiverBalance, err = logic.GetBalance(contractAddr, bm.Getblockchain())
			} else {
				receiverBalance, err = logic.GetBalance(receiverAccount.GetAddress(), bm.Getblockchain())
			}
			assert.Equal(t, tc.expectedTransfer, receiverBalance)
		})
	}
	logic.RemoveAccountTestFile()
}

//test logic.Send to invalid address
func TestSendToInvalidAddress(t *testing.T) {
	store := storage.NewRamStorage()
	defer store.Close()

	//this is internally set. Dont modify
	mineReward := transaction.Subsidy
	//Transfer ammount
	transferAmount := common.NewAmount(25)
	tip := common.NewAmount(5)
	//create a account address
	account1, err := logic.CreateAccountWithPassphrase("test", logic.GetTestAccountPath())
	assert.NotEmpty(t, account1)
	addr1 := account1.GetAddress()

	node := network.FakeNodeWithPidAndAddr(store, "test", "test")
	//create a blockchain
	bc, err := logic.CreateBlockchain(addr1, store, nil, transactionpool.NewTransactionPool(node, 128), nil, 1000000)
	assert.Nil(t, err)
	assert.NotNil(t, bc)

	//The balance should be 10 after creating a blockchain
	balance1, err := logic.GetBalance(addr1, bc)
	assert.Nil(t, err)
	assert.Equal(t, mineReward, balance1)
	//pool := blockchain.NewBlockPool(nil)

	//bm := lblockchain.NewBlockchainManager(bc, pool, node)

	//logic.Send 5 coins from addr1 to an invalid address
	_, _, err = logic.Send(account1, account.NewAddress(InvalidAddress), transferAmount, tip, common.NewAmount(0), common.NewAmount(0), "", bc)

	assert.NotNil(t, err)

	//the balance of the first account should be still be 10
	balance1, err = logic.GetBalance(addr1, bc)
	assert.Nil(t, err)
	assert.Equal(t, mineReward, balance1)

	logic.RemoveAccountTestFile()
}

//insufficient fund
func TestSendInsufficientBalance(t *testing.T) {
	logic.RemoveAccountTestFile()

	store := storage.NewRamStorage()
	defer store.Close()

	tip := common.NewAmount(5)

	//this is internally set. Dont modify
	mineReward := transaction.Subsidy
	//Transfer ammount is larger than the balance
	transferAmount := common.NewAmount(25000000000)

	//create a account address
	account1, err := logic.CreateAccountWithPassphrase("test", logic.GetTestAccountPath())
	assert.NotEmpty(t, account1)
	addr1 := account1.GetAddress()

	node := network.FakeNodeWithPidAndAddr(store, "test", "test")

	//create a blockchain
	bc, err := logic.CreateBlockchain(addr1, store, nil, transactionpool.NewTransactionPool(node, 128), nil, 1000000)
	assert.Nil(t, err)
	assert.NotNil(t, bc)

	//The balance should be 10 after creating a blockchain
	balance1, err := logic.GetBalance(addr1, bc)
	assert.Nil(t, err)
	assert.Equal(t, mineReward, balance1)

	//Create a second account
	account2, err := logic.CreateAccountWithPassphrase("test", logic.GetTestAccountPath())
	assert.NotEmpty(t, account2)
	assert.Nil(t, err)
	addr2 := account2.GetAddress()

	//The balance should be 0
	balance2, err := logic.GetBalance(addr2, bc)
	assert.Nil(t, err)
	assert.Equal(t, common.NewAmount(0), balance2)
	//pool := blockchain.NewBlockPool(nil)

	//bm := lblockchain.NewBlockchainManager(bc, pool, node)

	//logic.Send 5 coins from addr1 to addr2
	_, _, err = logic.Send(account1, addr2, transferAmount, tip, common.NewAmount(0), common.NewAmount(0), "", bc)
	assert.NotNil(t, err)

	//the balance of the first account should be still be 10
	balance1, err = logic.GetBalance(addr1, bc)
	assert.Nil(t, err)
	assert.Equal(t, mineReward, balance1)

	//the balance of the second account should be 0
	balance2, err = logic.GetBalance(addr2, bc)
	assert.Nil(t, err)
	assert.Equal(t, common.NewAmount(0), balance2)

	logic.RemoveAccountTestFile()
}

func TestDownloadRequestCh(t *testing.T) {
	var bps []*blockproducer.BlockProducer
	var bms []*lblockchain.BlockchainManager
	var dms []*downloadmanager.DownloadManager
	var dbs []storage.Storage
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
		bm, bp := CreateProducer(addr, addr, db, transactionpool.NewTransactionPool(node, 128), node)
		dm := downloadmanager.NewDownloadManager(node, bm, 2, nil)

		node.Start(testport_fork+i, "")
		nodes = append(nodes, node)
		bms = append(bms, bm)
		bps = append(bps, bp)
		dms = append(dms, dm)
	}
	defer nodes[0].Stop()
	defer nodes[1].Stop()

	desiredHeight := uint64(10)
	bms[0].Getblockchain().SetState(blockchain.BlockchainReady)
	bps[0].Start()
	util.WaitDoneOrTimeout(func() bool {
		return bms[0].Getblockchain().GetMaxHeight() > desiredHeight
	}, 10)
	bps[0].Stop()

	connectNodes(nodes[0], nodes[1])
	//a short delay for connect
	time.Sleep(time.Millisecond * 500)
	util.WaitDone(func() bool {
		return bps[0].GetProduceBlockStatus()
	})
	// node[1] start downloading
	dms[1].Start()
	bms[1].RequestDownloadBlockchain()

	// Make sure downloading starts on node[1]
	util.WaitDoneOrTimeout(func() bool {
		return bms[1].Getblockchain().GetState() == blockchain.BlockchainDownloading
	}, 10)

	// Make sure downloading ends on node[1]
	util.WaitDoneOrTimeout(func() bool {
		return bms[1].Getblockchain().GetState() != blockchain.BlockchainDownloading
	}, 30)

	assert.Equal(t, bms[0].Getblockchain().GetMaxHeight(), bms[1].Getblockchain().GetMaxHeight())
	assert.True(t, isSameBlockChain(bms[0].Getblockchain(), bms[1].Getblockchain()))
}

func TestForkChoice(t *testing.T) {
	var bps []*blockproducer.BlockProducer
	var bms []*lblockchain.BlockchainManager
	var dbs []storage.Storage
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
		bm, bp := CreateProducer(addr, addr, db, transactionpool.NewTransactionPool(node, 128), node)

		node.Start(testport_fork+i, "")
		nodes = append(nodes, node)
		bms = append(bms, bm)
		bps = append(bps, bp)
	}
	defer nodes[0].Stop()
	defer nodes[1].Stop()

	// Mine more blocks on node[0] than on node[1]
	bps[1].Start()
	util.WaitDoneOrTimeout(func() bool {
		return bms[1].Getblockchain().GetMaxHeight() > 4
	}, 10)
	bps[1].Stop()
	util.WaitDone(func() bool {
		return bps[1].GetProduceBlockStatus()
	})

	desiredHeight := uint64(10)
	if bms[1].Getblockchain().GetMaxHeight() > 10 {
		desiredHeight = bms[1].Getblockchain().GetMaxHeight() + 1
	}
	bps[0].Start()
	util.WaitDoneOrTimeout(func() bool {
		return bms[0].Getblockchain().GetMaxHeight() > desiredHeight
	}, 10)
	bps[0].Stop()

	util.WaitDone(func() bool {
		return bps[0].GetProduceBlockStatus()
	})
	// Trigger fork choice in node[1] by broadcasting tail block of node[0]
	tailBlk, _ := bms[0].Getblockchain().GetTailBlock()
	connectNodes(nodes[0], nodes[1])
	bms[0].BroadcastBlock(tailBlk)
	// Make sure syncing starts on node[1]
	util.WaitDoneOrTimeout(func() bool {
		return bms[1].Getblockchain().GetState() == blockchain.BlockchainSync
	}, 10)
	// Make sure syncing ends on node[1]
	util.WaitDoneOrTimeout(func() bool {
		return bms[1].Getblockchain().GetState() != blockchain.BlockchainSync
	}, 30)

	assert.Equal(t, bms[0].Getblockchain().GetMaxHeight(), bms[1].Getblockchain().GetMaxHeight())
	assert.True(t, isSameBlockChain(bms[0].Getblockchain(), bms[1].Getblockchain()))
}

func TestForkSegmentHandling(t *testing.T) {
	var bms []*lblockchain.BlockchainManager
	var dbs []storage.Storage
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
		bm, _ := CreateProducer(addr, addr, db, transactionpool.NewTransactionPool(node, 128), node)

		node.Start(testport_fork+i, "")
		nodes = append(nodes, node)
		bms = append(bms, bm)
	}
	defer nodes[0].Stop()
	defer nodes[1].Stop()

	blk1 := &block.Block{}
	blk2 := &block.Block{}

	lblockchain.AddBlockToGeneratedBlockchain(bms[0].Getblockchain(), 12)

	lblockchain.AddBlockToGeneratedBlockchain(bms[1].Getblockchain(), 3)

	// Pick 2 blocks from blockchain[0] which can trigger syncing on node[1]
	blk1, _ = bms[0].Getblockchain().GetBlockByHeight(7)
	blk2, _ = bms[0].Getblockchain().GetTailBlock()

	connectNodes(nodes[0], nodes[1])
	fmt.Println(bms[0].Getblockchain().GetMaxHeight())
	fmt.Println(bms[1].Getblockchain().GetMaxHeight())
	bms[0].BroadcastBlock(blk1)
	// Wait for node[1] to start syncing
	util.WaitDoneOrTimeout(func() bool {
		return bms[1].Getblockchain().GetState() == blockchain.BlockchainSync
	}, 10)
	// Make sure previous syncing ends
	util.WaitDoneOrTimeout(func() bool {
		return bms[1].Getblockchain().GetState() != blockchain.BlockchainSync
	}, 10)
	// node[0] broadcast higher block on the same fork and should trigger another sync on node[1]
	bms[0].BroadcastBlock(blk2)

	// Make sure node[1] is syncing again
	util.WaitDoneOrTimeout(func() bool {
		return bms[1].Getblockchain().GetState() == blockchain.BlockchainSync
	}, 10)
	// Make sure syncing ends
	util.WaitDoneOrTimeout(func() bool {
		return bms[1].Getblockchain().GetState() != blockchain.BlockchainSync
	}, 10)

	assert.True(t, isSameBlockChain(bms[0].Getblockchain(), bms[1].Getblockchain()))
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
		{"Add zero", common.NewAmount(0), common.NewAmount(0), logic.ErrInvalidAmount},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {

			store := storage.NewRamStorage()
			defer store.Close()

			// Create a coinbase address
			key := "bb23d2ff19f5b16955e8a24dca34dd520980fe3bddca2b3e1b56663f0ec1aa7e"
			minerKeyPair := account.GenerateKeyPairByPrivateKey(key)
			minerAccount := account.NewAccountByKey(minerKeyPair)

			addr := minerAccount.GetAddress()
			node := network.FakeNodeWithPidAndAddr(store, "a", "b")

			bm, bp := CreateProducer(addr, addr, store, transactionpool.NewTransactionPool(node, 128), node)

			// Create a new account address for testing
			testAddr := account.NewAddress("dGDrVKjCG3sdXtDUgWZ7Fp3Q97tLhqWivf")

			logic.SetMinerKeyPair(key)
			bp.Start()

			for bm.Getblockchain().GetMaxHeight() <= 1 {
			}

			// Add `addAmount` to the balance of the new account
			_, _, err := logic.SendFromMiner(testAddr, tc.addAmount, bm.Getblockchain())
			height := bm.Getblockchain().GetMaxHeight()
			assert.Equal(t, err, tc.expectedErr)
			for bm.Getblockchain().GetMaxHeight()-height <= 1 {
			}

			bp.Stop()

			// The account balance should be the expected difference
			balance, err := logic.GetBalance(testAddr, bm.Getblockchain())
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
			bc, err := logic.CreateBlockchain(addr, db, nil, transactionpool.NewTransactionPool(node, 128), nil, 1000000)
			assert.Nil(t, err)

			_, _, err = logic.SendFromMiner(account.NewAddress(tc.address), common.NewAmount(8), bc)
			assert.Equal(t, logic.ErrInvalidRcverAddress, err)
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
	senderAccount, err := logic.CreateAccountWithPassphrase("test", logic.GetTestAccountPath())
	minerAccount, err := logic.CreateAccountWithPassphrase("test", logic.GetTestAccountPath())
	assert.Nil(t, err)
	node := network.FakeNodeWithPidAndAddr(store, "test", "test")
	bm, bps := CreateProducer(minerAccount.GetAddress(), senderAccount.GetAddress(), store, transactionpool.NewTransactionPool(node, 128), node)

	//deploy smart contract
	_, _, err = logic.Send(senderAccount, account.NewAddress(""), common.NewAmount(1), common.NewAmount(0), common.NewAmount(10000), common.NewAmount(1), contract, bm.Getblockchain())

	assert.Nil(t, err)

	txp := bm.Getblockchain().GetTxPool().GetTransactions()[0]
	contractAddr := ltransaction.NewTxContract(txp).GetContractAddress()

	// Create a miner account; Balance is 0 initially

	if err != nil {
		panic(err)
	}

	//a short delay before mining starts
	time.Sleep(time.Millisecond * 500)

	// Make logic.Sender the miner and mine for 1 block (which should include the transaction)
	bps.Start()
	for bm.Getblockchain().GetMaxHeight() < 1 {
	}
	bps.Stop()

	//a short delay before mining starts
	time.Sleep(time.Millisecond * 500)

	//store data
	functionCall := `{"function":"set","args":["testKey","222"]}`
	_, _, err = logic.Send(senderAccount, contractAddr, common.NewAmount(1), common.NewAmount(0), common.NewAmount(100), common.NewAmount(1), functionCall, bm.Getblockchain())

	assert.Nil(t, err)
	currentHeight := bm.Getblockchain().GetMaxHeight()
	bps.Start()
	for bm.Getblockchain().GetMaxHeight() < currentHeight+1 {
	}
	bps.Stop()

	//get data
	functionCall = `{"function":"get","args":["testKey"]}`
	_, _, err = logic.Send(senderAccount, contractAddr, common.NewAmount(1), common.NewAmount(0), common.NewAmount(100), common.NewAmount(1), functionCall, bm.Getblockchain())

	assert.Nil(t, err)
	currentHeight = bm.Getblockchain().GetMaxHeight()
	bps.Start()
	for bm.Getblockchain().GetMaxHeight() < currentHeight+1 {
	}
	bps.Stop()
	logic.RemoveAccountTestFile()
}

func connectNodes(node1 *network.Node, node2 *network.Node) {
	node1.GetNetwork().ConnectToSeed(node2.GetHostPeerInfo())
}

func CreateProducer(producerAddr, addr account.Address, db *storage.RamStorage, txPool *transactionpool.TransactionPool, node *network.Node) (*lblockchain.BlockchainManager, *blockproducer.BlockProducer) {
	producer := blockproducerinfo.NewBlockProducerInfo(producerAddr.String())

	libPolicy := &blockchainMock.LIBPolicy{}
	libPolicy.On("GetProducers").Return(nil)
	libPolicy.On("GetMinConfirmationNum").Return(6)
	libPolicy.On("IsBypassingLibCheck").Return(true)
	consensus := &blockchainMock.Consensus{}
	consensus.On("Validate", mock.Anything).Return(true)
	bc := lblockchain.CreateBlockchain(addr, db, libPolicy, txPool, nil, 100000)
	bm := lblockchain.NewBlockchainManager(bc, blockchain.NewBlockPool(nil), node, consensus)

	bpConsensus := &mocks.Consensus{}
	bpConsensus.On("Validate", mock.Anything).Return(true)
	bpConsensus.On("ProduceBlock", mock.Anything).Run(func(args mock.Arguments) {
		args.Get(0).(func(process func(*block.Block), deadline deadline.Deadline))(
			func(blk *block.Block) {
				hash := lblock.CalculateHash(blk)
				blk.SetHash(hash)
			},
			deadline.NewUnlimitedDeadline(),
		)
	})

	blockproducer := blockproducer.NewBlockProducer(bm, bpConsensus, producer)
	return bm, blockproducer
}

func TestDoubleMint(t *testing.T) {
	var SendNode *network.Node
	var recvNode *network.Node
	var recvNodeBc *lblockchain.Blockchain
	var blks []*block.Block
	var parent *block.Block
	var dposArray []*consensus.DPOS
	var SendBm *lblockchain.BlockchainManager

	validProducerAddr := "dPGZmHd73UpZhrM6uvgnzu49ttbLp4AzU8"
	validProducerAccount := account.NewTransactionAccountByAddress(account.NewAddress(validProducerAddr))
	validProducerKey := "5a66b0fdb69c99935783059bb200e86e97b506ae443a62febd7d0750cd7fac55"

	dynasty := consensus.NewDynasty([]string{validProducerAddr}, len([]string{validProducerAddr}), 15)
	producerHash := validProducerAccount.GetPubKeyHash()
	tx := &transaction.Transaction{nil, []transactionbase.TXInput{{[]byte{}, -1, nil, nil}}, []transactionbase.TXOutput{{common.NewAmount(0), account.PubKeyHash(producerHash), ""}}, common.NewAmount(0), common.NewAmount(0), common.NewAmount(0), 0, transaction.TxTypeNormal}

	for i := 0; i < 3; i++ {
		blk := createValidBlock([]*transaction.Transaction{tx}, validProducerKey, validProducerAddr, parent)
		blks = append(blks, blk)
		parent = blk
	}
	//check all timestamps are equal
	for i := 0; i < len(blks)-1; i++ {
		assert.True(t, blks[i].GetTimestamp() == blks[i+1].GetTimestamp())
	}
	for i := 0; i < 2; i++ {

		dpos := consensus.NewDPOS(blockproducerinfo.NewBlockProducerInfo(validProducerAddr))
		dpos.SetDynasty(dynasty)

		db := storage.NewRamStorage()
		node := network.NewNode(db, nil)
		node.Start(testport_msg_relay_port3+i, "")

		bc := lblockchain.CreateBlockchain(validProducerAccount.GetAddress(), db, dpos, transactionpool.NewTransactionPool(node, 128), nil, 100000)
		pool := blockchain.NewBlockPool(nil)

		bm := lblockchain.NewBlockchainManager(bc, pool, node, dpos)

		dpos.SetKey(validProducerKey)
		if i == 0 {
			SendNode = node
			SendBm = bm
		} else {
			recvNode = node
			recvNode.GetNetwork().ConnectToSeed(SendNode.GetHostPeerInfo())
			recvNodeBc = bc
		}
		dposArray = append(dposArray, dpos)
	}

	defer recvNode.Stop()
	defer SendNode.Stop()

	for _, blk := range blks {
		SendBm.BroadcastBlock(blk)
	}

	time.Sleep(time.Second * 2)
	assert.True(t, recvNodeBc.GetMaxHeight() < 2)
	assert.False(t, dposArray[1].Validate(blks[1]))
}

func createValidBlock(tx []*transaction.Transaction, validProducerKey, validProducerAddr string, parent *block.Block) *block.Block {
	blk := block.NewBlock(tx, parent, validProducerAddr)
	blk.SetHash(lblock.CalculateHashWithNonce(blk))
	lblock.SignBlock(blk, validProducerKey)
	return blk
}

func TestSimultaneousSyncingAndBlockProducing(t *testing.T) {
	const genesisAddr = "121yKAXeG4cw6uaGCBYjWk9yTWmMkhcoDD"

	validProducerAddress := "dPGZmHd73UpZhrM6uvgnzu49ttbLp4AzU8"
	validProducerKey := "5a66b0fdb69c99935783059bb200e86e97b506ae443a62febd7d0750cd7fac55"

	producer := blockproducerinfo.NewBlockProducerInfo(validProducerAddress)
	dpos1 := consensus.NewDPOS(producer)
	dynasty := consensus.NewDynasty([]string{validProducerAddress}, 1, 1)
	dpos1.SetKey(validProducerKey)
	dpos1.SetDynasty(dynasty)

	db := storage.NewRamStorage()
	seedNode := network.NewNode(db, nil)
	seedNode.Start(testport_fork_syncing, "")
	defer seedNode.Stop()

	bc := lblockchain.CreateBlockchain(account.NewAddress(genesisAddr), db, dpos1, transactionpool.NewTransactionPool(seedNode, 128), nil, 100000)

	//create and start seed node
	pool := blockchain.NewBlockPool(nil)
	bm := lblockchain.NewBlockchainManager(bc, pool, seedNode, dpos1)

	// conss.SetKey(validProducerKey)
	bp := blockproducer.NewBlockProducer(bm, dpos1, producer)

	// seed node start mining
	bp.Start()
	util.WaitDoneOrTimeout(func() bool {
		return bc.GetMaxHeight() > 8
	}, 10)

	// set up another node for syncing
	dpos2 := consensus.NewDPOS(blockproducerinfo.NewBlockProducerInfo(validProducerAddress))
	dpos2.SetKey(validProducerKey)
	dpos2.SetDynasty(dynasty)
	db2 := storage.NewRamStorage()
	node2 := network.NewNode(db2, nil)
	node2.Start(testport_fork_syncing+1, "")
	defer node2.Stop()

	bc2 := lblockchain.CreateBlockchain(account.NewAddress(genesisAddr), db2, dpos2, transactionpool.NewTransactionPool(node2, 128), nil, 100000)
	lblockchain.NewBlockchainManager(bc2, blockchain.NewBlockPool(nil), node2, dpos2)

	// Trigger fork choice in node by broadcasting tail block of node[0]
	tailBlk, _ := bc.GetTailBlock()

	connectNodes(seedNode, node2)
	bm.BroadcastBlock(tailBlk)

	time.Sleep(time.Second * 5)
	bp.Stop()
	assert.True(t, bc.GetMaxHeight()-bc2.GetMaxHeight() <= 1)
}

func TestUpdate(t *testing.T) {
	var address1Bytes = []byte("address1000000000000000000000000")
	var address1TA = account.NewTransactionAccountByPubKey(address1Bytes)

	db := storage.NewRamStorage()
	defer db.Close()

	blk := core.GenerateUtxoMockBlockWithoutInputs()
	utxoIndex := lutxo.NewUTXOIndex(utxo.NewUTXOCache(db))
	utxoIndex.UpdateUtxos(blk.GetTransactions())
	utxoIndex.Save()
	utxoIndexInDB := lutxo.NewUTXOIndex(utxo.NewUTXOCache(db))

	// test updating UTXO index with non-dependent transactions
	// Assert that both the original instance and the database copy are updated correctly
	for _, index := range []lutxo.UTXOIndex{*utxoIndex, *utxoIndexInDB} {
		utxoTx := index.GetAllUTXOsByPubKeyHash(address1TA.GetPubKeyHash())
		assert.Equal(t, 2, utxoTx.Size())
		utxo0 := utxoTx.GetUtxo(blk.GetTransactions()[0].ID, 0)
		utx1 := utxoTx.GetUtxo(blk.GetTransactions()[0].ID, 1)
		assert.Equal(t, blk.GetTransactions()[0].ID, utxo0.Txid)
		assert.Equal(t, 0, utxo0.TxIndex)
		assert.Equal(t, blk.GetTransactions()[0].Vout[0].Value, utxo0.Value)
		assert.Equal(t, blk.GetTransactions()[0].ID, utx1.Txid)
		assert.Equal(t, 1, utx1.TxIndex)
		assert.Equal(t, blk.GetTransactions()[0].Vout[1].Value, utx1.Value)
	}

	// test updating UTXO index with dependent transactions
	var prikey1 = "bb23d2ff19f5b16955e8a24dca34dd520980fe3bddca2b3e1b56663f0ec1aa71"
	var ta1 = account.NewAccountByPrivateKey(prikey1)
	var prikey2 = "bb23d2ff19f5b16955e8a24dca34dd520980fe3bddca2b3e1b56663f0ec1aa72"
	var ta2 = account.NewAccountByPrivateKey(prikey2)
	var prikey3 = "bb23d2ff19f5b16955e8a24dca34dd520980fe3bddca2b3e1b56663f0ec1aa73"
	var ta3 = account.NewAccountByPrivateKey(prikey3)
	var prikey4 = "bb23d2ff19f5b16955e8a24dca34dd520980fe3bddca2b3e1b56663f0ec1aa74"
	var ta4 = account.NewAccountByPrivateKey(prikey4)
	var prikey5 = "bb23d2ff19f5b16955e8a24dca34dd520980fe3bddca2b3e1b56663f0ec1aa75"
	var ta5 = account.NewAccountByPrivateKey(prikey5)
	var dependentTx1 = &transaction.Transaction{
		ID: nil,
		Vin: []transactionbase.TXInput{
			{util.GenerateRandomAoB(1), 1, nil, ta1.GetKeyPair().GetPublicKey()},
		},
		Vout: []transactionbase.TXOutput{
			{common.NewAmount(5), ta1.GetPubKeyHash(), ""},
			{common.NewAmount(10), ta2.GetPubKeyHash(), ""},
		},
		Tip: common.NewAmount(3),
	}
	dependentTx1.ID = dependentTx1.Hash()

	var dependentTx2 = &transaction.Transaction{
		ID: nil,
		Vin: []transactionbase.TXInput{
			{dependentTx1.ID, 1, nil, ta2.GetKeyPair().GetPublicKey()},
		},
		Vout: []transactionbase.TXOutput{
			{common.NewAmount(5), ta3.GetPubKeyHash(), ""},
			{common.NewAmount(3), ta4.GetPubKeyHash(), ""},
		},
		Tip: common.NewAmount(2),
	}
	dependentTx2.ID = dependentTx2.Hash()

	var dependentTx3 = &transaction.Transaction{
		ID: nil,
		Vin: []transactionbase.TXInput{
			{dependentTx2.ID, 0, nil, ta3.GetKeyPair().GetPublicKey()},
		},
		Vout: []transactionbase.TXOutput{
			{common.NewAmount(1), ta4.GetPubKeyHash(), ""},
		},
		Tip: common.NewAmount(4),
	}
	dependentTx3.ID = dependentTx3.Hash()

	var dependentTx4 = &transaction.Transaction{
		ID: nil,
		Vin: []transactionbase.TXInput{
			{dependentTx2.ID, 1, nil, ta4.GetKeyPair().GetPublicKey()},
			{dependentTx3.ID, 0, nil, ta4.GetKeyPair().GetPublicKey()},
		},
		Vout: []transactionbase.TXOutput{
			{common.NewAmount(3), ta1.GetPubKeyHash(), ""},
		},
		Tip: common.NewAmount(1),
	}
	dependentTx4.ID = dependentTx4.Hash()

	var dependentTx5 = &transaction.Transaction{
		ID: nil,
		Vin: []transactionbase.TXInput{
			{dependentTx1.ID, 0, nil, ta1.GetKeyPair().GetPublicKey()},
			{dependentTx4.ID, 0, nil, ta1.GetKeyPair().GetPublicKey()},
		},
		Vout: []transactionbase.TXOutput{
			{common.NewAmount(4), ta5.GetPubKeyHash(), ""},
		},
		Tip: common.NewAmount(4),
	}
	dependentTx5.ID = dependentTx5.Hash()

	utxoPk2 := &utxo.UTXO{dependentTx1.Vout[1], dependentTx1.ID, 1, utxo.UtxoNormal, []byte{}}
	utxoPk1 := &utxo.UTXO{dependentTx1.Vout[0], dependentTx1.ID, 0, utxo.UtxoNormal, []byte{}}

	utxoTxPk2 := utxo.NewUTXOTx()
	utxoTxPk2.PutUtxo(utxoPk2)

	utxoTxPk1 := utxo.NewUTXOTx()
	utxoTxPk1.PutUtxo(utxoPk1)

	utxoIndex2 := lutxo.NewUTXOIndex(utxo.NewUTXOCache(storage.NewRamStorage()))

	utxoIndex2.SetIndexAdd(map[string]*utxo.UTXOTx{
		ta2.GetPubKeyHash().String(): &utxoTxPk2,
		ta1.GetPubKeyHash().String(): &utxoTxPk1,
	})

	tx2Utxo1 := utxo.UTXO{dependentTx2.Vout[0], dependentTx2.ID, 0, utxo.UtxoNormal, []byte{}}
	tx2Utxo2 := utxo.UTXO{dependentTx2.Vout[1], dependentTx2.ID, 1, utxo.UtxoNormal, []byte{}}
	tx2Utxo3 := utxo.UTXO{dependentTx3.Vout[0], dependentTx3.ID, 0, utxo.UtxoNormal, []byte{}}
	tx2Utxo4 := utxo.UTXO{dependentTx1.Vout[0], dependentTx1.ID, 0, utxo.UtxoNormal, []byte{}}
	tx2Utxo5 := utxo.UTXO{dependentTx4.Vout[0], dependentTx4.ID, 0, utxo.UtxoNormal, []byte{}}
	ltransaction.NewTxDecorator(dependentTx2).Sign(account.GenerateKeyPairByPrivateKey(prikey2).GetPrivateKey(), utxoIndex2.GetAllUTXOsByPubKeyHash(ta2.GetPubKeyHash()).GetAllUtxos())
	ltransaction.NewTxDecorator(dependentTx3).Sign(account.GenerateKeyPairByPrivateKey(prikey3).GetPrivateKey(), []*utxo.UTXO{&tx2Utxo1})
	ltransaction.NewTxDecorator(dependentTx4).Sign(account.GenerateKeyPairByPrivateKey(prikey4).GetPrivateKey(), []*utxo.UTXO{&tx2Utxo2, &tx2Utxo3})
	ltransaction.NewTxDecorator(dependentTx5).Sign(account.GenerateKeyPairByPrivateKey(prikey1).GetPrivateKey(), []*utxo.UTXO{&tx2Utxo4, &tx2Utxo5})

	txsForUpdate := []*transaction.Transaction{dependentTx2, dependentTx3}
	utxoIndex2.UpdateUtxos(txsForUpdate)
	assert.Equal(t, 1, utxoIndex2.GetAllUTXOsByPubKeyHash(ta1.GetPubKeyHash()).Size())
	assert.Equal(t, 0, utxoIndex2.GetAllUTXOsByPubKeyHash(ta2.GetPubKeyHash()).Size())
	assert.Equal(t, 0, utxoIndex2.GetAllUTXOsByPubKeyHash(ta3.GetPubKeyHash()).Size())
	assert.Equal(t, 2, utxoIndex2.GetAllUTXOsByPubKeyHash(ta4.GetPubKeyHash()).Size())
	txsForUpdate = []*transaction.Transaction{dependentTx4}
	utxoIndex2.UpdateUtxos(txsForUpdate)
	assert.Equal(t, 2, utxoIndex2.GetAllUTXOsByPubKeyHash(ta1.GetPubKeyHash()).Size())
	assert.Equal(t, 0, utxoIndex2.GetAllUTXOsByPubKeyHash(ta2.GetPubKeyHash()).Size())
	assert.Equal(t, 0, utxoIndex2.GetAllUTXOsByPubKeyHash(ta3.GetPubKeyHash()).Size())
	txsForUpdate = []*transaction.Transaction{dependentTx5}
	utxoIndex2.UpdateUtxos(txsForUpdate)
	assert.Equal(t, 0, utxoIndex2.GetAllUTXOsByPubKeyHash(ta1.GetPubKeyHash()).Size())
	assert.Equal(t, 0, utxoIndex2.GetAllUTXOsByPubKeyHash(ta2.GetPubKeyHash()).Size())
	assert.Equal(t, 0, utxoIndex2.GetAllUTXOsByPubKeyHash(ta3.GetPubKeyHash()).Size())
	assert.Equal(t, 0, utxoIndex2.GetAllUTXOsByPubKeyHash(ta4.GetPubKeyHash()).Size())
	assert.Equal(t, 1, utxoIndex2.GetAllUTXOsByPubKeyHash(ta5.GetPubKeyHash()).Size())
}

func Test_MultipleMinersWithDPOS(t *testing.T) {
	const (
		timeBetweenBlock = 2
		dposRounds       = 2
	)

	miners := []string{
		"dPGZmHd73UpZhrM6uvgnzu49ttbLp4AzU8",
		"dQEooMsqp23RkPsvZXj3XbsRh9BUyGz2S9",
		"dastXXWLe5pxbRYFhcyUq8T3wb5srWkHKa",
		"dUuPPYshbBgkzUrgScEHWvdGbSxC8z4R12",
		"dPGD4t6ibpmyKZnXH1TNbbPw98EDaaZq8C",
	}
	keystrs := []string{
		"5a66b0fdb69c99935783059bb200e86e97b506ae443a62febd7d0750cd7fac55",
		"bb23d2ff19f5b16955e8a24dca34dd520980fe3bddca2b3e1b56663f0ec1aa7e",
		"300c0338c4b0d49edc66113e3584e04c6b907f9ded711d396d522aae6a79be1a",
		"da9282440fae188c371165e01615a2e1b14af68b3eaae51e6608c0bd86d4e6a6",
		"7c918ed7660d55759b7fc42b25f26bdab3caf8fc07586b2659a26470fb8dfc69",
	}
	dynasty := consensus.NewDynasty(miners, len(miners), timeBetweenBlock)

	var bcs []*lblockchain.Blockchain
	var bps []*blockproducer.BlockProducer

	var nodeArray []*network.Node

	for i, miner := range miners {
		producer := blockproducerinfo.NewBlockProducerInfo(miner)
		dpos := consensus.NewDPOS(producer)
		dpos.SetKey(keystrs[i])
		dpos.SetDynasty(dynasty)
		bc := lblockchain.CreateBlockchain(account.NewAddress(miners[0]), storage.NewRamStorage(), dpos, transactionpool.NewTransactionPool(nil, 128), nil, 100000)
		pool := blockchain.NewBlockPool(nil)

		node := network.NewNode(bc.GetDb(), nil)
		node.Start(21200+i, "")
		nodeArray = append(nodeArray, node)

		bm := lblockchain.NewBlockchainManager(bc, pool, node, dpos)
		bp := blockproducer.NewBlockProducer(bm, dpos, producer)
		bps = append(bps, bp)
		bcs = append(bcs, bc)
	}

	for i := range miners {
		bps[i].Start()
	}

	for i := range miners {
		for j := range miners {
			if i != j {
				nodeArray[i].GetNetwork().ConnectToSeed(nodeArray[j].GetHostPeerInfo())
			}
		}
	}

	time.Sleep(time.Second * time.Duration(dynasty.GetDynastyTime()*dposRounds))

	// assert before close node
	assertHeight := uint64(dynasty.GetDynastyTime() * dposRounds / timeBetweenBlock)
	for i := range miners {
		assert.Equal(t, assertHeight, bcs[i].GetMaxHeight())
	}

	for i := range miners {
		bps[i].Stop()
		nodeArray[i].Stop()
	}
}

func TestDPOS_UpdateLIB(t *testing.T) {
	const (
		timeBetweenBlock = 2
		dposRounds       = 2
	)

	miners := []string{
		"dPGZmHd73UpZhrM6uvgnzu49ttbLp4AzU8",
		"dQEooMsqp23RkPsvZXj3XbsRh9BUyGz2S9",
		"dastXXWLe5pxbRYFhcyUq8T3wb5srWkHKa",
		"dUuPPYshbBgkzUrgScEHWvdGbSxC8z4R12",
		"dPGD4t6ibpmyKZnXH1TNbbPw98EDaaZq8C",
	}
	keystrs := []string{
		"5a66b0fdb69c99935783059bb200e86e97b506ae443a62febd7d0750cd7fac55",
		"bb23d2ff19f5b16955e8a24dca34dd520980fe3bddca2b3e1b56663f0ec1aa7e",
		"300c0338c4b0d49edc66113e3584e04c6b907f9ded711d396d522aae6a79be1a",
		"da9282440fae188c371165e01615a2e1b14af68b3eaae51e6608c0bd86d4e6a6",
		"7c918ed7660d55759b7fc42b25f26bdab3caf8fc07586b2659a26470fb8dfc69",
	}
	dynasty := consensus.NewDynasty(miners, len(miners), timeBetweenBlock)

	var bcs []*lblockchain.Blockchain
	var bps []*blockproducer.BlockProducer

	var nodeArray []*network.Node

	for i, miner := range miners {
		producer := blockproducerinfo.NewBlockProducerInfo(miner)
		dpos := consensus.NewDPOS(producer)
		dpos.SetKey(keystrs[i])
		dpos.SetDynasty(dynasty)
		bc := lblockchain.CreateBlockchain(account.NewAddress(miners[0]), storage.NewRamStorage(), dpos, transactionpool.NewTransactionPool(nil, 128), nil, 100000)
		pool := blockchain.NewBlockPool(nil)

		node := network.NewNode(bc.GetDb(), nil)
		node.Start(21200+i, "")
		nodeArray = append(nodeArray, node)

		bm := lblockchain.NewBlockchainManager(bc, pool, node, dpos)
		bp := blockproducer.NewBlockProducer(bm, dpos, producer)
		bp.Start()
		bps = append(bps, bp)
		bcs = append(bcs, bc)
	}

	for i := range miners {
		for j := range miners {
			if i != j {
				nodeArray[i].GetNetwork().ConnectToSeed(nodeArray[j].GetHostPeerInfo())
			}
		}
	}

	time.Sleep(time.Second * time.Duration(dynasty.GetDynastyTime()*dposRounds))

	for i := range miners {
		bps[i].Stop()
		nodeArray[i].Stop()
	}

	//Waiting block sync to other nodes

	for i := range miners {
		util.WaitDoneOrTimeout(func() bool {
			return !bps[i].IsProducingBlock()
		}, 20)
	}

	block0, _ := bcs[0].GetLIB()
	assert.NotEqual(t, 0, block0.GetHeight())

	for i := range miners {
		block, _ := bcs[i].GetLIB()
		assert.Equal(t, block0.GetHash(), block.GetHash())
	}
}

// TestSmartContractOfContractTransfer tests contract transfer to normal address, using system interface 'blockchain.js'
func TestSmartContractOfContractTransfer(t *testing.T) {
	store := storage.NewRamStorage()
	defer store.Close()

	contract := `'use strict';

	var TransferTest = function(){

	};

	TransferTest.prototype = {
		transfer: function(to, amount, tip){
			return Blockchain.transfer(to, amount, tip);
		},
		dapp_schedule: function () {
		}
	};
	module.exports = new TransferTest();
	`

	// Create a account address
	senderAccount, err := logic.CreateAccountWithPassphrase("test", logic.GetTestAccountPath())
	minerAccount, err := logic.CreateAccountWithPassphrase("test", logic.GetTestAccountPath())
	assert.Nil(t, err)
	node := network.FakeNodeWithPidAndAddr(store, "test", "test")
	bm, bps := CreateProducer(minerAccount.GetAddress(), senderAccount.GetAddress(), store, transactionpool.NewTransactionPool(node, 128), node)

	//deploy smart contract
	_, _, err = logic.Send(senderAccount, account.NewAddress(""), common.NewAmount(30000), common.NewAmount(0), common.NewAmount(30000), common.NewAmount(1), contract, bm.Getblockchain())

	assert.Nil(t, err)

	txp := bm.Getblockchain().GetTxPool().GetTransactions()[0]
	contractAddr := ltransaction.NewTxContract(txp).GetContractAddress()

	if err != nil {
		panic(err)
	}

	//a short delay before mining starts
	time.Sleep(time.Millisecond * 500)

	// Make logic.Sender the miner and mine for 1 block (which should include the transaction)
	bps.Start()
	for bm.Getblockchain().GetMaxHeight() < 1 {
	}
	bps.Stop()

	time.Sleep(time.Millisecond * 500)

	receiverAddress := account.NewAddress("dYgmFyXLg5jSfbysWoZF7Zimnx95xg77Qo")
	receiverAmount := common.NewAmount(5)
	// transfer to receiverAddress
	functionCall := `{"function":"transfer","args":["` + receiverAddress.String() + `","` + receiverAmount.String() + `","1"]}`

	_, _, err = logic.Send(senderAccount, contractAddr, common.NewAmount(1), common.NewAmount(0), common.NewAmount(30000), common.NewAmount(1), functionCall, bm.Getblockchain())

	assert.Nil(t, err)

	currentHeight := bm.Getblockchain().GetMaxHeight()

	bps.Start()
	for bm.Getblockchain().GetMaxHeight() < currentHeight+1 {
	}
	bps.Stop()

	// query receiver balance
	balance, err := logic.GetBalance(receiverAddress, bm.Getblockchain())

	assert.Nil(t, err)
	assert.Equal(t, receiverAmount, balance)

	logic.RemoveAccountTestFile()
}

// TestSmartContractOfContractDelete tests contract destroy interface, using system interface 'blockchain.js'
func TestSmartContractOfContractDelete(t *testing.T) {
	store := storage.NewRamStorage()
	defer store.Close()

	contract := `'use strict';

	var DeleteTest = function(){

	};

	DeleteTest.prototype = {
		transfer: function(to, amount, tip){
			return Blockchain.transfer(to, amount, tip);
		},
		deleteContract: function(){
			return Blockchain.deleteContract();
		},
		dapp_schedule: function () {
		}
	};
	module.exports = new DeleteTest();
	`

	// Create a account address
	senderAccount, err := logic.CreateAccountWithPassphrase("test", logic.GetTestAccountPath())
	minerAccount, err := logic.CreateAccountWithPassphrase("test", logic.GetTestAccountPath())
	assert.Nil(t, err)
	node := network.FakeNodeWithPidAndAddr(store, "test", "test")
	bm, bps := CreateProducer(minerAccount.GetAddress(), senderAccount.GetAddress(), store, transactionpool.NewTransactionPool(node, 128), node)

	//deploy smart contract
	toAmount := common.NewAmount(30000)
	_, _, err = logic.Send(senderAccount, account.NewAddress(""), toAmount, common.NewAmount(0), common.NewAmount(30000), common.NewAmount(1), contract, bm.Getblockchain())

	assert.Nil(t, err)

	txp := bm.Getblockchain().GetTxPool().GetTransactions()[0]
	contractAddr := ltransaction.NewTxContract(txp).GetContractAddress()

	if err != nil {
		panic(err)
	}

	//a short delay before mining starts
	time.Sleep(time.Millisecond * 500)

	// Make logic.Sender the miner and mine for 1 block (which should include the transaction)
	bps.Start()
	for bm.Getblockchain().GetMaxHeight() < 1 {
	}
	bps.Stop()

	time.Sleep(time.Millisecond * 500)

	// query contract address balance
	balance, err := logic.GetBalance(contractAddr, bm.Getblockchain())

	assert.Nil(t, err)
	assert.Equal(t, toAmount, balance)

	// transfer to receiverAddress
	functionCall := `{"function":"deleteContract","args":[]}`
	_, _, err = logic.Send(senderAccount, contractAddr, common.NewAmount(1), common.NewAmount(0), common.NewAmount(30000), common.NewAmount(1), functionCall, bm.Getblockchain())

	assert.Nil(t, err)

	currentHeight := bm.Getblockchain().GetMaxHeight()

	bps.Start()
	for bm.Getblockchain().GetMaxHeight() < currentHeight+1 {
	}
	bps.Stop()

	// query contract address balance
	balance, err = logic.GetBalance(contractAddr, bm.Getblockchain())

	assert.Nil(t, err)
	assert.Equal(t, common.NewAmount(0), balance)

	logic.RemoveAccountTestFile()
}

// TestZeroGasPriceOfContractTransaction tests send contract tx with zero or negative gas price value
func TestZeroGasPriceOfContractTransaction(t *testing.T) {
	store := storage.NewRamStorage()
	defer store.Close()

	contract := `'use strict';

	var GasPriceTest = function(){

	};

	GasPriceTest.prototype = {
		transfer: function(to, amount, tip){
			return Blockchain.transfer(to, amount, tip);
		},
		dapp_schedule: function () {
		}
	};
	module.exports = new GasPriceTest();
	`

	// Create a account address
	senderAccount, err := logic.CreateAccountWithPassphrase("test", logic.GetTestAccountPath())
	minerAccount, err := logic.CreateAccountWithPassphrase("test", logic.GetTestAccountPath())
	assert.Nil(t, err)
	node := network.FakeNodeWithPidAndAddr(store, "test", "test")
	bm, bps := CreateProducer(minerAccount.GetAddress(), senderAccount.GetAddress(), store, transactionpool.NewTransactionPool(node, 128), node)

	//deploy smart contract
	toAmount := common.NewAmount(30000)
	_, _, err = logic.Send(senderAccount, account.NewAddress(""), toAmount, common.NewAmount(0), common.NewAmount(30000), common.NewAmount(1), contract, bm.Getblockchain())

	assert.Nil(t, err)

	txp := bm.Getblockchain().GetTxPool().GetTransactions()[0]
	contractAddr := ltransaction.NewTxContract(txp).GetContractAddress()

	if err != nil {
		panic(err)
	}

	//a short delay before mining starts
	time.Sleep(time.Millisecond * 500)

	// Make logic.Sender the miner and mine for 1 block (which should include the transaction)
	bps.Start()
	for bm.Getblockchain().GetMaxHeight() < 1 {
	}
	bps.Stop()

	time.Sleep(time.Millisecond * 500)

	// query contract address balance
	balance, err := logic.GetBalance(contractAddr, bm.Getblockchain())

	assert.Nil(t, err)
	assert.Equal(t, toAmount, balance)

	receiverAddress := account.NewAddress("dYgmFyXLg5jSfbysWoZF7Zimnx95xg77Qo")
	receiverAmount := common.NewAmount(5)
	// transfer to receiverAddress
	functionCall := `{"function":"transfer","args":["` + receiverAddress.String() + `","` + receiverAmount.String() + `","1"]}`

	_, _, err = logic.Send(senderAccount, contractAddr, common.NewAmount(1), common.NewAmount(0), common.NewAmount(30000), common.NewAmount(0), functionCall, bm.Getblockchain())

	assert.Nil(t, err)

	currentHeight := bm.Getblockchain().GetMaxHeight()

	bps.Start()
	for bm.Getblockchain().GetMaxHeight() < currentHeight+1 {
	}
	bps.Stop()

	// query receiver balance
	balance, err = logic.GetBalance(receiverAddress, bm.Getblockchain())

	assert.Nil(t, err)
	assert.Equal(t, common.NewAmount(0), balance)

	logic.RemoveAccountTestFile()
}

func isSameBlockChain(bc1, bc2 *lblockchain.Blockchain) bool {
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
