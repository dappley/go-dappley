package util

import (
	"github.com/dappley/go-dappley/core"
	"github.com/dappley/go-dappley/core/pb"
	"github.com/dappley/go-dappley/sdk"
	logger "github.com/sirupsen/logrus"
)

type DoubleSpendingTxSender struct {
	TxSender
}

func NewDoubleSpendingTxSender(dappSdk *sdk.DappSdk, wallet *sdk.DappSdkWallet) *DoubleSpendingTxSender {
	return &DoubleSpendingTxSender{
		TxSender{
			dappSdk: dappSdk,
			wallet:  wallet,
		},
	}
}

func (txSender *DoubleSpendingTxSender) Generate(params core.SendTxParam) {
	pkh, err := core.NewUserPubKeyHash(params.SenderKeyPair.PublicKey)

	if err != nil {
		logger.WithError(err).Panic("DoubleSpendingTx: Unable to hash sender public key")
	}

	prevUtxos, err := txSender.wallet.GetUtxoIndex().GetUTXOsByAmount(pkh, params.Amount)

	if err != nil {
		logger.WithError(err).Panic("DoubleSpendingTx: Unable to get UTXOs to match the amount")
	}

	vouts := prepareOutputLists(prevUtxos, params.From, params.To, params.Amount, params.Tip)
	txSender.tx = NewTransaction(prevUtxos, vouts, params.Tip, params.SenderKeyPair)
}

func (txSender *DoubleSpendingTxSender) Send() {

	_, err := txSender.dappSdk.SendTransaction(txSender.tx.ToProto().(*corepb.Transaction))

	if err == nil {
		logger.WithError(err).Info("DoubleSpendingTx: Sending transaction 1 succeeded")
	} else {
		logger.WithError(err).Panic("DoubleSpendingTx: Sending transaction 1 failed!")
	}

	_, err = txSender.dappSdk.SendTransaction(txSender.tx.ToProto().(*corepb.Transaction))

	if err == nil {
		logger.WithError(err).Info("DoubleSpendingTx: Sending transaction 2 succeeded")
	} else {
		logger.WithError(err).Error("DoubleSpendingTx: Sending transaction 2 failed!")
	}

}

func (txSender *DoubleSpendingTxSender) Print() {
	logger.Info("Sending double spending transactions")
}
