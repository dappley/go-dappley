package sdk

import (
	"github.com/dappley/go-dappley/common"
	logger "github.com/sirupsen/logrus"
)

type DappSdkFundRequest struct {
	conn *DappSdkGrpcClient
	sdk  *DappSdk
}

//NewDappSdkFundRequest creates a new DappSdkFundRequest instance
func NewDappSdkFundRequest(conn *DappSdkGrpcClient, blockchain *DappSdk) *DappSdkFundRequest {
	return &DappSdkFundRequest{conn, blockchain}
}

//Fund requests a fund from the miner if the requested amount is lower than the current balance
func (sdkfr *DappSdkFundRequest) Fund(fundAddr string, amount *common.Amount) {
	logger.Info("DappSdkFundRequest: Requesting fund from miner...")

	if fundAddr == "" {
		logger.Panic("DappSdkFundRequest: There is no wallet to receive fund.")
	}

	if _, isSufficient := sdkfr.checkSufficientInitialAmount(fundAddr, amount); isSufficient {
		return
	}

	sdkfr.sdk.RequestFund(fundAddr, amount)
}

//checkSufficientInitialAmount checks if the current balance of the address is sufficient according to minAmount
func (sdkfr *DappSdkFundRequest) checkSufficientInitialAmount(addr string, minAmount *common.Amount) (uint64, bool) {
	balance, err := sdkfr.sdk.GetBalance(addr)
	if err != nil {
		logger.WithError(err).WithFields(logger.Fields{
			"address": addr,
		}).Error("DappSdkFundRequest: Failed to get balance.")
		return 0, false
	}
	return uint64(balance), uint64(balance) >= minAmount.Uint64()
}
