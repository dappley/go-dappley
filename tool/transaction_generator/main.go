package main

import (
	"github.com/dappley/go-dappley/client"
	"github.com/dappley/go-dappley/common"
	"github.com/dappley/go-dappley/config"
	"github.com/dappley/go-dappley/core"
	"github.com/dappley/go-dappley/core/pb"
	"github.com/dappley/go-dappley/sdk"
	"github.com/dappley/go-dappley/tool/transaction_automator/pb"
	logger "github.com/sirupsen/logrus"
	"time"
)

var txSenders = []transctionSender{
	sendNormalTransaction,
	sendTransactionUsingUnexistingUTXO,
	sendTransactionUsingUnAuthorizedUTXO,
	sendTransactionWithInsufficientBalance,
	sendDoubleSpendingTransactions,
}

const configFilePath = "default.conf"

type transctionSender func(dappSdk *sdk.DappSdk, wallet *sdk.DappSdkWallet, addrs []core.Address, wm *client.WalletManager)

func main() {
	logger.SetFormatter(&logger.TextFormatter{
		FullTimestamp: true,
	})

	toolConfigs := &tx_automator_configpb.Config{}
	config.LoadConfig(configFilePath, toolConfigs)

	grpcClient := sdk.NewDappSdkGrpcClient(toolConfigs.GetPort())
	dappSdk := sdk.NewDappSdk(grpcClient)
	wallet := sdk.NewDappSdkWallet(
		toolConfigs.GetMaxWallet(),
		toolConfigs.GetPassword(),
		dappSdk,
	)

	addrs := wallet.GetAddrs()
	wm := wallet.GetWalletManager()
	fundRequest := sdk.NewDappSdkFundRequest(grpcClient, dappSdk)
	initialAmount := common.NewAmount(toolConfigs.GetInitialAmount())
	fundRequest.Fund(wallet.GetAddrs()[0].String(), initialAmount)
	fundRequest.Fund(wallet.GetAddrs()[2].String(), initialAmount)

	wallet.UpdateFromServer()

	ticker := time.NewTicker(time.Millisecond * 200).C
	currHeight := uint64(0)
	index := 0
	for {
		select {
		case <-ticker:
			height, err := dappSdk.GetBlockHeight()
			if err != nil {
				logger.Panic("Can not get block height from server")
			}

			if height > currHeight {
				currHeight = height
				wallet.UpdateFromServer()
				txSenders[index](dappSdk, wallet, addrs, wm)
				index = index + 1
				if index >= len(txSenders) {
					logger.Info("All transactions are sent. Exiting...")
					return
				}
			}
		}
	}
}

func sendTransactionUsingUnexistingUTXO(dappSdk *sdk.DappSdk, wallet *sdk.DappSdkWallet, addrs []core.Address, wm *client.WalletManager) {

	logger.Info("Sending a transaction with unexisitng utxo...")

	amount := common.NewAmount(10)
	tx := createTransactionUsingUnexistingUTXO(wallet,
		addrs,
		amount,
		common.NewAmount(0),
		wm)

	_, err := dappSdk.SendTransaction(tx.ToProto().(*corepb.Transaction))

	if err != nil {
		logger.WithError(err).Error("Unable to send transaction!")
	}
}

func sendNormalTransaction(dappSdk *sdk.DappSdk, wallet *sdk.DappSdkWallet, addrs []core.Address, wm *client.WalletManager) {

	amount := common.NewAmount(10)

	tx := createNormalTransaction(wallet,
		addrs,
		amount,
		common.NewAmount(0),
		wm)

	_, err := dappSdk.SendTransaction(tx.ToProto().(*corepb.Transaction))

	logger.Info("Sending a normal transaction...")

	if err != nil {
		logger.WithError(err).Panic("Unable to send transaction!")
	}

	logger.Info("Send is successful!")
}

func sendTransactionUsingUnAuthorizedUTXO(dappSdk *sdk.DappSdk, wallet *sdk.DappSdkWallet, addrs []core.Address, wm *client.WalletManager) {

	amount := common.NewAmount(10)
	tx := createTransactionUsingUnauthorizedUTXO(wallet,
		addrs,
		amount,
		common.NewAmount(0),
		wm)

	_, err := dappSdk.SendTransaction(tx.ToProto().(*corepb.Transaction))

	logger.Info("Sending a transaction with unauthorized utxo...")

	if err != nil {
		logger.WithError(err).Error("Unable to send transaction!")
	}
}

func sendTransactionWithInsufficientBalance(dappSdk *sdk.DappSdk, wallet *sdk.DappSdkWallet, addrs []core.Address, wm *client.WalletManager) {

	amount := common.NewAmount(10)
	tx := createTransactionWithInsufficientBalance(wallet,
		addrs,
		amount,
		common.NewAmount(0),
		wm)

	_, err := dappSdk.SendTransaction(tx.ToProto().(*corepb.Transaction))

	logger.Info("Sending a transaction with Insufficient Balance...")

	if err != nil {
		logger.WithError(err).Error("Unable to send transaction!")
	}
}

