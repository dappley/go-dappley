package util

import (
	"github.com/dappley/go-dappley/core/transaction"
	"github.com/dappley/go-dappley/sdk"
)

type TxSender struct {
	tx      *transaction.Transaction
	dappSdk *sdk.DappSdk
	account *sdk.DappSdkAccount
}

type TestTransaction interface {
	Generate(params transaction.SendTxParam)
	Send()
	Print()
}
