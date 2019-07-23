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

package rpc

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/dappley/go-dappley/client"
	"github.com/dappley/go-dappley/common"
	"github.com/dappley/go-dappley/consensus"
	"github.com/dappley/go-dappley/core"
	corepb "github.com/dappley/go-dappley/core/pb"
	"github.com/dappley/go-dappley/logic"
	"github.com/dappley/go-dappley/network"
	rpcpb "github.com/dappley/go-dappley/rpc/pb"
	"github.com/dappley/go-dappley/storage"
	"github.com/dappley/go-dappley/vm"
	logger "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type RpcTestContext struct {
	store      storage.Storage
	account    *client.Account
	consensus  core.Consensus
	bc         *core.Blockchain
	node       *network.Node
	rpcServer  *Server
	serverPort uint32
}

func TestServer_StartRPC(t *testing.T) {

	pid := "QmWsMUDBeWxwU4R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ"
	addr := "/ip4/127.0.0.1/tcp/10000"
	node := network.FakeNodeWithPeer(pid, addr)
	//start grpc server
	server := NewGrpcServer(node, "temp")
	server.Start(defaultRpcPort)
	defer server.Stop()

	time.Sleep(time.Millisecond * 100)
	//prepare grpc client
	var conn *grpc.ClientConn
	conn, err := grpc.Dial(fmt.Sprint(":", defaultRpcPort), grpc.WithInsecure())
	assert.Nil(t, err)
	defer conn.Close()

	c := rpcpb.NewAdminServiceClient(conn)
	response, err := c.RpcGetPeerInfo(context.Background(), &rpcpb.GetPeerInfoRequest{})
	assert.Nil(t, err)
	assert.Equal(t, 0, len(response.GetPeerList()))
}

func TestRpcSend(t *testing.T) {
	logger.SetLevel(logger.WarnLevel)
	// Create storage
	store := storage.NewRamStorage()
	defer store.Close()
	client.RemoveAccountFile()

	// Create accounts
	senderAccount, err := logic.CreateAccount(strings.Replace(client.GetAccountFilePath(), "accounts", "accounts_test", -1), "test")
	if err != nil {
		panic(err)
	}
	receiverAccount, err := logic.CreateAccount(strings.Replace(client.GetAccountFilePath(), "accounts", "accounts_test", -1), "test")
	if err != nil {
		panic(err)
	}

	minerAccount, err := logic.CreateAccount(strings.Replace(client.GetAccountFilePath(), "accounts", "accounts_test", -1), "test")
	if err != nil {
		panic(err)
	}

	// Create a blockchain with PoW consensus and sender account as coinbase (so its balance starts with 10)
	pow := consensus.NewProofOfWork()
	scManager := vm.NewV8EngineManager(client.Address{})
	bc, err := logic.CreateBlockchain(senderAccount.GetKeyPair().GenerateAddress(), store, pow, 1280000, scManager, 1000000)
	if err != nil {
		panic(err)
	}

	// Prepare a PoW node that put mining reward to the sender's address
	pool := core.NewBlockPool(0)
	node := network.FakeNodeWithPidAndAddr(pool, bc, "a", "b")
	pow.Setup(node, minerAccount.GetKeyPair().GenerateAddress().String())
	pow.SetTargetBit(0)

	// Start a grpc server
	server := NewGrpcServer(node, "temp")
	server.Start(defaultRpcPort + 1) // use a different port as other integration tests
	defer server.Stop()

	time.Sleep(100 * time.Millisecond)

	// Create a grpc connection and a client
	conn, err := grpc.Dial(fmt.Sprint(":", defaultRpcPort+1), grpc.WithInsecure())
	if err != nil {
		panic(err)
	}
	defer conn.Close()
	c := rpcpb.NewAdminServiceClient(conn)

	// Initiate a RPC send request
	_, err = c.RpcSend(context.Background(), &rpcpb.SendRequest{
		From:        senderAccount.GetKeyPair().GenerateAddress().String(),
		To:          receiverAccount.GetKeyPair().GenerateAddress().String(),
		Amount:      common.NewAmount(7).Bytes(),
		AccountPath: strings.Replace(client.GetAccountFilePath(), "accounts", "accounts_test", -1),
		Tip:         common.NewAmount(2).Bytes(),
		Data:        "",
	})
	assert.Nil(t, err)

	// Start mining to approve the transaction
	pow.Start()
	for bc.GetMaxHeight() < 1 {
	}
	pow.Stop()

	time.Sleep(100 * time.Millisecond)

	// Check balance
	minedReward := common.NewAmount(10000000)
	senderBalance, err := logic.GetBalance(senderAccount.GetKeyPair().GenerateAddress(), bc)
	assert.Nil(t, err)
	receiverBalance, err := logic.GetBalance(receiverAccount.GetKeyPair().GenerateAddress(), bc)
	assert.Nil(t, err)
	minerBalance, err := logic.GetBalance(minerAccount.GetKeyPair().GenerateAddress(), bc)
	assert.Nil(t, err)

	leftBalance, _ := minedReward.Sub(common.NewAmount(7))
	leftBalance, _ = leftBalance.Sub(common.NewAmount(2))
	minerRewardBalance := minedReward.Times(bc.GetMaxHeight()).Add(common.NewAmount(2))
	assert.Equal(t, leftBalance, senderBalance)
	assert.Equal(t, common.NewAmount(7), receiverBalance)
	assert.Equal(t, minerRewardBalance, minerBalance)
	client.RemoveAccountFile()
}

