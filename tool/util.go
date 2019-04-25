package tool

import (
	"context"
	"github.com/dappley/go-dappley/common"
	"github.com/dappley/go-dappley/rpc/pb"
	logger "github.com/sirupsen/logrus"
	"time"
)

const (
	fundTimeout = time.Duration(time.Minute * 5)
)

func FundFromMiner(adminClient rpcpb.AdminServiceClient, rpcClient rpcpb.RpcServiceClient, fundAddr string, amount *common.Amount) {
	logger.Info("Requesting fund from miner...")

	if fundAddr == "" {
		logger.Panic("There is no wallet to receive fund.")
	}

	requestFundFromMiner(adminClient, fundAddr, amount)
	bal, isSufficient := checkSufficientInitialAmount(rpcClient, fundAddr, amount)
	if isSufficient {
		//continue if the initial amount is sufficient
		return
	}
	logger.WithFields(logger.Fields{
		"address":     fundAddr,
		"balance":     bal,
		"target_fund": amount.String(),
	}).Info("Current wallet balance is insufficient. Waiting for more funds...")
	waitTilInitialAmountIsSufficient(adminClient, rpcClient, fundAddr, amount)
}

func checkSufficientInitialAmount(rpcClient rpcpb.RpcServiceClient, addr string, amount *common.Amount) (uint64, bool) {
	balance, err := GetBalance(rpcClient, addr)
	if err != nil {
		logger.WithError(err).WithFields(logger.Fields{
			"address": addr,
		}).Panic("Failed to get balance.")
	}
	return uint64(balance), uint64(balance) >= amount.Uint64()
}

func waitTilInitialAmountIsSufficient(adminClient rpcpb.AdminServiceClient, rpcClient rpcpb.RpcServiceClient, addr string, amount *common.Amount) {
	checkBalanceTicker := time.NewTicker(time.Second * 5).C
	timeout := time.NewTicker(fundTimeout).C
	for {
		select {
		case <-checkBalanceTicker:
			bal, isSufficient := checkSufficientInitialAmount(rpcClient, addr, amount)
			if isSufficient {
				//continue if the initial amount is sufficient
				return
			}
			logger.WithFields(logger.Fields{
				"address":     addr,
				"balance":     bal,
				"target_fund": amount,
			}).Info("Current wallet balance is insufficient. Waiting for more funds...")
			requestFundFromMiner(adminClient, addr, amount)
		case <-timeout:
			logger.WithFields(logger.Fields{
				"target_fund": amount,
			}).Panic("Timed out while waiting for sufficient fund from miner!")
		}
	}
}

func requestFundFromMiner(adminClient rpcpb.AdminServiceClient, fundAddr string, amount *common.Amount) {

	sendFromMinerRequest := &rpcpb.SendFromMinerRequest{To: fundAddr, Amount: amount.Bytes()}
	adminClient.RpcSendFromMiner(context.Background(), sendFromMinerRequest)
}

func GetBalance(rpcClient rpcpb.RpcServiceClient, address string) (int64, error) {
	response, err := rpcClient.RpcGetBalance(context.Background(), &rpcpb.GetBalanceRequest{Address: address})
	return response.Amount, err
}
