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
	"errors"
	"fmt"
	"github.com/dappley/go-dappley/common/deadline"
	"github.com/dappley/go-dappley/consensus"
	"github.com/dappley/go-dappley/core/block"
	"github.com/dappley/go-dappley/core/blockchain"
	"github.com/dappley/go-dappley/core/utxo"
	blockchainMock "github.com/dappley/go-dappley/logic/lblockchain/mocks"
	"github.com/dappley/go-dappley/logic/ltransaction"

	"github.com/dappley/go-dappley/core/blockproducerinfo"
	"github.com/dappley/go-dappley/core/scState"
	"github.com/dappley/go-dappley/core/transaction"
	"github.com/dappley/go-dappley/core/transaction/pb"
	"github.com/dappley/go-dappley/core/utxo/pb"
	"github.com/dappley/go-dappley/logic/blockproducer"
	"github.com/dappley/go-dappley/logic/blockproducer/mocks"
	"github.com/dappley/go-dappley/logic/lblock"
	"github.com/dappley/go-dappley/logic/lblockchain"
	"github.com/dappley/go-dappley/logic/lutxo"
	"github.com/dappley/go-dappley/logic/transactionpool"
	"github.com/stretchr/testify/mock"
	"testing"
	"time"

	"github.com/dappley/go-dappley/util"

	"github.com/dappley/go-dappley/common"
	"github.com/dappley/go-dappley/core/account"

	"github.com/dappley/go-dappley/logic"
	"github.com/dappley/go-dappley/network"
	"github.com/dappley/go-dappley/rpc/pb"
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
	account    *account.Account
	bp         *blockproducer.BlockProducer
	bm         *lblockchain.BlockchainManager
	node       *network.Node
	rpcServer  *Server
	serverPort uint32
}

