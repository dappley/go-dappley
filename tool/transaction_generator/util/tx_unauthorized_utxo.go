package util

import (
	"github.com/dappley/go-dappley/core"
	"github.com/dappley/go-dappley/core/pb"
	"github.com/dappley/go-dappley/sdk"
	logger "github.com/sirupsen/logrus"
)

type UnauthorizedUtxoTxSender struct {
	TxSender
	unauthorizedAddrPkh core.PubKeyHash
}

func NewUnauthorizedUtxoTxSender(dappSdk *sdk.DappSdk, wallet *sdk.DappSdkWallet, unauthorizedAddr core.Address) *UnauthorizedUtxoTxSender {
	unauthorizedpkh, _ := core.NewUserPubKeyHash(wallet.GetWalletManager().GetKeyPairByAddress(unauthorizedAddr).PublicKey)

	return &UnauthorizedUtxoTxSender{
		TxSender{
			dappSdk: dappSdk,
			wallet:  wallet,
		},
		unauthorizedpkh,
	}
}

func (txSender *UnauthorizedUtxoTxSender) Generate(params core.SendTxParam) {
	pkh, err := core.NewUserPubKeyHash(params.SenderKeyPair.PublicKey)

	if err != nil {
		logger.WithError(err).Panic("UnauthorizedUtxoTx: Unable to hash sender public key")
	}

	prevUtxos, err := txSender.wallet.GetUtxoIndex().GetUTXOsByAmount(pkh, params.Amount)

	if err != nil {
		logger.WithError(err).Panic("UnauthorizedUtxoTx: Unable to get UTXOs to match the amount")
	}

	unauthorizedUtxo := txSender.wallet.GetUtxoIndex().GetAllUTXOsByPubKeyHash(txSender.unauthorizedAddrPkh).GetAllUtxos()
	prevUtxos = append(prevUtxos, unauthorizedUtxo[0])

	vouts := prepareOutputLists(prevUtxos, params.From, params.To, params.Amount, params.Tip)
	txSender.tx = NewTransaction(prevUtxos, vouts, params.Tip, params.SenderKeyPair)
}

func (txSender *UnauthorizedUtxoTxSender) Send() {

	_, err := txSender.dappSdk.SendTransaction(txSender.tx.ToProto().(*corepb.Transaction))

	if err != nil {
		logger.WithError(err).Error("UnauthorizedUtxoTx: Unable to send transaction!")
	}
}

func (txSender *UnauthorizedUtxoTxSender) Print() {
	logger.Info("Sending a transaction with an unauthrized utxo...")
}
