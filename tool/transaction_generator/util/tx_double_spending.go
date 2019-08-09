package util

import (
	"github.com/dappley/go-dappley/core/account"
	"github.com/dappley/go-dappley/core/transaction"
	transactionpb "github.com/dappley/go-dappley/core/transaction/pb"
	"github.com/dappley/go-dappley/sdk"
	logger "github.com/sirupsen/logrus"
)

type DoubleSpendingTxSender struct {
	TxSender
}

func NewDoubleSpendingTxSender(dappSdk *sdk.DappSdk, account *sdk.DappSdkAccount) *DoubleSpendingTxSender {
	return &DoubleSpendingTxSender{
		TxSender{
			dappSdk: dappSdk,
			account: account,
		},
	}
}

func (txSender *DoubleSpendingTxSender) Generate(params transaction.SendTxParam) {
	pkh, err := account.NewUserPubKeyHash(params.SenderKeyPair.GetPublicKey())

	if err != nil {
		logger.WithError(err).Panic("DoubleSpendingTx: Unable to hash sender public key")
	}

	prevUtxos, err := txSender.account.GetUtxoIndex().GetUTXOsByAmount(pkh, params.Amount)

	if err != nil {
		logger.WithError(err).Panic("DoubleSpendingTx: Unable to get UTXOs to match the amount")
	}

	vouts := prepareOutputLists(prevUtxos, params.From, params.To, params.Amount, params.Tip)
	txSender.tx = NewTransaction(prevUtxos, vouts, params.Tip, params.SenderKeyPair)
}

func (txSender *DoubleSpendingTxSender) Send() {

	_, err := txSender.dappSdk.SendTransaction(txSender.tx.ToProto().(*transactionpb.Transaction))

	if err == nil {
		logger.WithError(err).Info("DoubleSpendingTx: Sending transaction 1 succeeded")
	} else {
		logger.WithError(err).Panic("DoubleSpendingTx: Sending transaction 1 failed!")
	}

	_, err = txSender.dappSdk.SendTransaction(txSender.tx.ToProto().(*transactionpb.Transaction))

	if err == nil {
		logger.WithError(err).Info("DoubleSpendingTx: Sending transaction 2 succeeded")
	} else {
		logger.WithError(err).Error("DoubleSpendingTx: Sending transaction 2 failed!")
	}

}

func (txSender *DoubleSpendingTxSender) Print() {
	logger.Info("Sending double spending transactions")
}
