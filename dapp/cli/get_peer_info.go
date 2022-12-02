package main

import (
	"context"
	"fmt"

	rpcpb "github.com/dappley/go-dappley/rpc/pb"
	"github.com/golang/protobuf/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func getPeerInfoCommandHandler(ctx context.Context, account interface{}, flags cmdFlags) {
	response, err := account.(rpcpb.AdminServiceClient).RpcGetPeerInfo(ctx, &rpcpb.GetPeerInfoRequest{})
	if err != nil {
		switch status.Code(err) {
		case codes.Unavailable:
			fmt.Println("Error: server is not reachable!")
		default:
			fmt.Println("Error: ", err.Error())
		}
		return
	}
	fmt.Println(proto.MarshalTextString(response))
}
