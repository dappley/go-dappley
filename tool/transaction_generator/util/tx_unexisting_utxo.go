package util

import (
	"github.com/dappley/go-dappley/common"
	"github.com/dappley/go-dappley/core"
	"github.com/dappley/go-dappley/core/pb"
	"github.com/dappley/go-dappley/sdk"
	logger "github.com/sirupsen/logrus"
)

type UnexistingUtxoTxSender struct {
	TxSender
}

func NewUnexistingUtxoTxSender(dappSdk *sdk.DappSdk, wallet *sdk.DappSdkWallet) *UnexistingUtxoTxSender {
	return &UnexistingUtxoTxSender{
		TxSender{
			dappSdk: dappSdk,
			wallet:  wallet,
		},
	}
}

func (txSender *UnexistingUtxoTxSender) Generate(params core.SendTxParam) {
	pkh, err := core.NewUserPubKeyHash(params.SenderKeyPair.PublicKey)

	if err != nil {
		logger.WithError(err).Panic("UnexisitingUtxoTx: Unable to hash sender public key")
	}

	prevUtxos, err := txSender.wallet.GetUtxoIndex().GetUTXOsByAmount(pkh, params.Amount)

	if err != nil {
		logger.WithError(err).Panic("UnexisitingUtxoTx: Unable to get UTXOs to match the amount")
	}

	unexistingUtxo := &core.UTXO{
		TXOutput: *core.NewTXOutput(common.NewAmount(10), params.From),
		Txid:     []byte("FakeTxId"),
		TxIndex:  0,
		UtxoType: core.UtxoNormal,
	}
	prevUtxos = append(prevUtxos, unexistingUtxo)

	vouts := prepareOutputLists(prevUtxos, params.From, params.To, params.Amount, params.Tip)
	txSender.tx = NewTransaction(prevUtxos, vouts, params.Tip, params.SenderKeyPair)
}

func (txSender *UnexistingUtxoTxSender) Send() {

	_, err := txSender.dappSdk.SendTransaction(txSender.tx.ToProto().(*corepb.Transaction))

	if err != nil {
		logger.WithError(err).Error("UnexisitingUtxoTx: Sending transaction failed!")
	}
}

func (txSender *UnexistingUtxoTxSender) Print() {
	logger.Info("Sending a transaction with an unexisting utxo...")
}
