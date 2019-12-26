package util

import (
	"github.com/dappley/go-dappley/common"
	"github.com/dappley/go-dappley/core/account"
	"github.com/dappley/go-dappley/core/transaction"
	"github.com/dappley/go-dappley/core/transactionbase"
	"github.com/dappley/go-dappley/core/utxo"
	logger "github.com/sirupsen/logrus"
	"time"
)

func NewTransaction(prevUtxos []*utxo.UTXO, vouts []transactionbase.TXOutput, tip *common.Amount, senderKeyPair *account.KeyPair) *transaction.Transaction {
	tx := &transaction.Transaction{
		nil,
		prepareInputLists(prevUtxos, senderKeyPair.GetPublicKey(), nil),
		vouts,
		tip,
		common.NewAmount(0),
		common.NewAmount(0),
		time.Now().UnixNano() / 1e6, transaction.TxTypeDefault}
	tx.ID = tx.Hash()

	err := transaction.NewTxDecorator(tx).Sign(senderKeyPair.GetPrivateKey(), prevUtxos)
	if err != nil {
		logger.Panic("Sign transaction failed. Terminating...")
	}
	return tx
}

func prepareInputLists(utxos []*utxo.UTXO, publicKey []byte, signature []byte) []transactionbase.TXInput {
	var inputs []transactionbase.TXInput

	// Build a list of inputs
	for _, utxo := range utxos {
		input := transactionbase.TXInput{utxo.Txid, utxo.TxIndex, signature, publicKey}
		inputs = append(inputs, input)
	}

	return inputs
}

func calculateUtxoSum(utxos []*utxo.UTXO) *common.Amount {
	sum := common.NewAmount(0)
	for _, utxo := range utxos {
		sum = sum.Add(utxo.Value)
	}
	return sum
}

func calculateChange(input, amount, tip *common.Amount) *common.Amount {
	change, err := input.Sub(amount)
	if err != nil {
		logger.Panic("Insufficient input")
	}

	change, err = change.Sub(tip)
	if err != nil {
		logger.Panic("Insufficient input")
	}
	return change
}

func prepareOutputLists(prevUtxos []*utxo.UTXO, from, to *account.TransactionAccount, amount *common.Amount, tip *common.Amount) []transactionbase.TXOutput {
	sum := calculateUtxoSum(prevUtxos)
	change := calculateChange(sum, amount, tip)

	var outputs []transactionbase.TXOutput

	outputs = append(outputs, *transactionbase.NewTXOutput(amount, to))
	if !change.IsZero() {
		outputs = append(outputs, *transactionbase.NewTXOutput(change, from))
	}
	return outputs
}
