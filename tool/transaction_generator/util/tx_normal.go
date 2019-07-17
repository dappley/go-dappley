package util

import (
	"github.com/dappley/go-dappley/core"
	corepb "github.com/dappley/go-dappley/core/pb"
	"github.com/dappley/go-dappley/sdk"
	logger "github.com/sirupsen/logrus"
)

type NormalTxSender struct {
	TxSender
}

func NewNormalTransaction(dappSdk *sdk.DappSdk, account *sdk.DappSdkAccount) *NormalTxSender {
	return &NormalTxSender{
		TxSender{
			dappSdk: dappSdk,
			account: account,
		},
	}
}

func (txSender *NormalTxSender) Generate(params core.SendTxParam) {
	pkh, err := core.NewUserPubKeyHash(params.SenderKeyPair.PublicKey)

	if err != nil {
		logger.WithError(err).Panic("NormalTx: Unable to hash sender public key")
	}

	prevUtxos, err := txSender.account.GetUtxoIndex().GetUTXOsByAmount(pkh, params.Amount)

	if err != nil {
		logger.WithError(err).Panic("NormalTx: Unable to get UTXOs to match the amount")
	}

	vouts := prepareOutputLists(prevUtxos, params.From, params.To, params.Amount, params.Tip)
	txSender.tx = NewTransaction(prevUtxos, vouts, params.Tip, params.SenderKeyPair)
}

func (txSender *NormalTxSender) Send() {

	_, err := txSender.dappSdk.SendTransaction(txSender.tx.ToProto().(*corepb.Transaction))

	if err != nil {
		logger.WithError(err).Panic("NormalTx: Unable to send transaction!")
	}

	logger.Info("Sending is successful!")
}

func (txSender *NormalTxSender) Print() {
	logger.Info("Sending a normal transaction ...")
}
