package util

import (
	"github.com/dappley/go-dappley/client"
	"github.com/dappley/go-dappley/common"
	"github.com/dappley/go-dappley/core"
	logger "github.com/sirupsen/logrus"
)

func NewTransaction(prevUtxos []*core.UTXO, vouts []core.TXOutput, tip *common.Amount, senderKeyPair *client.KeyPair) *core.Transaction {
	tx := &core.Transaction{
		nil,
		prepareInputLists(prevUtxos, senderKeyPair.PublicKey, nil),
		vouts,
		tip,
		common.NewAmount(0),
		common.NewAmount(0)}
	tx.ID = tx.Hash()

	err := tx.Sign(senderKeyPair.PrivateKey, prevUtxos)
	if err != nil {
		logger.Panic("Sign transaction failed. Terminating...")
	}
	return tx
}

func prepareInputLists(utxos []*core.UTXO, publicKey []byte, signature []byte) []core.TXInput {
	var inputs []core.TXInput

	// Build a list of inputs
	for _, utxo := range utxos {
		input := core.TXInput{utxo.Txid, utxo.TxIndex, signature, publicKey}
		inputs = append(inputs, input)
	}

	return inputs
}

func calculateUtxoSum(utxos []*core.UTXO) *common.Amount {
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

func prepareOutputLists(prevUtxos []*core.UTXO, from, to client.Address, amount *common.Amount, tip *common.Amount) []core.TXOutput {

	sum := calculateUtxoSum(prevUtxos)
	change := calculateChange(sum, amount, tip)

	var outputs []core.TXOutput

	outputs = append(outputs, *core.NewTXOutput(amount, to))
	if !change.IsZero() {
		outputs = append(outputs, *core.NewTXOutput(change, from))
	}
	return outputs
}
