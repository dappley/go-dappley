package main

import (
	"context"
	"fmt"
	"io/ioutil"

	logger "github.com/sirupsen/logrus"
	"google.golang.org/grpc"

	"github.com/dappley/go-dappley/common"
	"github.com/dappley/go-dappley/wallet"
	rpcpb "github.com/dappley/go-dappley/rpc/pb"
)

func main() {
	conn, _ := grpc.Dial(":50051", grpc.WithInsecure())
	rpc := rpcpb.NewRpcServiceClient(conn)

	c, err := rpc.RpcSubscribe(context.Background(), &rpcpb.SubscribeRequest{Topics: []string{"topic1", "topic2", "topic3"}})
	if err != nil {
		logger.Panic(err)
	}
	admin := rpcpb.NewAdminServiceClient(conn)
	raw, err := ioutil.ReadFile("test_event.js")
	if err != nil {
		logger.Panic(err)
	}

	resp, err := admin.RpcSend(context.Background(), &rpcpb.SendRequest{
		From:        "dHvB2CF9PUtih7VM1VUZmf3g25ZGfNym5A",
		To:          "",
		Amount:      common.NewAmount(1).Bytes(),
		Tip:         common.NewAmount(0).Bytes(),
		AccountPath: wallet.GetAccountFilePath(),
		Data:        string(raw),
	})
	if err != nil {
		logger.Panic(err)
	}
	fmt.Println("Contract addr:", resp.ContractAddress)
	contractAddr := resp.ContractAddress
	//contractAddr := "cTuDnSBeqDuqwfQiRrS2UrCRywEfKQJeGs"
	count := 0
	for {
		logger.Info("Sending event triggering...")
		_, err := admin.RpcSend(context.Background(), &rpcpb.SendRequest{
			From:        "dHvB2CF9PUtih7VM1VUZmf3g25ZGfNym5A",
			To:          contractAddr,
			Amount:      common.NewAmount(1).Bytes(),
			Tip:         common.NewAmount(0).Bytes(),
			AccountPath: wallet.GetAccountFilePath(),
			Data:        fmt.Sprintf("{\"function\":\"trigger\",\"args\":[\"topic%d\",\"data%d\"]}", count%3+1, count),
		})
		if err != nil {
			logger.Panic(err)
		}
		logger.Info("Waiting for event triggering event triggering...")
		resp, err := c.Recv()
		if err != nil {
			logger.Panic(err)
		}
		logger.WithFields(logger.Fields{
			"data": resp.Data,
		}).Info("Received data!")
		count += 1
	}
}
