package util

import (
	"github.com/dappley/go-dappley/common"
	"github.com/dappley/go-dappley/core/account"
	"github.com/dappley/go-dappley/core/transaction"
	transactionpb "github.com/dappley/go-dappley/core/transaction/pb"
	"github.com/dappley/go-dappley/core/transaction_base"
	"github.com/dappley/go-dappley/core/utxo"
	"github.com/dappley/go-dappley/sdk"
	logger "github.com/sirupsen/logrus"
)

type UnexistingUtxoTxSender struct {
	TxSender
}

func NewUnexistingUtxoTxSender(dappSdk *sdk.DappSdk, account *sdk.DappSdkAccount) *UnexistingUtxoTxSender {
	return &UnexistingUtxoTxSender{
		TxSender{
			dappSdk: dappSdk,
			account: account,
		},
	}
}

func (txSender *UnexistingUtxoTxSender) Generate(params transaction.SendTxParam) {
	pkh, err := account.NewUserPubKeyHash(params.SenderKeyPair.GetPublicKey())

	if err != nil {
		logger.WithError(err).Panic("UnexisitingUtxoTx: Unable to hash sender public key")
	}

	prevUtxos, err := txSender.account.GetUtxoIndex().GetUTXOsByAmount(pkh, params.Amount)

	if err != nil {
		logger.WithError(err).Panic("UnexisitingUtxoTx: Unable to get UTXOs to match the amount")
	}

	unexistingUtxo := &utxo.UTXO{
		TXOutput: *transaction_base.NewTXOutput(common.NewAmount(10), params.From),
		Txid:     []byte("FakeTxId"),
		TxIndex:  0,
		UtxoType: utxo.UtxoNormal,
	}
	prevUtxos = append(prevUtxos, unexistingUtxo)

	vouts := prepareOutputLists(prevUtxos, params.From, params.To, params.Amount, params.Tip)
	txSender.tx = NewTransaction(prevUtxos, vouts, params.Tip, params.SenderKeyPair)
}

func (txSender *UnexistingUtxoTxSender) Send() {

	_, err := txSender.dappSdk.SendTransaction(txSender.tx.ToProto().(*transactionpb.Transaction))

	if err != nil {
		logger.WithError(err).Error("UnexisitingUtxoTx: Sending transaction failed!")
	}
}

func (txSender *UnexistingUtxoTxSender) Print() {
	logger.Info("Sending a transaction with an unexisting utxo...")
}
