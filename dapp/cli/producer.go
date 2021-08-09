package main

import (
	"context"
	"fmt"

	"github.com/dappley/go-dappley/core/account"
	rpcpb "github.com/dappley/go-dappley/rpc/pb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func cliaddProducerCommandHandler(ctx context.Context, c interface{}, flags cmdFlags) {
	producerAddress := *(flags[flagProducerAddr].(*string))
	height := *(flags[flagBlockHeight].(*uint64))
	if len(producerAddress) == 0 {
		fmt.Println("\n Please enter an address. Example: cli addProducer -address 1MeSBgufmzwpiJNLemUe1emxAussBnz7a7 -height 100")
		fmt.Println()
		return
	}
	addressAccount := account.NewTransactionAccountByAddress(account.NewAddress(producerAddress))

	if !addressAccount.IsValid() {
		fmt.Println("Error: address is invalid")
		return
	}

	_, err := c.(rpcpb.AdminServiceClient).RpcChangeProducer(ctx, &rpcpb.ChangeProducerRequest{
		Addresses: producerAddress,
		Height:    height,
		Kind:      uint64(2),
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
	fmt.Println("Producer will be added.")
}

func clichangeProducerCommandHandler(ctx context.Context, c interface{}, flags cmdFlags) {
	producerAddress := *(flags[flagProducerAddr].(*string))
	height := *(flags[flagBlockHeight].(*uint64))
	if len(producerAddress) == 0 {
		fmt.Println("\n Please enter an address. Example: cli changeProducer -address 1MeSBgufmzwpiJNLemUe1emxAussBnz7a7 -height 100")
		fmt.Println()
		return
	}
	addressAccount := account.NewTransactionAccountByAddress(account.NewAddress(producerAddress))

	if !addressAccount.IsValid() {
		fmt.Println("Error: address is invalid")
		return
	}

	_, err := c.(rpcpb.AdminServiceClient).RpcChangeProducer(ctx, &rpcpb.ChangeProducerRequest{
		Addresses: producerAddress,
		Height:    height,
		Kind:      uint64(1),
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
	fmt.Println("Producer will be changed.")
}

func clideleteProducerCommandHandler(ctx context.Context, c interface{}, flags cmdFlags) {

	height := *(flags[flagBlockHeight].(*uint64))
	if height == 0 {
		fmt.Println("\n Please enter an address. Example: cli deleteProducer -height 100")
		fmt.Println()
		return
	}
	_, err := c.(rpcpb.AdminServiceClient).RpcChangeProducer(ctx, &rpcpb.ChangeProducerRequest{
		Addresses: "",
		Height:    height,
		Kind:      uint64(3),
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