func CreateProducer(producerAddr, addr account.Address, db storage.Storage, txPool *transactionpool.TransactionPool, node *network.Node) (*lblockchain.BlockchainManager, *blockproducer.BlockProducer) {
	producer := blockproducerinfo.NewBlockProducerInfo(producerAddr.String())

	libPolicy := &blockchainMock.LIBPolicy{}
	libPolicy.On("GetProducers").Return(nil)
	libPolicy.On("GetMinConfirmationNum").Return(6)
	libPolicy.On("IsBypassingLibCheck").Return(true)
	consensus := &blockchainMock.Consensus{}
	consensus.On("Validate", mock.Anything).Return(true)

	bc := lblockchain.CreateBlockchain(addr, db, libPolicy, txPool, vm.NewV8EngineManager(account.Address{}), 100000)
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

func TestServer_StartRPC(t *testing.T) {

	pid := "QmWsMUDBeWxwU4R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ"
	addr := "/ip4/127.0.0.1/tcp/10000"
	node := network.FakeNodeWithPeer(pid, addr)
	//start grpc server
	server := NewGrpcServer(node, nil, consensus.NewDPOS(nil), "temp")
	server.Start(defaultRpcPort)
	defer server.Stop()

	time.Sleep(time.Millisecond * 100)
	//prepare grpc account
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

	// Create accounts
	senderAccount, err := logic.CreateAccountWithPassphrase("test", logic.GetTestAccountPath())
	if err != nil {
		panic(err)
	}
	receiverAccount, err := logic.CreateAccountWithPassphrase("test", logic.GetTestAccountPath())
	if err != nil {
		panic(err)
	}
	minerAccount, err := logic.CreateAccountWithPassphrase("test", logic.GetTestAccountPath())
	if err != nil {
		panic(err)
	}
	node := network.FakeNodeWithPidAndAddr(store, "a", "b")

	// Create a blockchain with PoW consensus and sender account as coinbase (so its balance starts with 10)
	bm, bp := CreateProducer(
		minerAccount.GetAddress(),
		senderAccount.GetAddress(),
		store,
		transactionpool.NewTransactionPool(node, 128000),
		node,
	)

	// Start a grpc server
	server := NewGrpcServer(node, bm, consensus.NewDPOS(nil), "temp")
	server.Start(defaultRpcPort + 1) // use a different port as other integration tests
	defer server.Stop()

	time.Sleep(100 * time.Millisecond)

	// Create a grpc connection and a account
	conn, err := grpc.Dial(fmt.Sprint(":", defaultRpcPort+1), grpc.WithInsecure())
	if err != nil {
		panic(err)
	}
	defer conn.Close()
	c := rpcpb.NewAdminServiceClient(conn)

	// Initiate a RPC send request
	_, err = c.RpcSend(context.Background(), &rpcpb.SendRequest{
		From:        senderAccount.GetAddress().String(),
		To:          receiverAccount.GetAddress().String(),
		Amount:      common.NewAmount(7).Bytes(),
		AccountPath: logic.GetTestAccountPath(),
		Tip:         common.NewAmount(2).Bytes(),
		Data:        "",
	})
	assert.Nil(t, err)

	// Start mining to approve the transaction
	bp.Start()
	for bm.Getblockchain().GetMaxHeight() < 1 {
	}
	bp.Stop()

	time.Sleep(100 * time.Millisecond)

	// Check balance
	minedReward := transaction.Subsidy
	senderBalance, err := logic.GetBalance(senderAccount.GetAddress(), bm.Getblockchain())
	assert.Nil(t, err)
	receiverBalance, err := logic.GetBalance(receiverAccount.GetAddress(), bm.Getblockchain())
	assert.Nil(t, err)
	minerBalance, err := logic.GetBalance(minerAccount.GetAddress(), bm.Getblockchain())
	assert.Nil(t, err)

	leftBalance, _ := minedReward.Sub(common.NewAmount(7))
	leftBalance, _ = leftBalance.Sub(common.NewAmount(2))
	minerRewardBalance := minedReward.Times(bm.Getblockchain().GetMaxHeight()).Add(common.NewAmount(2))
	assert.Equal(t, leftBalance, senderBalance)
	assert.Equal(t, common.NewAmount(7), receiverBalance)
	assert.Equal(t, minerRewardBalance, minerBalance)
	logic.RemoveAccountTestFile()
}

func TestRpcSendContract(t *testing.T) {

	logger.SetLevel(logger.WarnLevel)
	// Create storage
	store := storage.NewRamStorage()
	defer store.Close()

	// Create accounts
	senderAccount, err := logic.CreateAccountWithPassphrase("test", logic.GetTestAccountPath())
	if err != nil {
		panic(err)
	}

	minerAccount, err := logic.CreateAccountWithPassphrase("test", logic.GetTestAccountPath())
	if err != nil {
		panic(err)
	}

	node := network.FakeNodeWithPidAndAddr(store, "a", "b")
	bm, bp := CreateProducer(
		minerAccount.GetAddress(),
		senderAccount.GetAddress(),
		store,
		transactionpool.NewTransactionPool(node, 128000),
		node,
	)

	// Start a grpc server
	server := NewGrpcServer(node, bm, consensus.NewDPOS(nil), "temp")
	server.Start(defaultRpcPort + 10) // use a different port as other integration tests
	defer server.Stop()

	time.Sleep(100 * time.Millisecond)

	// Create a grpc connection and a account
	conn, err := grpc.Dial(fmt.Sprint(":", defaultRpcPort+10), grpc.WithInsecure())
	if err != nil {
		panic(err)
	}
	defer conn.Close()
	c := rpcpb.NewAdminServiceClient(conn)

	contract := "dapp_schedule"
	// Initiate a RPC send request
	_, err = c.RpcSend(context.Background(), &rpcpb.SendRequest{
		From:        senderAccount.GetAddress().String(),
		To:          "",
		Amount:      common.NewAmount(7).Bytes(),
		AccountPath: logic.GetTestAccountPath(),
		Tip:         common.NewAmount(2).Bytes(),
		Data:        contract,
		GasLimit:    common.NewAmount(30000).Bytes(),
		GasPrice:    common.NewAmount(1).Bytes(),
	})
	assert.Nil(t, err)

	// Start mining to approve the transaction
	bp.Start()
	for bm.Getblockchain().GetMaxHeight() < 1 {
	}
	bp.Stop()

	time.Sleep(time.Second)

	//check smart contract deployment
	res := string("")
loop:
	for i := bm.Getblockchain().GetMaxHeight(); i > 0; i-- {
		blk, err := bm.Getblockchain().GetBlockByHeight(i)
		assert.Nil(t, err)
		for _, tx := range blk.GetTransactions() {
			ctx := ltransaction.NewTxContract(tx)
			if ctx != nil {
				res = ctx.GetContract()
				break loop
			}
		}
	}
	assert.Equal(t, contract, res)

	logic.RemoveAccountTestFile()
}

func TestRpcGetVersion(t *testing.T) {
	rpcContext, err := createRpcTestContext(2)
	if err != nil {
		panic(err)
	}
	defer rpcContext.destroyContext()

	time.Sleep(100 * time.Millisecond)

	// Create a grpc connection and a account
	conn, err := grpc.Dial(fmt.Sprint(":", rpcContext.serverPort), grpc.WithInsecure())
	if err != nil {
		panic(err)
	}
	defer conn.Close()
	c := rpcpb.NewRpcServiceClient(conn)

	// Test GetVersion with support account version
	response, err := c.RpcGetVersion(context.Background(), &rpcpb.GetVersionRequest{ProtoVersion: "1.0.0"})
	assert.Nil(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, "1.0.0", response.ProtoVersion, "1.0.0")

	// Test GetVersion with unsupport account version -- invalid version length
	response, err = c.RpcGetVersion(context.Background(), &rpcpb.GetVersionRequest{ProtoVersion: "1.0.0.0"})
	assert.Nil(t, response)

	assert.Equal(t, codes.InvalidArgument, status.Code(err))
	assert.Equal(t, "proto version not supported", status.Convert(err).Message())

	// Test GetVersion with unsupport account version
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

	rpcContext.bp.Start()

	for rpcContext.bm.Getblockchain().GetMaxHeight() < 5 {

	}

	rpcContext.bp.Stop()
	util.WaitDoneOrTimeout(func() bool {
		return !rpcContext.bp.IsProducingBlock()
	}, 20)
	time.Sleep(time.Second)

	// Create a grpc connection and a account
	conn, err := grpc.Dial(fmt.Sprint(":", rpcContext.serverPort), grpc.WithInsecure())
	if err != nil {
		panic(err)
	}
	defer conn.Close()
	c := rpcpb.NewRpcServiceClient(conn)
	response, err := c.RpcGetBlockchainInfo(context.Background(), &rpcpb.GetBlockchainInfoRequest{})
	assert.Nil(t, err)

	tailBlock, err := rpcContext.bm.Getblockchain().GetTailBlock()
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

	receiverAccount, err := logic.CreateAccountWithPassphrase("test", logic.GetTestAccountPath())
	if err != nil {
		panic(err)
	}

	logic.Send(rpcContext.account, receiverAccount.GetAddress(), common.NewAmount(6), common.NewAmount(0), common.NewAmount(0), common.NewAmount(0), "", rpcContext.bm.Getblockchain())

	rpcContext.bp.Start()

	for rpcContext.bm.Getblockchain().GetMaxHeight() < MinUtxoBlockHeaderCount {

	}

	rpcContext.bp.Stop()
	util.WaitDoneOrTimeout(func() bool {
		return !rpcContext.bp.IsProducingBlock()
	}, 20)
	time.Sleep(time.Second)

	// Create a grpc connection and a account
	conn, err := grpc.Dial(fmt.Sprint(":", rpcContext.serverPort), grpc.WithInsecure())
	if err != nil {
		panic(err)
	}
	defer conn.Close()
	c := rpcpb.NewRpcServiceClient(conn)

	senderResponse, err := c.RpcGetUTXO(context.Background(), &rpcpb.GetUTXORequest{Address: rpcContext.account.GetAddress().String()})
	assert.Nil(t, err)
	assert.NotNil(t, senderResponse)
	minedReward := transaction.Subsidy
	leftAmount, err := minedReward.Times(rpcContext.bm.Getblockchain().GetMaxHeight() + 1).Sub(common.NewAmount(6))
	assert.Equal(t, leftAmount, getBalance(senderResponse.Utxos))

	tailBlock, err := rpcContext.bm.Getblockchain().GetTailBlock()
	assert.Equal(t, int(MinUtxoBlockHeaderCount), len(senderResponse.BlockHeaders))
	assert.Equal(t, []byte(tailBlock.GetHash()), senderResponse.BlockHeaders[0].GetHash())

	receiverResponse, err := c.RpcGetUTXO(context.Background(), &rpcpb.GetUTXORequest{Address: receiverAccount.GetAddress().String()})
	assert.Nil(t, err)
	assert.NotNil(t, receiverResponse)
	assert.Equal(t, common.NewAmount(6), getBalance(receiverResponse.Utxos))
	logic.RemoveAccountTestFile()
}

func TestRpcGetBlocks(t *testing.T) {
	rpcContext, err := createRpcTestContext(5)
	if err != nil {
		panic(err)
	}
	defer rpcContext.destroyContext()

	rpcContext.bp.Start()

	for rpcContext.bm.Getblockchain().GetMaxHeight() < 500 {
	}

	rpcContext.bp.Stop()
	util.WaitDoneOrTimeout(func() bool {
		return !rpcContext.bp.IsProducingBlock()
	}, 20)
	time.Sleep(time.Second)

	genesisBlock := lblockchain.NewGenesisBlock(rpcContext.account.GetAddress(), transaction.Subsidy)
	// Create a grpc connection and a account
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
	block1, err := rpcContext.bm.Getblockchain().GetBlockByHeight(1)
	assert.Equal(t, []byte(block1.GetHash()), response.Blocks[0].GetHeader().GetHash())
	block20, err := rpcContext.bm.Getblockchain().GetBlockByHeight(uint64(maxGetBlocksCount))
	assert.Equal(t, []byte(block20.GetHash()), response.Blocks[19].GetHeader().GetHash())

	// Check query loop
	var startBlockHashes [][]byte
	queryCount := (int(rpcContext.bm.Getblockchain().GetMaxHeight())+maxGetBlocksCount-1)/maxGetBlocksCount - 1
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
			leftCount := int(rpcContext.bm.Getblockchain().GetMaxHeight()) - queryCount*maxGetBlocksCount
			assert.Equal(t, leftCount, len(response.Blocks))
		} else {
			assert.Equal(t, maxGetBlocksCount, len(response.Blocks))
		}
	}

	tailBlock, err := rpcContext.bm.Getblockchain().GetTailBlock()
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

	rpcContext.bp.Start()

	for rpcContext.bm.Getblockchain().GetMaxHeight() < 50 {
	}

	rpcContext.bp.Stop()
	util.WaitDoneOrTimeout(func() bool {
		return !rpcContext.bp.IsProducingBlock()
	}, 20)
	time.Sleep(time.Second)

	// Create a grpc connection and a account
	conn, err := grpc.Dial(fmt.Sprint(":", rpcContext.serverPort), grpc.WithInsecure())
	if err != nil {
		panic(err)
	}
	defer conn.Close()
	c := rpcpb.NewRpcServiceClient(conn)

	block20, err := rpcContext.bm.Getblockchain().GetBlockByHeight(20)
	response, err := c.RpcGetBlockByHash(context.Background(), &rpcpb.GetBlockByHashRequest{Hash: block20.GetHash()})
	assert.Nil(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, []byte(block20.GetHash()), response.Block.GetHeader().GetHash())

	tailBlock, err := rpcContext.bm.Getblockchain().GetTailBlock()
	response, err = c.RpcGetBlockByHash(context.Background(), &rpcpb.GetBlockByHashRequest{Hash: tailBlock.GetHash()})
	assert.Nil(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, []byte(tailBlock.GetHash()), response.Block.GetHeader().GetHash())

	response, err = c.RpcGetBlockByHash(context.Background(), &rpcpb.GetBlockByHashRequest{Hash: []byte("noexists")})
	assert.Nil(t, response)
	assert.Equal(t, codes.NotFound, status.Code(err))
	assert.Equal(t, lblockchain.ErrBlockDoesNotExist.Error(), status.Convert(err).Message())
}

