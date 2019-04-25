package main

import (
	"context"
	"github.com/dappley/go-dappley/client"
	"github.com/dappley/go-dappley/common"
	"github.com/dappley/go-dappley/config"
	"github.com/dappley/go-dappley/config/pb"
	"github.com/dappley/go-dappley/core"
	"github.com/dappley/go-dappley/core/pb"
	"github.com/dappley/go-dappley/logic"
	"github.com/dappley/go-dappley/rpc/pb"
	"github.com/dappley/go-dappley/tool"
	logger "github.com/sirupsen/logrus"
	"time"
)

var (
	password      = "testpassword"
	maxWallet     = 5
	initialAmount = common.NewAmount(100000)
	fundTimeout   = time.Duration(time.Minute * 5)
	txSenders     = []transctionSender{
		sendNormalTransaction,
		sendTransactionUsingUnexistingUTXO,
		sendTransactionUsingUnAuthorizedUTXO,
		sendTransactionWithInsufficientBalance,
		sendDoubleSpendingTransactions,
	}
)

const (
	rpcFilePath = "default.conf"
)

type transctionSender func(rpcClient rpcpb.RpcServiceClient, utxoIndex *core.UTXOIndex, addrs []core.Address, wm *client.WalletManager)

func main() {
	logger.SetFormatter(&logger.TextFormatter{
		FullTimestamp: true,
	})

	var utxoIndex *core.UTXOIndex

	cliConfig := &configpb.CliConfig{}
	config.LoadConfig(rpcFilePath, cliConfig)
	conn := tool.InitRpcClient(int(cliConfig.GetPort()))

	adminClient := rpcpb.NewAdminServiceClient(conn)
	rpcClient := rpcpb.NewRpcServiceClient(conn)

	addrs := createWallet()
	wm, err := logic.GetWalletManager(client.GetWalletFilePath())
	if err != nil {
		logger.Panic("Can not get access to wallet")
	}
	tool.FundFromMiner(adminClient, rpcClient, addrs[0].String(), initialAmount)
	tool.FundFromMiner(adminClient, rpcClient, addrs[2].String(), initialAmount)

	ticker := time.NewTicker(time.Millisecond * 200).C
	currHeight := uint64(0)
	index := 0
	for {
		select {
		case <-ticker:
			height := tool.GetBlockHeight(rpcClient)
			if height > currHeight {
				currHeight = height
				displayBalances(rpcClient, addrs)
				utxoIndex = tool.UpdateUtxoIndex(rpcClient, addrs)
				logger.Info("Transaction ", index)
				txSenders[index](rpcClient, utxoIndex, addrs, wm)
				index = index + 1
				if index >= len(txSenders) {
					logger.Info("All transactions are sent. Exiting...")
					return
				}
			}
		}
	}
}

func sendTransactionUsingUnexistingUTXO(rpcClient rpcpb.RpcServiceClient, utxoIndex *core.UTXOIndex, addrs []core.Address, wm *client.WalletManager) {

	amount := common.NewAmount(10)
	tx := createTransactionUsingUnexistingUTXO(utxoIndex,
		addrs,
		amount,
		common.NewAmount(0),
		wm)

	txpb := tx.ToProto().(*corepb.Transaction)

	_, err := rpcClient.RpcSendTransaction(
		context.Background(),
		&rpcpb.SendTransactionRequest{Transaction: txpb},
	)

	logger.Info("Sending a transaction with unexisitng utxo...")

	if err != nil {
		logger.WithError(err).Warn("Unable to send transaction!")
	}
}

func sendNormalTransaction(rpcClient rpcpb.RpcServiceClient, utxoIndex *core.UTXOIndex, addrs []core.Address, wm *client.WalletManager) {

	amount := common.NewAmount(10)

	tx := createNormalTransaction(utxoIndex,
		addrs,
		amount,
		common.NewAmount(0),
		wm)

	txpb := tx.ToProto().(*corepb.Transaction)

	_, err := rpcClient.RpcSendTransaction(
		context.Background(),
		&rpcpb.SendTransactionRequest{Transaction: txpb},
	)

	logger.Info("Sending a normal transaction...")

	if err != nil {
		logger.WithError(err).Panic("Unable to send transaction!")
	}
}

func sendTransactionUsingUnAuthorizedUTXO(rpcClient rpcpb.RpcServiceClient, utxoIndex *core.UTXOIndex, addrs []core.Address, wm *client.WalletManager) {

	amount := common.NewAmount(10)
	tx := createTransactionUsingUnauthorizedUTXO(utxoIndex,
		addrs,
		amount,
		common.NewAmount(0),
		wm)

	txpb := tx.ToProto().(*corepb.Transaction)

	_, err := rpcClient.RpcSendTransaction(
		context.Background(),
		&rpcpb.SendTransactionRequest{Transaction: txpb},
	)

	logger.Info("Sending a transaction with unauthorized utxo...")

	if err != nil {
		logger.WithError(err).Warn("Unable to send transaction!")
	}
}

func sendTransactionWithInsufficientBalance(rpcClient rpcpb.RpcServiceClient, utxoIndex *core.UTXOIndex, addrs []core.Address, wm *client.WalletManager) {

	amount := common.NewAmount(10)
	tx := createTransactionWithInsufficientBalance(utxoIndex,
		addrs,
		amount,
		common.NewAmount(0),
		wm)

	txpb := tx.ToProto().(*corepb.Transaction)

	_, err := rpcClient.RpcSendTransaction(
		context.Background(),
		&rpcpb.SendTransactionRequest{Transaction: txpb},
	)

	logger.Info("Sending a transaction with Insufficient Balance...")

	if err != nil {
		logger.WithError(err).Warn("Unable to send transaction!")
	}
}

