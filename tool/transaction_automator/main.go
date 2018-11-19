package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/dappley/go-dappley/client"
	"github.com/dappley/go-dappley/common"
	"github.com/dappley/go-dappley/config"
	"github.com/dappley/go-dappley/config/pb"
	"github.com/dappley/go-dappley/core"
	"github.com/dappley/go-dappley/logic"
	"github.com/dappley/go-dappley/rpc/pb"
	logger "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"math/rand"
	"time"
)

var(
	password			= "testpassword"
	maxWallet 			= 10
	initialAmount 		= uint64(10)
	maxSendAmount 		= uint64(3)
	sendInternalInMs 	= time.Duration(1000)
)

func main() {
	logger.SetFormatter(&logger.TextFormatter{
		FullTimestamp: true,
	})

	var filePath string
	flag.StringVar(&filePath, "f", "default.conf", "CLI config file path")
	flag.Parse()

	cliConfig := &configpb.CliConfig{}
	config.LoadConfig(filePath, cliConfig)
	conn := initRpcClient(int(cliConfig.GetPort()))

	adminClient := rpcpb.NewAdminServiceClient(conn)
	rpcClient := rpcpb.NewRpcServiceClient(conn)

	addresses := createWallet()

	fundAllFromMiner(adminClient, addresses)
	displayBalances(rpcClient, addresses)

	ticker := time.NewTicker(time.Millisecond*sendInternalInMs).C
	getBalanceTicker := time.NewTicker(time.Minute).C
	for{
		select{
		case <- ticker:
			sendRandomTransactions(adminClient, rpcClient, addresses)
		case <- getBalanceTicker:
			displayBalances(rpcClient, addresses)
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

func createWallet() []core.Address{
	logger.Info("Creating Wallet...")
	wm, err := logic.GetWalletManager(client.GetWalletFilePath())
	if err!= nil{
		logger.Panic("Cant get wallet")
	}
	addresses := wm.GetAddresses()
	numOfWallets := len(addresses)
	if numOfWallets<10 {
		for i:=numOfWallets; i<maxWallet;i++ {
			_, err := logic.CreateWalletWithpassphrase(password)
			if err!= nil{
				logger.Panic("Please delete your original wallet file in go-dappley/bin")
			}
		}
	}
	for _, addr:=range wm.GetAddresses(){
		logger.WithFields(logger.Fields{
			"address"	: addr.String(),
		}).Info("Current Wallet Addresses")
	}
	return addresses
}

func fundAllFromMiner(adminClient rpcpb.AdminServiceClient, addresses []core.Address){
	for _, addr := range addresses{
		fundFromMiner(adminClient, addr.String())
	}
}

func fundFromMiner(adminClient rpcpb.AdminServiceClient, fundAddr string) {

	sendFromMinerRequest := rpcpb.SendFromMinerRequest{}
	sendFromMinerRequest.To = fundAddr
	sendFromMinerRequest.Amount = common.NewAmount(initialAmount).Bytes()

	_, err := adminClient.RpcSendFromMiner(context.Background(), &sendFromMinerRequest)
	if err!= nil {
		logger.WithFields(logger.Fields{
			"error": err,
		}).Panic("Insufficient test fund from miner")
	}
}

func sendRandomTransactions(adminClient rpcpb.AdminServiceClient, rpcClient rpcpb.RpcServiceClient, addresses []core.Address){

	fromIndex := getAddrWithBalance(rpcClient, addresses)
	toIndex := rand.Intn(maxWallet)
	for toIndex == fromIndex{
		toIndex = rand.Intn(maxWallet)
	}
	sendAmount := calcSendAmount(rpcClient, addresses[fromIndex].String(), addresses[toIndex].String())
	err :=  sendTransaction(adminClient, addresses[fromIndex].String(), addresses[toIndex].String(), sendAmount)
	if err!=nil {
		logger.WithFields(logger.Fields{
			"From"	: addresses[fromIndex].String(),
			"to"	: addresses[toIndex].String(),
			"amount": sendAmount,
			"error" : err,
		}).Warn("send transaction failed!")
		return
	}
	logger.WithFields(logger.Fields{
		"From"	: addresses[fromIndex].String(),
		"to"	: addresses[toIndex].String(),
		"amount": sendAmount,
		"error" : err,
	}).Info("Transaction Sent!")
}

func calcSendAmount(rpcClient rpcpb.RpcServiceClient, from, to string) uint64{
	fromBalance, _  := getBalance(rpcClient, from)
	toBalance, _  := getBalance(rpcClient, to)
	difference := fromBalance-toBalance
	amount := uint64(0)
	if difference < 0 {
		amount = 1
	}else if difference==0 {
		amount = uint64(fromBalance - 1)
	}else{
		amount = uint64(difference/3)
	}

	if amount == 0 {
		amount = rand.Uint64()%maxSendAmount + 1
	}
	return amount
}

func getAddrWithBalance(rpcClient rpcpb.RpcServiceClient, addresses []core.Address) int{
	fromIndex := rand.Intn(maxWallet)
	amount, err := getBalance(rpcClient, addresses[fromIndex].String())
	//TODO: add time out to this loop
	for  err!=nil || amount <= int64(maxSendAmount+1) {
		fromIndex = rand.Intn(maxWallet)
		amount, err = getBalance(rpcClient, addresses[fromIndex].String())
	}
	return fromIndex
}

func sendTransaction(adminClient rpcpb.AdminServiceClient, from, to string, amount uint64) error{
	_, err := adminClient.RpcSend(context.Background(), &rpcpb.SendRequest{
		From:       from,
		To:         to,
		Amount:     common.NewAmount(amount).Bytes(),
		Tip:        0,
		Walletpath: client.GetWalletFilePath(),
		Contract:   "",
	})
	return err
}

func displayBalances(rpcClient rpcpb.RpcServiceClient, addresses []core.Address) {
	for _, addr := range addresses{
		amount, err := getBalance(rpcClient, addr.String())
		if err!=nil{
			logger.WithFields(logger.Fields{
				"address" 	: addr.String(),
				"amount"	: amount,
				"err"		: err,
			}).Warn("Get wallet balance failed")
		}
		logger.WithFields(logger.Fields{
			"address" 	: addr.String(),
			"amount"	: amount,
		}).Info("wallet balance")
	}
}

func getBalance(rpcClient rpcpb.RpcServiceClient, address string) (int64, error){
	getBalanceRequest := rpcpb.GetBalanceRequest{}
	getBalanceRequest.Name = "getBalance"
	getBalanceRequest.Address = address
	response, err := rpcClient.RpcGetBalance(context.Background(), &getBalanceRequest)
	return response.Amount, err
}