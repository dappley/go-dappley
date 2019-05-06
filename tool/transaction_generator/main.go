package main

import (
	"github.com/dappley/go-dappley/common"
	"github.com/dappley/go-dappley/config"
	"github.com/dappley/go-dappley/core"
	"github.com/dappley/go-dappley/sdk"
	"github.com/dappley/go-dappley/tool/transaction_automator/pb"
	"github.com/dappley/go-dappley/tool/transaction_generator/util"
	logger "github.com/sirupsen/logrus"
	"time"
)

const configFilePath = "default.conf"

func main() {
	logger.SetFormatter(&logger.TextFormatter{
		FullTimestamp: true,
	})

	logger.Info("*******************************************")
	logger.Info("*Invalid transaction generator tool starts*")
	logger.Info("*******************************************")

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
	fromAddr := addrs[0]
	toAddr := addrs[1]
	unauthorizedAddr := addrs[2]

	fundRequest := sdk.NewDappSdkFundRequest(grpcClient, dappSdk)
	initialAmount := common.NewAmount(toolConfigs.GetInitialAmount())
	fundRequest.Fund(fromAddr.String(), initialAmount)
	fundRequest.Fund(unauthorizedAddr.String(), initialAmount)

	txSenders := []util.TestTransaction{
		util.NewNormalTransaction(dappSdk, wallet),
		util.NewUnexistingUtxoTxSender(dappSdk, wallet),
		util.NewInsufficientBalanceTxSender(dappSdk, wallet),
		util.NewDoubleSpendingTxSender(dappSdk, wallet),
		util.NewUnauthorizedUtxoTxSender(dappSdk, wallet, unauthorizedAddr),
	}

	params := core.SendTxParam{
		fromAddr,
		wallet.GetWalletManager().GetKeyPairByAddress(fromAddr),
		toAddr,
		common.NewAmount(10),
		common.NewAmount(0),
		"",
	}

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
				logger.Info("Running test case #", index)
				txSenders[index].Generate(params)
				txSenders[index].Send()
				index = index + 1
				logger.Info("Waiting for next block...")
				if index >= len(txSenders) {
					logger.Info("All transactions are sent. Exiting...")
					return
				}
			}
		}
	}
}