func sendDoubleSpendingTransactions(rpcClient rpcpb.RpcServiceClient, utxoIndex *core.UTXOIndex, addrs []core.Address, wm *client.WalletManager) {

	amount := common.NewAmount(10)

	tx := createNormalTransaction(utxoIndex,
		addrs,
		amount,
		common.NewAmount(0),
		wm)

	txpb := tx.ToProto().(*corepb.Transaction)

	_, err := rpcClient.RpcSendTransaction(
		context.Background(),
		&rpcpb.SendTransactionRequest{Transaction: txpb},
	)

	logger.Info("Sending double spending transactions: Sending Tx 1")

	if err != nil {
		logger.WithError(err).Warn("Unable to send transaction!")
	}

	_, err = rpcClient.RpcSendTransaction(
		context.Background(),
		&rpcpb.SendTransactionRequest{Transaction: txpb},
	)

	logger.Info("Sending double spending transactions: Sending Tx 2")

	if err != nil {
		logger.WithError(err).Warn("Unable to send transaction!")
	}
}

func createWallet() []core.Address {
	wm, err := logic.GetWalletManager(client.GetWalletFilePath())
	if err != nil {
		logger.Panic("Cannot get wallet manager.")
	}
	addresses := wm.GetAddresses()
	numOfWallets := len(addresses)
	for i := numOfWallets; i < maxWallet; i++ {
		_, err := logic.CreateWalletWithpassphrase(password)
		if err != nil {
			logger.WithError(err).Panic("Cannot create new wallet.")
		}
	}

	addresses = wm.GetAddresses()
	logger.WithFields(logger.Fields{
		"addresses": addresses,
	}).Info("Wallets are created")
	return addresses
}

func createNormalTransaction(utxoIndex *core.UTXOIndex, addrs []core.Address, amount, tip *common.Amount, wm *client.WalletManager) *core.Transaction {

	from := addrs[0]
	senderKeyPair := wm.GetKeyPairByAddress(from)
	to := addrs[1]
	pkh, err := core.NewUserPubKeyHash(senderKeyPair.PublicKey)

	if err != nil {
		logger.WithError(err).Panic("Unable to hash sender public key")
	}
	prevUtxos, err := utxoIndex.GetUTXOsByAmount(pkh, amount)
	return createTransaction(prevUtxos, from, to, amount, tip, senderKeyPair)
}

func createTransactionUsingUnexistingUTXO(utxoIndex *core.UTXOIndex, addrs []core.Address, amount, tip *common.Amount, wm *client.WalletManager) *core.Transaction {

	from := addrs[0]
	senderKeyPair := wm.GetKeyPairByAddress(from)
	to := addrs[1]
	pkh, err := core.NewUserPubKeyHash(senderKeyPair.PublicKey)

	if err != nil {
		logger.WithError(err).Panic("Unable to hash sender public key")
	}
	prevUtxos, err := utxoIndex.GetUTXOsByAmount(pkh, amount)
	unexistingUtxo := &core.UTXO{
		TXOutput: *core.NewTXOutput(common.NewAmount(10), from),
		Txid:     []byte("FakeTxId"),
		TxIndex:  0,
		UtxoType: core.UtxoNormal,
	}
	prevUtxos = append(prevUtxos, unexistingUtxo)
	return createTransaction(prevUtxos, from, to, amount, tip, senderKeyPair)
}

func createTransactionUsingUnauthorizedUTXO(utxoIndex *core.UTXOIndex, addrs []core.Address, amount, tip *common.Amount, wm *client.WalletManager) *core.Transaction {

	from := addrs[0]
	senderKeyPair := wm.GetKeyPairByAddress(from)
	to := addrs[1]
	pkh, err := core.NewUserPubKeyHash(senderKeyPair.PublicKey)

	if err != nil {
		logger.WithError(err).Panic("Unable to hash sender public key")
	}
	prevUtxos, err := utxoIndex.GetUTXOsByAmount(pkh, amount)
	unauthorizedpkh, err := core.NewUserPubKeyHash(wm.GetKeyPairByAddress(addrs[2]).PublicKey)
	unauthorizedUtxo := utxoIndex.GetAllUTXOsByPubKeyHash(unauthorizedpkh).GetAllUtxos()
	prevUtxos = append(prevUtxos, unauthorizedUtxo[0])

	return createTransaction(prevUtxos, from, to, amount, tip, senderKeyPair)
}

func createTransactionWithInsufficientBalance(utxoIndex *core.UTXOIndex, addrs []core.Address, amount, tip *common.Amount, wm *client.WalletManager) *core.Transaction {

	from := addrs[0]
	senderKeyPair := wm.GetKeyPairByAddress(from)
	to := addrs[1]
	pkh, err := core.NewUserPubKeyHash(senderKeyPair.PublicKey)

	if err != nil {
		logger.WithError(err).Panic("Unable to hash sender public key")
	}
	prevUtxos, err := utxoIndex.GetUTXOsByAmount(pkh, amount)
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

func displayBalances(rpcClient rpcpb.RpcServiceClient, addresses []core.Address) {
	for _, addr := range addresses {
		amount, err := tool.GetBalance(rpcClient, addr.String())
		balanceLogger := logger.WithFields(logger.Fields{
			"address": addr.String(),
			"amount":  amount,
		})
		if err != nil {
			balanceLogger.WithError(err).Warn("Failed to get wallet balance.")
		}
		balanceLogger.Info("Displaying wallet balance...")
	}
}
