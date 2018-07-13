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
	"container/heap"
	"fmt"
	"sync"
)

// An TransactionPool is a max-heap of Transactions.
type TransactionPool struct {
	messageCh    chan string
	exitCh       chan bool
	size         int
	transactions []Transaction
}

var instance *TransactionPool
var once sync.Once

func (pool TransactionPool) Len() int { return len(pool.transactions) }

//Compares Transaction Tips
func (pool TransactionPool) Less(i, j int) bool {
	return pool.transactions[i].Tip > pool.transactions[j].Tip
}
func (pool TransactionPool) Swap(i, j int) {
	pool.transactions[i], pool.transactions[j] = pool.transactions[j], pool.transactions[i]
}

//func NewTransactionPool(size int) (*TransactionPool) {
//	txPool := &TransactionPool{
//		messageCh:    make(chan string, size),
//		size:         size,
//	}
//	heap.Init(txPool)
//	return txPool
//}
func (pool *TransactionPool) Push(x interface{}) {
	// Push and Pop use pointer receivers because they modify the slice's length,
	// not just its contents.
	pool.transactions = append(pool.transactions, x.(Transaction))
}

func (pool *TransactionPool) Pop() interface{} {
	old := pool.transactions
	length := len(old)
	last := old[length-1]
	pool.transactions = old[0 : length-1]
	return last
}

func GetTxnPoolInstance() *TransactionPool {
	once.Do(func() {
		//instance = &TransactionPool{}
		instance = &TransactionPool{
			messageCh: make(chan string, 128),
			size:      128,
		}
	})
	heap.Init(instance)
	return instance
}

func (pool *TransactionPool) GetSortedTransactions() []*Transaction {
	sortedTransactions := []*Transaction{}

	for GetTxnPoolInstance().Len() > 0 {
		if len(sortedTransactions) < TransactionPoolLimit {
			var transaction = heap.Pop(GetTxnPoolInstance()).(Transaction)
			sortedTransactions = append(sortedTransactions, &transaction)
		}
	}
	return sortedTransactions
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
