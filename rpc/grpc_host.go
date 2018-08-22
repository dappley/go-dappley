package logic

import (
	"log"
	"net"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"github.com/dappley/go-dappley/rpc/pb"
)

const (
	port = ":50051"
)

// server is used to implement helloworld.GreeterServer.
type server struct{}

// SayHello implements helloworld.GreeterServer
func (s *server) RpcCreateWallet(ctx context.Context, in *rpc.CreateWalletRequest) (*rpc.CreateWalletReply, error) {
	return &rpc.CreateWalletReply{Message: "Hello " + in.Name}, nil
}

func (s *server) RpcGetBalance(ctx context.Context, in *rpc.GetBalanceRequest) (*rpc.GetBalanceReply, error) {
	return &rpc.GetBalanceReply{Message: "Hello " + in.Name}, nil
}

func (s *server) RpcSend(ctx context.Context, in *rpc.SendRequest) (*rpc.SendReply, error) {
	return &rpc.SendReply{Message: "Hello " + in.Name}, nil
}

func startRPC() {
	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	rpc.RegisterConnectServer(s, &server{})
	s.Serve(lis)
}