func TestRpcSendContract(t *testing.T) {

	logger.SetLevel(logger.WarnLevel)
	// Create storage
	store := storage.NewRamStorage()
	defer store.Close()
	client.RemoveAccountFile()

	// Create accounts
	senderAccount, err := logic.CreateAccount(strings.Replace(client.GetAccountFilePath(), "accounts", "accounts_test", -1), "test")
	if err != nil {
		panic(err)
	}

	minerAccount, err := logic.CreateAccount(strings.Replace(client.GetAccountFilePath(), "accounts", "accounts_test", -1), "test")
	if err != nil {
		panic(err)
	}

	// Create a blockchain with PoW consensus and sender account as coinbase (so its balance starts with 10)
	pow := consensus.NewProofOfWork()
	scManager := vm.NewV8EngineManager(client.Address{})
	bc, err := logic.CreateBlockchain(senderAccount.GetKeyPair().GenerateAddress(), store, pow, 1280000, scManager, 1000000)
	if err != nil {
		panic(err)
	}

	// Prepare a PoW node that put mining reward to the sender's address
	pool := core.NewBlockPool(0)
	node := network.FakeNodeWithPidAndAddr(pool, bc, "a", "b")
	pow.Setup(node, minerAccount.GetKeyPair().GenerateAddress().String())
	pow.SetTargetBit(0)

	// Start a grpc server
	server := NewGrpcServer(node, "temp")
	server.Start(defaultRpcPort + 10) // use a different port as other integration tests
	defer server.Stop()

	time.Sleep(100 * time.Millisecond)

	// Create a grpc connection and a client
	conn, err := grpc.Dial(fmt.Sprint(":", defaultRpcPort+10), grpc.WithInsecure())
	if err != nil {
		panic(err)
	}
	defer conn.Close()
	c := rpcpb.NewAdminServiceClient(conn)

	contract := "dapp_schedule"
	// Initiate a RPC send request
	_, err = c.RpcSend(context.Background(), &rpcpb.SendRequest{
		From:        senderAccount.GetKeyPair().GenerateAddress().String(),
		To:          "",
		Amount:      common.NewAmount(7).Bytes(),
		AccountPath: strings.Replace(client.GetAccountFilePath(), "accounts", "accounts_test", -1),
		Tip:         common.NewAmount(2).Bytes(),
		Data:        contract,
		GasLimit:    common.NewAmount(30000).Bytes(),
		GasPrice:    common.NewAmount(1).Bytes(),
	})
	assert.Nil(t, err)

	// Start mining to approve the transaction
	pow.Start()
	for bc.GetMaxHeight() < 1 {
	}
	pow.Stop()

	time.Sleep(time.Second)

	//check smart contract deployment
	res := string("")
	contractAddr := client.NewAddress("")
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
	assert.Equal(t, contract, res)

	client.RemoveAccountFile()
}

func TestRpcGetVersion(t *testing.T) {
	rpcContext, err := createRpcTestContext(2)
	if err != nil {
		panic(err)
	}
	defer rpcContext.destroyContext()

	time.Sleep(100 * time.Millisecond)

	// Create a grpc connection and a client
	conn, err := grpc.Dial(fmt.Sprint(":", rpcContext.serverPort), grpc.WithInsecure())
	if err != nil {
		panic(err)
	}
	defer conn.Close()
	c := rpcpb.NewRpcServiceClient(conn)

	// Test GetVersion with support client version
	response, err := c.RpcGetVersion(context.Background(), &rpcpb.GetVersionRequest{ProtoVersion: "1.0.0"})
	assert.Nil(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, "1.0.0", response.ProtoVersion, "1.0.0")

	// Test GetVersion with unsupport client version -- invalid version length
	response, err = c.RpcGetVersion(context.Background(), &rpcpb.GetVersionRequest{ProtoVersion: "1.0.0.0"})
	assert.Nil(t, response)

	assert.Equal(t, codes.InvalidArgument, status.Code(err))
	assert.Equal(t, "proto version not supported", status.Convert(err).Message())

	// Test GetVersion with unsupport client version
	response, err = c.RpcGetVersion(context.Background(), &rpcpb.GetVersionRequest{ProtoVersion: "2.0.0"})
	assert.Nil(t, response)
	assert.Equal(t, codes.Unimplemented, status.Code(err))
	assert.Equal(t, "major version mismatch", status.Convert(err).Message())
}

func TestRpcGetBlockchainInfo(t *testing.T) {
	rpcContext, err := createRpcTestContext(3)
	if err != nil {
		panic(err)
	}
	defer rpcContext.destroyContext()

	rpcContext.consensus.Setup(rpcContext.node, rpcContext.account.GetKeyPair().GenerateAddress().String())
	rpcContext.consensus.Start()

	for rpcContext.bc.GetMaxHeight() < 5 {

	}

	rpcContext.consensus.Stop()
	core.WaitDoneOrTimeout(func() bool {
		return !rpcContext.consensus.IsProducingBlock()
	}, 20)
	time.Sleep(time.Second)

	// Create a grpc connection and a client
	conn, err := grpc.Dial(fmt.Sprint(":", rpcContext.serverPort), grpc.WithInsecure())
	if err != nil {
		panic(err)
	}
	defer conn.Close()
	c := rpcpb.NewRpcServiceClient(conn)
	response, err := c.RpcGetBlockchainInfo(context.Background(), &rpcpb.GetBlockchainInfoRequest{})
	assert.Nil(t, err)

	tailBlock, err := rpcContext.bc.GetTailBlock()
	assert.Nil(t, err)

	assert.Equal(t, []byte(tailBlock.GetHash()), response.TailBlockHash)
	assert.Equal(t, tailBlock.GetHeight(), response.BlockHeight)
	assert.Equal(t, 0, len(response.Producers))
}

func TestRpcGetUTXO(t *testing.T) {
	rpcContext, err := createRpcTestContext(4)
	if err != nil {
		panic(err)
	}
	defer rpcContext.destroyContext()

	receiverAccount, err := logic.CreateAccount(strings.Replace(client.GetAccountFilePath(), "accounts", "accounts_test", -1), "test")
	if err != nil {
		panic(err)
	}

	logic.Send(rpcContext.account, receiverAccount.GetKeyPair().GenerateAddress(), common.NewAmount(6), common.NewAmount(0), common.NewAmount(0), common.NewAmount(0), "", rpcContext.bc, rpcContext.node)

	rpcContext.consensus.Setup(rpcContext.node, rpcContext.account.GetKeyPair().GenerateAddress().String())
	rpcContext.consensus.Start()

	for rpcContext.bc.GetMaxHeight() < MinUtxoBlockHeaderCount {

	}

	rpcContext.consensus.Stop()
	core.WaitDoneOrTimeout(func() bool {
		return !rpcContext.consensus.IsProducingBlock()
	}, 20)
	time.Sleep(time.Second)

	// Create a grpc connection and a client
	conn, err := grpc.Dial(fmt.Sprint(":", rpcContext.serverPort), grpc.WithInsecure())
	if err != nil {
		panic(err)
	}
	defer conn.Close()
	c := rpcpb.NewRpcServiceClient(conn)

	senderResponse, err := c.RpcGetUTXO(context.Background(), &rpcpb.GetUTXORequest{Address: rpcContext.account.GetKeyPair().GenerateAddress().String()})
	assert.Nil(t, err)
	assert.NotNil(t, senderResponse)
	minedReward := common.NewAmount(10000000)
	leftAmount, err := minedReward.Times(rpcContext.bc.GetMaxHeight() + 1).Sub(common.NewAmount(6))
	assert.Equal(t, leftAmount, getBalance(senderResponse.Utxos))

	tailBlock, err := rpcContext.bc.GetTailBlock()
	assert.Equal(t, int(MinUtxoBlockHeaderCount), len(senderResponse.BlockHeaders))
	assert.Equal(t, []byte(tailBlock.GetHash()), senderResponse.BlockHeaders[0].GetHash())

	receiverResponse, err := c.RpcGetUTXO(context.Background(), &rpcpb.GetUTXORequest{Address: receiverAccount.GetKeyPair().GenerateAddress().String()})
	assert.Nil(t, err)
	assert.NotNil(t, receiverResponse)
	assert.Equal(t, common.NewAmount(6), getBalance(receiverResponse.Utxos))
}

