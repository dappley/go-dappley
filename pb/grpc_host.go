package logic

import (
	"log"
	"net"

	pb "github.com/dappworks/go-dappworks/logic"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

const (
	port = ":50051"
)

// server is used to implement helloworld.GreeterServer.
type server struct{}

// SayHello implements helloworld.GreeterServer
func (s *server) RpcCreateWallet(ctx context.Context, in *pb.CreateWalletRequest) (*pb.CreateWalletReply, error) {
	return &pb.CreateWalletReply{Message: "Hello " + in.Name}, nil
}

func (s *server) RpcGetBalance(ctx context.Context, in *pb.GetBalanceRequest) (*pb.GetBalanceReply, error) {
	return &pb.GetBalanceReply{Message: "Hello " + in.Name}, nil
}

func (s *server) RpcSend(ctx context.Context, in *pb.SendRequest) (*pb.SendReply, error) {
	return &pb.SendReply{Message: "Hello " + in.Name}, nil
}

func startRPC() {
	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	pb.RegisterConnectServer(s, &server{})
	s.Serve(lis)
}
