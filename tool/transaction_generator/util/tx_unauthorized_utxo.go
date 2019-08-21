package util

import (
	"github.com/dappley/go-dappley/core"
	"github.com/dappley/go-dappley/core/account"
	corepb "github.com/dappley/go-dappley/core/pb"
	"github.com/dappley/go-dappley/sdk"
	logger "github.com/sirupsen/logrus"
)

type UnauthorizedUtxoTxSender struct {
	TxSender
	unauthorizedAddrPkh account.PubKeyHash
}

func NewUnauthorizedUtxoTxSender(dappSdk *sdk.DappSdk, acc *sdk.DappSdkAccount, unauthorizedAddr account.Address) *UnauthorizedUtxoTxSender {
	unauthorizedpkh, _ := account.NewUserPubKeyHash(acc.GetAccountManager().GetKeyPairByAddress(unauthorizedAddr).GetPublicKey())

	return &UnauthorizedUtxoTxSender{
		TxSender{
			dappSdk: dappSdk,
			account: acc,
		},
		unauthorizedpkh,
	}
}

func (txSender *UnauthorizedUtxoTxSender) Generate(params core.SendTxParam) {
	pkh, err := account.NewUserPubKeyHash(params.SenderKeyPair.GetPublicKey())

	if err != nil {
		logger.WithError(err).Panic("UnauthorizedUtxoTx: Unable to hash sender public key")
	}

	prevUtxos, err := txSender.account.GetUtxoIndex().GetUTXOsByAmount(pkh, params.Amount)

	if err != nil {
		logger.WithError(err).Panic("UnauthorizedUtxoTx: Unable to get UTXOs to match the amount")
	}

	unauthorizedUtxo := txSender.account.GetUtxoIndex().GetAllUTXOsByPubKeyHash(txSender.unauthorizedAddrPkh).GetAllUtxos()
	prevUtxos = append(prevUtxos, unauthorizedUtxo[0])

	vouts := prepareOutputLists(prevUtxos, params.From, params.To, params.Amount, params.Tip)
	txSender.tx = NewTransaction(prevUtxos, vouts, params.Tip, params.SenderKeyPair)
}

func (txSender *UnauthorizedUtxoTxSender) Send() {

	_, err := txSender.dappSdk.SendTransaction(txSender.tx.ToProto().(*corepb.Transaction))

	if err != nil {
		logger.WithError(err).Error("UnauthorizedUtxoTx: Sending transaction failed!")
	}
}

func (txSender *UnauthorizedUtxoTxSender) Print() {
	logger.Info("Sending a transaction with an unauthrized utxo...")
}