func TestRpcGetBlocks(t *testing.T) {
	rpcContext, err := createRpcTestContext(5)
	if err != nil {
		panic(err)
	}
	defer rpcContext.destroyContext()

	rpcContext.consensus.Setup(rpcContext.node, rpcContext.account.GetKeyPair().GenerateAddress().String())
	rpcContext.consensus.Start()

	for rpcContext.bc.GetMaxHeight() < 500 {
	}

	rpcContext.consensus.Stop()
	core.WaitDoneOrTimeout(func() bool {
		return !rpcContext.consensus.IsProducingBlock()
	}, 20)
	time.Sleep(time.Second)

	genesisBlock := core.NewGenesisBlock(rpcContext.account.GetKeyPair().GenerateAddress())
	// Create a grpc connection and a client
	conn, err := grpc.Dial(fmt.Sprint(":", rpcContext.serverPort), grpc.WithInsecure())
	if err != nil {
		panic(err)
	}
	defer conn.Close()
	c := rpcpb.NewRpcServiceClient(conn)

	//Check first query
	maxGetBlocksCount := 20
	response, err := c.RpcGetBlocks(context.Background(), &rpcpb.GetBlocksRequest{StartBlockHashes: [][]byte{genesisBlock.GetHash()}, MaxCount: int32(maxGetBlocksCount)})
	assert.Nil(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, maxGetBlocksCount, len(response.Blocks))
	block1, err := rpcContext.bc.GetBlockByHeight(1)
	assert.Equal(t, []byte(block1.GetHash()), response.Blocks[0].GetHeader().GetHash())
	block20, err := rpcContext.bc.GetBlockByHeight(uint64(maxGetBlocksCount))
	assert.Equal(t, []byte(block20.GetHash()), response.Blocks[19].GetHeader().GetHash())

	// Check query loop
	var startBlockHashes [][]byte
	queryCount := (int(rpcContext.bc.GetMaxHeight())+maxGetBlocksCount-1)/maxGetBlocksCount - 1
	startHashCount := 3 // suggest value is 2/3 * producersnum +1

	for i := 0; i < queryCount; i++ {
		startBlockHashes = nil
		lastBlocksCount := len(response.Blocks)
		for j := 0; j < startHashCount; j++ {
			startBlockHashes = append(startBlockHashes, response.Blocks[lastBlocksCount-1-j].GetHeader().GetHash())
		}
		response, err = c.RpcGetBlocks(context.Background(), &rpcpb.GetBlocksRequest{StartBlockHashes: startBlockHashes, MaxCount: int32(maxGetBlocksCount)})
		assert.Nil(t, err)
		assert.NotNil(t, response)
		if i == (queryCount - 1) {
			leftCount := int(rpcContext.bc.GetMaxHeight()) - queryCount*maxGetBlocksCount
			assert.Equal(t, leftCount, len(response.Blocks))
		} else {
			assert.Equal(t, maxGetBlocksCount, len(response.Blocks))
		}
	}

	tailBlock, err := rpcContext.bc.GetTailBlock()
	assert.Nil(t, err)
	assert.Equal(t, []byte(tailBlock.GetHash()), response.Blocks[len(response.Blocks)-1].GetHeader().GetHash())

	// Check query reach tailblock
	response, err = c.RpcGetBlocks(context.Background(), &rpcpb.GetBlocksRequest{StartBlockHashes: [][]byte{tailBlock.GetHash()}, MaxCount: int32(maxGetBlocksCount)})
	assert.Nil(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, 0, len(response.Blocks))

	// Check maxGetBlocksCount overflow
	maxGetBlocksCount = int(MaxGetBlocksCount) + 1
	response, err = c.RpcGetBlocks(context.Background(), &rpcpb.GetBlocksRequest{StartBlockHashes: [][]byte{genesisBlock.GetHash()}, MaxCount: int32(maxGetBlocksCount)})
	assert.Nil(t, response)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
	assert.Equal(t, "block count overflow", status.Convert(err).Message())
}

func TestRpcGetBlockByHash(t *testing.T) {
	rpcContext, err := createRpcTestContext(6)
	if err != nil {
		panic(err)
	}
	defer rpcContext.destroyContext()

	rpcContext.consensus.Setup(rpcContext.node, rpcContext.account.GetKeyPair().GenerateAddress().String())
	rpcContext.consensus.Start()

	for rpcContext.bc.GetMaxHeight() < 50 {
	}

	rpcContext.consensus.Stop()
	core.WaitDoneOrTimeout(func() bool {
		return !rpcContext.consensus.IsProducingBlock()
	}, 20)
	time.Sleep(time.Second)

	// Create a grpc connection and a client
	conn, err := grpc.Dial(fmt.Sprint(":", rpcContext.serverPort), grpc.WithInsecure())
	if err != nil {
		panic(err)
	}
	defer conn.Close()
	c := rpcpb.NewRpcServiceClient(conn)

	block20, err := rpcContext.bc.GetBlockByHeight(20)
	response, err := c.RpcGetBlockByHash(context.Background(), &rpcpb.GetBlockByHashRequest{Hash: block20.GetHash()})
	assert.Nil(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, []byte(block20.GetHash()), response.Block.GetHeader().GetHash())

	tailBlock, err := rpcContext.bc.GetTailBlock()
	response, err = c.RpcGetBlockByHash(context.Background(), &rpcpb.GetBlockByHashRequest{Hash: tailBlock.GetHash()})
	assert.Nil(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, []byte(tailBlock.GetHash()), response.Block.GetHeader().GetHash())

	response, err = c.RpcGetBlockByHash(context.Background(), &rpcpb.GetBlockByHashRequest{Hash: []byte("noexists")})
	assert.Nil(t, response)
	assert.Equal(t, codes.NotFound, status.Code(err))
	assert.Equal(t, core.ErrBlockDoesNotExist.Error(), status.Convert(err).Message())
}

