package ltransaction

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/dappley/go-dappley/common"
	"github.com/dappley/go-dappley/core/account"
	"github.com/dappley/go-dappley/core/block"
	"github.com/dappley/go-dappley/core/scState"
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

// VerifyInEstimate returns whether the current tx in estimate mode is valid.
func VerifyInEstimate(utxoIndex *lutxo.UTXOIndex, ctx *transaction.ContractTx) error {
	utxos := getPrevUTXOs(&ctx.Transaction, utxoIndex)
	if ctx.IsScheduleContract() && !IsContractDeployed(utxoIndex, ctx) {
		return errors.New("Transaction: contract state check failed")
	}

	err := ctx.Verify(utxos)

	return err
}

// VerifyContractTx ensures signature of transactions is correct or verifies against blockHeight if it's a coinbase transactions
func verifyContractTx(utxoIndex *lutxo.UTXOIndex, ctx *transaction.ContractTx) error {
	utxos := getPrevUTXOs(&ctx.Transaction, utxoIndex)
	err := VerifyInEstimate(utxoIndex, ctx)
	if err != nil {
		return err
	}
	totalBalance := ctx.GetTotalBalance(utxos)
	return ctx.VerifyGas(totalBalance)
}

// VerifyTransaction ensures signature of transactions is correct or verifies against blockHeight if it's a coinbase transactions
func VerifyTransaction(utxoIndex *lutxo.UTXOIndex, tx *transaction.Transaction, blockHeight uint64) error {
	ctx := tx.ToContractTx()
	if ctx != nil {
		return verifyContractTx(utxoIndex, ctx)
	}
	if tx.IsCoinbase() {
		//TODO coinbase vout check need add tip
		if tx.Vout[0].Value.Cmp(transaction.Subsidy) < 0 {
			return errors.New("Transaction: subsidy check failed")
		}
		bh := binary.BigEndian.Uint64(tx.Vin[0].Signature)
		if blockHeight != bh {
			return fmt.Errorf("Transaction: block height check failed expected=%v actual=%v", blockHeight, bh)
		}
		return nil
	}
	if tx.IsRewardTx() || tx.IsGasRewardTx() || tx.IsGasChangeTx() {
		//TODO: verify reward tx here
		return nil
	}
	utxos := getPrevUTXOs(tx, utxoIndex)
	err := tx.Verify(utxos)

	return err
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
			inputAmount = inputAmount.Add(usedUTXO.Value)
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

// Returns related previous UTXO for current transaction
func getPrevUTXOs(tx *transaction.Transaction, utxoIndex *lutxo.UTXOIndex) []*utxo.UTXO {
	var prevUtxos []*utxo.UTXO
	tempUtxoTxMap := make(map[string]*utxo.UTXOTx)
	for _, vin := range tx.Vin {
		if ok, _ := account.IsValidPubKey(vin.PubKey); !ok {
			logger.WithFields(logger.Fields{
				"tx_id":          hex.EncodeToString(tx.ID),
				"vin_tx_id":      hex.EncodeToString(vin.Txid),
				"vin_public_key": hex.EncodeToString(vin.PubKey),
			}).Warn("Transaction: failed to get PubKeyHash of vin.")
			return nil
		}

		ta := account.NewTransactionAccountByPubKey(vin.PubKey)
		tempUtxoTx, ok := tempUtxoTxMap[string(ta.GetPubKeyHash())]
		if !ok {
			tempUtxoTx = utxoIndex.GetAllUTXOsByPubKeyHash(ta.GetPubKeyHash())
			tempUtxoTxMap[string(ta.GetPubKeyHash())] = tempUtxoTx
		}
		utxo := tempUtxoTx.GetUtxo(vin.Txid, vin.Vout)
		if utxo == nil {
			logger.WithFields(logger.Fields{
				"tx_id":      hex.EncodeToString(tx.ID),
				"vin_tx_id":  hex.EncodeToString(vin.Txid),
				"vin_index":  vin.Vout,
				"pubKeyHash": ta.GetPubKeyHash().String(),
			}).Warn("Transaction: cannot find vin.")
			return nil
		}
		prevUtxos = append(prevUtxos, utxo)
	}
	return prevUtxos
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

// IsContractDeployed returns if the current contract is deployed
func IsContractDeployed(utxoIndex *lutxo.UTXOIndex, ctx *transaction.ContractTx) bool {
	pubkeyhash := ctx.GetContractPubKeyHash()
	if pubkeyhash == nil {
		return false
	}

	contractUtxoTx := utxoIndex.GetAllUTXOsByPubKeyHash(pubkeyhash)
	return contractUtxoTx.Size() > 0
}

//Execute executes the smart contract the transaction points to. it doesnt do anything if is a contract deploy transaction
func Execute(ctx *transaction.ContractTx, prevUtxos []*utxo.UTXO,
	isContractDeployed bool,
	index lutxo.UTXOIndex,
	scStorage *scState.ScState,
	rewards map[string]string,
	engine ScEngine,
	currblkHeight uint64,
	parentBlk *block.Block) (uint64, []*transaction.Transaction, error) {

	if engine == nil {
		return 0, nil, nil
	}
	if !isContractDeployed {
		return 0, nil, nil
	}

	vout := ctx.Vout[transaction.ContractTxouputIndex]

	function, args := util.DecodeScInput(vout.Contract)
	if function == "" {
		return 0, nil, ErrUnsupportedSourceType
	}
	if err := engine.SetExecutionLimits(ctx.GasLimit.Uint64(), 0); err != nil {
		return 0, nil, ErrInvalidGasLimit
	}

	totalArgs := util.PrepareArgs(args)
	address := vout.GetAddress()
	logger.WithFields(logger.Fields{
		"contract_address": address.String(),
		"invoked_function": function,
		"arguments":        totalArgs,
	}).Debug("Transaction: is executing the smart contract...")

	createContractUtxo, invokeUtxos := index.SplitContractUtxo([]byte(vout.PubKeyHash))

	engine.ImportSourceCode(createContractUtxo.Contract)
	engine.ImportLocalStorage(scStorage)
	engine.ImportContractAddr(address)
	engine.ImportUTXOs(invokeUtxos)
	engine.ImportSourceTXID(ctx.ID)
	engine.ImportRewardStorage(rewards)
	engine.ImportTransaction(&ctx.Transaction)
	engine.ImportContractCreateUTXO(createContractUtxo)
	engine.ImportPrevUtxos(prevUtxos)
	engine.ImportCurrBlockHeight(currblkHeight)
	engine.ImportSeed(parentBlk.GetTimestamp())
	_, err := engine.Execute(function, totalArgs)
	gasCount := engine.ExecutionInstructions()
	// record base gas
	baseGas, _ := ctx.GasCountOfTxBase()
	gasCount += baseGas.Uint64()
	if err != nil {
		return gasCount, nil, err
	}
	return gasCount, engine.GetGeneratedTXs(), err
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
