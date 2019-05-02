package util

import (
	"encoding/hex"
	"github.com/dappley/go-dappley/common"
	"github.com/dappley/go-dappley/core"
	"github.com/dappley/go-dappley/core/pb"
	"github.com/dappley/go-dappley/sdk"
	logger "github.com/sirupsen/logrus"
	"math/rand"
	"time"
)

const (
	contractFunctionCall = "{\"function\":\"record\",\"args\":[\"dEhFf5mWTSe67mbemZdK3WiJh8FcCayJqm\",\"4\"]}"
	TimeBetweenBatch1    = time.Duration(1000)
)

type BatchTxSender struct {
	tps         uint32
	wallet      *sdk.DappSdkWallet
	utxoIndex   *sdk.DappSdkUtxoIndex
	blockchain  *sdk.DappSdkBlockchain
	isRunning   bool
	pendingTxs  []*corepb.Transaction
	isScEnabled bool
	scAddr      string
	scSendFreq  uint32
	scCounter   uint32
}

func NewBatchTxSender(tps uint32, wallet *sdk.DappSdkWallet, utxoIndex *sdk.DappSdkUtxoIndex, blockchain *sdk.DappSdkBlockchain, smartContractSendFreq uint32, scAddr string) *BatchTxSender {
	return &BatchTxSender{
		tps:         tps,
		wallet:      wallet,
		utxoIndex:   utxoIndex,
		blockchain:  blockchain,
		isRunning:   false,
		isScEnabled: false,
		scAddr:      scAddr,
		scSendFreq:  smartContractSendFreq,
		scCounter:   0,
	}
}

func (sender *BatchTxSender) Start() {
	sender.isRunning = true
}

func (sender *BatchTxSender) Stop() {
	sender.isRunning = false
	sender.ClearPendingTx()
}

func (sender *BatchTxSender) EnableSmartContract() {
	sender.isScEnabled = true
}

func (sender *BatchTxSender) IsRunning() bool {
	return sender.isRunning
}

func (sender *BatchTxSender) ClearPendingTx() {
	sender.pendingTxs = []*corepb.Transaction{}
}

func (sender *BatchTxSender) AddTxToPendingTxs(tx *corepb.Transaction) {
	sender.pendingTxs = append(sender.pendingTxs, tx)
}

func (sender *BatchTxSender) IsPendingTxsReady() bool {
	return len(sender.pendingTxs) >= int(sender.tps)
}

func (sender *BatchTxSender) Run() {
	ticker := time.NewTicker(time.Millisecond * TimeBetweenBatch1).C
	go func() {
		for {
			select {
			case <-ticker:

				if !sender.IsRunning() {
					continue
				}

				if sender.IsPendingTxsReady() {
					if sender.blockchain.SendBatchTransactions(sender.pendingTxs) != nil {
						sender.utxoIndex.Update()
					}
					sender.ClearPendingTx()
				}

			default:
				if !sender.IsRunning() {
					continue
				}

				if sender.IsPendingTxsReady() {
					continue
				}

				tx := sender.createRandomTransaction()
				if tx != nil {
					sender.AddTxToPendingTxs(tx)
				}
			}
		}
	}()
}

func (sender *BatchTxSender) createRandomTransaction() *corepb.Transaction {

	data := ""

	fromIndex := sender.getAddrWithBalance()
	toIndex := getDifferentIndex(fromIndex, len(sender.wallet.GetAddrs()))
	toAddr := sender.wallet.GetAddrs()[toIndex].String()
	sendAmount := uint64(1)

	if sender.IsTheTurnToSendSmartContractTransaction() && sender.isScEnabled {
		data = contractFunctionCall
		toAddr = sender.scAddr
	}

	senderKeyPair := sender.wallet.GetWalletManager().GetKeyPairByAddress(sender.wallet.GetAddrs()[fromIndex])
	tx := sender.createTransaction(sender.wallet.GetAddrs()[fromIndex], core.NewAddress(toAddr), common.NewAmount(sendAmount), common.NewAmount(0), data, senderKeyPair)
	if tx == nil {
		return nil
	}

	sender.wallet.AddToBalance(toIndex, sendAmount)
	sender.wallet.SubstractFromBalance(fromIndex, sendAmount)

	return tx.ToProto().(*corepb.Transaction)
}

func (sender *BatchTxSender) getAddrWithBalance() int {
	fromIndex := rand.Intn(len(sender.wallet.GetAddrs()))
	amount := sender.wallet.GetBalances()[fromIndex]
	//TODO: add time out to this loop
	for amount <= 1 {
		fromIndex = rand.Intn(len(sender.wallet.GetAddrs()))
		amount = sender.wallet.GetBalances()[fromIndex]
	}
	return fromIndex
}

func getDifferentIndex(index int, maxIndex int) int {
	newIndex := rand.Intn(maxIndex)
	for newIndex == index {
		newIndex = rand.Intn(maxIndex)
	}
	return newIndex
}

func (sender *BatchTxSender) createTransaction(from, to core.Address, amount, tip *common.Amount, contract string, senderKeyPair *core.KeyPair) *core.Transaction {

	pkh, err := core.NewUserPubKeyHash(senderKeyPair.PublicKey)
	if err != nil {
		logger.WithError(err).Panic("Unable to hash sender public key")
	}
	prevUtxos, err := sender.utxoIndex.GetUtxoIndex().GetUTXOsByAmount(pkh, amount)

	if err != nil {
		return nil
	}
	sendTxParam := core.NewSendTxParam(from, senderKeyPair, to, amount, tip, contract)
	tx, err := core.NewUTXOTransaction(prevUtxos, sendTxParam)

	sendTXLogger := logger.WithFields(logger.Fields{
		"from":   from.String(),
		"to":     to.String(),
		"amount": amount.String(),
		"txid":   "",
		"data":   contract,
	})

	if err != nil {
		sendTXLogger.WithError(err).Error("Failed to send transaction!")
		return nil
	}
	sendTXLogger.Data["txid"] = hex.EncodeToString(tx.ID)

	sender.utxoIndex.GetUtxoIndex().UpdateUtxo(&tx)
	return &tx
}

func (sender *BatchTxSender) IsTheTurnToSendSmartContractTransaction() bool {
	sender.scCounter += 1
	result := false
	if sender.scCounter == sender.scSendFreq {
		result = true
		sender.scCounter = 0
	}
	return result
}