func TestRpcGetBlockByHeight(t *testing.T) {
	rpcContext, err := createRpcTestContext(7)
	if err != nil {
		panic(err)
	}
	defer rpcContext.destroyContext()

	rpcContext.consensus.Setup(rpcContext.node, rpcContext.account.GetKeyPair().GenerateAddress().String())
	rpcContext.consensus.Start()

	for rpcContext.bc.GetMaxHeight() < 50 {
	}

	rpcContext.consensus.Stop()
	core.WaitDoneOrTimeout(func() bool {
		return !rpcContext.consensus.IsProducingBlock()
	}, 20)
	time.Sleep(time.Second)

	// Create a grpc connection and a client
	conn, err := grpc.Dial(fmt.Sprint(":", rpcContext.serverPort), grpc.WithInsecure())
	if err != nil {
		panic(err)
	}
	defer conn.Close()
	c := rpcpb.NewRpcServiceClient(conn)

	block20, err := rpcContext.bc.GetBlockByHeight(20)
	response, err := c.RpcGetBlockByHeight(context.Background(), &rpcpb.GetBlockByHeightRequest{Height: 20})
	assert.Nil(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, []byte(block20.GetHash()), response.Block.GetHeader().GetHash())

	tailBlock, err := rpcContext.bc.GetTailBlock()
	response, err = c.RpcGetBlockByHeight(context.Background(), &rpcpb.GetBlockByHeightRequest{Height: tailBlock.GetHeight()})
	assert.Nil(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, []byte(tailBlock.GetHash()), response.Block.GetHeader().GetHash())

	response, err = c.RpcGetBlockByHeight(context.Background(), &rpcpb.GetBlockByHeightRequest{Height: tailBlock.GetHeight() + 1})
	assert.Nil(t, response)
	assert.Equal(t, codes.NotFound, status.Code(err))
	assert.Equal(t, core.ErrBlockDoesNotExist.Error(), status.Convert(err).Message())
}

func TestRpcSendTransaction(t *testing.T) {
	rpcContext, err := createRpcTestContext(8)
	if err != nil {
		panic(err)
	}
	defer rpcContext.destroyContext()

	receiverAccount, err := logic.CreateAccount(strings.Replace(client.GetAccountFilePath(), "accounts", "accounts_test", -1), "test")
	if err != nil {
		panic(err)
	}

	rpcContext.consensus.Setup(rpcContext.node, rpcContext.account.GetKeyPair().GenerateAddress().String())
	rpcContext.consensus.Start()

	maxHeight := rpcContext.bc.GetMaxHeight()
	for maxHeight < 2 {
		maxHeight = rpcContext.bc.GetMaxHeight()
	}
	// Create a grpc connection and a client
	conn, err := grpc.Dial(fmt.Sprint(":", rpcContext.serverPort), grpc.WithInsecure())
	if err != nil {
		panic(err)
	}
	defer conn.Close()
	c := rpcpb.NewRpcServiceClient(conn)

	pubKeyHash, _ := rpcContext.account.GetKeyPair().GenerateAddress().GetPubKeyHash()
	utxos, err := core.NewUTXOIndex(rpcContext.bc.GetUtxoCache()).GetUTXOsByAmount(pubKeyHash, common.NewAmount(6))
	assert.Nil(t, err)

	sendTxParam := core.NewSendTxParam(rpcContext.account.GetKeyPair().GenerateAddress(),
		rpcContext.account.GetKeyPair(),
		receiverAccount.GetKeyPair().GenerateAddress(),
		common.NewAmount(6),
		common.NewAmount(0),
		common.NewAmount(0),
		common.NewAmount(0),
		"")
	transaction, err := core.NewUTXOTransaction(utxos, sendTxParam)
	successResponse, err := c.RpcSendTransaction(context.Background(), &rpcpb.SendTransactionRequest{Transaction: transaction.ToProto().(*corepb.Transaction)})
	assert.Nil(t, err)
	assert.NotNil(t, successResponse)

	maxHeight = rpcContext.bc.GetMaxHeight()
	for (rpcContext.bc.GetMaxHeight() - maxHeight) < 2 {
	}

	utxos2, err := core.NewUTXOIndex(rpcContext.bc.GetUtxoCache()).GetUTXOsByAmount(pubKeyHash, common.NewAmount(6))
	sendTxParam2 := core.NewSendTxParam(rpcContext.account.GetKeyPair().GenerateAddress(),
		rpcContext.account.GetKeyPair(),
		receiverAccount.GetKeyPair().GenerateAddress(),
		common.NewAmount(6),
		common.NewAmount(0),
		common.NewAmount(0),
		common.NewAmount(0),
		"")
	errTransaction, err := core.NewUTXOTransaction(utxos2, sendTxParam2)
	errTransaction.Vin[0].Signature = []byte("invalid")
	failedResponse, err := c.RpcSendTransaction(context.Background(), &rpcpb.SendTransactionRequest{Transaction: errTransaction.ToProto().(*corepb.Transaction)})
	assert.Nil(t, failedResponse)
	assert.Equal(t, codes.FailedPrecondition, status.Code(err))
	assert.Equal(t, core.ErrTransactionVerifyFailed.Error(), status.Convert(err).Message())

	maxHeight = rpcContext.bc.GetMaxHeight()
	for (rpcContext.bc.GetMaxHeight() - maxHeight) < 2 {
	}

	rpcContext.consensus.Stop()
	core.WaitDoneOrTimeout(func() bool {
		return !rpcContext.consensus.IsProducingBlock()
	}, 20)
	time.Sleep(time.Second)

	minedReward := common.NewAmount(10000000)
	leftAmount, err := minedReward.Times(rpcContext.bc.GetMaxHeight() + 1).Sub(common.NewAmount(6))
	realAmount, err := logic.GetBalance(rpcContext.account.GetKeyPair().GenerateAddress(), rpcContext.bc)
	assert.Equal(t, leftAmount, realAmount)
	recvAmount, err := logic.GetBalance(receiverAccount.GetKeyPair().GenerateAddress(), rpcContext.bc)
	assert.Equal(t, common.NewAmount(6), recvAmount)
}

