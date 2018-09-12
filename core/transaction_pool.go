// Copyright (C) 2018 go-dappley authors
//
// This file is part of the go-dappley library.
//
// the go-dappley library is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// the go-dappley library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with the go-dappley library.  If not, see <http://www.gnu.org/licenses/>.
//

package core

import (
	"bytes"
	"github.com/dappley/go-dappley/common/sorted"
	"fmt"
)


const TransactionPoolLimit = 5

type TransactionPool struct {
	messageCh    chan string
	exitCh       chan bool
	size         int
	Transactions sorted.Slice
}

func NewTransactionPool() *TransactionPool{
	txPool := &TransactionPool{
		messageCh: make(chan string, 128),
		size:      128,
	}
	txPool.Transactions = *sorted.NewSlice(CompareTransactionTips, txPool.StructDelete, txPool.StructPush)
	return txPool
}

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

func (txPool *TransactionPool) StructDelete(tx interface{}) {
	for k, v := range txPool.Transactions.Get() {
		if bytes.Compare(v.(Transaction).ID, tx.(Transaction).ID) == 0 {

			var content []interface{}
			content = append(content, txPool.Transactions.Get()[k+1:]...)
			content = append(txPool.Transactions.Get()[0:k], content...)
			txPool.Transactions.Set(content)
			return
		}
	}
}

// Push a new value into slice
func (txPool *TransactionPool) StructPush(val interface{}) {
	if txPool.Transactions.Len() == 0 {
		txPool.Transactions.AddSliceItem(val)
		return
	}

	start, end := 0, txPool.Transactions.Len()-1
	result, mid := 0, 0
	for start <= end {
		mid = (start + end) / 2
		cmp := txPool.Transactions.GetSliceCmp()
		result = cmp(txPool.Transactions.Index(mid), val)
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
		content = append(content, txPool.Transactions.Get()[mid:]...)
		content = append(txPool.Transactions.Get()[0:mid], content...)
	} else {
		content = append(content, txPool.Transactions.Get()[mid+1:]...)
		content = append(txPool.Transactions.Get()[0:mid+1], content...)

	}
	txPool.Transactions.Set(content)
}


func (txPool *TransactionPool) RemoveMultipleTransactions(txs []*Transaction){
	for _,tx := range txs {
		txPool.StructDelete(*tx)
	}
}

//function f should return true if the transaction needs to be pushed back to the pool
func (txPool *TransactionPool) Traverse(txHandler func(tx Transaction) bool){

	for _,v := range txPool.Transactions.Get(){
		tx := v.(Transaction)
		if !txHandler(tx) {
			txPool.Transactions.StructDelete(tx)
		}
	}
}

func (txPool *TransactionPool) FilterAllTransactions(utxoPool UtxoIndex) {
	txPool.Traverse(func(tx Transaction) bool{
		return tx.Verify(utxoPool) // TODO: also check if amount is valid
	})
}

//need to optimize
func (txPool *TransactionPool) PopSortedTransactions() []*Transaction {
	sortedTransactions := []*Transaction{}
	for txPool.Transactions.Len() > 0 {
		tx := txPool.Transactions.PopRight().(Transaction)
		sortedTransactions = append(sortedTransactions, &tx)
	}
	return sortedTransactions
}

func (txPool *TransactionPool) ConditionalAdd(tx Transaction){
	//get smallest tip tx

	if(txPool.Transactions.Len() >= TransactionPoolLimit){
		compareTx:= txPool.Transactions.PopLeft().(Transaction)
		greaterThanLeastTip:= tx.Tip > compareTx.Tip
		if(greaterThanLeastTip){
			txPool.Transactions.StructPush(tx)
		}else{// do nothing, push back popped tx
			txPool.Transactions.StructPush(compareTx)
		}
	}else{
		txPool.Transactions.StructPush(tx)
	}
}

func (txPool *TransactionPool) Start() {
	go txPool.messageLoop()
}

func (txPool *TransactionPool) Stop() {
	txPool.exitCh <- true
}

//todo: will change the input from string to transaction
func (txPool *TransactionPool) PushTransaction(msg string) {
	//func (txPool *TransactionPool) PushTransaction(tx *Transaction){
	//	txPool.Push(tx)
	fmt.Println(msg)
}

func (txPool *TransactionPool) messageLoop() {
	for {
		select {
		case <-txPool.exitCh:
			fmt.Println("Quit Transaction Pool")
			return
		case msg := <-txPool.messageCh:
			txPool.PushTransaction(msg)
		}
	}
}

