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

	"github.com/asaskevich/EventBus"
	"github.com/dappley/go-dappley/common/sorted"
	logger "github.com/sirupsen/logrus"
)

const (
	NewTransactionTopic   = "NewTransaction"
	EvictTransactionTopic = "EvictTransaction"
)

type TransactionPool struct {
	Transactions sorted.Slice
	index 		 map[string]*Transaction
	limit        uint32
	EventBus     EventBus.Bus
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
		EventBus:     EventBus.New(),
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

	for _, tx := range txPool.index {
		if contains(tx, validTxs) || contains(tx, invalidTxs) {
			continue
		}

		if tx.Verify(utxoIndex, txPool, 0) {
			dependentTxs := txPool.getDependentTxs(tx.ID, []*Transaction{})
			for _, dependentTx := range dependentTxs {
				if !contains(dependentTx, validTxs) {
					validTxs = append(validTxs, dependentTx)
				}
			}
		} else {
			invalidTxs = append(invalidTxs, tx)
		}
	}

	txPool.RemoveMultipleTransactions(validTxs)

	return validTxs
}

func (txPool *TransactionPool) GetAllTransactions() []*Transaction{
	txs := []*Transaction{}
	for _, v := range txPool.Transactions.Get() {
		tx := v.(Transaction)
		txs = append(txs, &tx)
	}
	return txs
}

func (txPool *TransactionPool) deepCopy() TransactionPool {
	txPoolCopy := TransactionPool{
		Transactions: *sorted.NewSlice(compareTxTips, match),
		index:        make(map[string]*Transaction),
		limit:        txPool.limit,
		EventBus:     EventBus.New(),
	}

	for _, tx := range txPool.index {
		txPoolCopy.Push(tx.DeepCopy())
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

	hashTxs := map[*Transaction]bool{tx: true}
	for _, vin := range tx.Vin {
		parentTxs := txPool.getDependentTxs(vin.Txid, dependentTxs)
		for _, parentTx := range parentTxs {
			if _, exists := hashTxs[parentTx]; !exists {
				hashTxs[parentTx] = true
				dependentTxs = append(dependentTxs, parentTx)
			}
		}
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
		txPool.EventBus.Publish(EvictTransactionTopic, &leastTipTx)
	}

	if _, exists := txPool.index[string(tx.ID)]; exists {
		logger.Warn("TransactionPool: transaction not pushed to pool because transaction ID already exists")
	}

	txPool.Transactions.Push(tx)
	txPool.index[string(tx.ID)] = &tx
	txPool.EventBus.Publish(NewTransactionTopic, &tx)
}

func (txPool *TransactionPool) GetTxByID(txId []byte) *Transaction {
	if _, exists := txPool.index[string(txId)]; !exists {
		logger.Warn("TransactionPool: transaction does not exist")
		return nil
	}

	return txPool.index[string(txId)]
}
