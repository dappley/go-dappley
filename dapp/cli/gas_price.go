package main

import (
	"context"
	"fmt"

	"github.com/dappley/go-dappley/common"
	rpcpb "github.com/dappley/go-dappley/rpc/pb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func gasPriceCommandHandler(ctx context.Context, account interface{}, flags cmdFlags) {
	gasPriceRequest := &rpcpb.GasPriceRequest{}
	gasPriceResponse, err := account.(rpcpb.RpcServiceClient).RpcGasPrice(ctx, gasPriceRequest)
	if err != nil {
		switch status.Code(err) {
		case codes.Unavailable:
			fmt.Println("Error: server is not reachable!")
		default:
			fmt.Println("Error:", status.Convert(err).Message())
		}
		return
	}
	gasPrice := gasPriceResponse.GasPrice
	fmt.Println("Gas price: ", common.NewAmountFromBytes(gasPrice).String())
}
