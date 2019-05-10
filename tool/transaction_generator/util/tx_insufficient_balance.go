package util

import (
	"github.com/dappley/go-dappley/common"
	"github.com/dappley/go-dappley/core"
	"github.com/dappley/go-dappley/core/pb"
	"github.com/dappley/go-dappley/sdk"
	logger "github.com/sirupsen/logrus"
)

type InsufficientBalanceTxSender struct {
	TxSender
}

func NewInsufficientBalanceTxSender(dappSdk *sdk.DappSdk, wallet *sdk.DappSdkWallet) *InsufficientBalanceTxSender {
	return &InsufficientBalanceTxSender{
		TxSender{
			dappSdk: dappSdk,
			wallet:  wallet,
		},
	}
}

func (txSender *InsufficientBalanceTxSender) Generate(params core.SendTxParam) {
	pkh, err := core.NewUserPubKeyHash(params.SenderKeyPair.PublicKey)

	if err != nil {
		logger.WithError(err).Panic("InsufficientBalanceTx: Unable to hash sender public key")
	}

	prevUtxos, err := txSender.wallet.GetUtxoIndex().GetUTXOsByAmount(pkh, params.Amount)

	if err != nil {
		logger.WithError(err).Panic("InsufficientBalanceTx: Unable to get UTXOs to match the amount")
	}

	vouts := prepareOutputLists(prevUtxos, params.From, params.To, params.Amount, params.Tip)
	vouts[0].Value = vouts[0].Value.Add(common.NewAmount(1))

	txSender.tx = NewTransaction(prevUtxos, vouts, params.Tip, params.SenderKeyPair)
}

func (txSender *InsufficientBalanceTxSender) Send() {

	_, err := txSender.dappSdk.SendTransaction(txSender.tx.ToProto().(*corepb.Transaction))

	if err != nil {
		logger.WithError(err).Error("InsufficientBalanceTx: Unable to send transaction!")
	}
}

func (txSender *InsufficientBalanceTxSender) Print() {
	logger.Info("Sending a transaction with insufficient balance...")
}
