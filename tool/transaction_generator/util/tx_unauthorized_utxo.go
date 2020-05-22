package util

import (
	"github.com/dappley/go-dappley/core/account"
	"github.com/dappley/go-dappley/core/transaction"
	transactionpb "github.com/dappley/go-dappley/core/transaction/pb"
	"github.com/dappley/go-dappley/sdk"
	logger "github.com/sirupsen/logrus"
)

type UnauthorizedUtxoTxSender struct {
	TxSender
	unauthorizedAddrPkh account.PubKeyHash
}

func NewUnauthorizedUtxoTxSender(dappSdk *sdk.DappSdk, acc *sdk.DappSdkAccount, unauthorizedAddr account.Address) *UnauthorizedUtxoTxSender {
	unauthorizedTA := account.NewAccountByKey(acc.GetAccountManager().GetKeyPairByAddress(unauthorizedAddr))

	return &UnauthorizedUtxoTxSender{
		TxSender{
			dappSdk: dappSdk,
			account: acc,
		},
		unauthorizedTA.GetPubKeyHash(),
	}
}

func (txSender *UnauthorizedUtxoTxSender) Generate(params transaction.SendTxParam) {
	if ok, err := account.IsValidPubKey(params.SenderKeyPair.GetPublicKey()); !ok {
		logger.WithError(err).Panic("UnexisitingUtxoTx: Unable to hash sender public key")
	}
	ta := account.NewAccountByKey(params.SenderKeyPair)

	prevUtxos, err := txSender.account.GetUtxoIndex().GetUTXOsByAmount(ta.GetPubKeyHash(), params.Amount)

	if err != nil {
		logger.WithError(err).Panic("UnauthorizedUtxoTx: Unable to get UTXOs to match the amount")
	}

	unauthorizedUtxo := txSender.account.GetUtxoIndex().GetAllUTXOsByPubKeyHash(txSender.unauthorizedAddrPkh).GetAllUtxos()
	prevUtxos = append(prevUtxos, unauthorizedUtxo[0])
	fromTA := account.NewTransactionAccountByAddress(params.From)
	toTA := account.NewTransactionAccountByAddress(params.To)
	vouts := prepareOutputLists(prevUtxos, fromTA, toTA, params.Amount, params.Tip)
	txSender.tx = NewTransaction(prevUtxos, vouts, params.Tip, params.SenderKeyPair)
}

func (txSender *UnauthorizedUtxoTxSender) Send() {

	_, err := txSender.dappSdk.SendTransaction(txSender.tx.ToProto().(*transactionpb.Transaction))

	if err != nil {
		logger.WithError(err).Error("UnauthorizedUtxoTx: Sending transaction failed!")
	}
}

func (txSender *UnauthorizedUtxoTxSender) Print() {
	logger.Info("Sending a transaction with an unauthrized utxo...")
}
