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
	"github.com/dappley/go-dappley/common"
	"testing"
	"time"
	"google.golang.org/grpc"
	"golang.org/x/net/context"
	"github.com/stretchr/testify/assert"
	"github.com/dappley/go-dappley/network"
	"github.com/dappley/go-dappley/rpc/pb"
	"github.com/dappley/go-dappley/storage"
	"github.com/dappley/go-dappley/client"
	"github.com/dappley/go-dappley/logic"
	"github.com/dappley/go-dappley/consensus"
	logger "github.com/sirupsen/logrus"
)

func TestServer_StartRPC(t *testing.T) {

	pid := "QmWsMUDBeWxwU4R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ"
	addr := "/ip4/127.0.0.1/tcp/10000"
	node := network.FakeNodeWithPeer(pid, addr)
	//start grpc server
	server := NewGrpcServer(node,"temp")
	server.Start(defaultRpcPort)
	defer server.Stop()

	time.Sleep(time.Millisecond*100)
	//prepare grpc client
	var conn *grpc.ClientConn
	conn, err := grpc.Dial(fmt.Sprint(":",defaultRpcPort), grpc.WithInsecure())
	assert.Nil(t, err)
	defer conn.Close()

	c := rpcpb.NewRpcServiceClient(conn)
	response, err := c.RpcGetPeerInfo(context.Background(), &rpcpb.GetPeerInfoRequest{})
	assert.Nil(t, err)

	ret := &network.PeerList{}
	ret.FromProto(response.PeerList)
	assert.Equal(t,node.GetPeerList(),ret)

}

func TestRpcSend(t *testing.T) {
	logger.SetLevel(logger.WarnLevel)
	// Create storage
	store := storage.NewRamStorage()
	defer store.Close()
	client.RemoveWalletFile()
	client.RemoveTestWalletFile()

	// Create wallets
	senderWallet, err := logic.CreateWallet()
	if err != nil {
		panic(err)
	}
	receiverWallet, err := logic.CreateWallet()
	if err != nil {
		panic(err)
	}

	// Create a blockchain with PoW consensus and sender wallet as coinbase (so its balance starts with 10)
	pow := consensus.NewProofOfWork()
	bc, err := logic.CreateBlockchain(senderWallet.GetAddress(), store, pow)
	if err != nil {
		panic(err)
	}

	// Prepare a PoW node that put mining reward to the sender's address
	node := network.FakeNodeWithPidAndAddr(bc, "a", "b")
	pow.Setup(node, senderWallet.GetAddress().Address)
	pow.SetTargetBit(0)

	// Start a grpc server
	server := NewGrpcServer(node, "temp")
	server.Start(defaultRpcPort + 1) // use a different port as other integration tests
	defer server.Stop()

	time.Sleep(100 * time.Millisecond)

	// Create a grpc connection and a client
	conn, err := grpc.Dial(fmt.Sprint(":", defaultRpcPort + 1), grpc.WithInsecure())
	if err != nil {
		panic(err)
	}
	defer conn.Close()
	c := rpcpb.NewRpcServiceClient(conn)

	// Initiate a RPC send request
	_, err = c.RpcSend(context.Background(), &rpcpb.SendRequest{
		From: senderWallet.GetAddress().Address,
		To: receiverWallet.GetAddress().Address,
		Amount: common.NewAmount(7).Bytes(),
	})
	assert.Nil(t, err)

	// Start mining to approve the transaction
	pow.Start()
	for bc.GetMaxHeight() < 1 {}
	pow.Stop()

	time.Sleep(100 * time.Millisecond)

	// Check balance
	minedReward := common.NewAmount(10)
	senderBalance, err := logic.GetBalance(senderWallet.GetAddress(), store)
	assert.Nil(t, err)
	receiverBalance, err := logic.GetBalance(receiverWallet.GetAddress(), store)
	assert.Nil(t, err)
	leftBalance,_ := minedReward.Times(bc.GetMaxHeight() + 1).Sub(common.NewAmount(7))
	assert.Equal(t, leftBalance, senderBalance) // minedReward * (blockHeight + 1) - (Send Balance)
	assert.Equal(t, common.NewAmount(7), receiverBalance)

	client.RemoveWalletFile()
	client.RemoveTestWalletFile()
}