package main

import (
	"context"
	"fmt"

	rpcpb "github.com/dappley/go-dappley/rpc/pb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func clideleteProducerCommandHandler(ctx context.Context, c interface{}, flags cmdFlags) {

	height := *(flags[flagBlockHeight].(*uint64))
	if height == 0 {
		fmt.Println("\n Please enter an address. Example: cli deleteProducer -height 100")
		fmt.Println()
		return
	}
	_, err := c.(rpcpb.AdminServiceClient).RpcDeleteProducer(ctx, &rpcpb.DeleteProducerRequest{
		Height: height,
	})

	if err != nil {
		switch status.Code(err) {
		case codes.Unavailable:
			fmt.Println("Error: server is not reachable!")
		default:
			fmt.Println("Error: ", status.Convert(err).Message())
		}
		return
	}
	fmt.Println("Producer will be deleted.")
}
