package transaction_logic

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/dappley/go-dappley/common"
	"github.com/dappley/go-dappley/core/account"
	"github.com/dappley/go-dappley/core/block"
	"github.com/dappley/go-dappley/core/scState"
	"github.com/dappley/go-dappley/core/transaction"
	"github.com/dappley/go-dappley/core/transaction_base"
	"github.com/dappley/go-dappley/core/utxo"
	"github.com/dappley/go-dappley/crypto/keystore/secp256k1"
	"github.com/dappley/go-dappley/logic/utxo_logic"
	"github.com/dappley/go-dappley/util"
	logger "github.com/sirupsen/logrus"
)

const SCDestroyAddress = "dRxukNqeADQrAvnHD52BVNdGg6Bgmyuaw4"

var (
	ErrInvalidGasPrice = errors.New("invalid gas price, should be in (0, 10^12]")
	ErrInvalidGasLimit = errors.New("invalid gas limit, should be in (0, 5*10^10]")

	// vm error
	ErrExecutionFailed       = errors.New("execution failed")
	ErrUnsupportedSourceType = errors.New("unsupported source type")
)

// VerifyInEstimate returns whether the current tx in estimate mode is valid.
func VerifyInEstimate(utxoIndex *utxo_logic.UTXOIndex, ctx *transaction.ContractTx) error {
	if ctx.IsExecutionContract() && !IsContractDeployed(utxoIndex, ctx) {
		return errors.New("Transaction: contract state check failed")
	}

	_, err := verify(&ctx.Transaction, utxoIndex)
	if err != nil {
		return err
	}
	return nil
}

// VerifyContractTx ensures signature of transactions is correct or verifies against blockHeight if it's a coinbase transactions
func VerifyContractTx(utxoIndex *utxo_logic.UTXOIndex, ctx *transaction.ContractTx) (bool, error) {
	if ctx.IsExecutionContract() && !IsContractDeployed(utxoIndex, ctx) {
		return false, errors.New("Transaction: contract state check failed")
	}

	totalBalance, err := verify(&ctx.Transaction, utxoIndex)
	if err != nil {
		return false, err
	}
	return ctx.VerifyGas(totalBalance)
}