func TestRpcGetBlockByHeight(t *testing.T) {
	rpcContext, err := createRpcTestContext(7)
	if err != nil {
		panic(err)
	}
	defer rpcContext.destroyContext()

	rpcContext.bp.Start()

	for rpcContext.bm.Getblockchain().GetMaxHeight() < 50 {
	}

	rpcContext.bp.Stop()
	util.WaitDoneOrTimeout(func() bool {
		return !rpcContext.bp.IsProducingBlock()
	}, 20)
	time.Sleep(time.Second)

	// Create a grpc connection and a account
	conn, err := grpc.Dial(fmt.Sprint(":", rpcContext.serverPort), grpc.WithInsecure())
	if err != nil {
		panic(err)
	}
	defer conn.Close()
	c := rpcpb.NewRpcServiceClient(conn)

	block20, err := rpcContext.bm.Getblockchain().GetBlockByHeight(20)
	response, err := c.RpcGetBlockByHeight(context.Background(), &rpcpb.GetBlockByHeightRequest{Height: 20})
	assert.Nil(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, []byte(block20.GetHash()), response.Block.GetHeader().GetHash())

	tailBlock, err := rpcContext.bm.Getblockchain().GetTailBlock()
	response, err = c.RpcGetBlockByHeight(context.Background(), &rpcpb.GetBlockByHeightRequest{Height: tailBlock.GetHeight()})
	assert.Nil(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, []byte(tailBlock.GetHash()), response.Block.GetHeader().GetHash())

	response, err = c.RpcGetBlockByHeight(context.Background(), &rpcpb.GetBlockByHeightRequest{Height: tailBlock.GetHeight() + 1})
	assert.Nil(t, response)
	assert.Equal(t, codes.NotFound, status.Code(err))
	assert.Equal(t, lblockchain.ErrBlockDoesNotExist.Error(), status.Convert(err).Message())
}

func GetUTXOsfromAmount(inputUTXOs []*utxo.UTXO, amount *common.Amount, tip *common.Amount, gasLimit *common.Amount, gasPrice *common.Amount) ([]*utxo.UTXO, error) {
	if tip != nil {
		amount = amount.Add(tip)
	}
	if gasLimit != nil {
		limitedFee := gasLimit.Mul(gasPrice)
		amount = amount.Add(limitedFee)
	}
	var retUtxos []*utxo.UTXO
	sum := common.NewAmount(0)
	for _, u := range inputUTXOs {
		sum = sum.Add(u.Value)
		retUtxos = append(retUtxos, u)
		if sum.Cmp(amount) >= 0 {
			break
		}
	}

	if sum.Cmp(amount) < 0 {
		//return nil, "ErrInsufficientFund"
		return nil, errors.New("cli: the balance is insufficient")
	}

	return retUtxos, nil
}

