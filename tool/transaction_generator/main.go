package main

import (
	"github.com/dappley/go-dappley/common"
	"github.com/dappley/go-dappley/config"
	"github.com/dappley/go-dappley/core/account"
	"github.com/dappley/go-dappley/core/transaction"
	"github.com/dappley/go-dappley/sdk"
	"github.com/dappley/go-dappley/tool/tool_util"
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

	dappSdk, account := initial_setup()

	testTransactions := prepareTestTransactions(dappSdk, account)

	sendTestTransactions(dappSdk, account, testTransactions)

	logger.Info("**************************************")
	logger.Info("**All transactions are sent. Exiting**")
	logger.Info("**************************************")

}

func initial_setup() (*sdk.DappSdk, *sdk.DappSdkAccount) {
	toolConfigs := &tx_automator_configpb.Config{}
	config.LoadConfig(configFilePath, toolConfigs)

	grpcClient := sdk.NewDappSdkGrpcClient(toolConfigs.GetPort())
	dappSdk := sdk.NewDappSdk(grpcClient)
	account := sdk.NewDappSdkAccount(
		toolConfigs.GetMaxAccount(),
		toolConfigs.GetPassword(),
		dappSdk,
	)

	addrs := account.GetAddrs()
	fromAddr := addrs[0]
	unauthorizedAddr := addrs[2]

	fundRequest := tool.NewFundRequest(dappSdk)
	initialAmount := common.NewAmount(toolConfigs.GetInitialAmount())
	fundRequest.Fund(fromAddr.String(), initialAmount)
	fundRequest.Fund(unauthorizedAddr.String(), initialAmount)

	return dappSdk, account
}

func getUnauthroizedAddr(account *sdk.DappSdkAccount) account.Address {
	return account.GetAddrs()[2]
}

func prepareTestTransactions(dappSdk *sdk.DappSdk, account *sdk.DappSdkAccount) []util.TestTransaction {
	return []util.TestTransaction{
		util.NewNormalTransaction(dappSdk, account),
		util.NewUnexistingUtxoTxSender(dappSdk, account),
		util.NewInsufficientBalanceTxSender(dappSdk, account),
		util.NewDoubleSpendingTxSender(dappSdk, account),
		util.NewUnauthorizedUtxoTxSender(dappSdk, account, getUnauthroizedAddr(account)),
	}
}

func prepareSendParameters(account *sdk.DappSdkAccount) transaction.SendTxParam {

	fromAddr := account.GetAddrs()[0]
	toAddr := account.GetAddrs()[1]
	return transaction.SendTxParam{
		fromAddr,
		account.GetAccountManager().GetKeyPairByAddress(fromAddr),
		toAddr,
		common.NewAmount(10),
		common.NewAmount(0),
		common.NewAmount(0),
		common.NewAmount(0),
		"",
	}
}

func sendTestTransactions(dappSdk *sdk.DappSdk, account *sdk.DappSdkAccount, testTransactions []util.TestTransaction) {
	nextBlockTicker := tool.NewNextBlockTicker(dappSdk)
	nextBlockTicker.Run()

	for i, testTx := range testTransactions {
		logger.Info("Waiting for next block...")
		<-nextBlockTicker.GetTickerChan()
		logger.Info("")
		logger.Info("Running test #", i)
		testTx.Print()
		account.Update()
		testTx.Generate(prepareSendParameters(account))
		testTx.Send()
	}
}
