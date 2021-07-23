package main

import (
	"context"
	"fmt"

	rpcpb "github.com/dappley/go-dappley/rpc/pb"
	"github.com/golang/protobuf/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func addPeerCommandHandler(ctx context.Context, account interface{}, flags cmdFlags) {
	req := &rpcpb.AddPeerRequest{
		FullAddress: *(flags[flagPeerFullAddr].(*string)),
	}
	response, err := account.(rpcpb.AdminServiceClient).RpcAddPeer(ctx, req)
	if err != nil {
		switch status.Code(err) {
		case codes.Unavailable:
			fmt.Println("Error: server is not reachable!")
		default:
			fmt.Println("Error: ", status.Convert(err).Message())
		}
		return
	}
	fmt.Println(proto.MarshalTextString(response))
}
