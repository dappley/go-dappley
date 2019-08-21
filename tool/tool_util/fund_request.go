package tool

import (
	"github.com/dappley/go-dappley/common"
	"github.com/dappley/go-dappley/sdk"
	logger "github.com/sirupsen/logrus"
)

type FundRequest struct {
	sdk *sdk.DappSdk
}

//NewDappSdkFundRequest creates a new FundRequest instance
func NewFundRequest(sdk *sdk.DappSdk) *FundRequest {
	return &FundRequest{sdk}
}

//Fund requests a fund from the miner if the requested amount is lower than the current balance
func (fr *FundRequest) Fund(fundAddr string, amount *common.Amount) {
	logger.Info("FundRequest: Requesting fund from miner...")

	if fundAddr == "" {
		logger.Panic("FundRequest: There is no account to receive fund.")
	}

	if _, isSufficient := fr.checkSufficientInitialAmount(fundAddr, amount); isSufficient {
		return
	}

	fr.sdk.RequestFund(fundAddr, amount)
}

//checkSufficientInitialAmount checks if the current balance of the address is sufficient according to minAmount
func (fr *FundRequest) checkSufficientInitialAmount(addr string, minAmount *common.Amount) (uint64, bool) {
	balance, err := fr.sdk.GetBalance(addr)
	if err != nil {
		logger.WithError(err).WithFields(logger.Fields{
			"address": addr,
		}).Error("FundRequest: Failed to get balance.")
		return 0, false
	}
	return uint64(balance), uint64(balance) >= minAmount.Uint64()
}