func TestRpcVerifyTransaction(t *testing.T) {
	rpcContext, err := createRpcTestContext(18)
	defer rpcContext.destroyContext()
	if err != nil {
		panic(err)
	}
	fromAcc, err := logic.CreateAccountWithPassphrase("test", logic.GetTestAccountPath())
	if err != nil {
		panic(err)
	}
	toAcc, err := logic.CreateAccountWithPassphrase("test", logic.GetTestAccountPath())
	if err != nil {
		panic(err)
	}
	//
	gctx, err := ltransaction.NewGasChangeTx(account.NewTransactionAccountByAddress(fromAcc.GetAddress()), 0, common.NewAmount(uint64(0)), common.NewAmount(uint64(3000)), common.NewAmount(uint64(1)), 1)
	utxoIndex := lutxo.NewUTXOIndex(rpcContext.bm.Getblockchain().GetUtxoCache())
	utxoIndex.UpdateUtxo(&gctx)
	utxoIndex.Save()
	// Create a grpc connection and a account
	conn, err := grpc.Dial(fmt.Sprint(":", rpcContext.serverPort), grpc.WithInsecure())
	if err != nil {
		panic(err)
	}
	defer conn.Close()
	c := rpcpb.NewRpcServiceClient(conn)
	senderResponse, err := c.RpcGetUTXO(context.Background(), &rpcpb.GetUTXORequest{Address: fromAcc.GetAddress().String()})
	assert.Nil(t, err)
	assert.NotNil(t, senderResponse)

	utxos := senderResponse.GetUtxos()
	var inputUtxos []*utxo.UTXO
	for _, u := range utxos {
		uu := utxo.UTXO{}
		uu.Value = common.NewAmountFromBytes(u.Amount)
		uu.Txid = u.Txid
		uu.PubKeyHash = account.PubKeyHash(u.PublicKeyHash)
		uu.TxIndex = int(u.TxIndex)
		inputUtxos = append(inputUtxos, &uu)
	}
	tip := common.NewAmount(0)
	gasLimit := common.NewAmount(0)
	gasPrice := common.NewAmount(0)

	tx_utxos, err := GetUTXOsfromAmount(inputUtxos, common.NewAmount(3000), tip, gasLimit, gasPrice)
	if err != nil {
		panic(err)
	}
	sendTxParam := transaction.NewSendTxParam(fromAcc.GetAddress(), fromAcc.GetKeyPair(),
		toAcc.GetAddress(), common.NewAmount(3000), tip, gasLimit, gasPrice, "")
	tx, err := ltransaction.NewUTXOTransaction(tx_utxos, sendTxParam)
	sendTransactionRequest := &rpcpb.SendTransactionRequest{Transaction: tx.ToProto().(*transactionpb.Transaction)}
	_, err = c.(rpcpb.RpcServiceClient).RpcSendTransaction(context.Background(), sendTransactionRequest)
	rpcContext.bp.Start()
	maxHeight := rpcContext.bm.Getblockchain().GetMaxHeight()
	for maxHeight < 2 {
		maxHeight = rpcContext.bm.Getblockchain().GetMaxHeight()
	}
	rpcContext.bp.Stop()
	util.WaitDoneOrTimeout(func() bool {
		return !rpcContext.bp.IsProducingBlock()
	}, 20)
	//send second transaction
	gctx2, err := ltransaction.NewGasChangeTx(account.NewTransactionAccountByAddress(fromAcc.GetAddress()), 0, common.NewAmount(uint64(0)), common.NewAmount(uint64(1000)), common.NewAmount(uint64(1)), 1)
	utxoIndex = lutxo.NewUTXOIndex(rpcContext.bm.Getblockchain().GetUtxoCache())
	utxoIndex.UpdateUtxo(&gctx2)
	utxoIndex.Save()
	rpcContext.bm.Getblockchain()
	senderResponse2, err := c.RpcGetUTXO(context.Background(), &rpcpb.GetUTXORequest{Address: fromAcc.GetAddress().String()})
	assert.Nil(t, err)
	assert.NotNil(t, senderResponse2)

	utxos2 := senderResponse2.GetUtxos()
	var inputUtxos2 []*utxo.UTXO
	for _, u := range utxos2 {
		uu := utxo.UTXO{}
		uu.Value = common.NewAmountFromBytes(u.Amount)
		uu.Txid = u.Txid
		uu.PubKeyHash = account.PubKeyHash(u.PublicKeyHash)
		uu.TxIndex = int(u.TxIndex)
		inputUtxos2 = append(inputUtxos2, &uu)
	}

	tx_utxos2, err := GetUTXOsfromAmount(inputUtxos2, common.NewAmount(1000), common.NewAmount(0), common.NewAmount(0), common.NewAmount(0))
	if err != nil {
		panic(err)
	}
	sendTxParam2 := transaction.NewSendTxParam(fromAcc.GetAddress(), fromAcc.GetKeyPair(),
		toAcc.GetAddress(), common.NewAmount(1000), tip, gasLimit, gasPrice, "")
	txTmp, err := ltransaction.NewUTXOTransaction(tx_utxos2, sendTxParam2)
	sendTransactionRequest2 := &rpcpb.SendTransactionRequest{Transaction: txTmp.ToProto().(*transactionpb.Transaction)}
	_, err = c.(rpcpb.RpcServiceClient).RpcSendTransaction(context.Background(), sendTransactionRequest2)
	assert.Nil(t, err)
	logic.RemoveAccountTestFile()
}

func TestRpcSendTransaction(t *testing.T) {
	rpcContext, err := createRpcTestContext(8)
	if err != nil {
		panic(err)
	}
	defer rpcContext.destroyContext()

	receiverAccount, err := logic.CreateAccountWithPassphrase("test", logic.GetTestAccountPath())
	if err != nil {
		panic(err)
	}

	rpcContext.bp.Start()

	maxHeight := rpcContext.bm.Getblockchain().GetMaxHeight()
	for maxHeight < 2 {
		maxHeight = rpcContext.bm.Getblockchain().GetMaxHeight()
	}
	// Create a grpc connection and a account
	conn, err := grpc.Dial(fmt.Sprint(":", rpcContext.serverPort), grpc.WithInsecure())
	if err != nil {
		panic(err)
	}
	defer conn.Close()
	c := rpcpb.NewRpcServiceClient(conn)

	pubKeyHash := rpcContext.account.GetPubKeyHash()
	utxos, err := lutxo.NewUTXOIndex(rpcContext.bm.Getblockchain().GetUtxoCache()).GetUTXOsByAmount(pubKeyHash, common.NewAmount(6))
	assert.Nil(t, err)

	sendTxParam := transaction.NewSendTxParam(rpcContext.account.GetAddress(),
		rpcContext.account.GetKeyPair(),
		receiverAccount.GetAddress(),
		common.NewAmount(6),
		common.NewAmount(0),
		common.NewAmount(0),
		common.NewAmount(0),
		"")
	tx, err := ltransaction.NewUTXOTransaction(utxos, sendTxParam)
	successResponse, err := c.RpcSendTransaction(context.Background(), &rpcpb.SendTransactionRequest{Transaction: tx.ToProto().(*transactionpb.Transaction)})
	assert.Nil(t, err)
	assert.NotNil(t, successResponse)

	maxHeight = rpcContext.bm.Getblockchain().GetMaxHeight()
	for (rpcContext.bm.Getblockchain().GetMaxHeight() - maxHeight) < 2 {
	}

	utxos2, err := lutxo.NewUTXOIndex(rpcContext.bm.Getblockchain().GetUtxoCache()).GetUTXOsByAmount(pubKeyHash, common.NewAmount(6))
	sendTxParam2 := transaction.NewSendTxParam(rpcContext.account.GetAddress(),
		rpcContext.account.GetKeyPair(),
		receiverAccount.GetAddress(),
		common.NewAmount(6),
		common.NewAmount(0),
		common.NewAmount(0),
		common.NewAmount(0),
		"")
	errTransaction, err := ltransaction.NewUTXOTransaction(utxos2, sendTxParam2)
	errTransaction.Vin[0].Signature = []byte("invalid")
	failedResponse, err := c.RpcSendTransaction(context.Background(), &rpcpb.SendTransactionRequest{Transaction: errTransaction.ToProto().(*transactionpb.Transaction)})
	assert.Nil(t, failedResponse)
	assert.Equal(t, codes.FailedPrecondition, status.Code(err))
	assert.Equal(t, lblockchain.ErrTransactionVerifyFailed.Error(), status.Convert(err).Message())

	maxHeight = rpcContext.bm.Getblockchain().GetMaxHeight()
	for (rpcContext.bm.Getblockchain().GetMaxHeight() - maxHeight) < 2 {
	}

	rpcContext.bp.Stop()
	util.WaitDoneOrTimeout(func() bool {
		return !rpcContext.bp.IsProducingBlock()
	}, 20)
	time.Sleep(time.Second)

	minedReward := transaction.Subsidy
	leftAmount, err := minedReward.Times(rpcContext.bm.Getblockchain().GetMaxHeight() + 1).Sub(common.NewAmount(6))
	realAmount, err := logic.GetBalance(rpcContext.account.GetAddress(), rpcContext.bm.Getblockchain())
	assert.Equal(t, leftAmount, realAmount)
	recvAmount, err := logic.GetBalance(receiverAccount.GetAddress(), rpcContext.bm.Getblockchain())
	assert.Equal(t, common.NewAmount(6), recvAmount)
	logic.RemoveAccountTestFile()
}

