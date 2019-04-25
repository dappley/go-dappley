package main

import (
	"context"
	"fmt"
	"github.com/dappley/go-dappley/client"
	"github.com/dappley/go-dappley/common"
	"github.com/dappley/go-dappley/config"
	"github.com/dappley/go-dappley/config/pb"
	"github.com/dappley/go-dappley/core"
	"github.com/dappley/go-dappley/core/pb"
	"github.com/dappley/go-dappley/logic"
	"github.com/dappley/go-dappley/rpc/pb"
	"github.com/dappley/go-dappley/storage"
	logger "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"time"
)

var (
	password      = "testpassword"
	maxWallet     = 5
	initialAmount = uint64(100000)
	fundTimeout   = time.Duration(time.Minute * 5)
	txSenders     = []transctionSender{
		sendNormalTransaction,
		sendTransactionUsingUnexistingUTXO,
		sendTransactionUsingUnAuthorizedUTXO,
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
	conn := initRpcClient(int(cliConfig.GetPort()))

	adminClient := rpcpb.NewAdminServiceClient(conn)
	rpcClient := rpcpb.NewRpcServiceClient(conn)

	addrs := createWallet()
	wm, err := logic.GetWalletManager(client.GetWalletFilePath())
	if err != nil {
		logger.Panic("Can not get access to wallet")
	}

	fundFromMiner(adminClient, rpcClient, addrs[0].String())
	fundFromMiner(adminClient, rpcClient, addrs[2].String())

	ticker := time.NewTicker(time.Millisecond * 200).C
	currHeight := uint64(0)
	index := 0
	for {
		select {
		case <-ticker:
			height := getBlockHeight(rpcClient)
			if height > currHeight {
				currHeight = height
				displayBalances(rpcClient, addrs)
				utxoIndex = updateUtxoIndex(rpcClient, addrs)
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

func initRpcClient(port int) *grpc.ClientConn {
	//prepare grpc client
	var conn *grpc.ClientConn
	conn, err := grpc.Dial(fmt.Sprint(":", port), grpc.WithInsecure())
	if err != nil {
		logger.WithError(err).Panic("Connection to RPC server failed.")
	}
	return conn
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

func updateUtxoIndex(serviceClient rpcpb.RpcServiceClient, addrs []core.Address) *core.UTXOIndex {
	wm, err := logic.GetWalletManager(client.GetWalletFilePath())
	if err != nil {
		logger.WithError(err).Panic("updateUtxoIndex: Unable to get wallet")
	}

	utxoIndex := core.NewUTXOIndex(core.NewUTXOCache(storage.NewRamStorage()))

	for _, addr := range addrs {
		kp := wm.GetKeyPairByAddress(addr)
		_, err := core.NewUserPubKeyHash(kp.PublicKey)
		if err != nil {
			logger.WithError(err).Panic("updateUtxoIndex: Unable to get public key hash")
		}

		utxos := getUtxoByAddr(serviceClient, addr)
		for _, utxoPb := range utxos {
			utxo := core.UTXO{}
			utxo.FromProto(utxoPb)
			utxoIndex.AddUTXO(utxo.TXOutput, utxo.Txid, utxo.TxIndex)
		}
	}
	return utxoIndex
}

func getUtxoByAddr(serviceClient rpcpb.RpcServiceClient, addr core.Address) []*corepb.Utxo {
	resp, err := serviceClient.RpcGetUTXO(context.Background(), &rpcpb.GetUTXORequest{
		Address: addr.String(),
	})
	if err != nil {
		logger.WithError(err).WithFields(logger.Fields{
			"addr": addr.String(),
		}).Error("Can not update utxo")
	}
	return resp.Utxos
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

func fundFromMiner(adminClient rpcpb.AdminServiceClient, rpcClient rpcpb.RpcServiceClient, fundAddr string) {
	logger.Info("Requesting fund from miner...")

	if fundAddr == "" {
		logger.Panic("There is no wallet to receive fund.")
	}

	requestFundFromMiner(adminClient, fundAddr)
	bal, isSufficient := checkSufficientInitialAmount(rpcClient, fundAddr)
	if isSufficient {
		//continue if the initial amount is sufficient
		return
	}
	logger.WithFields(logger.Fields{
		"address":     fundAddr,
		"balance":     bal,
		"target_fund": initialAmount,
	}).Info("Current wallet balance is insufficient. Waiting for more funds...")
	waitTilInitialAmountIsSufficient(adminClient, rpcClient, fundAddr)
}

func requestFundFromMiner(adminClient rpcpb.AdminServiceClient, fundAddr string) {

	sendFromMinerRequest := &rpcpb.SendFromMinerRequest{To: fundAddr, Amount: common.NewAmount(initialAmount).Bytes()}

	adminClient.RpcSendFromMiner(context.Background(), sendFromMinerRequest)
}

func checkSufficientInitialAmount(rpcClient rpcpb.RpcServiceClient, addr string) (uint64, bool) {
	balance, err := getBalance(rpcClient, addr)
	if err != nil {
		logger.WithError(err).WithFields(logger.Fields{
			"address": addr,
		}).Panic("Failed to get balance.")
	}
	return uint64(balance), uint64(balance) >= initialAmount
}

func waitTilInitialAmountIsSufficient(adminClient rpcpb.AdminServiceClient, rpcClient rpcpb.RpcServiceClient, addr string) {
	checkBalanceTicker := time.NewTicker(time.Second * 5).C
	timeout := time.NewTicker(fundTimeout).C
	for {
		select {
		case <-checkBalanceTicker:
			bal, isSufficient := checkSufficientInitialAmount(rpcClient, addr)
			if isSufficient {
				//continue if the initial amount is sufficient
				return
			}
			logger.WithFields(logger.Fields{
				"address":     addr,
				"balance":     bal,
				"target_fund": initialAmount,
			}).Info("Current wallet balance is insufficient. Waiting for more funds...")
			requestFundFromMiner(adminClient, addr)
		case <-timeout:
			logger.WithFields(logger.Fields{
				"target_fund": initialAmount,
			}).Panic("Timed out while waiting for sufficient fund from miner!")
		}
	}
}

func getBalance(rpcClient rpcpb.RpcServiceClient, address string) (int64, error) {
	response, err := rpcClient.RpcGetBalance(context.Background(), &rpcpb.GetBalanceRequest{Address: address})
	return response.Amount, err
}

func displayBalances(rpcClient rpcpb.RpcServiceClient, addresses []core.Address) {
	for _, addr := range addresses {
		amount, err := getBalance(rpcClient, addr.String())
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

func getBlockHeight(rpcClient rpcpb.RpcServiceClient) uint64 {
	resp, err := rpcClient.RpcGetBlockchainInfo(
		context.Background(),
		&rpcpb.GetBlockchainInfoRequest{})
	if err != nil {
		logger.WithError(err).Panic("Cannot get block height.")
	}
	return resp.BlockHeight
}
