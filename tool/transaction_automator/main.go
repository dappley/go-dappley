package main

import (
	"context"
	"encoding/csv"
	"encoding/hex"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"time"

	"github.com/dappley/go-dappley/storage"

	logger "github.com/sirupsen/logrus"
	"google.golang.org/grpc"

	"github.com/dappley/go-dappley/client"
	"github.com/dappley/go-dappley/common"
	"github.com/dappley/go-dappley/config"
	"github.com/dappley/go-dappley/config/pb"
	"github.com/dappley/go-dappley/core"
	"github.com/dappley/go-dappley/core/pb"
	"github.com/dappley/go-dappley/logic"
	"github.com/dappley/go-dappley/rpc/pb"
	_ "net/http/pprof"
)

var (
	password             = "testpassword"
	maxWallet            = 10
	initialAmount        = uint64(1000)
	maxDefaultSendAmount = uint64(5)
	sendInterval         = time.Duration(5000) //ms
	fundTimeout          = time.Duration(time.Minute * 5)
	currBalance          = make(map[string]uint64)
	tempBalance			 = make(map[string]uint64)
	sentTxs              = make(map[string]int)
	utxoIndex            = &core.UTXOIndex{}
	smartContractAddr    = ""
	smartContractCounter = 0
	isContractDeployed   = false
	currHeight           = uint64(0)
)

const(
	cmdStart = iota
	cmdStop
)


const (
	smartContractSendFreq  = 10000000000
	contractAddrFilePath   = "contract/contractAddr"
	contractFilePath       = "contract/test_contract.js"
	contractFunctionCall   = "{\"function\":\"record\",\"args\":[\"dEhFf5mWTSe67mbemZdK3WiJh8FcCayJqm\",\"4\"]}"
	transactionLogFilePath = "log/tx.csv"
	failedTxLogFilePath    = "log/failedTx.csv"
	numOfTxPerBlk          = 30
	TimeBetweenBatch       = time.Duration(1000)
)

