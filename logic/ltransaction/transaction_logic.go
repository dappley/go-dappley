package ltransaction

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/dappley/go-dappley/core/block"
	"github.com/dappley/go-dappley/core/scState"

	"github.com/dappley/go-dappley/common"
	"github.com/dappley/go-dappley/core/account"
	"github.com/dappley/go-dappley/core/transaction"
	"github.com/dappley/go-dappley/core/transactionbase"
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
	ErrLoadError             = errors.New("contract load error")
)

// VerifyTransaction ensures signature of transactions is correct or verifies against blockHeight if it's a coinbase transactions
func VerifyTransaction(utxoIndex *lutxo.UTXOIndex, tx *transaction.Transaction, blockHeight uint64) error {
	txDecorator := NewTxDecorator(tx)
	if txDecorator != nil {
		return txDecorator.Verify(utxoIndex, blockHeight)
	}
	return nil
}

// VerifyContractTransaction ensures the generated transactions from smart contract are the same with those in block
func VerifyContractTransaction(utxoIndex *lutxo.UTXOIndex, tx *TxContract, scState *scState.ScState, scEngine ScEngine, currBlkHeight uint64, parentBlk *block.Block, rewards map[string]string) (gasCount uint64, generatedTxs []*transaction.Transaction, err error) {
	// Run the contract and collect generated transactions
	if scEngine == nil {
		return 0, nil, errors.New("VerifyContractTransaction: is missing SCEngineManager when verifying transactions.")
	}

	prevUtxos, err := lutxo.FindVinUtxosInUtxoPool(utxoIndex, tx.Transaction)
	if err != nil {
		logger.WithError(err).WithFields(logger.Fields{
			"txid": hex.EncodeToString(tx.ID),
		}).Warn("VerifyContractTransaction: cannot find vin while executing smart contract")
		return 0, nil, err
	}

	isContractDeployed := tx.IsContractDeployed(utxoIndex)
	utxoIndex.UpdateUtxo(tx.Transaction)

	if err := scEngine.SetExecutionLimits(1000, 0); err != nil {
		return 0, nil, err
	}
	gasCount, generatedTxs, err = tx.Execute(prevUtxos, isContractDeployed, utxoIndex, scState, rewards, scEngine, currBlkHeight, parentBlk)
	return gasCount, generatedTxs, nil
}

// DescribeTransaction reverse-engineers the high-level description of a transaction
func DescribeTransaction(utxoIndex *lutxo.UTXOIndex, tx *transaction.Transaction) (sender, recipient *account.Address, amount, tip *common.Amount, error error) {
	var receiverAddress account.Address
	vinPubKey := tx.Vin[0].PubKey
	ta := account.NewContractTransactionAccount()
	inputAmount := common.NewAmount(0)
	outputAmount := common.NewAmount(0)
	payoutAmount := common.NewAmount(0)
	adaptedTx := transaction.NewTxAdapter(tx)
	for _, vin := range tx.Vin {
		if bytes.Compare(vin.PubKey, vinPubKey) == 0 {
			switch {
			case adaptedTx.IsRewardTx():
				ta = account.NewTransactionAccountByPubKey(transaction.RewardTxData)
				continue
			case adaptedTx.IsContractSend():
				// vinPubKey is pubKeyHash of contract address if it is a sc generated tx
				ta = account.NewContractAccountByPubKeyHash(vinPubKey)
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

// Returns DAG structure of batch transaction
func SplitToTxDags(txs []*transaction.Transaction) [][]*transaction.Transaction {
	// key: prevTxId, value: currentTxId
	relationMap := make(map[string][]string)
	txIdMap := make(map[string]*transaction.Transaction)

	// build data
	for _, tx := range txs {
		txId := hex.EncodeToString(tx.ID)
		vins := tx.Vin
		for _, vin := range vins {
			refTxId := hex.EncodeToString(vin.Txid)
			addToRelationMap(relationMap, refTxId, txId)
		}
		txIdMap[txId] = tx
	}

	// resolve relation
	var dags [][]*transaction.Transaction
	for _, tx := range txs {
		txId := hex.EncodeToString(tx.ID)
		var oneDag []*transaction.Transaction
		if _, existed := txIdMap[txId]; existed {
			oneDag = append(oneDag, tx)
			delete(txIdMap, txId)
			if childIds, exists := relationMap[hex.EncodeToString(tx.ID)]; exists {
				oneDag = childTxDagRecursion(oneDag, childIds, relationMap, txIdMap)
			}
		}
		if len(oneDag) > 0 {
			dags = append(dags, oneDag)
		}
	}
	return dags
}

// Parsing relation between two transactions
func addToRelationMap(m map[string][]string, refTxId string, txId string) {
	ids, exists := m[refTxId]
	if exists {
		for _, item := range ids {
			if txId == item {
				return
			}
		}
	}
	if !exists {
		ids = []string{}
	}
	ids = append(ids, txId)
	m[refTxId] = ids
}

// Recursion to add tx to dag for child transactions depend on parent transaction
func childTxDagRecursion(oneDag []*transaction.Transaction, childIds []string, relationMap map[string][]string,
	txIdMap map[string]*transaction.Transaction) []*transaction.Transaction {
	if len(childIds) == 0 {
		return oneDag
	}
	for _, childId := range childIds {
		if tx, existed := txIdMap[childId]; existed {
			oneDag = append(oneDag, tx)
			delete(txIdMap, childId)
		}
		// use recursion to handle deep dependency
		if nextIds, existed := relationMap[childId]; existed {
			oneDag = childTxDagRecursion(oneDag, nextIds, relationMap, txIdMap)
		}
	}
	return oneDag
}