func TestRpcService_RpcSendBatchTransaction(t *testing.T) {
	logger.SetLevel(logger.DebugLevel)
	rpcContext, err := createRpcTestContext(99)
	if err != nil {
		panic(err)
	}
	defer rpcContext.destroyContext()

	receiverAccount1, err := logic.CreateAccount(strings.Replace(client.GetAccountFilePath(), "accounts", "accounts_test", -1), "test1")
	if err != nil {
		panic(err)
	}
	receiverAccount2, err := logic.CreateAccount(strings.Replace(client.GetAccountFilePath(), "accounts", "accounts_test", -1), "test2")
	if err != nil {
		panic(err)
	}
	receiverAccount4, err := logic.CreateAccount(strings.Replace(client.GetAccountFilePath(), "accounts", "accounts_test", -1), "test4")
	if err != nil {
		panic(err)
	}

	rpcContext.consensus.Setup(rpcContext.node, rpcContext.account.GetKeyPair().GenerateAddress().String())
	rpcContext.consensus.Start()

	maxHeight := rpcContext.bc.GetMaxHeight()
	for maxHeight < 2 {
		maxHeight = rpcContext.bc.GetMaxHeight()
	}

	// Create a grpc connection and a client
	conn, err := grpc.Dial(fmt.Sprint(":", rpcContext.serverPort), grpc.WithInsecure())
	if err != nil {
		panic(err)
	}
	defer conn.Close()
	c := rpcpb.NewRpcServiceClient(conn)

	pubKeyHash, _ := rpcContext.account.GetKeyPair().GenerateAddress().GetPubKeyHash()
	utxoIndex := core.NewUTXOIndex(rpcContext.bc.GetUtxoCache())
	utxos, err := utxoIndex.GetUTXOsByAmount(pubKeyHash, common.NewAmount(3))
	assert.Nil(t, err)

	sendTxParam1 := core.NewSendTxParam(rpcContext.account.GetKeyPair().GenerateAddress(),
		rpcContext.account.GetKeyPair(),
		receiverAccount1.GetKeyPair().GenerateAddress(),
		common.NewAmount(3),
		common.NewAmount(0),
		common.NewAmount(0),
		common.NewAmount(0),
		"")
	transaction1, err := core.NewUTXOTransaction(utxos, sendTxParam1)
	utxoIndex.UpdateUtxoState([]*core.Transaction{&transaction1})
	utxos, err = utxoIndex.GetUTXOsByAmount(pubKeyHash, common.NewAmount(2))
	sendTxParam2 := core.NewSendTxParam(rpcContext.account.GetKeyPair().GenerateAddress(),
		rpcContext.account.GetKeyPair(),
		receiverAccount2.GetKeyPair().GenerateAddress(),
		common.NewAmount(2),
		common.NewAmount(0),
		common.NewAmount(0),
		common.NewAmount(0),
		"")
	transaction2, err := core.NewUTXOTransaction(utxos, sendTxParam2)
	utxoIndex.UpdateUtxoState([]*core.Transaction{&transaction2})
	pubKeyHash1, _ := receiverAccount1.GetKeyPair().GenerateAddress().GetPubKeyHash()
	utxos, err = utxoIndex.GetUTXOsByAmount(pubKeyHash1, common.NewAmount(1))
	sendTxParam3 := core.NewSendTxParam(receiverAccount1.GetKeyPair().GenerateAddress(),
		receiverAccount1.GetKeyPair(),
		receiverAccount2.GetKeyPair().GenerateAddress(),
		common.NewAmount(1),
		common.NewAmount(0),
		common.NewAmount(0),
		common.NewAmount(0),
		"")
	transaction3, err := core.NewUTXOTransaction(utxos, sendTxParam3)
	utxoIndex.UpdateUtxoState([]*core.Transaction{&transaction3})

	rpcContext.consensus.Stop()
	time.Sleep(time.Second)

	successResponse, err := c.RpcSendBatchTransaction(context.Background(), &rpcpb.SendBatchTransactionRequest{Transactions: []*corepb.Transaction{transaction1.ToProto().(*corepb.Transaction), transaction2.ToProto().(*corepb.Transaction), transaction3.ToProto().(*corepb.Transaction)}})
	assert.Nil(t, err)
	assert.NotNil(t, successResponse)

	rpcContext.consensus.Start()
	maxHeight = rpcContext.bc.GetMaxHeight()
	for (rpcContext.bc.GetMaxHeight() - maxHeight) < 2 {
	}
	rpcContext.consensus.Stop()
	time.Sleep(time.Second)

	utxos2, err := utxoIndex.GetUTXOsByAmount(pubKeyHash, common.NewAmount(3))
	sendTxParamErr := core.NewSendTxParam(rpcContext.account.GetKeyPair().GenerateAddress(),
		rpcContext.account.GetKeyPair(),
		receiverAccount4.GetKeyPair().GenerateAddress(),
		common.NewAmount(3),
		common.NewAmount(0),
		common.NewAmount(0),
		common.NewAmount(0),
		"")
	errTransaction, err := core.NewUTXOTransaction(utxos2, sendTxParamErr)

	sendTxParam4 := core.NewSendTxParam(rpcContext.account.GetKeyPair().GenerateAddress(),
		rpcContext.account.GetKeyPair(),
		receiverAccount4.GetKeyPair().GenerateAddress(),
		common.NewAmount(3),
		common.NewAmount(0),
		common.NewAmount(0),
		common.NewAmount(0),
		"")
	transaction4, err := core.NewUTXOTransaction(utxos2, sendTxParam4)
	errTransaction.Vin[0].Signature = []byte("invalid")
	failedResponse, err := c.RpcSendBatchTransaction(context.Background(), &rpcpb.SendBatchTransactionRequest{Transactions: []*corepb.Transaction{errTransaction.ToProto().(*corepb.Transaction), transaction4.ToProto().(*corepb.Transaction)}})
	assert.Nil(t, failedResponse)
	st := status.Convert(err)
	assert.Equal(t, codes.Unknown, st.Code())

	detail0 := st.Details()[1].(*rpcpb.SendTransactionStatus)
	detail1 := st.Details()[0].(*rpcpb.SendTransactionStatus)
	assert.Equal(t, errTransaction.ID, detail0.Txid)
	assert.Equal(t, uint32(codes.FailedPrecondition), detail0.Code)
	assert.Equal(t, uint32(codes.OK), detail1.Code)

	rpcContext.consensus.Start()
	maxHeight = rpcContext.bc.GetMaxHeight()
	for (rpcContext.bc.GetMaxHeight() - maxHeight) < 2 {
	}

	rpcContext.consensus.Stop()
	time.Sleep(time.Second)

	core.WaitDoneOrTimeout(func() bool {
		return !rpcContext.consensus.IsProducingBlock()
	}, 20)

	minedReward := common.NewAmount(10000000)
	leftAmount, err := minedReward.Times(rpcContext.bc.GetMaxHeight() + 1).Sub(common.NewAmount(8))
	realAmount, err := logic.GetBalance(rpcContext.account.GetKeyPair().GenerateAddress(), rpcContext.bc)
	assert.Equal(t, leftAmount, realAmount)
	recvAmount1, err := logic.GetBalance(receiverAccount1.GetKeyPair().GenerateAddress(), rpcContext.bc)
	recvAmount2, err := logic.GetBalance(receiverAccount2.GetKeyPair().GenerateAddress(), rpcContext.bc)
	recvAmount4, err := logic.GetBalance(receiverAccount4.GetKeyPair().GenerateAddress(), rpcContext.bc)
	assert.Equal(t, common.NewAmount(2), recvAmount1)
	assert.Equal(t, common.NewAmount(3), recvAmount2)
	assert.Equal(t, common.NewAmount(3), recvAmount4)
}

