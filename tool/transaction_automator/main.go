package main

import (
	"github.com/dappley/go-dappley/sdk"
	"github.com/dappley/go-dappley/tool/transaction_automator/pb"
	"github.com/dappley/go-dappley/tool/transaction_automator/util"
	"io/ioutil"
	_ "net/http/pprof"
	"os"
	"time"

	"github.com/dappley/go-dappley/common"
	"github.com/dappley/go-dappley/config"
	logger "github.com/sirupsen/logrus"
)

const (
	contractAddrFilePath = "contract/contractAddr"
	contractFilePath     = "contract/test_contract.js"
	configFilePath       = "default.conf"
)

func main() {
	logger.SetFormatter(&logger.TextFormatter{
		FullTimestamp: true,
	})

	toolConfigs := &tx_automator_configpb.Config{}
	config.LoadConfig(configFilePath, toolConfigs)

	grpcClient := sdk.NewDappSdkGrpcClient(toolConfigs.GetPort())
	dappSdk := sdk.NewDappSdk(grpcClient)
	wallet := sdk.NewDappleySdkWallet(
		toolConfigs.GetMaxWallet(),
		toolConfigs.GetPassword(),
		dappSdk,
	)

	fundAddr := wallet.GetAddrs()[0].String()
	fundRequest := sdk.NewDappSdkFundRequest(grpcClient, dappSdk)
	initialAmount := toolConfigs.GetInitialAmount()
	fundRequest.Fund(fundAddr, common.NewAmount(initialAmount))

	logger.WithFields(logger.Fields{
		"initial_total_amount": initialAmount,
	}).Info("Funding is completed. Script starts.")

	wallet.UpdateFromServer()
	wallet.DisplayBalances()

	isScDeployed, scAddr := deploySmartContract(dappSdk, fundAddr)

	currHeight, err := dappSdk.GetBlockHeight()
	if err != nil {
		logger.WithError(err).Panic("Get Blockheight failed!")
	}

	ticker := time.NewTicker(time.Millisecond * 200).C

	sender := util.NewBatchTxSender(toolConfigs.GetTps(), wallet, dappSdk, toolConfigs.GetScFreq(), scAddr)
	if isScDeployed {
		sender.EnableSmartContract()
	}
	sender.Run()
	sender.Start()

	for {
		select {
		case <-ticker:
			height, err := dappSdk.GetBlockHeight()
			if err != nil {
				logger.WithError(err).Panic("Get Blockheight failed!")
			}

			if height > currHeight {
				sender.Stop()

				sender.EnableSmartContract()
				currHeight = height

				wallet.UpdateFromServer()
				wallet.DisplayBalances()

			} else {
				sender.Start()
			}
		}
	}
}

func deploySmartContract(dappSdk *sdk.DappSdk, from string) (bool, string) {

	smartContractAddr := getSmartContractAddr()
	if smartContractAddr != "" {
		logger.WithFields(logger.Fields{
			"contractAddr": smartContractAddr,
		}).Info("Smart contract has already been deployed. If you are sure it is not deployed, empty the file:", contractAddrFilePath)
		return true, smartContractAddr
	}

	data, err := ioutil.ReadFile(contractFilePath)
	if err != nil {
		logger.WithError(err).WithFields(logger.Fields{
			"file_path": contractFilePath,
		}).Panic("Unable to read smart contract file!")
	}

	contract := string(data)
	resp, err := dappSdk.SendTransaction(from, "", 1, contract)
	if err != nil {
		logger.WithError(err).WithFields(logger.Fields{
			"file_path":     contractFilePath,
			"contract_addr": smartContractAddr,
		}).Panic("Deploy smart contract failed!")
	}
	smartContractAddr = resp.ContractAddress

	recordSmartContractAddr(smartContractAddr)

	logger.WithFields(logger.Fields{
		"contract_addr": smartContractAddr,
	}).Info("Smart contract has been deployed")

	return false, smartContractAddr
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