func main() {
	logger.SetFormatter(&logger.TextFormatter{
		FullTimestamp: true,
	})

	var filePath string
	flag.StringVar(&filePath, "f", "conf/default_cli.conf", "CLI config file path")
	flag.Parse()

	go func() {
		log.Println(http.ListenAndServe("localhost:6061", nil))
	}()

	cliConfig := &configpb.CliConfig{}
	config.LoadConfig(filePath, cliConfig)
	conn := initRpcClient(int(cliConfig.GetPort()))

	adminClient := rpcpb.NewAdminServiceClient(conn)
	rpcClient := rpcpb.NewRpcServiceClient(conn)

	addresses := createWallet()

	fundAddr := addresses[0].String()
	fundFromMiner(adminClient, rpcClient, fundAddr)
	logger.WithFields(logger.Fields{
		"initial_total_amount": initialAmount,
		"send_interval":        fmt.Sprintf("%d ms", sendInterval),
	}).Info("Funding is completed. Script starts.")
	displayBalances(rpcClient, addresses, true)

	utxoIndex = core.NewUTXOIndex(core.NewUTXOCache(storage.NewRamStorage()))
	updateUtxoIndex(rpcClient, addresses)

	deploySmartContract(adminClient, fundAddr)
	currHeight = getBlockHeight(rpcClient)

	ticker := time.NewTicker(time.Millisecond*200).C

	cmdChan := make(chan int, 5)
	go sendRandomTransactions(rpcClient, addresses, cmdChan)
	cmdChan <- cmdStart

	for {
		select {
		case <-ticker:
			height := getBlockHeight(rpcClient)
			if height > currHeight {
				cmdChan <- cmdStop

				isContractDeployed = true
				currHeight = height

				displayBalances(rpcClient, addresses, true)
				updateUtxoIndex(rpcClient, addresses)

				//blk := getTailBlock(rpcClient, currHeight)
				//logger.WithFields(logger.Fields{
				//	"height": currHeight,
				//}).Info("New Block Height")
				//verifyTransactions(blk.GetTransactions())
				//recordTransactions(blk.GetTransactions(), currHeight)

			}else{
				cmdChan <- cmdStart
			}
		}
	}
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

func updateUtxoIndex(serviceClient rpcpb.RpcServiceClient, addrs []core.Address) {
	wm, err := logic.GetWalletManager(client.GetWalletFilePath())
	if err != nil {
		logger.WithError(err).Panic("updateUtxoIndex: Unable to get wallet")
	}
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
}

func recordTransactions(txs []*corepb.Transaction, height uint64) {
	f, err := os.OpenFile(transactionLogFilePath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		logger.Panic("Open file failed while recording transactions")
	}
	w := csv.NewWriter(f)
	for _, tx := range txs {
		vinStr := ""
		for _, vin := range tx.GetVin() {
			vinStr += hex.EncodeToString(vin.GetTxid()) + ":" + fmt.Sprint(vin.GetVout()) + ",\n"
		}
		voutStr := ""
		for _, vout := range tx.GetVout() {
			voutStr += core.PubKeyHash(vout.GetPublicKeyHash()).GenerateAddress().String() + ":" + common.NewAmountFromBytes(vout.GetValue()).String() + ",\n"
		}
		w.Write([]string{fmt.Sprint(height), hex.EncodeToString(tx.GetId()), vinStr, voutStr, common.NewAmountFromBytes(tx.GetTip()).String()})
	}
	w.Flush()
}

func getSmartContractAddr() string {
	bytes, err := ioutil.ReadFile(contractAddrFilePath)
	if err != nil {
		logger.WithError(err).WithFields(logger.Fields{
			"file_path": contractAddrFilePath,
		}).Panic("Unable to read file!")
	}
	return string(bytes)
}

func recordSmartContractAddr(addr string) {
	err := ioutil.WriteFile(contractAddrFilePath, []byte(addr), os.FileMode(777))
	if err != nil {
		logger.WithError(err).WithFields(logger.Fields{
			"file_path":     contractAddrFilePath,
			"contract_addr": addr,
		}).Panic("Unable to record smart contract address!")
	}
}

func deploySmartContract(serviceClient rpcpb.AdminServiceClient, from string) {

	smartContractAddr = getSmartContractAddr()
	if smartContractAddr != "" {
		logger.WithFields(logger.Fields{
			"contractAddr": smartContractAddr,
		}).Info("Smart contract has already been deployed. If you are sure it is not deployed, empty the file:", contractAddrFilePath)
		isContractDeployed = true
		return
	}

	data, err := ioutil.ReadFile(contractFilePath)
	if err != nil {
		logger.WithError(err).WithFields(logger.Fields{
			"file_path": contractFilePath,
		}).Panic("Unable to read smart contract file!")
	}

	contract := string(data)
	resp, err := sendTransaction(serviceClient, from, "", 1, contract)
	smartContractAddr = resp.ContractAddress
	if err != nil {
		logger.WithError(err).WithFields(logger.Fields{
			"file_path":     contractFilePath,
			"contract_addr": smartContractAddr,
		}).Panic("Deploy smart contract failed!")
	}

	recordSmartContractAddr(smartContractAddr)

	logger.WithFields(logger.Fields{
		"contract_addr": smartContractAddr,
	}).Info("Smart contract has been deployed")
}

func getTailBlock(serviceClient rpcpb.RpcServiceClient, blkHeight uint64) *corepb.Block {
	resp, err := serviceClient.RpcGetBlockByHeight(context.Background(), &rpcpb.GetBlockByHeightRequest{Height: blkHeight})
	for err!=nil {
		logger.WithError(err).Error("GetTailBlock failed")
		resp, err = serviceClient.RpcGetBlockByHeight(context.Background(), &rpcpb.GetBlockByHeightRequest{Height: blkHeight})
	}
	return resp.Block
}

func verifyTransactions(txs []*corepb.Transaction) {
	logger.WithFields(logger.Fields{
		"num_of_tx": len(txs),
	}).Info("Transactions mined in previous block.")
	logger.WithFields(logger.Fields{
		"num_of_tx": len(sentTxs),
	}).Info("Transactions recorded")
	for _, tx := range txs {
		delete(sentTxs, hex.EncodeToString(tx.GetId()))
	}
	for txid, count := range sentTxs {
		sentTxs[txid]++
		if count > 0 {
			logger.WithFields(logger.Fields{
				"txid":  txid,
				"count": count,
			}).Warn("Transaction is not found in previous block!")
		}
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

func requestFundFromMiner(adminClient rpcpb.AdminServiceClient, fundAddr string) {

	sendFromMinerRequest := &rpcpb.SendFromMinerRequest{To: fundAddr, Amount: common.NewAmount(initialAmount).Bytes()}

	adminClient.RpcSendFromMiner(context.Background(), sendFromMinerRequest)
}

func sendBatchTransactions(client rpcpb.RpcServiceClient, txs []*corepb.Transaction) error{

	_, err := client.RpcSendBatchTransaction(context.Background(),&rpcpb.SendBatchTransactionRequest{
		Transactions:  txs,
	})

	if err != nil{
		logger.WithError(err).Error("Unable to send batch transactions!")
		return err
	}

	logger.WithFields(logger.Fields{
		"num_of_txs" : len(txs),
	}).Info("Batch Transactions are sent!")

	return nil
}

func updateRecordAfterSend(txs []*corepb.Transaction){
	for _, tx := range txs{
		sentTxs[hex.EncodeToString(tx.GetId())] = 0
	}
}

func sendRandomTransactions(rpcClient rpcpb.RpcServiceClient, addresses []core.Address, cmdChan chan int){
	txs := []*corepb.Transaction{}
	isRunning := false

	count := 0

	ticker := time.NewTicker(time.Millisecond*TimeBetweenBatch).C
	wm, err := logic.GetWalletManager(client.GetWalletFilePath())
	if err!=nil {
		logger.WithError(err).Panic("Unable to get wallet")
	}
	for{
		select{
		case cmd := <- cmdChan:
			switch(cmd){
			case cmdStart:
				isRunning = true
			case cmdStop:
				isRunning = false
				txs = []*corepb.Transaction{}
			}
		case <-ticker:
			if !isRunning {
				continue
			}
			if len(txs) >= numOfTxPerBatch {
				logger.WithFields(logger.Fields{
					"invalidTxCount": count,
				}).Info("Send Batch Txs!")
				count = 0
				if sendBatchTransactions(rpcClient, txs) == nil{
					updateRecordAfterSend(txs)
					updateCurrBal()
				}else{
					//if the send failed, the current utxo is not up to date
					updateUtxoIndex(rpcClient, addresses)
					//isRunning = false
				}
				txs = []*corepb.Transaction{}
				tempBalance = getCurrBalanceDeepCopy(currBalance)
			}
		default:
			if !isRunning {
				continue
			}
			if len(txs) >= numOfTxPerBatch {
				continue
			}
			tx := createRandomTransaction(addresses, wm)
			if tx!=nil{
				txs = append(txs, tx)
			}else{
				count++
			}
		}
	}
}

func updateCurrBal(){
	currBalance = make(map[string]uint64)
	currBalance = getCurrBalanceDeepCopy(tempBalance)
}

func getCurrBalanceDeepCopy(original map[string]uint64 ) map[string]uint64{
	temp := make(map[string]uint64)
	for key,val := range original{
		temp[key] = val
	}
	return temp
}

func createRandomTransaction(addresses []core.Address, wm *client.WalletManager) *corepb.Transaction{

	data := ""

	fromIndex := getAddrWithBalance(addresses)
	toIndex := rand.Intn(maxWallet)
	for toIndex == fromIndex {
		toIndex = rand.Intn(maxWallet)
	}
	toAddr := addresses[toIndex].String()
	sendAmount := calcSendAmount(addresses[fromIndex].String(), addresses[toIndex].String())

	if IsTheTurnToSendSmartContractTransaction() && isContractDeployed {
		data = contractFunctionCall
		toAddr = smartContractAddr
	}

	senderKeyPair := wm.GetKeyPairByAddress(addresses[fromIndex])
	tx := createTransaction(addresses[fromIndex], core.NewAddress(toAddr),common.NewAmount(sendAmount), common.NewAmount(0), data ,senderKeyPair)
	if tx==nil {
		return nil
	}

	tempBalance[addresses[fromIndex].String()] -= sendAmount
	tempBalance[core.NewAddress(toAddr).String()] += sendAmount

	return tx.ToProto().(*corepb.Transaction)
}

func createTransaction(from, to core.Address, amount, tip *common.Amount, contract string, senderKeyPair *core.KeyPair) *core.Transaction{

	pkh, err := core.NewUserPubKeyHash(senderKeyPair.PublicKey)
	if err != nil {
		logger.WithError(err).Panic("Unable to hash sender public key")
	}
	prevUtxos, err := utxoIndex.GetUTXOsByAmount(pkh, amount)

	if err!=nil {
		//logger.WithError(err).WithFields(logger.Fields{
		//	"pkh" : hex.EncodeToString(pkh),
		//	"addr": pkh.GenerateAddress().String(),
		//	"amount": amount.String(),
		//}).Warn("Unable to get previous utxos")
		return nil
	}
	tx, err := core.NewUTXOTransaction(prevUtxos, from, to, amount, senderKeyPair, tip, contract)

	sendTXLogger := logger.WithFields(logger.Fields{
		"from":             from.String(),
		"to":               to.String(),
		"amount":           amount.String(),
		"sender_balance":   currBalance[from.String()],
		"receiver_balance": currBalance[to.String()],
		"txid":             "",
		"data":             contract,
	})

	if err != nil {
		sendTXLogger.WithError(err).Error("Failed to send transaction!")
		return nil
	}
	sendTXLogger.Data["txid"] = hex.EncodeToString(tx.ID)

	//sendTXLogger.Info("Transaction is created!")

	utxoIndex.UpdateUtxo(&tx)
	return &tx
}

func IsTheTurnToSendSmartContractTransaction() bool {
	smartContractCounter += 1
	return smartContractCounter%smartContractSendFreq == 0
}

func calcSendAmount(from, to string) uint64 {
	fromBalance, _ := tempBalance[from]
	toBalance, _ := tempBalance[to]
	amount := uint64(0)
	if fromBalance < toBalance {
		amount = 1
	} else if fromBalance == toBalance {
		amount = fromBalance - 1
	} else {
		amount = (fromBalance - toBalance) / 3
	}

	if amount == 0 {
		amount = 1
	}
	return amount
}

func getAddrWithBalance(addresses []core.Address) int {
	fromIndex := rand.Intn(maxWallet)
	amount := tempBalance[addresses[fromIndex].String()]
	//TODO: add time out to this loop
	for amount <= maxDefaultSendAmount+1 {
		fromIndex = rand.Intn(maxWallet)
		amount = tempBalance[addresses[fromIndex].String()]
	}
	return fromIndex
}

func sendTransaction(adminClient rpcpb.AdminServiceClient, from, to string, amount uint64, data string) (*rpcpb.SendResponse, error) {

	resp, err := adminClient.RpcSend(context.Background(), &rpcpb.SendRequest{
		From:       from,
		To:         to,
		Amount:     common.NewAmount(amount).Bytes(),
		Tip:        common.NewAmount(0).Bytes(),
		WalletPath: client.GetWalletFilePath(),
		Data:       data,
	})

	if err != nil {
		recordFailedTransaction(err, from, to, amount, data)
		return resp, err
	}
	sentTxs[resp.Txid] = 0
	currBalance[from] -= amount
	currBalance[to] += amount
	return resp, nil
}

func recordFailedTransaction(txErr error, from, to string, amount uint64, data string) {
	f, err := os.OpenFile(failedTxLogFilePath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		logger.Panic("Open file failed while recording failed transactions")
	}
	w := csv.NewWriter(f)

	w.Write([]string{fmt.Sprint(currHeight), from, to, fmt.Sprint(amount), txErr.Error(), data})

	w.Flush()
}

func displayBalances(rpcClient rpcpb.RpcServiceClient, addresses []core.Address, update bool) {
	for _, addr := range addresses {
		amount, err := getBalance(rpcClient, addr.String())
		balanceLogger := logger.WithFields(logger.Fields{
			"address": addr.String(),
			"amount":  amount,
			"record":  currBalance[addr.String()],
		})
		if err != nil {
			balanceLogger.WithError(err).Warn("Failed to get wallet balance.")
		}
		balanceLogger.Info("Displaying wallet balance...")
		if update {
			currBalance[addr.String()] = uint64(amount)
			tempBalance[addr.String()] = uint64(amount)
		}
	}
}

func getBalance(rpcClient rpcpb.RpcServiceClient, address string) (int64, error) {
	response, err := rpcClient.RpcGetBalance(context.Background(), &rpcpb.GetBalanceRequest{Address: address})
	return response.Amount, err
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
