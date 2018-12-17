package main

import (
	"context"
	"flag"
	"fmt"
	"math/rand"
	"time"

	"github.com/dappley/go-dappley/client"
	"github.com/dappley/go-dappley/common"
	"github.com/dappley/go-dappley/config"
	"github.com/dappley/go-dappley/config/pb"
	"github.com/dappley/go-dappley/core"
	"github.com/dappley/go-dappley/logic"
	"github.com/dappley/go-dappley/rpc/pb"
	logger "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

var (
	password             = "testpassword"
	maxWallet            = 10
	initialAmount        = uint64(10)
	maxDefaultSendAmount = uint64(5)
	sendInterval         = time.Duration(1000) //ms
	checkBalanceInterval = time.Duration(10)   //s
	fundTimeout          = time.Duration(time.Minute * 5)
	currBalance          = make(map[string]uint64)
)

func main() {
	logger.SetFormatter(&logger.TextFormatter{
		FullTimestamp: true,
	})

	var filePath string
	flag.StringVar(&filePath, "f", "conf/default_cli.conf", "CLI config file path")
	flag.Parse()

	cliConfig := &configpb.CliConfig{}
	config.LoadConfig(filePath, cliConfig)
	conn := initRpcClient(int(cliConfig.GetPort()))

	adminClient := rpcpb.NewAdminServiceClient(conn)
	rpcClient := rpcpb.NewRpcServiceClient(conn)

	addresses := createWallet()

	fundFromMiner(adminClient, rpcClient, addresses)
	logger.WithFields(logger.Fields{
		"initial total amount": initialAmount,
		"send interval(ms)":    sendInterval,
	}).Info("Funding completed. Script Starts.")
	displayBalances(rpcClient, addresses)

	ticker := time.NewTicker(time.Millisecond * sendInterval).C
	currHeight := getBlockHeight(rpcClient)
	for {
		select {
		case <-ticker:
			height := getBlockHeight(rpcClient)
			if height > currHeight {
				displayBalances(rpcClient, addresses)
				currHeight = height
			} else {
				sendRandomTransactions(adminClient, addresses)
			}
		}
	}
}

func initRpcClient(port int) *grpc.ClientConn {
	//prepare grpc client
	var conn *grpc.ClientConn
	conn, err := grpc.Dial(fmt.Sprint(":", port), grpc.WithInsecure())
	if err != nil {
		logger.Panic("ERROR: Not able to connect to RPC server. ERR:", err)
	}
	return conn
}

func createWallet() []core.Address {
	logger.Info("Creating Wallet...")
	wm, err := logic.GetWalletManager(client.GetWalletFilePath())
	if err != nil {
		logger.Panic("Cant get wallet")
	}
	addresses := wm.GetAddresses()
	numOfWallets := len(addresses)
	if numOfWallets < 10 {
		for i := numOfWallets; i < maxWallet; i++ {
			_, err := logic.CreateWalletWithpassphrase(password)
			if err != nil {
				logger.Panic("Please delete your original wallet file in go-dappley/bin")
			}
		}
	}
	wm, err = logic.GetWalletManager(client.GetWalletFilePath())
	addresses = wm.GetAddresses()
	for _, addr := range addresses {
		logger.WithFields(logger.Fields{
			"address": addr.String(),
		}).Info("Current Wallet Addresses")
	}
	return addresses
}

func fundFromMiner(adminClient rpcpb.AdminServiceClient, rpcClient rpcpb.RpcServiceClient, addresses []core.Address) {
	logger.Info("Requesting fund from miner...")

	fundAddr := addresses[0].String()
	if len(addresses) == 0 {
		logger.Panic("Wallet is not created! Can not get fund from miner!")
	}

	requestFundFromMiner(adminClient, fundAddr)
	bal, isSufficient := checkSufficientInitialAmount(rpcClient, fundAddr)
	if isSufficient {
		//continue if the initial amount is sufficient
		return
	}
	logger.WithFields(logger.Fields{
		"address":    fundAddr,
		"balance":    bal,
		"targetFund": initialAmount,
	}).Info("Current wallet balance is insufficient. Waiting for more funds...")
	waitTilInitialAmountIsSufficient(adminClient, rpcClient, fundAddr)
}

func checkSufficientInitialAmount(rpcClient rpcpb.RpcServiceClient, addr string) (uint64, bool) {
	balance, err := getBalance(rpcClient, addr)
	if err != nil {
		logger.WithFields(logger.Fields{
			"err": err,
		}).Panic("Not able to get balance")
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
				"address":    addr,
				"balance":    bal,
				"targetFund": initialAmount,
			}).Info("Current wallet balance is insufficient. Waiting for more funds...")
			requestFundFromMiner(adminClient, addr)
		case <-timeout:
			logger.WithFields(logger.Fields{
				"targetFund": initialAmount,
			}).Panic("Time out. Not able to get sufficient fund from miner!")
		}
	}
}

