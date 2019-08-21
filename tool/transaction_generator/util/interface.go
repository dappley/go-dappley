package util

import (
	"github.com/dappley/go-dappley/core"
	"github.com/dappley/go-dappley/sdk"
)

type TxSender struct {
	tx      *core.Transaction
	dappSdk *sdk.DappSdk
	account *sdk.DappSdkAccount
}

type TestTransaction interface {
	Generate(params core.SendTxParam)
	Send()
	Print()
}