func sendDoubleSpendingTransactions(dappSdk *sdk.DappSdk, wallet *sdk.DappSdkWallet, addrs []core.Address, wm *client.WalletManager) {

	amount := common.NewAmount(10)

	tx := createNormalTransaction(wallet,
		addrs,
		amount,
		common.NewAmount(0),
		wm)

	_, err := dappSdk.SendTransaction(tx.ToProto().(*corepb.Transaction))

	logger.Info("Sending double spending transactions: Sending Tx 1")

	if err != nil {
		logger.WithError(err).Panic("Unable to send transaction!")
	}

	_, err = dappSdk.SendTransaction(tx.ToProto().(*corepb.Transaction))

	logger.Info("Sending double spending transactions: Sending Tx 2")

	if err != nil {
		logger.WithError(err).Error("Unable to send transaction!")
	}
}

func createNormalTransaction(wallet *sdk.DappSdkWallet, addrs []core.Address, amount, tip *common.Amount, wm *client.WalletManager) *core.Transaction {

	from := addrs[0]
	senderKeyPair := wm.GetKeyPairByAddress(from)
	to := addrs[1]
	pkh, err := core.NewUserPubKeyHash(senderKeyPair.PublicKey)

	if err != nil {
		logger.WithError(err).Panic("Unable to hash sender public key")
	}
	prevUtxos, err := wallet.GetUtxoIndex().GetUTXOsByAmount(pkh, amount)
	return createTransaction(prevUtxos, from, to, amount, tip, senderKeyPair)
}

func createTransactionUsingUnexistingUTXO(wallet *sdk.DappSdkWallet, addrs []core.Address, amount, tip *common.Amount, wm *client.WalletManager) *core.Transaction {

	from := addrs[0]
	senderKeyPair := wm.GetKeyPairByAddress(from)
	to := addrs[1]
	pkh, err := core.NewUserPubKeyHash(senderKeyPair.PublicKey)

	if err != nil {
		logger.WithError(err).Panic("Unable to hash sender public key")
	}
	prevUtxos, err := wallet.GetUtxoIndex().GetUTXOsByAmount(pkh, amount)
	unexistingUtxo := &core.UTXO{
		TXOutput: *core.NewTXOutput(common.NewAmount(10), from),
		Txid:     []byte("FakeTxId"),
		TxIndex:  0,
		UtxoType: core.UtxoNormal,
	}
	prevUtxos = append(prevUtxos, unexistingUtxo)
	return createTransaction(prevUtxos, from, to, amount, tip, senderKeyPair)
}

func createTransactionUsingUnauthorizedUTXO(wallet *sdk.DappSdkWallet, addrs []core.Address, amount, tip *common.Amount, wm *client.WalletManager) *core.Transaction {

	from := addrs[0]
	senderKeyPair := wm.GetKeyPairByAddress(from)
	to := addrs[1]
	pkh, err := core.NewUserPubKeyHash(senderKeyPair.PublicKey)

	if err != nil {
		logger.WithError(err).Panic("Unable to hash sender public key")
	}
	prevUtxos, err := wallet.GetUtxoIndex().GetUTXOsByAmount(pkh, amount)
	unauthorizedpkh, err := core.NewUserPubKeyHash(wm.GetKeyPairByAddress(addrs[2]).PublicKey)
	unauthorizedUtxo := wallet.GetUtxoIndex().GetAllUTXOsByPubKeyHash(unauthorizedpkh).GetAllUtxos()
	prevUtxos = append(prevUtxos, unauthorizedUtxo[0])

	return createTransaction(prevUtxos, from, to, amount, tip, senderKeyPair)
}

func createTransactionWithInsufficientBalance(wallet *sdk.DappSdkWallet, addrs []core.Address, amount, tip *common.Amount, wm *client.WalletManager) *core.Transaction {

	from := addrs[0]
	senderKeyPair := wm.GetKeyPairByAddress(from)
	to := addrs[1]
	pkh, err := core.NewUserPubKeyHash(senderKeyPair.PublicKey)

	if err != nil {
		logger.WithError(err).Panic("Unable to hash sender public key")
	}
	prevUtxos, err := wallet.GetUtxoIndex().GetUTXOsByAmount(pkh, amount)
	tx := createTransaction(prevUtxos, from, to, amount, tip, senderKeyPair)
	tx.Vin = tx.Vin[:len(tx.Vin)-1]

	return tx
}

func createTransaction(prevUtxos []*core.UTXO, from, to core.Address, amount, tip *common.Amount, senderKeyPair *core.KeyPair) *core.Transaction {

	sum := calculateUtxoSum(prevUtxos)
	change := calculateChange(sum, amount, tip)
	vouts := prepareOutputLists(from, to, amount, change)

	tx := newTransaction(prevUtxos, vouts, tip, senderKeyPair)
	logger.WithFields(logger.Fields{
		"From":   from,
		"To":     to,
		"Amount": amount,
	}).Info("Creating a transaction...")
	return tx
}

func newTransaction(prevUtxos []*core.UTXO, vouts []core.TXOutput, tip *common.Amount, senderKeyPair *core.KeyPair) *core.Transaction {
	tx := &core.Transaction{
		nil,
		prepareInputLists(prevUtxos, senderKeyPair.PublicKey, nil),
		vouts,
		tip}
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

func prepareOutputLists(from, to core.Address, amount *common.Amount, change *common.Amount) []core.TXOutput {

	var outputs []core.TXOutput
	toAddr := to

	outputs = append(outputs, *core.NewTXOutput(amount, toAddr))
	if !change.IsZero() {
		outputs = append(outputs, *core.NewTXOutput(change, from))
	}
	return outputs
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
