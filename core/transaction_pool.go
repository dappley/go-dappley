// Copyright (C) 2018 go-dappley authors
//
// This file is part of the go-dappley library.
//
// the go-dappley library is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either pubKeyHash 3 of the License, or
// (at your option) any later pubKeyHash.
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
	index 		 map[string]*Transaction
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
		index: 		  make(map[string]*Transaction),
		limit:        limit,
	}
}

func (txPool *TransactionPool) RemoveMultipleTransactions(txs []*Transaction) {
	for _, tx := range txs {
		txPool.Transactions.Del(*tx)
		delete(txPool.index, string(tx.ID))
	}
}

func (txPool *TransactionPool) PopValidTxs(utxoIndex UTXOIndex) []*Transaction {
	var validTxs []*Transaction
	var invalidTxs []*Transaction

	tempTxPool := txPool.deepCopy()
	tempUtxoIndex := utxoIndex.DeepCopy()
	for txId := range txPool.index {
		tx := txPool.index[txId]

		if contains(tx, validTxs) || contains(tx, invalidTxs) {
			continue
		}

		if tx.Verify(tempUtxoIndex, tempTxPool, 0) {
			dependentTxs := txPool.getDependentTxs(tx.ID, []*Transaction{})
			validTxs = append(validTxs, dependentTxs...)
		} else {
			invalidTxs = append(invalidTxs, tx)
		}
	}

	txPool.RemoveMultipleTransactions(validTxs)
	txPool.RemoveMultipleTransactions(invalidTxs)

	return validTxs
}

func (txPool *TransactionPool) deepCopy() TransactionPool {
	txPoolCopy := TransactionPool{
		Transactions: *sorted.NewSlice(compareTxTips, match),
		index:        make(map[string]*Transaction),
		limit:        txPool.limit,
	}

	for _, tx := range txPool.index {
		txPoolCopy.Push(tx.TrimmedCopy())
	}

	return txPoolCopy
}

func contains(targetTx *Transaction, txs []*Transaction) bool {
	for _, tx := range txs {
		if bytes.Equal(targetTx.ID, tx.ID) {
			return true
		}
	}
	return false
}

func (txPool *TransactionPool) getDependentTxs(txID []byte, dependentTxs []*Transaction) []*Transaction {
	if _, exists := txPool.index[string(txID)]; !exists {
		return dependentTxs
	}
	tx := txPool.index[string(txID)]
	dependentTxs = append(dependentTxs, tx)
	for _, vin := range tx.Vin {
		dependentTxs = txPool.getDependentTxs(vin.Txid, dependentTxs)
	}
	return dependentTxs
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

		leastTipTx := txPool.Transactions.Left().(Transaction)
		if tx.Tip <= leastTipTx.Tip {
			return
		}

		txPool.Transactions.PopLeft()
	}

	if _, exists := txPool.index[string(tx.ID)]; exists {
		logger.Warn("TransactionPool: transaction not pushed to pool because transaction ID already exists")
	}

	txPool.Transactions.Push(tx)
	txPool.index[string(tx.ID)] = &tx
}

func (txPool *TransactionPool) GetTxByID(txId []byte) *Transaction {
	if _, exists := txPool.index[string(txId)]; !exists {
		logger.Warn("TransactionPool: transaction does not exists")
		return nil
	}

	return txPool.index[string(txId)]
}