func TestRpcService_RpcSendBatchTransaction(t *testing.T) {
	logger.SetLevel(logger.DebugLevel)
	rpcContext, err := createRpcTestContext(99)
	if err != nil {
		panic(err)
	}
	defer rpcContext.destroyContext()

	receiverAccount1, err := logic.CreateAccountWithPassphrase("test", logic.GetTestAccountPath())
	if err != nil {
		panic(err)
	}
	receiverAccount2, err := logic.CreateAccountWithPassphrase("test", logic.GetTestAccountPath())
	if err != nil {
		panic(err)
	}
	receiverAccount4, err := logic.CreateAccountWithPassphrase("test", logic.GetTestAccountPath())
	if err != nil {
		panic(err)
	}

	rpcContext.bp.Start()

	maxHeight := rpcContext.bm.Getblockchain().GetMaxHeight()
	for maxHeight < 2 {
		maxHeight = rpcContext.bm.Getblockchain().GetMaxHeight()
	}

	// Create a grpc connection and a account
	conn, err := grpc.Dial(fmt.Sprint(":", rpcContext.serverPort), grpc.WithInsecure())
	if err != nil {
		panic(err)
	}
	defer conn.Close()
	c := rpcpb.NewRpcServiceClient(conn)

	pubKeyHash := rpcContext.account.GetPubKeyHash()
	utxoIndex := lutxo.NewUTXOIndex(rpcContext.bm.Getblockchain().GetUtxoCache())
	utxos, err := utxoIndex.GetUTXOsByAmount(pubKeyHash, common.NewAmount(3))
	assert.Nil(t, err)

	sendTxParam1 := transaction.NewSendTxParam(rpcContext.account.GetAddress(),
		rpcContext.account.GetKeyPair(),
		receiverAccount1.GetAddress(),
		common.NewAmount(3),
		common.NewAmount(0),
		common.NewAmount(0),
		common.NewAmount(0),
		"")
	transaction1, err := ltransaction.NewUTXOTransaction(utxos, sendTxParam1)
	utxoIndex.UpdateUtxos([]*transaction.Transaction{&transaction1})
	utxos, err = utxoIndex.GetUTXOsByAmount(pubKeyHash, common.NewAmount(2))
	sendTxParam2 := transaction.NewSendTxParam(rpcContext.account.GetAddress(),
		rpcContext.account.GetKeyPair(),
		receiverAccount2.GetAddress(),
		common.NewAmount(2),
		common.NewAmount(0),
		common.NewAmount(0),
		common.NewAmount(0),
		"")
	transaction2, err := ltransaction.NewUTXOTransaction(utxos, sendTxParam2)
	utxoIndex.UpdateUtxos([]*transaction.Transaction{&transaction2})
	pubKeyHash1 := receiverAccount1.GetPubKeyHash()
	utxos, err = utxoIndex.GetUTXOsByAmount(pubKeyHash1, common.NewAmount(1))
	sendTxParam3 := transaction.NewSendTxParam(receiverAccount1.GetAddress(),
		receiverAccount1.GetKeyPair(),
		receiverAccount2.GetAddress(),
		common.NewAmount(1),
		common.NewAmount(0),
		common.NewAmount(0),
		common.NewAmount(0),
		"")
	transaction3, err := ltransaction.NewUTXOTransaction(utxos, sendTxParam3)
	utxoIndex.UpdateUtxos([]*transaction.Transaction{&transaction3})

	rpcContext.bp.Stop()
	time.Sleep(time.Second)

	successResponse, err := c.RpcSendBatchTransaction(context.Background(), &rpcpb.SendBatchTransactionRequest{Transactions: []*transactionpb.Transaction{transaction1.ToProto().(*transactionpb.Transaction), transaction2.ToProto().(*transactionpb.Transaction), transaction3.ToProto().(*transactionpb.Transaction)}})
	assert.Nil(t, err)
	assert.NotNil(t, successResponse)

	rpcContext.bp.Start()
	maxHeight = rpcContext.bm.Getblockchain().GetMaxHeight()
	for (rpcContext.bm.Getblockchain().GetMaxHeight() - maxHeight) < 2 {
	}
	rpcContext.bp.Stop()
	time.Sleep(time.Second)

	utxos2, err := utxoIndex.GetUTXOsByAmount(pubKeyHash, common.NewAmount(3))
	sendTxParamErr := transaction.NewSendTxParam(rpcContext.account.GetAddress(),
		rpcContext.account.GetKeyPair(),
		receiverAccount4.GetAddress(),
		common.NewAmount(3),
		common.NewAmount(0),
		common.NewAmount(0),
		common.NewAmount(0),
		"")
	errTransaction, err := ltransaction.NewUTXOTransaction(utxos2, sendTxParamErr)

	sendTxParam4 := transaction.NewSendTxParam(rpcContext.account.GetAddress(),
		rpcContext.account.GetKeyPair(),
		receiverAccount4.GetAddress(),
		common.NewAmount(3),
		common.NewAmount(0),
		common.NewAmount(0),
		common.NewAmount(0),
		"")
	transaction4, err := ltransaction.NewUTXOTransaction(utxos2, sendTxParam4)
	errTransaction.Vin[0].Signature = []byte("invalid")
	failedResponse, err := c.RpcSendBatchTransaction(context.Background(), &rpcpb.SendBatchTransactionRequest{Transactions: []*transactionpb.Transaction{errTransaction.ToProto().(*transactionpb.Transaction), transaction4.ToProto().(*transactionpb.Transaction)}})
	assert.Nil(t, failedResponse)
	st := status.Convert(err)
	assert.Equal(t, codes.Unknown, st.Code())

	detail0 := st.Details()[0].(*rpcpb.SendTransactionStatus)
	detail1 := st.Details()[1].(*rpcpb.SendTransactionStatus)
	assert.Equal(t, errTransaction.ID, detail0.Txid)
	assert.Equal(t, uint32(codes.FailedPrecondition), detail0.Code)
	assert.Equal(t, uint32(codes.OK), detail1.Code)

	rpcContext.bp.Start()
	maxHeight = rpcContext.bm.Getblockchain().GetMaxHeight()
	for (rpcContext.bm.Getblockchain().GetMaxHeight() - maxHeight) < 2 {
	}

	rpcContext.bp.Stop()
	time.Sleep(time.Second)

	util.WaitDoneOrTimeout(func() bool {
		return !rpcContext.bp.IsProducingBlock()
	}, 20)

	minedReward := transaction.Subsidy
	leftAmount, err := minedReward.Times(rpcContext.bm.Getblockchain().GetMaxHeight() + 1).Sub(common.NewAmount(8))
	realAmount, err := logic.GetBalance(rpcContext.account.GetAddress(), rpcContext.bm.Getblockchain())
	assert.Equal(t, leftAmount, realAmount)
	recvAmount1, err := logic.GetBalance(receiverAccount1.GetAddress(), rpcContext.bm.Getblockchain())
	recvAmount2, err := logic.GetBalance(receiverAccount2.GetAddress(), rpcContext.bm.Getblockchain())
	recvAmount4, err := logic.GetBalance(receiverAccount4.GetAddress(), rpcContext.bm.Getblockchain())
	assert.Equal(t, common.NewAmount(2), recvAmount1)
	assert.Equal(t, common.NewAmount(3), recvAmount2)
	assert.Equal(t, common.NewAmount(3), recvAmount4)
	logic.RemoveAccountTestFile()
}