func TestGetNewTransaction(t *testing.T) {
	logger.SetLevel(logger.WarnLevel)
	rpcContext, err := createRpcTestContext(11)
	if err != nil {
		panic(err)
	}
	defer rpcContext.destroyContext()

	receiverAccount, err := logic.CreateAccount(strings.Replace(client.GetAccountFilePath(), "accounts", "accounts_test", -1), "test")
	if err != nil {
		panic(err)
	}

	rpcContext.consensus.Setup(rpcContext.node, rpcContext.account.GetKeyPair().GenerateAddress().String())
	rpcContext.consensus.Start()

	// Create a grpc connection and a client
	conn1, err := grpc.Dial(fmt.Sprint(":", rpcContext.serverPort), grpc.WithInsecure())
	if err != nil {
		panic(err)
	}
	c1 := rpcpb.NewRpcServiceClient(conn1)

	var tx1ID []byte
	var tx2ID []byte
	var conn1Step1 = false
	var conn1Step2 = false
	var conn2Step1 = false

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)
		defer cancel()
		stream, err := c1.RpcGetNewTransaction(ctx, &rpcpb.GetNewTransactionRequest{})
		if err != nil {
			return
		}

		response1, err := stream.Recv()
		conn1Step1 = true
		assert.Nil(t, err)
		assert.NotEqual(t, 0, len(tx1ID))
		assert.Equal(t, tx1ID, response1.GetTransaction().GetId())

		response2, err := stream.Recv()
		conn1Step2 = true
		assert.Nil(t, err)
		assert.NotEqual(t, 0, len(tx2ID))
		assert.Equal(t, tx2ID, response2.GetTransaction().GetId())
	}()

	// Create a grpc connection and a client
	conn2, err := grpc.Dial(fmt.Sprint(":", rpcContext.serverPort), grpc.WithInsecure())
	if err != nil {
		panic(err)
	}
	c2 := rpcpb.NewRpcServiceClient(conn2)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)
		defer cancel()
		stream, err := c2.RpcGetNewTransaction(ctx, &rpcpb.GetNewTransactionRequest{})
		if err != nil {
			return
		}
		response1, err := stream.Recv()
		conn2Step1 = true
		assert.Nil(t, err)
		assert.NotEqual(t, 0, len(tx1ID))
		assert.Equal(t, tx1ID, response1.GetTransaction().GetId())
	}()
	time.Sleep(time.Second)

	tx1ID, _, err = logic.Send(rpcContext.account, receiverAccount.GetKeyPair().GenerateAddress(), common.NewAmount(6), common.NewAmount(0), common.NewAmount(0), common.NewAmount(0), "", rpcContext.bc, rpcContext.node)
	assert.Nil(t, err)
	time.Sleep(time.Second)
	assert.Equal(t, true, conn1Step1)
	assert.Equal(t, false, conn1Step2)
	assert.Equal(t, true, conn2Step1)
	conn2.Close()

	tx2ID, _, err = logic.Send(rpcContext.account, receiverAccount.GetKeyPair().GenerateAddress(), common.NewAmount(6), common.NewAmount(0), common.NewAmount(0), common.NewAmount(0), "", rpcContext.bc, rpcContext.node)
	time.Sleep(time.Second)
	assert.Equal(t, true, conn1Step2)
	conn1.Close()

	_, _, err = logic.Send(rpcContext.account, receiverAccount.GetKeyPair().GenerateAddress(), common.NewAmount(4), common.NewAmount(0), common.NewAmount(0), common.NewAmount(0), "", rpcContext.bc, rpcContext.node)
	time.Sleep(time.Second)
	assert.Equal(t, false, rpcContext.bc.GetTxPool().EventBus.HasCallback(core.NewTransactionTopic))

	rpcContext.consensus.Stop()
	core.WaitDoneOrTimeout(func() bool {
		return !rpcContext.consensus.IsProducingBlock()
	}, 20)
	time.Sleep(time.Second)
}

func TestRpcGetAllTransactionsFromTxPool(t *testing.T) {
	rpcContext, err := createRpcTestContext(12)
	if err != nil {
		panic(err)
	}
	defer rpcContext.destroyContext()

	receiverAccount, err := logic.CreateAccount(strings.Replace(client.GetAccountFilePath(), "accounts", "accounts_test", -1), "test")
	if err != nil {
		panic(err)
	}

	conn1, err := grpc.Dial(fmt.Sprint(":", rpcContext.serverPort), grpc.WithInsecure())
	if err != nil {
		panic(err)
	}
	c1 := rpcpb.NewRpcServiceClient(conn1)

	// generate new transaction
	pubKeyHash, _ := rpcContext.account.GetKeyPair().GenerateAddress().GetPubKeyHash()
	utxos, err := core.NewUTXOIndex(rpcContext.bc.GetUtxoCache()).GetUTXOsByAmount(pubKeyHash, common.NewAmount(6))
	assert.Nil(t, err)

	sendTxParam := core.NewSendTxParam(rpcContext.account.GetKeyPair().GenerateAddress(),
		rpcContext.account.GetKeyPair(),
		receiverAccount.GetKeyPair().GenerateAddress(),
		common.NewAmount(6),
		common.NewAmount(0),
		common.NewAmount(0),
		common.NewAmount(0),
		"")
	transaction, err := core.NewUTXOTransaction(utxos, sendTxParam)
	// put a tx into txpool
	c1.RpcSendTransaction(context.Background(), &rpcpb.SendTransactionRequest{Transaction: transaction.ToProto().(*corepb.Transaction)})

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)
	defer cancel()
	// get a tx from txpool
	result, err := c1.RpcGetAllTransactionsFromTxPool(ctx, &rpcpb.GetAllTransactionsRequest{})
	assert.Nil(t, err)
	// assert result
	assert.NotNil(t, result)
	assert.Equal(t, 1, len(result.Transactions))

	time.Sleep(time.Second)
}

