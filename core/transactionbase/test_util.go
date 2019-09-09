package transactionbase

import (
	"github.com/dappley/go-dappley/common"
	"github.com/dappley/go-dappley/core/account"
	"github.com/dappley/go-dappley/util"
)

func MockTxInputs() []TXInput {
	return []TXInput{
		{util.GenerateRandomAoB(2),
			6,
			util.GenerateRandomAoB(2),
			[]byte("12345678901234567890123456789013")},
		{util.GenerateRandomAoB(2),
			2,
			util.GenerateRandomAoB(2),
			[]byte("12345678901234567890123456789014")},
	}
}

func MockTxOutputs() []TXOutput {
	return []TXOutput{
		{common.NewAmount(5), account.PubKeyHash(util.GenerateRandomAoB(2)), ""},
		{common.NewAmount(7), account.PubKeyHash(util.GenerateRandomAoB(2)), ""},
	}
}

func getAoB(length int64) []byte {
	return util.GenerateRandomAoB(length)
}

func GenerateFakeTxInputs() []TXInput {
	return []TXInput{
		{getAoB(2), 10, getAoB(2), getAoB(2)},
		{getAoB(2), 5, getAoB(2), getAoB(2)},
	}
}

func GenerateFakeTxOutputs() []TXOutput {
	return []TXOutput{
		{common.NewAmount(1), account.PubKeyHash(getAoB(2)), ""},
		{common.NewAmount(2), account.PubKeyHash(getAoB(2)), ""},
	}
}