func TestGetNewTransaction(t *testing.T) {
	logger.SetLevel(logger.WarnLevel)
	rpcContext, err := createRpcTestContext(11)
	if err != nil {
		panic(err)
	}
	defer rpcContext.destroyContext()

	receiverAccount, err := logic.CreateAccountWithPassphrase("test", logic.GetTestAccountPath())
	if err != nil {
		panic(err)
	}

	rpcContext.bp.Start()

	// Create a grpc connection and a account
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

	// Create a grpc connection and a account
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

	tx1ID, _, err = logic.Send(rpcContext.account, receiverAccount.GetAddress(), common.NewAmount(6), common.NewAmount(0), common.NewAmount(0), common.NewAmount(0), "", rpcContext.bm.Getblockchain())

	assert.Nil(t, err)
	time.Sleep(time.Second)
	assert.Equal(t, true, conn1Step1)
	assert.Equal(t, false, conn1Step2)
	assert.Equal(t, true, conn2Step1)
	conn2.Close()

	tx2ID, _, err = logic.Send(rpcContext.account, receiverAccount.GetAddress(), common.NewAmount(6), common.NewAmount(0), common.NewAmount(0), common.NewAmount(0), "", rpcContext.bm.Getblockchain())

	time.Sleep(time.Second)
	assert.Equal(t, true, conn1Step2)
	conn1.Close()

	_, _, err = logic.Send(rpcContext.account, receiverAccount.GetAddress(), common.NewAmount(4), common.NewAmount(0), common.NewAmount(0), common.NewAmount(0), "", rpcContext.bm.Getblockchain())

	time.Sleep(time.Second)
	assert.Equal(t, false, rpcContext.bm.Getblockchain().GetTxPool().EventBus.HasCallback(transactionpool.NewTransactionTopic))

	rpcContext.bp.Stop()
	util.WaitDoneOrTimeout(func() bool {
		return !rpcContext.bp.IsProducingBlock()
	}, 20)
	time.Sleep(time.Second)
	logic.RemoveAccountTestFile()
}

func TestRpcGetAllTransactionsFromTxPool(t *testing.T) {
	rpcContext, err := createRpcTestContext(12)
	if err != nil {
		panic(err)
	}
	defer rpcContext.destroyContext()

	receiverAccount, err := logic.CreateAccountWithPassphrase("test", logic.GetTestAccountPath())
	if err != nil {
		panic(err)
	}

	conn1, err := grpc.Dial(fmt.Sprint(":", rpcContext.serverPort), grpc.WithInsecure())
	if err != nil {
		panic(err)
	}
	c1 := rpcpb.NewRpcServiceClient(conn1)

	// generate new transaction
	pubKeyHash := rpcContext.account.GetPubKeyHash()
	utxos, err := lutxo.NewUTXOIndex(rpcContext.bm.Getblockchain().GetUtxoCache()).GetUTXOsByAmount(pubKeyHash, common.NewAmount(6))
	assert.Nil(t, err)

	sendTxParam := transaction.NewSendTxParam(rpcContext.account.GetAddress(),
		rpcContext.account.GetKeyPair(),
		receiverAccount.GetAddress(),
		common.NewAmount(6),
		common.NewAmount(0),
		common.NewAmount(0),
		common.NewAmount(0),
		"")
	transaction, err := ltransaction.NewUTXOTransaction(utxos, sendTxParam)
	// put a tx into txpool
	c1.RpcSendTransaction(context.Background(), &rpcpb.SendTransactionRequest{Transaction: transaction.ToProto().(*transactionpb.Transaction)})

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)
	defer cancel()
	// get a tx from txpool
	result, err := c1.RpcGetAllTransactionsFromTxPool(ctx, &rpcpb.GetAllTransactionsRequest{})
	assert.Nil(t, err)
	// assert result
	assert.NotNil(t, result)
	assert.Equal(t, 1, len(result.Transactions))

	time.Sleep(time.Second)
	logic.RemoveAccountTestFile()
}

