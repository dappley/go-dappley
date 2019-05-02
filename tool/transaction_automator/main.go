package main

import (
	"context"
	"fmt"
	"github.com/dappley/go-dappley/sdk"
	"github.com/dappley/go-dappley/tool/transaction_automator/pb"
	"github.com/dappley/go-dappley/tool/transaction_automator/util"
	"io/ioutil"
	_ "net/http/pprof"
	"os"
	"time"

	"github.com/dappley/go-dappley/client"
	"github.com/dappley/go-dappley/common"
	"github.com/dappley/go-dappley/config"
	"github.com/dappley/go-dappley/rpc/pb"
	logger "github.com/sirupsen/logrus"
)

var (
	currBalance           = make(map[string]uint64)
	smartContractAddr     = ""
	numOfTxPerBatch       = uint32(1000)
	smartContractSendFreq = uint32(1000000000)
)

const (
	contractAddrFilePath = "contract/contractAddr"
	contractFilePath     = "contract/test_contract.js"
	password             = "testpassword"
	maxWallet            = 10
	initialAmount        = uint64(1000)
	sendInterval         = time.Duration(5000) //ms
	configFilePath       = "default.conf"
)

func main() {
	logger.SetFormatter(&logger.TextFormatter{
		FullTimestamp: true,
	})

	toolConfigs := &tx_automator_configpb.Config{}
	config.LoadConfig(configFilePath, toolConfigs)
	numOfTxPerBatch = toolConfigs.GetTps()
	smartContractSendFreq = toolConfigs.GetScFreq()

	conn := sdk.NewDappSdk(toolConfigs.GetPort())
	wallet := sdk.NewDappleySdkWallet(conn, maxWallet, password)
	blockchain := sdk.NewDappSdkBlockchain(conn)
	utxoIndex := sdk.NewDappSdkUtxoIndex(conn, wallet)

	fundAddr := wallet.GetAddrs()[0].String()
	fundRequest := sdk.NewDappSdkFundRequest(conn, blockchain)
	fundRequest.Fund(fundAddr, common.NewAmount(initialAmount))

	logger.WithFields(logger.Fields{
		"initial_total_amount": initialAmount,
		"send_interval":        fmt.Sprintf("%d ms", sendInterval),
	}).Info("Funding is completed. Script starts.")
	wallet.UpdateBalancesFromServer(blockchain)

	utxoIndex.Update()

	isDeployedAlready := deploySmartContract(conn.GetAdminClient(), fundAddr)

	currHeight, err := blockchain.GetBlockHeight()
	if err != nil {
		logger.WithError(err).Panic("Get Blockheight failed!")
	}

	ticker := time.NewTicker(time.Millisecond * 200).C

	sender := util.NewBatchTxSender(numOfTxPerBatch, wallet, utxoIndex, blockchain, smartContractSendFreq, smartContractAddr)
	if isDeployedAlready {
		sender.EnableSmartContract()
	}
	sender.Run()
	sender.Start()

	for {
		select {
		case <-ticker:
			height, err := blockchain.GetBlockHeight()
			if err != nil {
				logger.WithError(err).Panic("Get Blockheight failed!")
			}

			if height > currHeight {
				sender.Stop()

				sender.EnableSmartContract()
				currHeight = height

				wallet.UpdateBalancesFromServer(blockchain)
				utxoIndex.Update()

			} else {
				sender.Start()
			}
		}
	}
}

func deploySmartContract(serviceClient rpcpb.AdminServiceClient, from string) bool {

	smartContractAddr = getSmartContractAddr()
	if smartContractAddr != "" {
		logger.WithFields(logger.Fields{
			"contractAddr": smartContractAddr,
		}).Info("Smart contract has already been deployed. If you are sure it is not deployed, empty the file:", contractAddrFilePath)
		return true
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

	return false
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
		return resp, err
	}
	currBalance[from] -= amount
	currBalance[to] += amount
	return resp, nil
}
