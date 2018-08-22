package rpc

import (
	"log"
	"net"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"github.com/dappley/go-dappley/rpc/pb"
	"github.com/dappley/go-dappley/network"
	"github.com/dappley/go-dappley/network/pb"
)

const (
	port = ":50051"
)

// Server is used to implement helloworld.GreeterServer.
type Server struct{
	node *network.Node
}

func NewGrpcServer(node *network.Node) *Server{
	return &Server{node}
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

func StartRpc(srv rpcpb.ConnectServer) {
	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	rpcpb.RegisterConnectServer(s, srv)
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %s", err)
	}
}
