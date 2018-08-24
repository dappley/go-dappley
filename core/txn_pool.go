package core

import (
	"bytes"
	"sync"

	"github.com/dappley/go-dappley/common/sorted"
	"fmt"
)

const UtxoMapKey = "utxo"
const UtxoForkMapKey = "utxoFork"
const TransactionPoolLimit = 5

type TransactionPool struct {
	messageCh    chan string
	exitCh       chan bool
	size         int
	Transactions sorted.Slice
}

var instance *TransactionPool
var once sync.Once

func CompareTransactionTips(a interface{}, b interface{}) int {
	ai := a.(Transaction)
	bi := b.(Transaction)
	if ai.Tip < bi.Tip {
		return -1
	} else if ai.Tip > bi.Tip {
		return 1
	} else {
		return 0
	}
}

func (txnPool *TransactionPool) StructDelete(txn interface{}) {
	for k, v := range txnPool.Transactions.Get() {
		if bytes.Compare(v.(Transaction).ID, txn.(Transaction).ID) == 0 {

			var content []interface{}
			content = append(content, txnPool.Transactions.Get()[k+1:]...)
			content = append(txnPool.Transactions.Get()[0:k], content...)
			txnPool.Transactions.Set(content)
			return
		}
	}
}

// Push a new value into slice
func (txnPool *TransactionPool) StructPush(val interface{}) {
	if txnPool.Transactions.Len() == 0 {
		txnPool.Transactions.AddSliceItem(val)
		return
	}

	start, end := 0, txnPool.Transactions.Len()-1
	result, mid := 0, 0
	for start <= end {
		mid = (start + end) / 2
		cmp := txnPool.Transactions.GetSliceCmp()
		result = cmp(txnPool.Transactions.Index(mid), val)
		if result > 0 {
			end = mid - 1
		} else if result < 0 {
			start = mid + 1
		} else {
			break
		}
	}
	content := []interface{}{val}
	if result > 0 {
		content = append(content, txnPool.Transactions.Get()[mid:]...)
		content = append(txnPool.Transactions.Get()[0:mid], content...)
	} else {
		content = append(content, txnPool.Transactions.Get()[mid+1:]...)
		content = append(txnPool.Transactions.Get()[0:mid+1], content...)

	}
	txnPool.Transactions.Set(content)
}

func NewTxnPool() *TransactionPool{
	txnPool := &TransactionPool{
		messageCh: make(chan string, 128),
		size:      128,
	}
	txnPool.Transactions = *sorted.NewSlice(CompareTransactionTips, txnPool.StructDelete, txnPool.StructPush)
	return txnPool
}

func (txnPool *TransactionPool) RemoveMultipleTransactions(txs []*Transaction){
	for _,tx := range txs {
		txnPool.StructDelete(*tx)
	}
}

//function f should return true if the transaction needs to be pushed back to the pool
func (txnPool *TransactionPool) Traverse(txHandler func(tx Transaction) bool){

	for _,v := range txnPool.Transactions.Get(){
		txn := v.(Transaction)
		if !txHandler(txn) {
			txnPool.Transactions.StructDelete(txn)
		}
	}
}

func (txnPool *TransactionPool) FilterAllTransactions(utxoPool utxoIndex) {
	txnPool.Traverse(func(tx Transaction) bool{
		return tx.Verify(utxoPool)
	})
}

//need to optimize
func (txnPool *TransactionPool) GetSortedTransactions() []*Transaction {
	sortedTransactions := []*Transaction{}
	for txnPool.Transactions.Len() > 0 {
		txn:= txnPool.Transactions.PopRight().(Transaction)
		sortedTransactions = append(sortedTransactions, &txn)
	}
	return sortedTransactions
}

func (txnPool *TransactionPool) ConditionalAdd(tx Transaction){
	//get smallest tip txn

	if(txnPool.Transactions.Len() >= TransactionPoolLimit){
		compareTx:= txnPool.Transactions.PopLeft().(Transaction)
		greaterThanLeastTip:= tx.Tip > compareTx.Tip
		if(greaterThanLeastTip){
			txnPool.Transactions.StructPush(tx)
		}else{// do nothing, push back popped tx
			txnPool.Transactions.StructPush(compareTx)
		}
	}else{
		txnPool.Transactions.StructPush(tx)
	}
}

func (txnPool *TransactionPool) Start() {
	go txnPool.messageLoop()
}

func (txnPool *TransactionPool) Stop() {
	txnPool.exitCh <- true
}

//todo: will change the input from string to transaction
func (txnPool *TransactionPool) PushTransaction(msg string) {
	//func (txnPool *TransactionPool) PushTransaction(tx *Transaction){
	//	txnPool.Push(tx)
	fmt.Println(msg)
}

func (txnPool *TransactionPool) messageLoop() {
	for {
		select {
		case <-txnPool.exitCh:
			fmt.Println("Quit Transaction Pool")
			return
		case msg := <-txnPool.messageCh:
			txnPool.PushTransaction(msg)
		}
	}
}

