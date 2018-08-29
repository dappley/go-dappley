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
	"golang.org/x/net/context"
	"github.com/dappley/go-dappley/rpc/pb"
	"github.com/dappley/go-dappley/network"
	"github.com/dappley/go-dappley/network/pb"
	"net"
	"log"
	"google.golang.org/grpc"
	"fmt"
)

const (
	defaultRpcPort = 50051
)

// Server is used to implement helloworld.GreeterServer.
type Server struct{
	srv 	*grpc.Server
	node 	*network.Node
}

func NewGrpcServer(node *network.Node) *Server{
	return &Server{grpc.NewServer(),node}
}

// SayHello implements helloworld.GreeterServer
func (s *Server) RpcCreateWallet(ctx context.Context, in *rpcpb.CreateWalletRequest) (*rpcpb.CreateWalletResponse, error) {
	return &rpcpb.CreateWalletResponse{Message: "Hello " + in.Name}, nil
}

func (s *Server) RpcGetBalance(ctx context.Context, in *rpcpb.GetBalanceRequest) (*rpcpb.GetBalanceResponse, error) {
	return &rpcpb.GetBalanceResponse{Message: "Hello " + in.Name}, nil
}

func (s *Server) RpcSend(ctx context.Context, in *rpcpb.SendRequest) (*rpcpb.SendResponse, error) {
	return &rpcpb.SendResponse{Message: "Hello " + in.Name}, nil
}

func (s *Server) RpcGetPeerInfo(ctx context.Context, in *rpcpb.GetPeerInfoRequest) (*rpcpb.GetPeerInfoResponse, error) {
	return &rpcpb.GetPeerInfoResponse{
		PeerList:s.node.GetPeerList().ToProto().(*networkpb.Peerlist),
	}, nil
}

func (s *Server) Start(port uint32) {
	go func(){
		if port == 0{
			port = defaultRpcPort
		}
		lis, err := net.Listen("tcp", fmt.Sprint(":",port))
		if err != nil {
			log.Fatalf("failed to listen: %v", err)
		}
		srv := grpc.NewServer()
		rpcpb.RegisterConnectServer(srv, s)
		if err := srv.Serve(lis); err != nil {
			log.Fatalf("failed to serve: %s", err)
		}
	}()
}

func (s *Server) Stop() {
	s.srv.Stop()
}
