package main

import (
	"context"
	"fmt"

	"github.com/dappley/go-dappley/core/account"
	rpcpb "github.com/dappley/go-dappley/rpc/pb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func contractQueryCommandHandler(ctx context.Context, c interface{}, flags cmdFlags) {
	contractAddr := *(flags[flagContractAddr].(*string))
	queryKey := *(flags[flagKey].(*string))
	queryValue := *(flags[flagValue].(*string))
	contractAccount := account.NewTransactionAccountByAddress(account.NewAddress(contractAddr))

	if !contractAccount.IsValid() {
		fmt.Println("Error: contract address is not valid!")
		return
	}
	if queryKey == "" && queryValue == "" {
		fmt.Println("Error: query key and value cannot be null at the same time!")
		return
	}
	response, err := c.(rpcpb.RpcServiceClient).RpcContractQuery(ctx, &rpcpb.ContractQueryRequest{
		ContractAddr: contractAddr,
		Key:          queryKey,
		Value:        queryValue,
	})
	if err != nil {
		switch status.Code(err) {
		case codes.Unavailable:
			fmt.Println("Error: server is not reachable!")
		default:
			fmt.Println("Error:", status.Convert(err).Message())
		}
		return
	}
	resultKey := response.GetKey()
	resultValue := response.GetValue()

	fmt.Println("Contract query result: key=", resultKey, ", value=", resultValue)
}
