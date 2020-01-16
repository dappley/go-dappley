package ltransaction

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/dappley/go-dappley/common"
	"github.com/dappley/go-dappley/core/account"
	"github.com/dappley/go-dappley/core/transaction"
	"github.com/dappley/go-dappley/core/transactionbase"
	"github.com/dappley/go-dappley/core/utxo"
	"github.com/dappley/go-dappley/logic/lutxo"
	"github.com/dappley/go-dappley/util"
	logger "github.com/sirupsen/logrus"
)

var (
	ErrInvalidGasPrice = errors.New("invalid gas price, should be in (0, 10^12]")
	ErrInvalidGasLimit = errors.New("invalid gas limit, should be in (0, 5*10^10]")

	// vm error
	ErrExecutionFailed       = errors.New("execution failed")
	ErrUnsupportedSourceType = errors.New("unsupported source type")
)

// VerifyTransaction ensures signature of transactions is correct or verifies against blockHeight if it's a coinbase transactions
func VerifyTransaction(utxoIndex *lutxo.UTXOIndex, tx *transaction.Transaction, blockHeight uint64) error {
	txDecorator := NewTxDecorator(tx)
	if txDecorator != nil && txDecorator.IsNeedVerify() {
		return txDecorator.Verify(utxoIndex, blockHeight)
	}
	return nil
}

// DescribeTransaction reverse-engineers the high-level description of a transaction
func DescribeTransaction(utxoIndex *lutxo.UTXOIndex, tx *transaction.Transaction) (sender, recipient *account.Address, amount, tip *common.Amount, error error) {
	var receiverAddress account.Address
	vinPubKey := tx.Vin[0].PubKey
	ta := account.NewContractTransactionAccount()
	inputAmount := common.NewAmount(0)
	outputAmount := common.NewAmount(0)
	payoutAmount := common.NewAmount(0)
	for _, vin := range tx.Vin {
		if bytes.Compare(vin.PubKey, vinPubKey) == 0 {
			switch {
			case tx.IsRewardTx():
				ta = account.NewTransactionAccountByPubKey(transaction.RewardTxData)
				continue
			case IsFromContract(utxoIndex, tx):
				// vinPubKey is the ta if it is a sc generated tx
				ta = account.NewTransactionAccountByPubKey(vinPubKey)
			default:
				if ok, err := account.IsValidPubKey(vin.PubKey); !ok {
					logger.WithError(err).Warn("DPoS: cannot compute the public key hash!")
					return nil, nil, nil, nil, err
				}

				ta = account.NewTransactionAccountByPubKey(vin.PubKey)

			}
			usedUTXO := utxoIndex.FindUTXOByVin([]byte(ta.GetPubKeyHash()), vin.Txid, vin.Vout)
			if usedUTXO != nil {
				inputAmount = inputAmount.Add(usedUTXO.Value)
			}
		} else {
			logger.Debug("Transaction: using UTXO from multiple accounts.")
		}
	}
	for _, vout := range tx.Vout {
		if bytes.Compare([]byte(vout.PubKeyHash), vinPubKey) == 0 {
			outputAmount = outputAmount.Add(vout.Value)
		} else {
			receiverAddress = vout.GetAddress()
			payoutAmount = payoutAmount.Add(vout.Value)
		}
	}
	tip, err := inputAmount.Sub(outputAmount)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	senderAddress := ta.GetAddress()

	return &senderAddress, &receiverAddress, payoutAmount, tip, nil
}

// IsFromContract returns true if tx is generated from a contract execution; false otherwise
func IsFromContract(utxoIndex *lutxo.UTXOIndex, tx *transaction.Transaction) bool {
	if len(tx.Vin) == 0 {
		return false
	}

	contractUtxos := utxoIndex.GetContractUtxos()

	for _, vin := range tx.Vin {
		pubKey := account.PubKeyHash(vin.PubKey)
		if isContract, _ := pubKey.IsContract(); !isContract {
			return false
		}

		if !isPubkeyInUtxos(contractUtxos, pubKey) {
			return false
		}
	}
	return true
}

func CheckContractSyntaxTransaction(engine ScEngine, tx *transaction.Transaction) error {
	TxOuts := tx.Vout
	for _, v := range TxOuts {
		err := CheckContractSyntax(engine, v)
		if err != nil {
			return err
		}
	}
	return nil
}

func CheckContractSyntax(sc ScEngine, out transactionbase.TXOutput) error {
	if out.Contract != "" {
		function, args := util.DecodeScInput(out.Contract)
		if function == "" {
			return sc.CheckContactSyntax(out.Contract)
		}
		totalArgs := util.PrepareArgs(args)
		functionCallScript := prepareFuncCallScript(function, totalArgs)
		return sc.CheckContactSyntax(functionCallScript)
	}
	return nil
}

func prepareFuncCallScript(function, args string) string {
	return fmt.Sprintf(
		`var instance = new _native_require();instance["%s"].apply(instance, [%s]);`,
		function,
		args,
	)
}

func isPubkeyInUtxos(contractUtxos []*utxo.UTXO, pubKey account.PubKeyHash) bool {
	for _, contractUtxo := range contractUtxos {
		if bytes.Compare(contractUtxo.PubKeyHash, pubKey) == 0 {
			return true
		}
	}
	return false
}