// VerifyTransaction ensures signature of transactions is correct or verifies against blockHeight if it's a coinbase transactions
func VerifyTransaction(utxoIndex *utxo_logic.UTXOIndex, tx *transaction.Transaction, blockHeight uint64) (bool, error) {
	ctx := tx.ToContractTx()
	if ctx != nil {
		return VerifyContractTx(utxoIndex, ctx)
	}
	if tx.IsCoinbase() {
		//TODO coinbase vout check need add tip
		if tx.Vout[0].Value.Cmp(transaction.Subsidy) < 0 {
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

func verify(tx *transaction.Transaction, utxoIndex *utxo_logic.UTXOIndex) (*common.Amount, error) {
	prevUtxos := getPrevUTXOs(tx, utxoIndex)
	if prevUtxos == nil {
		return nil, errors.New("Transaction: prevUtxos not found")
	}
	result, err := tx.VerifyID()
	if !result {
		return nil, err
	}

	result, err = verifyPublicKeyHash(prevUtxos, tx)
	if !result {
		return nil, err
	}

	totalPrev := calculateUtxoSum(prevUtxos)
	totalVoutValue, ok := tx.CalculateTotalVoutValue()
	if !ok {
		return nil, errors.New("Transaction: vout is invalid")
	}
	result, err = tx.VerifyAmount(totalPrev, totalVoutValue)
	if !result {
		return nil, err
	}
	result, err = tx.VerifyTip(totalPrev, totalVoutValue)
	if !result {
		logger.WithFields(logger.Fields{
			"tx_id": hex.EncodeToString(tx.ID),
		}).Warn("Transaction: tip is invalid.")
		return nil, err
	}
	result, err = verifySignatures(prevUtxos, tx)
	if !result {
		return nil, err
	}
	totalBalance, _ := totalPrev.Sub(totalVoutValue)
	totalBalance, _ = totalBalance.Sub(tx.Tip)
	return totalBalance, nil
}

// DescribeTransaction reverse-engineers the high-level description of a transaction
func DescribeTransaction(utxoIndex *utxo_logic.UTXOIndex, tx *transaction.Transaction) (sender, recipient *account.Address, amount, tip *common.Amount, error error) {
	var receiverAddress account.Address
	vinPubKey := tx.Vin[0].PubKey
	pubKeyHash := account.PubKeyHash([]byte(""))
	inputAmount := common.NewAmount(0)
	outputAmount := common.NewAmount(0)
	payoutAmount := common.NewAmount(0)
	for _, vin := range tx.Vin {
		if bytes.Compare(vin.PubKey, vinPubKey) == 0 {
			switch {
			case tx.IsRewardTx():
				pubKeyHash = account.PubKeyHash(transaction.RewardTxData)
				continue
			case IsFromContract(utxoIndex, tx):
				// vinPubKey is the pubKeyHash if it is a sc generated tx
				pubKeyHash = account.PubKeyHash(vinPubKey)
			default:
				pkh, err := account.NewUserPubKeyHash(vin.PubKey)
				if err != nil {
					return nil, nil, nil, nil, err
				}
				pubKeyHash = pkh
			}
			usedUTXO := utxoIndex.FindUTXOByVin([]byte(pubKeyHash), vin.Txid, vin.Vout)
			inputAmount = inputAmount.Add(usedUTXO.Value)
		} else {
			logger.Debug("Transaction: using UTXO from multiple accounts.")
		}
	}
	for _, vout := range tx.Vout {
		if bytes.Compare([]byte(vout.PubKeyHash), vinPubKey) == 0 {
			outputAmount = outputAmount.Add(vout.Value)
		} else {
			receiverAddress = vout.PubKeyHash.GenerateAddress()
			payoutAmount = payoutAmount.Add(vout.Value)
		}
	}
	tip, err := inputAmount.Sub(outputAmount)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	senderAddress := pubKeyHash.GenerateAddress()

	return &senderAddress, &receiverAddress, payoutAmount, tip, nil
}

// Returns related previous UTXO for current transaction
func getPrevUTXOs(tx *transaction.Transaction, utxoIndex *utxo_logic.UTXOIndex) []*utxo.UTXO {
	var prevUtxos []*utxo.UTXO
	tempUtxoTxMap := make(map[string]*utxo.UTXOTx)
	for _, vin := range tx.Vin {
		pubKeyHash, err := account.NewUserPubKeyHash(vin.PubKey)
		if err != nil {
			logger.WithFields(logger.Fields{
				"tx_id":          hex.EncodeToString(tx.ID),
				"vin_tx_id":      hex.EncodeToString(vin.Txid),
				"vin_public_key": hex.EncodeToString(vin.PubKey),
			}).Warn("Transaction: failed to get PubKeyHash of vin.")
			return nil
		}
		tempUtxoTx, ok := tempUtxoTxMap[string(pubKeyHash)]
		if !ok {
			tempUtxoTx = utxoIndex.GetAllUTXOsByPubKeyHash(pubKeyHash)
			tempUtxoTxMap[string(pubKeyHash)] = tempUtxoTx
		}
		utxo := tempUtxoTx.GetUtxo(vin.Txid, vin.Vout)
		if utxo == nil {
			logger.WithFields(logger.Fields{
				"tx_id":      hex.EncodeToString(tx.ID),
				"vin_tx_id":  hex.EncodeToString(vin.Txid),
				"vin_index":  vin.Vout,
				"pubKeyHash": pubKeyHash.String(),
			}).Warn("Transaction: cannot find vin.")
			return nil
		}
		prevUtxos = append(prevUtxos, utxo)
	}
	return prevUtxos
}

//verifyPublicKeyHash verifies if the public key in Vin is the original key for the public
//key hash in utxo
func verifyPublicKeyHash(prevUtxos []*utxo.UTXO, tx *transaction.Transaction) (bool, error) {

	for i, vin := range tx.Vin {
		if prevUtxos[i].PubKeyHash == nil {
			logger.Error("Transaction: previous transaction is not correct.")
			return false, errors.New("Transaction: prevUtxos not found")
		}

		isContract, err := prevUtxos[i].PubKeyHash.IsContract()
		if err != nil {
			return false, err
		}
		//if the utxo belongs to a Contract, the utxo is not verified through
		//public key hash. It will be verified through consensus
		if isContract {
			continue
		}

		pubKeyHash, err := account.NewUserPubKeyHash(vin.PubKey)
		if err != nil {
			return false, err
		}
		if !bytes.Equal([]byte(pubKeyHash), []byte(prevUtxos[i].PubKeyHash)) {
			return false, errors.New("Transaction: ID is invalid")
		}
	}
	return true, nil
}

// IsFromContract returns true if tx is generated from a contract execution; false otherwise
func IsFromContract(utxoIndex *utxo_logic.UTXOIndex, tx *transaction.Transaction) bool {
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

func NewSmartContractDestoryTX(utxos []*utxo.UTXO, contractAddr account.Address, sourceTXID []byte) transaction.Transaction {
	sum := calculateUtxoSum(utxos)
	tips := common.NewAmount(0)
	gasLimit := common.NewAmount(0)
	gasPrice := common.NewAmount(0)

	tx, _ := NewContractTransferTX(utxos, contractAddr, account.NewAddress(SCDestroyAddress), sum, tips, gasLimit, gasPrice, sourceTXID)
	return tx
}

// NewCoinbaseTX creates a new coinbase transaction
func NewCoinbaseTX(to account.Address, data string, blockHeight uint64, tip *common.Amount) transaction.Transaction {
	if data == "" {
		data = fmt.Sprintf("Reward to '%s'", to)
	}
	bh := make([]byte, 8)
	binary.BigEndian.PutUint64(bh, uint64(blockHeight))

	txin := transaction_base.TXInput{nil, -1, bh, []byte(data)}
	txout := transaction_base.NewTXOutput(transaction.Subsidy.Add(tip), to)
	tx := transaction.Transaction{nil, []transaction_base.TXInput{txin}, []transaction_base.TXOutput{*txout}, common.NewAmount(0), common.NewAmount(0), common.NewAmount(0)}
	tx.ID = tx.Hash()

	return tx
}

// NewUTXOTransaction creates a new transaction
func NewUTXOTransaction(utxos []*utxo.UTXO, sendTxParam transaction.SendTxParam) (transaction.Transaction, error) {

	sum := calculateUtxoSum(utxos)
	change, err := calculateChange(sum, sendTxParam.Amount, sendTxParam.Tip, sendTxParam.GasLimit, sendTxParam.GasPrice)
	if err != nil {
		return transaction.Transaction{}, err
	}
	tx := transaction.Transaction{
		nil,
		prepareInputLists(utxos, sendTxParam.SenderKeyPair.GetPublicKey(), nil),
		prepareOutputLists(sendTxParam.From, sendTxParam.To, sendTxParam.Amount, change, sendTxParam.Contract),
		sendTxParam.Tip,
		sendTxParam.GasLimit,
		sendTxParam.GasPrice,
	}
	tx.ID = tx.Hash()

	err = Sign(sendTxParam.SenderKeyPair.GetPrivateKey(), utxos, &tx)
	if err != nil {
		return transaction.Transaction{}, err
	}

	return tx, nil
}

func NewContractTransferTX(utxos []*utxo.UTXO, contractAddr, toAddr account.Address, amount, tip *common.Amount, gasLimit *common.Amount, gasPrice *common.Amount, sourceTXID []byte) (transaction.Transaction, error) {
	contractPubKeyHash, ok := account.GeneratePubKeyHashByAddress(contractAddr)
	if !ok {
		return transaction.Transaction{}, account.ErrInvalidAddress
	}
	if isContract, err := contractPubKeyHash.IsContract(); !isContract {
		return transaction.Transaction{}, err
	}

	sum := calculateUtxoSum(utxos)
	change, err := calculateChange(sum, amount, tip, gasLimit, gasPrice)
	if err != nil {
		return transaction.Transaction{}, err
	}

	// Intentionally set PubKeyHash as PubKey (to recognize it is from contract) and sourceTXID as signature in Vin
	tx := transaction.Transaction{
		nil,
		prepareInputLists(utxos, contractPubKeyHash, sourceTXID),
		prepareOutputLists(contractAddr, toAddr, amount, change, ""),
		tip,
		gasLimit,
		gasPrice,
	}
	tx.ID = tx.Hash()

	return tx, nil
}

//prepareInputLists prepares a list of txinputs for a new transaction
func prepareInputLists(utxos []*utxo.UTXO, publicKey []byte, signature []byte) []transaction_base.TXInput {
	var inputs []transaction_base.TXInput

	// Build a list of inputs
	for _, utxo := range utxos {
		input := transaction_base.TXInput{utxo.Txid, utxo.TxIndex, signature, publicKey}
		inputs = append(inputs, input)
	}

	return inputs
}

//preapreOutPutLists prepares a list of txoutputs for a new transaction
func prepareOutputLists(from, to account.Address, amount *common.Amount, change *common.Amount, contract string) []transaction_base.TXOutput {

	var outputs []transaction_base.TXOutput
	toAddr := to

	if toAddr.String() == "" {
		toAddr = account.NewContractPubKeyHash().GenerateAddress()
	}

	if contract != "" {
		outputs = append(outputs, *transaction_base.NewContractTXOutput(toAddr, contract))
	}

	outputs = append(outputs, *transaction_base.NewTXOutput(amount, toAddr))
	if !change.IsZero() {
		outputs = append(outputs, *transaction_base.NewTXOutput(change, from))
	}
	return outputs
}

// Sign signs each input of a Transaction
func Sign(privKey ecdsa.PrivateKey, prevUtxos []*utxo.UTXO, tx *transaction.Transaction) error {
	if tx.IsCoinbase() {
		logger.Warn("Transaction: will not sign a coinbase transaction_base.")
		return nil
	}

	if tx.IsRewardTx() {
		logger.Warn("Transaction: will not sign a reward transaction_base.")
		return nil
	}

	if tx.IsGasRewardTx() {
		logger.Warn("Transaction: will not sign a gas reward transaction_base.")
		return nil
	}

	if tx.IsGasChangeTx() {
		logger.Warn("Transaction: will not sign a gas change transaction_base.")
		return nil
	}

	txCopy := tx.TrimmedCopy(false)
	privData, err := secp256k1.FromECDSAPrivateKey(&privKey)
	if err != nil {
		logger.WithError(err).Error("Transaction: failed to get private key.")
		return err
	}

	for i, vin := range txCopy.Vin {
		txCopy.Vin[i].Signature = nil
		oldPubKey := vin.PubKey
		txCopy.Vin[i].PubKey = []byte(prevUtxos[i].PubKeyHash)
		txCopy.ID = txCopy.Hash()

		txCopy.Vin[i].PubKey = oldPubKey

		signature, err := secp256k1.Sign(txCopy.ID, privData)
		if err != nil {
			logger.WithError(err).Error("Transaction: failed to create a signature.")
			return err
		}

		tx.Vin[i].Signature = signature
	}
	return nil
}

func IsContractDeployed(utxoIndex *utxo_logic.UTXOIndex, ctx *transaction.ContractTx) bool {
	pubkeyhash := ctx.GetContractPubKeyHash()
	if pubkeyhash == nil {
		return false
	}

	contractUtxoTx := utxoIndex.GetAllUTXOsByPubKeyHash(pubkeyhash)
	return contractUtxoTx.Size() > 0
}

func verifySignatures(prevUtxos []*utxo.UTXO, tx *transaction.Transaction) (bool, error) {
	txCopy := tx.TrimmedCopy(false)

	for i, vin := range tx.Vin {
		txCopy.Vin[i].Signature = nil
		oldPubKey := txCopy.Vin[i].PubKey
		txCopy.Vin[i].PubKey = []byte(prevUtxos[i].PubKeyHash)
		txCopy.ID = txCopy.Hash()
		txCopy.Vin[i].PubKey = oldPubKey

		originPub := make([]byte, 1+len(vin.PubKey))
		originPub[0] = 4 // uncompressed point
		copy(originPub[1:], vin.PubKey)

		verifyResult, err := secp256k1.Verify(txCopy.ID, vin.Signature, originPub)

		if err != nil || verifyResult == false {
			return false, errors.New("Transaction: Signatures is invalid")
		}
	}

	return true, nil
}

//Execute executes the smart contract the transaction points to. it doesnt do anything if is a normal transaction
func Execute(ctx *transaction.ContractTx, prevUtxos []*utxo.UTXO,
	isSCUTXO bool,
	index utxo_logic.UTXOIndex,
	scStorage *scState.ScState,
	rewards map[string]string,
	engine ScEngine,
	currblkHeight uint64,
	parentBlk *block.Block) (uint64, []*transaction.Transaction, error) {

	if engine == nil {
		return 0, nil, nil
	}

	vout := ctx.Vout[transaction.ContractTxouputIndex]

	if isSCUTXO {
		return 0, nil, nil
	}

	function, args := util.DecodeScInput(vout.Contract)
	if function == "" {
		return 0, nil, ErrUnsupportedSourceType
	}

	totalArgs := util.PrepareArgs(args)
	address := vout.PubKeyHash.GenerateAddress()
	logger.WithFields(logger.Fields{
		"contract_address": address.String(),
		"invoked_function": function,
		"arguments":        totalArgs,
	}).Debug("Transaction: is executing the smart contract...")

	createContractUtxo, invokeUtxos := index.SplitContractUtxo([]byte(vout.PubKeyHash))
	if err := engine.SetExecutionLimits(ctx.GasLimit.Uint64(), 0); err != nil {
		return 0, nil, ErrInvalidGasLimit
	}
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

func CheckContractSyntax(sc ScEngine, out transaction_base.TXOutput) error {
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

//calculateUtxoSum calculates the total amount of all input utxos
func calculateUtxoSum(utxos []*utxo.UTXO) *common.Amount {
	sum := common.NewAmount(0)
	for _, utxo := range utxos {
		sum = sum.Add(utxo.Value)
	}
	return sum
}

//calculateChange calculates the change
func calculateChange(input, amount, tip *common.Amount, gasLimit *common.Amount, gasPrice *common.Amount) (*common.Amount, error) {
	change, err := input.Sub(amount)
	if err != nil {
		return nil, transaction.ErrInsufficientFund
	}

	change, err = change.Sub(tip)
	if err != nil {
		return nil, transaction.ErrInsufficientFund
	}
	change, err = change.Sub(gasLimit.Mul(gasPrice))
	if err != nil {
		return nil, transaction.ErrInsufficientFund
	}
	return change, nil
}
