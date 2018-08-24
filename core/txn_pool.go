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

func (s *TransactionPool) StructDelete(txn interface{}) {
	for k, v := range s.Transactions.Get() {
		if bytes.Compare(v.(Transaction).ID, txn.(Transaction).ID) == 0 {

			var content []interface{}
			content = append(content, s.Transactions.Get()[k+1:]...)
			content = append(s.Transactions.Get()[0:k], content...)
			s.Transactions.Set(content)
			return
		}
	}
}

// Push a new value into slice
func (s *TransactionPool) StructPush(val interface{}) {
	if s.Transactions.Len() == 0 {
		s.Transactions.AddSliceItem(val)
		return
	}

	start, end := 0, s.Transactions.Len()-1
	result, mid := 0, 0
	for start <= end {
		mid = (start + end) / 2
		cmp := s.Transactions.GetSliceCmp()
		result = cmp(s.Transactions.Index(mid), val)
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
		content = append(content, s.Transactions.Get()[mid:]...)
		content = append(s.Transactions.Get()[0:mid], content...)
	} else {
		content = append(content, s.Transactions.Get()[mid+1:]...)
		content = append(s.Transactions.Get()[0:mid+1], content...)

	}
	s.Transactions.Set(content)
}

func NewTxnPool() *TransactionPool{
	txnPool := &TransactionPool{
		messageCh: make(chan string, 128),
		size:      128,
	}
	txnPool.Transactions = *sorted.NewSlice(CompareTransactionTips, txnPool.StructDelete, txnPool.StructPush)
	return txnPool
}

func GetTxnPoolInstance() *TransactionPool {
	once.Do(func() {
		instance = NewTxnPool()
	})
	return instance
}

//function f should return true if the transaction needs to be pushed back to the pool
func (pool *TransactionPool) Traverse(txHandler func(tx Transaction) bool){

	for _,v := range pool.Transactions.Get(){
		txn := v.(Transaction)
		if !txHandler(txn) {
			pool.Transactions.StructDelete(txn)
		}
	}
}

func (pool *TransactionPool) FilterAllTransactions(utxoPool utxoIndex) {
	pool.Traverse(func(tx Transaction) bool{
		return tx.Verify(utxoPool)
	})
}

//need to optimize
func (pool *TransactionPool) GetSortedTransactions() []*Transaction {
	sortedTransactions := []*Transaction{}
	for GetTxnPoolInstance().Transactions.Len() > 0 {
		txn:= GetTxnPoolInstance().Transactions.PopRight().(Transaction)
		sortedTransactions = append(sortedTransactions, &txn)
	}
	return sortedTransactions
}

func (pool *TransactionPool) ConditionalAdd(tx Transaction){
	//get smallest tip txn

	if(pool.Transactions.Len() >= TransactionPoolLimit){
		compareTx:= pool.Transactions.PopLeft().(Transaction)
		greaterThanLeastTip:= tx.Tip > compareTx.Tip
		if(greaterThanLeastTip){
			pool.Transactions.StructPush(tx)
		}else{// do nothing, push back popped tx
			pool.Transactions.StructPush(compareTx)
		}
	}else{
		pool.Transactions.StructPush(tx)
	}
}

func (pool *TransactionPool) Start() {
	go pool.messageLoop()
}

func (pool *TransactionPool) Stop() {
	pool.exitCh <- true
}

//todo: will change the input from string to transaction
func (pool *TransactionPool) PushTransaction(msg string) {
	//func (pool *TransactionPool) PushTransaction(tx *Transaction){
	//	pool.Push(tx)
	fmt.Println(msg)
}

func (pool *TransactionPool) messageLoop() {
	for {
		select {
		case <-pool.exitCh:
			fmt.Println("Quit Transaction Pool")
			return
		case msg := <-pool.messageCh:
			pool.PushTransaction(msg)
		}
	}
}

