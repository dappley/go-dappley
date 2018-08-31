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
	"testing"
	"github.com/dappley/go-dappley/network"
	"google.golang.org/grpc"
	"github.com/dappley/go-dappley/rpc/pb"
	"golang.org/x/net/context"
	"time"
	"github.com/stretchr/testify/assert"
	"fmt"
	"google.golang.org/grpc/metadata"
)

func TestNewGrpcServer(t *testing.T) {
	node := network.NewNode(nil)
	grpcServer := NewGrpcServer(node, "password")
	assert.Equal(t,node,grpcServer.node)
	assert.Equal(t,"password",grpcServer.password)
}

//integration test
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
	md := metadata.Pairs("password", "temp")
	ctx := metadata.NewOutgoingContext(context.Background(), md)
	response, err := c.RpcGetPeerInfo(ctx,&rpcpb.GetPeerInfoRequest{})
	assert.Nil(t, err)

	ret := &network.PeerList{}
	ret.FromProto(response.PeerList)
	assert.Equal(t,node.GetPeerList(),ret)

}