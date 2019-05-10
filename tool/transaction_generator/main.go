package main

import (
	"github.com/dappley/go-dappley/common"
	"github.com/dappley/go-dappley/config"
	"github.com/dappley/go-dappley/core"
	"github.com/dappley/go-dappley/sdk"
	"github.com/dappley/go-dappley/tool/transaction_automator/pb"
	"github.com/dappley/go-dappley/tool/transaction_generator/util"
	logger "github.com/sirupsen/logrus"
)

const configFilePath = "default.conf"

func main() {
	logger.SetFormatter(&logger.TextFormatter{
		FullTimestamp: true,
	})

	logger.Info("*********************************************")
	logger.Info("**Invalid transaction generator tool starts**")
	logger.Info("*********************************************")

	dappSdk, wallet := initial_setup()

	testTransactions := prepareTestTransactions(dappSdk, wallet)

	sendTestTransactions(dappSdk, wallet, testTransactions)

	logger.Info("**************************************")
	logger.Info("**All transactions are sent. Exiting**")
	logger.Info("**************************************")

}

func initial_setup() (*sdk.DappSdk, *sdk.DappSdkWallet) {
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
	unauthorizedAddr := addrs[2]

	fundRequest := sdk.NewDappSdkFundRequest(grpcClient, dappSdk)
	initialAmount := common.NewAmount(toolConfigs.GetInitialAmount())
	fundRequest.Fund(fromAddr.String(), initialAmount)
	fundRequest.Fund(unauthorizedAddr.String(), initialAmount)

	return dappSdk, wallet
}

func getUnauthroizedAddr(wallet *sdk.DappSdkWallet) core.Address {
	return wallet.GetAddrs()[2]
}

func prepareTestTransactions(dappSdk *sdk.DappSdk, wallet *sdk.DappSdkWallet) []util.TestTransaction {
	return []util.TestTransaction{
		util.NewNormalTransaction(dappSdk, wallet),
		util.NewUnexistingUtxoTxSender(dappSdk, wallet),
		util.NewInsufficientBalanceTxSender(dappSdk, wallet),
		util.NewDoubleSpendingTxSender(dappSdk, wallet),
		util.NewUnauthorizedUtxoTxSender(dappSdk, wallet, getUnauthroizedAddr(wallet)),
	}
}

func prepareSendParameters(wallet *sdk.DappSdkWallet) core.SendTxParam {

	fromAddr := wallet.GetAddrs()[0]
	toAddr := wallet.GetAddrs()[1]
	return core.SendTxParam{
		fromAddr,
		wallet.GetWalletManager().GetKeyPairByAddress(fromAddr),
		toAddr,
		common.NewAmount(10),
		common.NewAmount(0),
		"",
	}
}

func sendTestTransactions(dappSdk *sdk.DappSdk, wallet *sdk.DappSdkWallet, testTransactions []util.TestTransaction) {
	nextBlockTicker := sdk.NewDappSdkNextBlockTicker(dappSdk)
	nextBlockTicker.Run()

	for i, testTx := range testTransactions {
		logger.Info("Waiting for next block...")
		<-nextBlockTicker.GetTickerChan()
		logger.Info("")
		logger.Info("Running test #", i)
		testTx.Print()
		wallet.UpdateFromServer()
		testTx.Generate(prepareSendParameters(wallet))
		testTx.Send()
	}
}
