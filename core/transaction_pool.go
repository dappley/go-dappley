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
	"encoding/gob"
	"sync"

	"github.com/asaskevich/EventBus"
	"github.com/dappley/go-dappley/common"
	"github.com/dappley/go-dappley/storage"
	logger "github.com/sirupsen/logrus"
)

const (
	NewTransactionTopic   = "NewTransaction"
	EvictTransactionTopic = "EvictTransaction"
	scheduleFuncName      = "dapp_schedule"
	TxPoolDbKey           = "txpool"
)

type TransactionNode struct {
	Children map[string]*Transaction
	Value    *Transaction
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
		newTx := tx.Value.DeepCopy()
		newTxNode := TransactionNode{Children: make(map[string]*Transaction), Value: &newTx}

		for childKey, childTx := range tx.Children {
			newTxNode.Children[childKey] = childTx
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

func (txPool *TransactionPool) GetFilteredTransactions(utxoIndex *UTXOIndex, blockHeight uint64) []*Transaction {
	txs := txPool.GetTransactions()
	tempUtxoCache := utxoIndex
	var validTxs []*Transaction
	var inValidTxs []*Transaction

	for _, tx := range txs {
		if tx.Verify(tempUtxoCache, blockHeight) {
			validTxs = append(validTxs, tx)
			tempUtxoCache.UpdateUtxo(tx)
		} else {
			inValidTxs = append(inValidTxs, tx)
		}
	}
	if len(inValidTxs) > 0 {
		txPool.CheckAndRemoveTransactions(inValidTxs)
	}

	return validTxs
}

func (txPool *TransactionPool) Push(tx Transaction) {
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
		if exist && tx.Tip.Cmp(minTx.Value.Tip) < 1 {
			return
		}

		toRemoveTxs := txPool.getToRemoveTxs(txPool.minTipTxId)
		if checkDependTxInMap(&tx, toRemoveTxs) {
			logger.Warn("TransactionPool: failed to push because dependent transactions are not removed from pool.")
			return
		}

		txPool.removeSelectedTransactions(toRemoveTxs)
		txPool.minTipTxId = ""
	}
	txPool.addTransaction(&tx)
}

func (txPool *TransactionPool) CheckAndRemoveTransactions(txs []*Transaction) {
	txPool.mutex.Lock()
	defer txPool.mutex.Unlock()

	for _, tx := range txs {
		txNode, ok := txPool.txs[string(tx.ID)]
		if !ok {
			continue
		}
		toRemoveTxs := map[string]*TransactionNode{string(tx.ID): txNode}
		txPool.removeSelectedTransactions(toRemoveTxs)
	}

	txPool.resetMinTipTransaction()
}

func (txPool *TransactionPool) getSortedTransactions() []*Transaction {
	nodes := make(map[string]*TransactionNode)
	isExecTxOkToInsert := true

	for key, node := range txPool.txs {
		nodes[key] = node
		if node.Value.IsContract() && !node.Value.IsExecutionContract() {
			isExecTxOkToInsert = false
		}
	}

	var sortedTxs []*Transaction
	for len(nodes) > 0 {
		for key, node := range nodes {
			if !checkDependTxInMap(node.Value, nodes) {
				if node.Value.IsContract() {
					if node.Value.IsExecutionContract() {
						if isExecTxOkToInsert {
							sortedTxs = append(sortedTxs, node.Value)
							delete(nodes, key)
						}
					} else {
						sortedTxs = append(sortedTxs, node.Value)
						delete(nodes, key)
						isExecTxOkToInsert = true
					}
				} else {
					sortedTxs = append(sortedTxs, node.Value)
					delete(nodes, key)
				}
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

func (txPool *TransactionPool) GetTransactionById(txid []byte) *Transaction {
	txPool.mutex.RLock()
	txPool.mutex.RUnlock()
	return txPool.txs[string(txid)].Value
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
		for key, _ := range currentTxNode.Children {
			toCheckTxs = append(toCheckTxs, txPool.txs[key])
		}
		toRemoveTxs[string(currentTxNode.Value.ID)] = currentTxNode
	}

	return toRemoveTxs
}

// The param toRemoveTxs must be calculate by function getToRemoveTxs
func (txPool *TransactionPool) removeSelectedTransactions(toRemoveTxs map[string]*TransactionNode) {
	for txId, txNode := range toRemoveTxs {
		for _, vin := range txNode.Value.Vin {
			parentTx, exist := txPool.txs[string(vin.Txid)]
			if exist {
				delete(parentTx.Children, txId)
			}
		}
		delete(txPool.txs, txId)
		txPool.EventBus.Publish(EvictTransactionTopic, txNode.Value)
	}
}

func (txPool *TransactionPool) addTransaction(tx *Transaction) {
	for _, vin := range tx.Vin {
		parentTx, exist := txPool.txs[string(vin.Txid)]
		if exist {
			parentTx.Children[string(tx.ID)] = tx
		}
	}

	txNode := TransactionNode{Children: make(map[string]*Transaction), Value: tx}
	txPool.txs[string(tx.ID)] = &txNode

	if minTx, exist := txPool.txs[txPool.minTipTxId]; exist {
		if tx.Tip.Cmp(minTx.Value.Tip) < 0 {
			txPool.minTipTxId = string(tx.ID)
		}
	} else {
		txPool.resetMinTipTransaction()
	}

	txPool.EventBus.Publish(NewTransactionTopic, tx)
}

func (txPool *TransactionPool) resetMinTipTransaction() {
	var minTip *common.Amount // = math.MaxUint64
	txPool.minTipTxId = ""
	first := true

	for txId, txNode := range txPool.txs {
		if first {
			first = false
			minTip = txNode.Value.Tip
			txPool.minTipTxId = txId
		} else {
			if txNode.Value.Tip.Cmp(minTip) < 0 {
				minTip = txNode.Value.Tip
				txPool.minTipTxId = txId
			}
		}
	}
}

func deserializeTxPool(d []byte) map[string]*TransactionNode {
	txs := make(map[string]*TransactionNode)
	decoder := gob.NewDecoder(bytes.NewReader(d))
	err := decoder.Decode(&txs)
	if err != nil {
		logger.WithError(err).Panic("TxPool: failed to deserialize TxPool transactions.")
	}
	return txs
}

func (txPool *TransactionPool) LoadFromDatabase(db storage.Storage) {
	txPool.mutex.Lock()
	defer txPool.mutex.Unlock()
	rawBytes, err := db.Get([]byte(TxPoolDbKey))
	if err != nil && err.Error() == storage.ErrKeyInvalid.Error() || len(rawBytes) == 0 {
		return
	}
	txPool.txs = deserializeTxPool(rawBytes)
}

func (txPool *TransactionPool) serialize() []byte {

	var encoded bytes.Buffer
	enc := gob.NewEncoder(&encoded)
	err := enc.Encode(txPool.txs)
	if err != nil {
		logger.WithError(err).Panic("TxPool: failed to serialize TxPool transactions.")
	}
	return encoded.Bytes()
}

func (txPool *TransactionPool) SaveToDatabase(db storage.Storage) error {
	txPool.mutex.Lock()
	defer txPool.mutex.Unlock()
	return db.Put([]byte(TxPoolDbKey), txPool.serialize())
}
