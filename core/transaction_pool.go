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
	logger "github.com/sirupsen/logrus"
)

type TransactionPool struct {
	Transactions sorted.Slice
	limit        uint32
}

func compareTxTips(tx1 interface{}, tx2 interface{}) int {
	t1 := tx1.(Transaction)
	t2 := tx2.(Transaction)
	if t1.Tip < t2.Tip {
		return -1
	} else if t1.Tip > t2.Tip {
		return 1
	} else {
		return 0
	}
}

// match returns true if tx1 and tx2 are Transactions and they have the same ID, false otherwise
func match(tx1 interface{}, tx2 interface{}) bool {
	return bytes.Compare(tx1.(Transaction).ID, tx2.(Transaction).ID) == 0
}

func NewTransactionPool(limit uint32) *TransactionPool {
	return &TransactionPool{
		Transactions: *sorted.NewSlice(compareTxTips, match),
		limit:        limit,
	}
}

func (txPool *TransactionPool) RemoveMultipleTransactions(txs []*Transaction) {
	for _, tx := range txs {
		txPool.Transactions.Del(*tx)
	}
}

// traverse iterates through the transaction pool and pass the transaction to txHandler callback in each iteration
func (txPool *TransactionPool) traverse(txHandler func(tx Transaction)) {
	for _, v := range txPool.Transactions.Get() {
		tx := v.(Transaction)
		txHandler(tx)
	}
}

// RemoveInvalidTransactions removes invalid transactions in transaction pool based on the existing UTXOs in utxoPool
func (txPool *TransactionPool) RemoveInvalidTransactions(utxoPool UTXOIndex) {
	txPool.traverse(func(tx Transaction) {
		if !tx.Verify(utxoPool, 0) { // all transactions in transaction pool have no blockHeight
			txPool.Transactions.Del(tx)
		}
	})
}

func (txPool *TransactionPool) Pop() []*Transaction {
	var sortedTransactions []*Transaction
	for txPool.Transactions.Len() > 0 {
		tx := txPool.Transactions.PopRight().(Transaction)
		sortedTransactions = append(sortedTransactions, &tx)
	}
	return sortedTransactions
}

func (txPool *TransactionPool) Push(tx Transaction) {
	if txPool.limit == 0 {
		logger.Warn("TransactionPool: transaction not pushed to pool because limit is set to 0")
		return
	}

	if txPool.Transactions.Len() >= int(txPool.limit) {
		logger.WithFields(logger.Fields{
			"limit": txPool.limit,
		}).Debug("TransactionPool: transaction pool limit reached")

		// Get tx with least tips
		compareTx := txPool.Transactions.PopLeft().(Transaction)
		greaterThanLeastTip := tx.Tip > compareTx.Tip
		if greaterThanLeastTip {
			txPool.Transactions.Push(tx)
		} else { // do nothing, push back popped tx
			txPool.Transactions.Push(compareTx)
		}
	} else {
		txPool.Transactions.Push(tx)
	}
}