func TestRpcService_RpcSubscribe(t *testing.T) {
	rpcContext, err := createRpcTestContext(13)
	if err != nil {
		panic(err)
	}
	defer rpcContext.destroyContext()

	time.Sleep(100 * time.Millisecond)

	// Create a grpc connection and a client
	conn1, err := grpc.Dial(fmt.Sprint(":", rpcContext.serverPort), grpc.WithInsecure())
	if err != nil {
		panic(err)
	}
	defer conn1.Close()
	c1 := rpcpb.NewRpcServiceClient(conn1)

	// Create a grpc connection and a client
	conn2, err := grpc.Dial(fmt.Sprint(":", rpcContext.serverPort), grpc.WithInsecure())
	if err != nil {
		panic(err)
	}
	//defer conn2.Close()
	c2 := rpcpb.NewRpcServiceClient(conn2)

	// Test GetVersion with support client version
	count1 := 0
	count2 := 0
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)
		defer cancel()
		cc, err := c1.RpcSubscribe(ctx, &rpcpb.SubscribeRequest{Topics: []string{"topic1", "topic2"}})
		assert.Nil(t, err)

		resp, err := cc.Recv()
		assert.Nil(t, err)
		assert.Equal(t, resp.Data, "data1")
		count1 += 1

		resp, err = cc.Recv()
		assert.Nil(t, err)
		assert.Equal(t, resp.Data, "data2")
		count1 += 1

		resp, err = cc.Recv()
		count2 += 1
	}()

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)
		defer cancel()
		cc, err := c2.RpcSubscribe(ctx, &rpcpb.SubscribeRequest{Topics: []string{"topic1", "topic3"}})
		assert.Nil(t, err)

		resp, err := cc.Recv()
		assert.Nil(t, err)
		assert.Equal(t, resp.Data, "data1")
		count2 += 1

		resp, err = cc.Recv()
		assert.Nil(t, err)
		assert.Equal(t, resp.Data, "data3")
		count2 += 1

		resp, err = cc.Recv()
		count2 += 1
	}()
	time.Sleep(time.Second)

	//publish topic 1. Both nodes will get the message
	rpcContext.bc.GetEventManager().Trigger([]*core.Event{core.NewEvent("topic1", "data1")})
	assert.Nil(t, err)
	time.Sleep(time.Second)
	assert.Equal(t, 1, count1)
	assert.Equal(t, 1, count2)

	//publish topic2. Only node 1 will get the message
	rpcContext.bc.GetEventManager().Trigger([]*core.Event{core.NewEvent("topic2", "data2")})
	assert.Nil(t, err)
	time.Sleep(time.Second)
	assert.Equal(t, 2, count1)
	assert.Equal(t, 1, count2)

	//publish topic3. Only node 2 will get the message
	rpcContext.bc.GetEventManager().Trigger([]*core.Event{core.NewEvent("topic3", "data3")})
	assert.Nil(t, err)
	time.Sleep(time.Second)
	assert.Equal(t, 2, count1)
	assert.Equal(t, 2, count2)

	//publish topic4. No nodes will get the message
	rpcContext.bc.GetEventManager().Trigger([]*core.Event{core.NewEvent("topic4", "data4")})
	assert.Nil(t, err)
	time.Sleep(time.Second)
	assert.Equal(t, 2, count1)
	assert.Equal(t, 2, count2)
}

func TestRpcGetLastIrreversibleBlock(t *testing.T) {
	rpcContext, err := createRpcTestContext(14)
	if err != nil {
		panic(err)
	}
	defer rpcContext.destroyContext()

	rpcContext.consensus.Setup(rpcContext.node, rpcContext.account.GetKeyPair().GenerateAddress().String())
	rpcContext.consensus.Start()

	for rpcContext.bc.GetMaxHeight() < 50 {
	}

	rpcContext.consensus.Stop()
	t.Log(rpcContext.bc.GetMaxHeight())
	core.WaitDoneOrTimeout(func() bool {
		return !rpcContext.consensus.IsProducingBlock()
	}, 20)
	time.Sleep(time.Second)

	// Create a grpc connection and a client
	conn, err := grpc.Dial(fmt.Sprint(":", rpcContext.serverPort), grpc.WithInsecure())
	if err != nil {
		panic(err)
	}
	defer conn.Close()
	c := rpcpb.NewRpcServiceClient(conn)

	block20, err := rpcContext.bc.GetBlockByHeight(20)
	assert.Nil(t, err)
	rpcContext.bc.SetLIBHash(block20.GetHash())

	response, err := c.RpcGetLastIrreversibleBlock(context.Background(), &rpcpb.GetLastIrreversibleBlockRequest{})
	assert.Nil(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, []byte(block20.GetHash()), response.Block.GetHeader().GetHash())
	assert.Equal(t, uint64(20), response.Block.GetHeader().GetHeight())

}

func createRpcTestContext(startPortOffset uint32) (*RpcTestContext, error) {
	context := RpcTestContext{}
	context.store = storage.NewRamStorage()

	client.RemoveAccountFile()

	// Create accounts
	account, err := logic.CreateAccount(strings.Replace(client.GetAccountFilePath(), "accounts", "accounts_test", -1), "test")
	if err != nil {
		context.destroyContext()
		panic(err)
	}
	context.account = account

	// Create a blockchain with PoW consensus and sender account as coinbase (so its balance starts with 10)
	context.consensus = consensus.NewProofOfWork()
	scManager := vm.NewV8EngineManager(client.Address{})
	bc, err := logic.CreateBlockchain(account.GetKeyPair().GenerateAddress(), context.store, context.consensus, 1280000, scManager, 1000000)
	if err != nil {
		context.destroyContext()
		panic(err)
	}
	context.bc = bc

	// Prepare a PoW node that put mining reward to the sender's address
	pool := core.NewBlockPool(0)
	context.node = network.FakeNodeWithPidAndAddr(pool, bc, "a", "b")

	// Start a grpc server
	context.rpcServer = NewGrpcServer(context.node, "temp")
	context.serverPort = defaultRpcPort + startPortOffset // use a different port as other integration tests
	context.rpcServer.Start(context.serverPort)
	return &context, nil
}

func (context *RpcTestContext) destroyContext() {
	if context.rpcServer != nil {
		context.rpcServer.Stop()
	}

	if context.store != nil {
		context.store.Close()
	}
}

func getBalance(utxos []*corepb.Utxo) *common.Amount {
	amount := common.NewAmount(0)
	for _, utxo := range utxos {
		amount = amount.Add(common.NewAmountFromBytes(utxo.Amount))
	}
	return amount
}