func requestFundFromMiner(adminClient rpcpb.AdminServiceClient, fundAddr string) {

	sendFromMinerRequest := rpcpb.SendFromMinerRequest{}
	sendFromMinerRequest.To = fundAddr
	sendFromMinerRequest.Amount = common.NewAmount(initialAmount).Bytes()

	_, err := adminClient.RpcSendFromMiner(context.Background(), &sendFromMinerRequest)
	if err != nil {
		logger.WithFields(logger.Fields{
			"error": err,
		}).Panic("Unable to get test fund from miner")
	}
}

func sendRandomTransactions(adminClient rpcpb.AdminServiceClient, addresses []core.Address) {

	fromIndex := getAddrWithBalance(addresses)
	toIndex := rand.Intn(maxWallet)
	for toIndex == fromIndex {
		toIndex = rand.Intn(maxWallet)
	}
	sendAmount := calcSendAmount(addresses[fromIndex].String(), addresses[toIndex].String())
	err := sendTransaction(adminClient, addresses[fromIndex].String(), addresses[toIndex].String(), sendAmount)
	if err != nil {
		logger.WithFields(logger.Fields{
			"From":    addresses[fromIndex].String(),
			"to":      addresses[toIndex].String(),
			"amount":  sendAmount,
			"error":   err,
			"fromBal": currBalance[addresses[fromIndex].String()],
			"toBal":   currBalance[addresses[toIndex].String()],
		}).Panic("send transaction failed!")
		return
	}
	logger.WithFields(logger.Fields{
		"From":    addresses[fromIndex].String(),
		"to":      addresses[toIndex].String(),
		"amount":  sendAmount,
		"error":   err,
		"fromBal": currBalance[addresses[fromIndex].String()],
		"toBal":   currBalance[addresses[toIndex].String()],
	}).Info("Transaction Sent!")
}

func calcSendAmount(from, to string) uint64 {
	fromBalance, _ := currBalance[from]
	toBalance, _ := currBalance[to]
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
	amount := currBalance[addresses[fromIndex].String()]
	//TODO: add time out to this loop
	for amount <= maxDefaultSendAmount+1 {
		fromIndex = rand.Intn(maxWallet)
		amount = currBalance[addresses[fromIndex].String()]
	}
	return fromIndex
}

func sendTransaction(adminClient rpcpb.AdminServiceClient, from, to string, amount uint64) error {
	_, err := adminClient.RpcSend(context.Background(), &rpcpb.SendRequest{
		From:       from,
		To:         to,
		Amount:     common.NewAmount(amount).Bytes(),
		Tip:        0,
		Walletpath: client.GetWalletFilePath(),
		Data:       "",
	})
	if err != nil {
		return err
	}
	currBalance[from] -= amount
	currBalance[to] += amount
	return nil
}

func displayBalances(rpcClient rpcpb.RpcServiceClient, addresses []core.Address) {
	for _, addr := range addresses {
		amount, err := getBalance(rpcClient, addr.String())
		if err != nil {
			logger.WithFields(logger.Fields{
				"address": addr.String(),
				"amount":  amount,
				"err":     err,
			}).Warn("Get wallet balance failed")
		}
		logger.WithFields(logger.Fields{
			"address": addr.String(),
			"amount":  amount,
			"record":  currBalance[addr.String()],
		}).Info("wallet balance")
		currBalance[addr.String()] = uint64(amount)
	}
}

func getBalance(rpcClient rpcpb.RpcServiceClient, address string) (int64, error) {
	getBalanceRequest := rpcpb.GetBalanceRequest{}
	getBalanceRequest.Name = "getBalance"
	getBalanceRequest.Address = address
	response, err := rpcClient.RpcGetBalance(context.Background(), &getBalanceRequest)
	return response.Amount, err
}

func getBlockHeight(rpcClient rpcpb.RpcServiceClient) uint64 {
	resp, err := rpcClient.RpcGetBlockchainInfo(
		context.Background(),
		&rpcpb.GetBlockchainInfoRequest{})
	if err != nil {
		logger.Panic("Can not get block height. err:", err)
	}
	return resp.BlockHeight
}

func isBalanceSufficient(addr string, amount uint64) bool {
	return currBalance[addr] >= amount
}
