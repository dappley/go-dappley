package main

import (
	"context"
	"fmt"

	"github.com/dappley/go-dappley/common"
	"github.com/dappley/go-dappley/core/account"
	"github.com/dappley/go-dappley/core/utxo"
	"github.com/dappley/go-dappley/logic"
	rpcpb "github.com/dappley/go-dappley/rpc/pb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func getBalanceCommandHandler(ctx context.Context, c interface{}, flags cmdFlags) {
	if len(*(flags[flagAddress].(*string))) == 0 {
		fmt.Println("\n Please enter an address. Example: cli getBalance -address 1MeSBgufmzwpiJNLemUe1emxAussBnz7a7")
		fmt.Println()
		return
	}
	response, err := logic.GetUtxoStream(c.(rpcpb.RpcServiceClient), &rpcpb.GetUTXORequest{
		Address: account.NewAddress(*(flags[flagAddress].(*string))).String(),
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
	utxos := response.GetUtxos()
	var inputUtxos []*utxo.UTXO
	for _, u := range utxos {
		utxo := utxo.UTXO{}
		utxo.FromProto(u)
		inputUtxos = append(inputUtxos, &utxo)
	}
	sum := common.NewAmount(0)
	for _, u := range inputUtxos {
		sum = sum.Add(u.Value)
	}
	fmt.Printf("The balance is: %d\n", sum)
}