func TestRpcService_RpcSubscribe(t *testing.T) {
	rpcContext, err := createRpcTestContext(13)
	if err != nil {
		panic(err)
	}
	defer rpcContext.destroyContext()

	time.Sleep(100 * time.Millisecond)

	// Create a grpc connection and a account
	conn1, err := grpc.Dial(fmt.Sprint(":", rpcContext.serverPort), grpc.WithInsecure())
	if err != nil {
		panic(err)
	}
	defer conn1.Close()
	c1 := rpcpb.NewRpcServiceClient(conn1)

	// Create a grpc connection and a account
	conn2, err := grpc.Dial(fmt.Sprint(":", rpcContext.serverPort), grpc.WithInsecure())
	if err != nil {
		panic(err)
	}
	//defer conn2.Close()
	c2 := rpcpb.NewRpcServiceClient(conn2)

	// Test GetVersion with support account version
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
	rpcContext.bm.Getblockchain().GetEventManager().Trigger([]*scState.Event{scState.NewEvent("topic1", "data1")})
	assert.Nil(t, err)
	time.Sleep(time.Second)
	assert.Equal(t, 1, count1)
	assert.Equal(t, 1, count2)

	//publish topic2. Only node 1 will get the message
	rpcContext.bm.Getblockchain().GetEventManager().Trigger([]*scState.Event{scState.NewEvent("topic2", "data2")})
	assert.Nil(t, err)
	time.Sleep(time.Second)
	assert.Equal(t, 2, count1)
	assert.Equal(t, 1, count2)

	//publish topic3. Only node 2 will get the message
	rpcContext.bm.Getblockchain().GetEventManager().Trigger([]*scState.Event{scState.NewEvent("topic3", "data3")})
	assert.Nil(t, err)
	time.Sleep(time.Second)
	assert.Equal(t, 2, count1)
	assert.Equal(t, 2, count2)

	//publish topic4. No nodes will get the message
	rpcContext.bm.Getblockchain().GetEventManager().Trigger([]*scState.Event{scState.NewEvent("topic4", "data4")})
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

	rpcContext.bp.Start()

	for rpcContext.bm.Getblockchain().GetMaxHeight() < 50 {
	}

	rpcContext.bp.Stop()
	t.Log(rpcContext.bm.Getblockchain().GetMaxHeight())
	util.WaitDoneOrTimeout(func() bool {
		return !rpcContext.bp.IsProducingBlock()
	}, 20)
	time.Sleep(time.Second)

	// Create a grpc connection and a account
	conn, err := grpc.Dial(fmt.Sprint(":", rpcContext.serverPort), grpc.WithInsecure())
	if err != nil {
		panic(err)
	}
	defer conn.Close()
	c := rpcpb.NewRpcServiceClient(conn)

	block20, err := rpcContext.bm.Getblockchain().GetBlockByHeight(20)
	assert.Nil(t, err)
	rpcContext.bm.Getblockchain().SetLIBHash(block20.GetHash())

	response, err := c.RpcGetLastIrreversibleBlock(context.Background(), &rpcpb.GetLastIrreversibleBlockRequest{})
	assert.Nil(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, []byte(block20.GetHash()), response.Block.GetHeader().GetHash())
	assert.Equal(t, uint64(20), response.Block.GetHeader().GetHeight())

}

func createRpcTestContext(startPortOffset uint32) (*RpcTestContext, error) {
	context := RpcTestContext{}
	context.store = storage.NewRamStorage()

	// Create accounts
	acc, err := logic.CreateAccountWithPassphrase("test", logic.GetTestAccountPath())
	if err != nil {
		context.destroyContext()
		panic(err)
	}
	context.account = acc

	context.node = network.FakeNodeWithPidAndAddr(context.store, "a", "b")
	context.bm, context.bp = CreateProducer(
		acc.GetAddress(),
		acc.GetAddress(),
		context.store,
		transactionpool.NewTransactionPool(context.node, 128000),
		context.node,
	)

	// Start a grpc server
	dpos := consensus.NewDPOS(nil)
	dpos.SetDynasty(consensus.NewDynasty(nil, 5, 15))
	context.rpcServer = NewGrpcServer(context.node, context.bm, dpos, "temp")
	context.serverPort = defaultRpcPort + startPortOffset // use a different port as other integration tests
	context.rpcServer.Start(context.serverPort)
	logic.RemoveAccountTestFile()
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

func getBalance(utxos []*utxopb.Utxo) *common.Amount {
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

	// Create accounts
	senderAccount, err := logic.CreateAccountWithPassphrase("test", logic.GetTestAccountPath())
	if err != nil {
		panic(err)
	}

	minerAccount, err := logic.CreateAccountWithPassphrase("test", logic.GetTestAccountPath())
	if err != nil {
		panic(err)
	}

	// Create a blockchain with PoW consensus and sender account as coinbase (so its balance starts with 10)
	node := network.FakeNodeWithPidAndAddr(store, "a", "b")
	bm, bp := CreateProducer(
		minerAccount.GetAddress(),
		senderAccount.GetAddress(),
		store,
		transactionpool.NewTransactionPool(node, 128000),
		node,
	)

	// Start a grpc server
	server := NewGrpcServer(node, bm, consensus.NewDPOS(nil), "temp")
	server.Start(defaultRpcPort + 100) // use a different port as other integration tests
	defer server.Stop()

	time.Sleep(100 * time.Millisecond)

	// Create a grpc connection and a account
	conn, err := grpc.Dial(fmt.Sprint(":", defaultRpcPort+100), grpc.WithInsecure())
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
		From:        senderAccount.GetAddress().String(),
		To:          "",
		Amount:      common.NewAmount(1).Bytes(),
		AccountPath: logic.GetTestAccountPath(),
		Tip:         common.NewAmount(0).Bytes(),
		Data:        contract,
		GasLimit:    common.NewAmount(30000).Bytes(),
		GasPrice:    common.NewAmount(1).Bytes(),
	})

	assert.Nil(t, err)
	contractAddr := sendResp.ContractAddress

	// Start mining to approve the transaction
	bp.Start()
	for bm.Getblockchain().GetMaxHeight() < 1 {
	}
	bp.Stop()

	time.Sleep(time.Second)
	// estimate contract
	contract = "{\"function\":\"record\",\"args\":[\"damnkW1X8KtnDLoKErLzAgaBtXDZKRywfF\",\"2000\"]}"
	pubKeyHash := senderAccount.GetPubKeyHash()
	utxos, err := lutxo.NewUTXOIndex(bm.Getblockchain().GetUtxoCache()).GetUTXOsByAmount(pubKeyHash, common.NewAmount(1))
	sendTxParam := transaction.NewSendTxParam(senderAccount.GetAddress(),
		senderAccount.GetKeyPair(),
		account.NewAddress(contractAddr),
		common.NewAmount(1),
		common.NewAmount(0),
		common.NewAmount(0),
		common.NewAmount(0),
		contract)
	tx, err := ltransaction.NewUTXOTransaction(utxos, sendTxParam)
	estimateGasRequest := &rpcpb.EstimateGasRequest{Transaction: tx.ToProto().(*transactionpb.Transaction)}
	gasResp, err := rpcClient.RpcEstimateGas(context.Background(), estimateGasRequest)
	assert.Nil(t, err)
	gasCount := gasResp.GasCount
	gas := common.NewAmountFromBytes(gasCount)

	assert.True(t, gas.Cmp(common.NewAmount(0)) > 0)

	logic.RemoveAccountTestFile()
}

