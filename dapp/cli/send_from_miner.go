package main

import (
	"context"
	"fmt"

	"github.com/dappley/go-dappley/common"
	"github.com/dappley/go-dappley/core/account"
	rpcpb "github.com/dappley/go-dappley/rpc/pb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func sendFromMinerCommandHandler(ctx context.Context, c interface{}, flags cmdFlags) {
	toAddr := *(flags[flagAddressBalance].(*string))
	if len(toAddr) == 0 {
		fmt.Println("\n Example: cli sendFromMiner -to 1MeSBgufmzwpiJNLemUe1emxAussBnz7a7 -amount 15")
		fmt.Println()
		return
	}
	amount := int64(*(flags[flagAmountBalance].(*int)))
	if amount <= 0 {
		fmt.Println("Error: amount must be greater than zero!")
		return
	}

	addressAccount := account.NewTransactionAccountByAddress(account.NewAddress(toAddr))
	if !addressAccount.IsValid() {
		fmt.Println("Error: address is invalid!")
		return
	}

	amountBytes := common.NewAmount(uint64(*(flags[flagAmountBalance].(*int)))).Bytes()
	sendFromMinerRequest := rpcpb.SendFromMinerRequest{To: toAddr, Amount: amountBytes}

	_, err := c.(rpcpb.AdminServiceClient).RpcSendFromMiner(ctx, &sendFromMinerRequest)
	if err != nil {
		switch status.Code(err) {
		case codes.Unavailable:
			fmt.Println("Error: server is not reachable!")
		default:
			fmt.Println("Error: ", err.Error())
		}
		return
	}
	fmt.Println("Requested amount is sent. Pending approval from network.")
}