func TestRpcService_RpcEstimateGas(t *testing.T) {
	logger.SetLevel(logger.WarnLevel)
	// Create storage
	store := storage.NewRamStorage()
	defer store.Close()
	client.RemoveAccountFile()

	// Create accounts
	senderAccount, err := logic.CreateAccount(strings.Replace(client.GetAccountFilePath(), "accounts", "accounts_test", -1), "test")
	if err != nil {
		panic(err)
	}

	minerAccount, err := logic.CreateAccount(strings.Replace(client.GetAccountFilePath(), "accounts", "accounts_test", -1), "test")
	if err != nil {
		panic(err)
	}

	// Create a blockchain with PoW consensus and sender account as coinbase (so its balance starts with 10)
	pow := consensus.NewProofOfWork()
	scManager := vm.NewV8EngineManager(client.Address{})
	bc, err := logic.CreateBlockchain(senderAccount.GetKeyPair().GenerateAddress(), store, pow, 1280000, scManager, 1000000)
	if err != nil {
		panic(err)
	}

	// Prepare a PoW node that put mining reward to the sender's address
	pool := core.NewBlockPool(0)
	node := network.FakeNodeWithPidAndAddr(pool, bc, "a", "b")
	pow.Setup(node, minerAccount.GetKeyPair().GenerateAddress().String())
	pow.SetTargetBit(0)

	// Start a grpc server
	server := NewGrpcServer(node, "temp")
	server.Start(defaultRpcPort + 15) // use a different port as other integration tests
	defer server.Stop()

	time.Sleep(100 * time.Millisecond)

	// Create a grpc connection and a client
	conn, err := grpc.Dial(fmt.Sprint(":", defaultRpcPort+15), grpc.WithInsecure())
	if err != nil {
		panic(err)
	}
	defer conn.Close()
	c := rpcpb.NewAdminServiceClient(conn)
	rpcClient := rpcpb.NewRpcServiceClient(conn)

	// deploy contract
	contract := "'usestrict';var StepRecorder=function(){};StepRecorder.prototype={record:function(addr,steps){var originalSteps=LocalStorage.get(addr);" +
		"LocalStorage.set(addr,originalSteps+steps);return _native_reward.record(addr,steps);},dapp_schedule:function(){}};module.exports = new StepRecorder();"
	// Initiate a RPC send request
	sendResp, err := c.RpcSend(context.Background(), &rpcpb.SendRequest{
		From:        senderAccount.GetKeyPair().GenerateAddress().String(),
		To:          "",
		Amount:      common.NewAmount(1).Bytes(),
		AccountPath: strings.Replace(client.GetAccountFilePath(), "accounts", "accounts_test", -1),
		Tip:         common.NewAmount(0).Bytes(),
		Data:        contract,
		GasLimit:    common.NewAmount(30000).Bytes(),
		GasPrice:    common.NewAmount(1).Bytes(),
	})

	assert.Nil(t, err)
	contractAddr := sendResp.ContractAddress

	// Start mining to approve the transaction
	pow.Start()
	for bc.GetMaxHeight() < 1 {
	}
	pow.Stop()

	time.Sleep(time.Second)
	// estimate contract
	contract = "{\"function\":\"record\",\"args\":[\"damnkW1X8KtnDLoKErLzAgaBtXDZKRywfF\",\"2000\"]}"
	pubKeyHash, _ := senderAccount.GetKeyPair().GenerateAddress().GetPubKeyHash()
	utxos, err := core.NewUTXOIndex(bc.GetUtxoCache()).GetUTXOsByAmount(pubKeyHash, common.NewAmount(1))
	sendTxParam := core.NewSendTxParam(senderAccount.GetKeyPair().GenerateAddress(),
		senderAccount.GetKeyPair(),
		client.NewAddress(contractAddr),
		common.NewAmount(1),
		common.NewAmount(0),
		common.NewAmount(0),
		common.NewAmount(0),
		contract)
	tx, err := core.NewUTXOTransaction(utxos, sendTxParam)
	estimateGasRequest := &rpcpb.EstimateGasRequest{Transaction: tx.ToProto().(*corepb.Transaction)}
	gasResp, err := rpcClient.RpcEstimateGas(context.Background(), estimateGasRequest)
	assert.Nil(t, err)
	gasCount := gasResp.GasCount
	gas := common.NewAmountFromBytes(gasCount)

	assert.True(t, gas.Cmp(common.NewAmount(0)) > 0)

	client.RemoveAccountFile()
}

func TestRpcService_RpcGasPrice(t *testing.T) {
	logger.SetLevel(logger.WarnLevel)
	// Create storage
	store := storage.NewRamStorage()
	defer store.Close()
	client.RemoveAccountFile()

	// Create accounts
	senderAccount, err := logic.CreateAccount(strings.Replace(client.GetAccountFilePath(), "accounts", "accounts_test", -1), "test")
	if err != nil {
		panic(err)
	}

	minerAccount, err := logic.CreateAccount(strings.Replace(client.GetAccountFilePath(), "accounts", "accounts_test", -1), "test")
	if err != nil {
		panic(err)
	}

	// Create a blockchain with PoW consensus and sender account as coinbase (so its balance starts with 10)
	pow := consensus.NewProofOfWork()
	scManager := vm.NewV8EngineManager(client.Address{})
	bc, err := logic.CreateBlockchain(senderAccount.GetKeyPair().GenerateAddress(), store, pow, 1280000, scManager, 1000000)
	if err != nil {
		panic(err)
	}

	// Prepare a PoW node that put mining reward to the sender's address
	pool := core.NewBlockPool(0)
	node := network.FakeNodeWithPidAndAddr(pool, bc, "a", "b")
	pow.Setup(node, minerAccount.GetKeyPair().GenerateAddress().String())
	pow.SetTargetBit(0)

	// Start a grpc server
	server := NewGrpcServer(node, "temp")
	server.Start(defaultRpcPort + 16) // use a different port as other integration tests
	defer server.Stop()

	time.Sleep(100 * time.Millisecond)

	// Create a grpc connection and a client
	conn, err := grpc.Dial(fmt.Sprint(":", defaultRpcPort+16), grpc.WithInsecure())
	if err != nil {
		panic(err)
	}
	defer conn.Close()
	rpcClient := rpcpb.NewRpcServiceClient(conn)

	// Start mining to approve the transaction
	pow.Start()
	for bc.GetMaxHeight() < 1 {
	}
	pow.Stop()

	time.Sleep(time.Second)

	gasPriceRequest := &rpcpb.GasPriceRequest{}
	gasPriceResponse, err := rpcClient.RpcGasPrice(context.Background(), gasPriceRequest)
	assert.Nil(t, err)
	gasPrice := gasPriceResponse.GasPrice
	price := common.NewAmountFromBytes(gasPrice)

	assert.True(t, price.Cmp(common.NewAmount(0)) > 0)

	client.RemoveAccountFile()
}
