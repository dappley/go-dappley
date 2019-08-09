package util

import (
	"math/rand"
	"time"

	"github.com/dappley/go-dappley/common"
	"github.com/dappley/go-dappley/core/account"
	"github.com/dappley/go-dappley/core/transaction"
	transactionpb "github.com/dappley/go-dappley/core/transaction/pb"
	"github.com/dappley/go-dappley/logic/transaction_logic"
	"github.com/dappley/go-dappley/sdk"
	logger "github.com/sirupsen/logrus"
)

const (
	contractFunctionCall = "{\"function\":\"record\",\"args\":[\"dEhFf5mWTSe67mbemZdK3WiJh8FcCayJqm\",\"4\"]}"
	TimeBetweenBatch1    = time.Duration(1000)
	timeoutInSec         = 1
)

type BatchTxSender struct {
	tps         uint32
	account     *sdk.DappSdkAccount
	dappSdk     *sdk.DappSdk
	isRunning   bool
	pendingTxs  []*transactionpb.Transaction
	isScEnabled bool
	scAddr      string
	scSendFreq  uint32
	scCounter   uint32
}

func NewBatchTxSender(tps uint32, account *sdk.DappSdkAccount, dappSdk *sdk.DappSdk, smartContractSendFreq uint32, scAddr string) *BatchTxSender {
	return &BatchTxSender{
		tps:         tps,
		account:     account,
		dappSdk:     dappSdk,
		isRunning:   false,
		isScEnabled: false,
		scAddr:      scAddr,
		scSendFreq:  smartContractSendFreq,
		scCounter:   0,
	}
}

func (sender *BatchTxSender) Resume() {
	sender.isRunning = true
}

func (sender *BatchTxSender) Pause() {
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
	sender.pendingTxs = []*transactionpb.Transaction{}
}

func (sender *BatchTxSender) AddTxToPendingTxs(tx *transactionpb.Transaction) {
	sender.pendingTxs = append(sender.pendingTxs, tx)
}

func (sender *BatchTxSender) IsPendingTxsReady() bool {
	return len(sender.pendingTxs) >= int(sender.tps)
}

func (sender *BatchTxSender) Run() {
	ticker := time.NewTicker(time.Millisecond * TimeBetweenBatch1).C
	sender.isRunning = true

	go func() {
		for {
			select {
			case <-ticker:
				if !sender.IsRunning() {
					continue
				}

				if sender.IsPendingTxsReady() {
					if sender.dappSdk.SendBatchTransactions(sender.pendingTxs) != nil {
						sender.account.Update()
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

				if sender.account.IsZeroBalance() {
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

func (sender *BatchTxSender) createRandomTransaction() *transactionpb.Transaction {

	data := ""

	fromIndex := sender.getAddrWithNoneZeroBalance()
	fromAddr := sender.account.GetAddrs()[fromIndex]

	toIndex := getDifferentIndex(fromIndex, len(sender.account.GetAddrs()))
	toAddr := sender.account.GetAddrs()[toIndex]

	sendAmount := uint64(1)

	if sender.IsTheTurnToSendSmartContractTransaction() && sender.isScEnabled {
		data = contractFunctionCall
		toAddr = account.NewAddress(sender.scAddr)
	}

	senderKeyPair := sender.account.GetAccountManager().GetKeyPairByAddress(sender.account.GetAddrs()[fromIndex])
	gasLimit := common.NewAmount(0)
	gasPrice := common.NewAmount(0)
	if data != "" {
		gasLimit = common.NewAmount(30000)
		gasPrice = common.NewAmount(1)
	}
	tx := sender.createTransaction(fromAddr, toAddr, common.NewAmount(sendAmount), common.NewAmount(0), gasLimit, gasPrice, data, senderKeyPair)
	if tx == nil {
		return nil
	}

	sender.account.GetUtxoIndex().UpdateUtxo(tx)
	sender.account.UpdateBalance(toAddr, sender.account.GetBalance(toAddr)+sendAmount)
	sender.account.UpdateBalance(fromAddr, sender.account.GetBalance(fromAddr)-sendAmount)

	return tx.ToProto().(*transactionpb.Transaction)
}

func (sender *BatchTxSender) getAddrWithNoneZeroBalance() int {
	fromIndex := rand.Intn(len(sender.account.GetAddrs()))
	fromAddr := sender.account.GetAddrs()[fromIndex]
	amount := sender.account.GetBalance(fromAddr)

	deadline := time.Now().Unix() + timeoutInSec

	for amount <= 1 && time.Now().Unix() < deadline {
		fromIndex = rand.Intn(len(sender.account.GetAddrs()))
		fromAddr = sender.account.GetAddrs()[fromIndex]
		amount = sender.account.GetBalance(fromAddr)
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

func (sender *BatchTxSender) createTransaction(from, to account.Address, amount, tip *common.Amount, gasLimit *common.Amount, gasPrice *common.Amount, contract string, senderKeyPair *account.KeyPair) *transaction.Transaction {

	pkh, err := account.NewUserPubKeyHash(senderKeyPair.GetPublicKey())
	if err != nil {
		logger.WithError(err).Panic("Unable to hash sender public key")
	}
	prevUtxos, err := sender.account.GetUtxoIndex().GetUTXOsByAmount(pkh, amount)

	if err != nil {
		return nil
	}
	sendTxParam := transaction.NewSendTxParam(from, senderKeyPair, to, amount, tip, gasLimit, gasPrice, contract)
	tx, err := transaction_logic.NewUTXOTransaction(prevUtxos, sendTxParam)

	if err != nil {
		logger.WithFields(logger.Fields{
			"from":   from.String(),
			"to":     to.String(),
			"amount": amount.String(),
			"txid":   "",
			"data":   contract,
		}).WithError(err).Error("Failed to send transaction!")
		return nil
	}

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
