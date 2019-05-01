package sdk

import (
	"context"
	"github.com/dappley/go-dappley/common"
	"github.com/dappley/go-dappley/rpc/pb"
	logger "github.com/sirupsen/logrus"
)

type DappSdkFundRequest struct {
	conn       *DappSdkConn
	blockchain *DappSdkBlockchain
}

func NewDappSdkFundRequest(conn *DappSdkConn, blockchain *DappSdkBlockchain) *DappSdkFundRequest {
	return &DappSdkFundRequest{conn, blockchain}
}

func (sdkfr *DappSdkFundRequest) Fund(fundAddr string, amount *common.Amount) {
	logger.Info("Requesting fund from miner...")

	if fundAddr == "" {
		logger.Panic("There is no wallet to receive fund.")
	}

	if _, isSufficient := sdkfr.checkSufficientInitialAmount(fundAddr, amount); isSufficient {
		return
	}

	sdkfr.requestFund(fundAddr, amount)
}

func (sdkfr *DappSdkFundRequest) requestFund(fundAddr string, amount *common.Amount) {
	sendFromMinerRequest := &rpcpb.SendFromMinerRequest{To: fundAddr, Amount: amount.Bytes()}
	sdkfr.conn.adminClient.RpcSendFromMiner(context.Background(), sendFromMinerRequest)
}

func (sdkfr *DappSdkFundRequest) checkSufficientInitialAmount(addr string, minAmount *common.Amount) (uint64, bool) {
	balance, err := sdkfr.blockchain.GetBalance(addr)
	if err != nil {
		logger.WithError(err).WithFields(logger.Fields{
			"address": addr,
		}).Error("Failed to get balance.")
		return 0, false
	}
	return uint64(balance), uint64(balance) >= minAmount.Uint64()
}
