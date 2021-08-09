package main

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"

	rpcpb "github.com/dappley/go-dappley/rpc/pb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func getBlockchainInfoCommandHandler(ctx context.Context, account interface{}, flags cmdFlags) {
	response, err := account.(rpcpb.RpcServiceClient).RpcGetBlockchainInfo(ctx, &rpcpb.GetBlockchainInfoRequest{})
	if err != nil {
		switch status.Code(err) {
		case codes.Unavailable:
			fmt.Println("Error: server is not reachable!")
		default:
			fmt.Println("Error: ", status.Convert(err).Message())
		}
		return
	}
	encodedResponse := map[string]interface{}{
		"TailBlockHash": hex.EncodeToString(response.TailBlockHash),
		"BlockHeight":   response.BlockHeight,
		"Producers":     response.Producers,
	}

	blockchainInfo, err := json.MarshalIndent(encodedResponse, "", "  ")
	if err != nil {
		fmt.Println("Error: ", err.Error())
		return
	}

	fmt.Println(string(blockchainInfo))
}
