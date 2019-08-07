package core

import (
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/dappley/go-dappley/common"
	logger "github.com/sirupsen/logrus"
)

// VerifyInEstimate returns whether the current tx in estimate mode is valid.
func VerifyInEstimate(utxoIndex *UTXOIndex, ctx *ContractTx) error {
	if ctx.IsExecutionContract() && !ctx.IsContractDeployed(utxoIndex) {
		return errors.New("Transaction: contract state check failed")
	}

	_, err := verify(&ctx.Transaction, utxoIndex)
	if err != nil {
		return err
	}
	return nil
}

// VerifyContractTx ensures signature of transactions is correct or verifies against blockHeight if it's a coinbase transactions
func VerifyContractTx(utxoIndex *UTXOIndex, ctx *ContractTx) (bool, error) {
	if ctx.IsExecutionContract() && !ctx.IsContractDeployed(utxoIndex) {
		return false, errors.New("Transaction: contract state check failed")
	}

	totalBalance, err := verify(&ctx.Transaction, utxoIndex)
	if err != nil {
		return false, err
	}
	return ctx.verifyGas(totalBalance)
}

// VerifyTransaction ensures signature of transactions is correct or verifies against blockHeight if it's a coinbase transactions
func VerifyTransaction(utxoIndex *UTXOIndex, tx *Transaction, blockHeight uint64) (bool, error) {
	ctx := tx.ToContractTx()
	if ctx != nil {
		return VerifyContractTx(utxoIndex, ctx)
	}
	if tx.IsCoinbase() {
		//TODO coinbase vout check need add tip
		if tx.Vout[0].Value.Cmp(subsidy) < 0 {
			return false, errors.New("Transaction: subsidy check failed")
		}
		bh := binary.BigEndian.Uint64(tx.Vin[0].Signature)
		if blockHeight != bh {
			return false, fmt.Errorf("Transaction: block height check failed expected=%v actual=%v", blockHeight, bh)
		}
		return true, nil
	}
	if tx.IsRewardTx() || tx.IsGasRewardTx() || tx.IsGasChangeTx() {
		//TODO: verify reward tx here
		return true, nil
	}

	_, err := verify(tx, utxoIndex)
	if err != nil {
		return false, err
	}
	return true, nil
}

func verify(tx *Transaction, utxoIndex *UTXOIndex) (*common.Amount, error) {
	prevUtxos := getPrevUTXOs(tx, utxoIndex)
	if prevUtxos == nil {
		return nil, errors.New("Transaction: prevUtxos not found")
	}
	result, err := tx.verifyID()
	if !result {
		return nil, err
	}

	result, err = tx.verifyPublicKeyHash(prevUtxos)
	if !result {
		return nil, err
	}

	totalPrev := calculateUtxoSum(prevUtxos)
	totalVoutValue, ok := tx.calculateTotalVoutValue()
	if !ok {
		return nil, errors.New("Transaction: vout is invalid")
	}
	result, err = tx.verifyAmount(totalPrev, totalVoutValue)
	if !result {
		return nil, err
	}
	result, err = tx.verifyTip(totalPrev, totalVoutValue)
	if !result {
		logger.WithFields(logger.Fields{
			"tx_id": hex.EncodeToString(tx.ID),
		}).Warn("Transaction: tip is invalid.")
		return nil, err
	}
	result, err = tx.verifySignatures(prevUtxos)
	if !result {
		return nil, err
	}
	totalBalance, _ := totalPrev.Sub(totalVoutValue)
	totalBalance, _ = totalBalance.Sub(tx.Tip)
	return totalBalance, nil
}