func TestRpcService_RpcGasPrice(t *testing.T) {
	logger.SetLevel(logger.WarnLevel)
	// Create storage
	store := storage.NewRamStorage()
	defer store.Close()

	// Create accounts
	senderAccount, err := logic.CreateAccountWithPassphrase("test", logic.GetTestAccountPath())
	if err != nil {
		panic(err)
	}

	minerAccount, err := logic.CreateAccountWithPassphrase("test", logic.GetTestAccountPath())
	if err != nil {
		panic(err)
	}

	node := network.FakeNodeWithPidAndAddr(store, "a", "b")
	bm, bp := CreateProducer(
		minerAccount.GetAddress(),
		senderAccount.GetAddress(),
		store,
		transactionpool.NewTransactionPool(node, 128000),
		node,
	)

	// Start a grpc server
	server := NewGrpcServer(node, nil, consensus.NewDPOS(nil), "temp")
	server.Start(defaultRpcPort + 16) // use a different port as other integration tests
	defer server.Stop()

	time.Sleep(100 * time.Millisecond)

	// Create a grpc connection and a account
	conn, err := grpc.Dial(fmt.Sprint(":", defaultRpcPort+16), grpc.WithInsecure())
	if err != nil {
		panic(err)
	}
	defer conn.Close()
	rpcClient := rpcpb.NewRpcServiceClient(conn)

	// Start mining to approve the transaction
	bp.Start()
	for bm.Getblockchain().GetMaxHeight() < 1 {
	}
	bp.Stop()

	time.Sleep(time.Second)

	gasPriceRequest := &rpcpb.GasPriceRequest{}
	gasPriceResponse, err := rpcClient.RpcGasPrice(context.Background(), gasPriceRequest)
	assert.Nil(t, err)
	gasPrice := gasPriceResponse.GasPrice
	price := common.NewAmountFromBytes(gasPrice)

	assert.True(t, price.Cmp(common.NewAmount(0)) > 0)

	logic.RemoveAccountTestFile()
}

func TestRpcService_RpcContractQuery(t *testing.T) {
	logger.SetLevel(logger.WarnLevel)
	// Create storage
	store := storage.NewRamStorage()
	defer store.Close()

	// Create accounts
	senderAccount, err := logic.CreateAccountWithPassphrase("test", logic.GetTestAccountPath())
	if err != nil {
		panic(err)
	}

	minerAccount, err := logic.CreateAccountWithPassphrase("test", logic.GetTestAccountPath())
	if err != nil {
		panic(err)
	}

	// Create a blockchain with PoW consensus and sender account as coinbase (so its balance starts with 10)
	node := network.FakeNodeWithPidAndAddr(store, "a", "b")
	bm, bp := CreateProducer(
		minerAccount.GetAddress(),
		senderAccount.GetAddress(),
		store,
		transactionpool.NewTransactionPool(node, 128000),
		node,
	)

	// Start a grpc server
	server := NewGrpcServer(node, bm, consensus.NewDPOS(nil), "temp")
	server.Start(defaultRpcPort + 17) // use a different port as other integration tests
	defer server.Stop()

	time.Sleep(100 * time.Millisecond)

	// Create a grpc connection and a account
	conn, err := grpc.Dial(fmt.Sprint(":", defaultRpcPort+17), grpc.WithInsecure())
	if err != nil {
		panic(err)
	}
	defer conn.Close()
	c := rpcpb.NewAdminServiceClient(conn)
	rpcClient := rpcpb.NewRpcServiceClient(conn)

	// deploy contract
	contract := "'use strict';var VideoSign=function(){};VideoSign.prototype={put_sign:function(key,value){LocalStorage.set(key,value)},get_sign:function(key){return LocalStorage.get(key)}," +
		"dapp_schedule:function(){}};module.exports=new VideoSign();"
	// Initiate a RPC send request
	sendResp, err := c.RpcSend(context.Background(), &rpcpb.SendRequest{
		From:        senderAccount.GetAddress().String(),
		To:          "",
		Amount:      common.NewAmount(1).Bytes(),
		AccountPath: logic.GetTestAccountPath(),
		Tip:         common.NewAmount(0).Bytes(),
		Data:        contract,
		GasLimit:    common.NewAmount(30000).Bytes(),
		GasPrice:    common.NewAmount(1).Bytes(),
	})

	assert.Nil(t, err)
	contractAddr := sendResp.ContractAddress

	// Start mining to approve the transaction
	bp.Start()
	for bm.Getblockchain().GetMaxHeight() < 1 {
	}
	bp.Stop()

	time.Sleep(time.Second)
	key := "k"
	value := "abc"
	// estimate contract
	contract = "{\"function\":\"put_sign\",\"args\":[\"" + key + "\",\"" + value + "\"]}"
	sendResp, err = c.RpcSend(context.Background(), &rpcpb.SendRequest{
		From:        senderAccount.GetAddress().String(),
		To:          contractAddr,
		Amount:      common.NewAmount(1).Bytes(),
		AccountPath: logic.GetTestAccountPath(),
		Tip:         common.NewAmount(0).Bytes(),
		Data:        contract,
		GasLimit:    common.NewAmount(30000).Bytes(),
		GasPrice:    common.NewAmount(1).Bytes(),
	})

	// Start mining to approve the transaction
	maxHeight := bm.Getblockchain().GetMaxHeight()
	bp.Start()
	for bm.Getblockchain().GetMaxHeight() < maxHeight+2 {
	}
	bp.Stop()

	// send query request
	queryRequest := &rpcpb.ContractQueryRequest{ContractAddr: contractAddr, Key: key}
	queryResp, err := rpcClient.RpcContractQuery(context.Background(), queryRequest)
	assert.Nil(t, err)

	assert.Equal(t, key, queryResp.Key, "RpcContractQuery get key failed")
	assert.Equal(t, value, queryResp.Value, "RpcContractQuery get value failed")

	queryRequest = &rpcpb.ContractQueryRequest{ContractAddr: contractAddr, Value: value}
	queryResp, err = rpcClient.RpcContractQuery(context.Background(), queryRequest)
	assert.Nil(t, err)

	assert.Equal(t, key, queryResp.Key, "RpcContractQuery get key failed")
	assert.Equal(t, value, queryResp.Value, "RpcContractQuery get value failed")

	logic.RemoveAccountTestFile()
}
