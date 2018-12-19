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
	"math"
	"sync"

	"github.com/asaskevich/EventBus"
	logger "github.com/sirupsen/logrus"
)

const (
	NewTransactionTopic   = "NewTransaction"
	EvictTransactionTopic = "EvictTransaction"
)

type TransactionNode struct {
	children map[string]*Transaction
	value    *Transaction
}

type TransactionPool struct {
	txs        map[string]*TransactionNode
	minTipTxId string
	limit      uint32
	EventBus   EventBus.Bus
	mutex      sync.RWMutex
}

func NewTransactionPool(limit uint32) *TransactionPool {
	return &TransactionPool{
		txs:      make(map[string]*TransactionNode),
		limit:    limit,
		EventBus: EventBus.New(),
		mutex:    sync.RWMutex{},
	}
}

func (txPool *TransactionPool) deepCopy() *TransactionPool {
	txPoolCopy := TransactionPool{
		txs:      make(map[string]*TransactionNode),
		limit:    txPool.limit,
		EventBus: EventBus.New(),
		mutex:    sync.RWMutex{},
	}

	for key, tx := range txPool.txs {
		newTx := tx.value.DeepCopy()
		newTxNode := TransactionNode{children: make(map[string]*Transaction), value: &newTx}

		for childKey, childTx := range tx.children {
			newTxNode.children[childKey] = childTx
		}
		txPoolCopy.txs[key] = &newTxNode
	}

	return &txPoolCopy
}

func (txPool *TransactionPool) GetAndResetTransactions() []*Transaction {
	txPool.mutex.Lock()
	defer txPool.mutex.Unlock()

	txs := txPool.getSortedTransactions()

	txPool.minTipTxId = ""
	txPool.txs = make(map[string]*TransactionNode)
	return txs
}

func (txPool *TransactionPool) GetTransactions() []*Transaction {
	txPool.mutex.RLock()
	defer txPool.mutex.RUnlock()

	return txPool.getSortedTransactions()
}

func (txPool *TransactionPool) Push(tx *Transaction) {
	txPool.mutex.Lock()
	defer txPool.mutex.Unlock()
	if txPool.limit == 0 {
		logger.Warn("TransactionPool: transaction is not pushed to pool because limit is set to 0.")
		return
	}

	if len(txPool.txs) >= int(txPool.limit) {
		logger.WithFields(logger.Fields{
			"limit": txPool.limit,
		}).Warn("TransactionPool: is full.")

		minTx, exist := txPool.txs[txPool.minTipTxId]
		if exist && tx.Tip <= minTx.value.Tip {
			return
		}

		toRemoveTxs := txPool.getToRemoveTxs(txPool.minTipTxId)
		if checkDependTxInMap(tx, toRemoveTxs) == true {
			logger.Warn("TransactionPool: failed to push because dependent transactions are not removed from pool.")
			return
		}

		txPool.removeSelectedTransactions(toRemoveTxs)
		txPool.minTipTxId = ""
		txPool.addTransaction(tx)
	} else {
		txPool.addTransaction(tx)
	}
}

func (txPool *TransactionPool) CheckAndRemoveTransactions(txs []*Transaction) {
	txPool.mutex.Lock()
	defer txPool.mutex.Unlock()

	for _, tx := range txs {
		toRemoveTxs := txPool.getToRemoveTxs(string(tx.ID))
		txPool.removeSelectedTransactions(toRemoveTxs)
	}

	txPool.resetMinTipTransaction()
}

func (txPool *TransactionPool) getSortedTransactions() []*Transaction {
	checkNodes := make(map[string]*TransactionNode)

	for key, node := range txPool.txs {
		checkNodes[key] = node
	}

	var sortedTxs []*Transaction
	for len(checkNodes) > 0 {
		for key, node := range checkNodes {
			if !checkDependTxInMap(node.value, checkNodes) {
				sortedTxs = append(sortedTxs, node.value)
				delete(checkNodes, key)
			}
		}
	}
	return sortedTxs
}

func checkDependTxInMap(tx *Transaction, existTxs map[string]*TransactionNode) bool {
	for _, vin := range tx.Vin {
		if _, exist := existTxs[string(vin.Txid)]; exist {
			return true
		}
	}
	return false
}

func (txPool *TransactionPool) getToRemoveTxs(startTxId string) map[string]*TransactionNode {
	txNode, ok := txPool.txs[startTxId]

	if !ok {
		return nil
	}

	toRemoveTxs := make(map[string]*TransactionNode)
	var toCheckTxs []*TransactionNode

	toCheckTxs = append(toCheckTxs, txNode)

	for len(toCheckTxs) > 0 {
		currentTxNode := toCheckTxs[0]
		toCheckTxs = toCheckTxs[1:]
		for key, _ := range currentTxNode.children {
			toCheckTxs = append(toCheckTxs, txPool.txs[key])
		}
		toRemoveTxs[string(currentTxNode.value.ID)] = currentTxNode
	}

	return toRemoveTxs
}

// The param toRemoveTxs must be calculate by function getToRemoveTxs
func (txPool *TransactionPool) removeSelectedTransactions(toRemoveTxs map[string]*TransactionNode) {
	for txId, txNode := range toRemoveTxs {
		for _, vin := range txNode.value.Vin {
			parentTx, exist := txPool.txs[string(vin.Txid)]
			if exist {
				delete(parentTx.children, txId)
			}
		}
		delete(txPool.txs, txId)
		txPool.EventBus.Publish(EvictTransactionTopic, txNode.value)
	}
}

func (txPool *TransactionPool) addTransaction(tx *Transaction) {
	for _, vin := range tx.Vin {
		parentTx, exist := txPool.txs[string(vin.Txid)]
		if exist {
			parentTx.children[string(tx.ID)] = tx
		}
	}

	txNode := TransactionNode{children: make(map[string]*Transaction), value: tx}
	txPool.txs[string(tx.ID)] = &txNode

	if minTx, exist := txPool.txs[txPool.minTipTxId]; exist {
		if tx.Tip < minTx.value.Tip {
			txPool.minTipTxId = string(tx.ID)
		}
	} else {
		txPool.resetMinTipTransaction()
	}

	txPool.EventBus.Publish(NewTransactionTopic, tx)
}

func (txPool *TransactionPool) resetMinTipTransaction() {
	var minTip uint64 = math.MaxUint64
	txPool.minTipTxId = ""
	for txId, txNode := range txPool.txs {
		if txNode.value.Tip < minTip {
			minTip = txNode.value.Tip
			txPool.minTipTxId = txId
		}
	}
}
