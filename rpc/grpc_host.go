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
	"net"
	"strings"

	logger "github.com/sirupsen/logrus"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"

	"github.com/dappley/go-dappley/network"
	"github.com/dappley/go-dappley/rpc/pb"
)

const (
	defaultRpcPort = 50051
	passwordToken  = "password"
)

type Server struct {
	srv      *grpc.Server
	node     *network.Node
	password string
}

func NewGrpcServer(node *network.Node, adminPassword string) *Server {
	return &Server{grpc.NewServer(), node, adminPassword}
}

func (s *Server) Start(port uint32) {
	go func() {
		if port == 0 {
			port = defaultRpcPort
		}
		lis, err := net.Listen("tcp", fmt.Sprint(":", port))
		if err != nil {
			logger.WithError(err).WithFields(logger.Fields{"port": port}).
				Panic("Server: failed to listen to TCP port")
		}

		srv := grpc.NewServer(grpc.UnaryInterceptor(s.AuthInterceptor))
		rpcpb.RegisterRpcServiceServer(srv, &RpcService{s.node})
		rpcpb.RegisterAdminServiceServer(srv, &AdminRpcService{s.node})
		if err := srv.Serve(lis); err != nil {
			logger.WithError(err).Fatal("Server: encounters an error while serving")
		}
	}()
}

func (s *Server) AuthInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	if info.Server.(Service).IsPrivate() {
		peer, ok := peer.FromContext(ctx)
		if !ok || len(peer.Addr.String()) == 0 {
			return nil, status.Errorf(codes.Unauthenticated, "unknown ip")
		}
		ip := strings.Split(peer.Addr.String(), ":")
		if ip[0] != "127.0.0.1" {
			return nil, status.Errorf(codes.Unauthenticated, "unauthorized access")
		}

	}
	return handler(ctx, req)
}

func (s *Server) Stop() {
	s.srv.Stop()
}
