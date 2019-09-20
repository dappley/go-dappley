package transaction

import (
	"github.com/dappley/go-dappley/common"
	"github.com/dappley/go-dappley/core/transactionbase"
	"github.com/dappley/go-dappley/util"
)

func MockTransaction() *Transaction {
	return &Transaction{
		ID:       util.GenerateRandomAoB(1),
		Vin:      transactionbase.MockTxInputs(),
		Vout:     transactionbase.MockTxOutputs(),
		Tip:      common.NewAmount(5),
		GasLimit: common.NewAmount(0),
		GasPrice: common.NewAmount(0),
	}
}
