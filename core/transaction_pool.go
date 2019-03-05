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
	"encoding/hex"
	"github.com/dappley/go-dappley/core/pb"
	"github.com/dappley/go-dappley/storage"
	"github.com/golang/protobuf/proto"
	"sort"
	"sync"

	"github.com/asaskevich/EventBus"
	"github.com/golang-collections/collections/stack"
	logger "github.com/sirupsen/logrus"
)

const (
	NewTransactionTopic   = "NewTransaction"
	EvictTransactionTopic = "EvictTransaction"
	scheduleFuncName      = "dapp_schedule"
	TxPoolDbKey           = "txpool"
)

type TransactionPool struct {
	txs       map[string]*TransactionNode
	tipOrder  []string
	sizeLimit uint32
	currSize  uint32
	EventBus  EventBus.Bus
	mutex     sync.RWMutex
}

func NewTransactionPool(limit uint32) *TransactionPool {
	return &TransactionPool{
		txs:       make(map[string]*TransactionNode),
		tipOrder:  make([]string, 0),
		sizeLimit: limit,
		currSize:  0,
		EventBus:  EventBus.New(),
		mutex:     sync.RWMutex{},
	}
}

func (txPool *TransactionPool) DeepCopy() *TransactionPool {
	txPoolCopy := TransactionPool{
		txs:       make(map[string]*TransactionNode),
		tipOrder:  make([]string, len(txPool.tipOrder)),
		sizeLimit: txPool.sizeLimit,
		currSize:  0,
		EventBus:  EventBus.New(),
		mutex:     sync.RWMutex{},
	}

	copy(txPoolCopy.tipOrder, txPool.tipOrder)

	for key, tx := range txPool.txs {
		newTx := tx.Value.DeepCopy()
		newTxNode := NewTransactionNode(&newTx)

		for childKey, childTx := range tx.Children {
			newTxNode.Children[childKey] = childTx
		}
		txPoolCopy.txs[key] = newTxNode
	}

	return &txPoolCopy
}

func (txPool *TransactionPool) GetTransactions() []*Transaction {
	txPool.mutex.RLock()
	defer txPool.mutex.RUnlock()

	return txPool.getSortedTransactions()
}

func (txPool *TransactionPool) GetPoolSize() int {
	txPool.mutex.RLock()
	defer txPool.mutex.RUnlock()

	return len(txPool.txs)
}

//PopTransactionsWithMostTips pops the transactions with the most tips
func (txPool *TransactionPool) PopTransactionsWithMostTips(utxoIndex *UTXOIndex, blockLimit int) []*Transaction {

	tempUtxoIndex := utxoIndex.DeepCopy()
	var validTxs []*Transaction
	var inValidTxs []*Transaction
	totalSize := 0;

	for totalSize < blockLimit && len(txPool.txs) > 0 {

		txNode := txPool.GetMaxTipTransaction()
		if txNode==nil{
			logger.WithFields(logger.Fields{
				"num_txs_in_pool" : len(txPool.txs),
				"num_txs_in_order": len(txPool.tipOrder),
			}).Warn("Transaction Pool: Pop max tip transaction failed!")
			break;
		}

		totalSize += txNode.Size
		txPool.tipOrder = txPool.tipOrder[1:]
		if txNode.Value.Verify(tempUtxoIndex, 0) {

			validTxs = append(validTxs, txNode.Value)
			tempUtxoIndex.UpdateUtxo(txNode.Value)
			txPool.insertChildrenIntoSortedWaitlist(txNode)
			txPool.removeTransaction(txNode)
		} else {
			inValidTxs = append(inValidTxs, txNode.Value)
			txPool.removeTransactionNodeAndChildren(txNode.Value)
		}
	}

	return validTxs
}

func (txPool *TransactionPool) Push(tx *Transaction) {
	txPool.mutex.Lock()
	defer txPool.mutex.Unlock()
	if txPool.sizeLimit == 0 {
		logger.Warn("TransactionPool: transaction is not pushed to pool because sizeLimit is set to 0.")
		return
	}

	txNode := NewTransactionNode(tx)

	for txPool.currSize!=0 && txPool.currSize + uint32(txNode.Size) >= txPool.sizeLimit {
		logger.WithFields(logger.Fields{
			"sizeLimit": txPool.sizeLimit,
		}).Warn("TransactionPool: is full.")

		minTx := txPool.GetMinTipTransaction()
		if minTx != nil && txNode.GetTipsPerByte().Cmp(minTx.GetTipsPerByte()) < 1 {
			return
		}

		toRemoveTxs := txPool.getDependentTxs(minTx)
		if checkDependTxInMap(tx, toRemoveTxs) == true {
			logger.Warn("TransactionPool: failed to push because dependent transactions are not removed from pool.")
			return
		}

		txPool.removeMinTipTx()
	}

	txPool.addTransaction(txNode)

}

//CleanUpMinedTxs updates the transaction pool when a new block is added to the blockchain.
//It removes the packed transactions from the txpool while keeping their children
func (txPool *TransactionPool) CleanUpMinedTxs(minedTxs []*Transaction) {
	txPool.mutex.Lock()
	defer txPool.mutex.Unlock()

	for _, tx := range minedTxs {

		txNode, ok := txPool.txs[hex.EncodeToString(tx.ID)]
		if !ok {
			continue
		}
		txPool.insertChildrenIntoSortedWaitlist(txNode)
		txPool.removeTransaction(txNode)
	}
	txPool.cleanUpTxSort()
}

func (txPool *TransactionPool) cleanUpTxSort() {
	newTxOrder := []string{}
	for _, txid := range txPool.tipOrder {
		if _, ok := txPool.txs[txid]; ok {
			newTxOrder = append(newTxOrder, txid)
		}
	}
	txPool.tipOrder = newTxOrder
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
		if _, exist := existTxs[hex.EncodeToString(vin.Txid)]; exist {
			return true
		}
	}
	return false
}

func (txPool *TransactionPool) GetTransactionById(txid []byte) *Transaction {
	txPool.mutex.RLock()
	txPool.mutex.RUnlock()
	txNode, ok := txPool.txs[hex.EncodeToString(txid)]
	if !ok {
		return nil
	}
	return txNode.Value
}

func (txPool *TransactionPool) getDependentTxs(txNode *TransactionNode) map[string]*TransactionNode {

	toRemoveTxs := make(map[string]*TransactionNode)
	toCheckTxs := []*TransactionNode{txNode}

	for len(toCheckTxs) > 0 {
		currentTxNode := toCheckTxs[0]
		toCheckTxs = toCheckTxs[1:]
		for key, _ := range currentTxNode.Children {
			toCheckTxs = append(toCheckTxs, txPool.txs[key])
		}
		toRemoveTxs[hex.EncodeToString(currentTxNode.Value.ID)] = currentTxNode
	}

	return toRemoveTxs
}

// The param toRemoveTxs must be calculate by function getDependentTxs
func (txPool *TransactionPool) removeSelectedTransactions(toRemoveTxs map[string]*TransactionNode) {
	for _, txNode := range toRemoveTxs {
		txPool.removeTransactionNodeAndChildren(txNode.Value)
	}
}

//removeTransactionNodeAndChildren removes the txNode from tx pool and all its children.
//Note: this function does not remove the node from tipOrder!
func (txPool *TransactionPool) removeTransactionNodeAndChildren(tx *Transaction) {

	txStack := stack.New()
	txStack.Push(hex.EncodeToString(tx.ID))
	for txStack.Len() > 0 {
		txid := txStack.Pop().(string)
		currTxNode, ok := txPool.txs[txid]
		if !ok {
			continue
		}
		for _, child := range currTxNode.Children {
			txStack.Push(hex.EncodeToString(child.ID))
		}
		txPool.EventBus.Publish(EvictTransactionTopic, txPool.txs[txid].Value)
		txPool.removeTransaction(currTxNode)
	}
}

//removeTransactionNodeAndChildren removes the txNode from tx pool.
//Note: this function does not remove the node from tipOrder!
func (txPool *TransactionPool) removeTransaction(txNode *TransactionNode) {
	txPool.disconnectFromParent(txNode.Value)
	txPool.EventBus.Publish(EvictTransactionTopic, txPool.txs[hex.EncodeToString(txNode.Value.ID)].Value)
	txPool.currSize -= uint32(txNode.Size)
	delete(txPool.txs, hex.EncodeToString(txNode.Value.ID))

}

//disconnectFromParent removes itself from its parent's node's children field
func (txPool *TransactionPool) disconnectFromParent(tx *Transaction) {
	for _, vin := range tx.Vin {
		if parentTx, exist := txPool.txs[hex.EncodeToString(vin.Txid)]; exist {
			delete(parentTx.Children, hex.EncodeToString(tx.ID))
		}
	}
}

func (txPool *TransactionPool) removeMinTipTx() {
	minTipTx := txPool.GetMinTipTransaction()
	if minTipTx == nil {
		return
	}
	txPool.removeTransactionNodeAndChildren(minTipTx.Value)
	txPool.tipOrder = txPool.tipOrder[:len(txPool.tipOrder)-1]
}

func (txPool *TransactionPool) addTransaction(txNode *TransactionNode) {
	isDependentOnParent := false
	for _, vin := range txNode.Value.Vin {
		parentTx, exist := txPool.txs[hex.EncodeToString(vin.Txid)]
		if exist {
			parentTx.Children[hex.EncodeToString(txNode.Value.ID)] = txNode.Value
			isDependentOnParent = true
		}
	}

	txPool.txs[hex.EncodeToString(txNode.Value.ID)] = txNode
	txPool.currSize += uint32(txNode.Size)

	txPool.EventBus.Publish(NewTransactionTopic, txNode.Value)

	//if it depends on another tx in txpool, the transaction will be not be included in the sorted list
	if isDependentOnParent {
		return
	}

	txPool.insertIntoSortedWaitlist(txNode)
}

func (txPool *TransactionPool) insertChildrenIntoSortedWaitlist(txNode *TransactionNode) {
	for _, child := range txNode.Children {
		parentTxidsInTxPool := txPool.GetParentTxidsInTxPool(child)
		if len(parentTxidsInTxPool)==1 {
			txPool.insertIntoSortedWaitlist(txPool.txs[hex.EncodeToString(child.ID)])
		}
	}
}

func (txPool *TransactionPool) GetParentTxidsInTxPool(tx *Transaction) []string{
	txids := []string{}
	for _, vin := range tx.Vin {
		txidStr := hex.EncodeToString(vin.Txid)
		if _, exist := txPool.txs[txidStr]; exist {
			txids = append(txids, txidStr)
		}
	}
	return txids
}

//insertIntoSortedWaitlist insert a transaction into txSort based on tip.
//If the transaction is a child of another transaction, the transaction will NOT be inserted
func (txPool *TransactionPool) insertIntoSortedWaitlist(txNode *TransactionNode) {
	index := sort.Search(len(txPool.tipOrder), func(i int) bool {
		if txPool.txs[txPool.tipOrder[i]] == nil{
			logger.WithFields(logger.Fields{
				"txid": txPool.tipOrder[i],
				"len_of_tip_order": len(txPool.tipOrder),
				"len_of_txs": len(txPool.txs),
			}).Warn("TransactionPool: the transaction in tip order does not exist in txs!")
		}
		if txPool.txs[txPool.tipOrder[i]].Value == nil{
			logger.WithFields(logger.Fields{
				"txid": txPool.tipOrder[i],
				"len_of_tip_order": len(txPool.tipOrder),
				"len_of_txs": len(txPool.txs),
			}).Warn("TransactionPool: the transaction in tip order does not exist in txs!")
		}
		return txPool.txs[txPool.tipOrder[i]].GetTipsPerByte().Cmp(txNode.GetTipsPerByte()) == -1
	})

	txPool.tipOrder = append(txPool.tipOrder, "")
	copy(txPool.tipOrder[index+1:], txPool.tipOrder[index:])
	txPool.tipOrder[index] = hex.EncodeToString(txNode.Value.ID)
}

func deserializeTxPool(d []byte) *TransactionPool {

	txPoolProto := &corepb.TransactionPool{}
	err := proto.Unmarshal(d, txPoolProto)
	if err != nil {
		logger.WithError(err).Panic("TxPool: failed to deserialize TxPool transactions.")
	}
	txPool := NewTransactionPool(1)
	txPool.FromProto(txPoolProto)

	return txPool
}

func LoadTxPoolFromDatabase(db storage.Storage, txPoolSize uint32) *TransactionPool {
	rawBytes, err := db.Get([]byte(TxPoolDbKey))
	if err != nil && err.Error() == storage.ErrKeyInvalid.Error() || len(rawBytes) == 0 {
		return NewTransactionPool(txPoolSize)
	}
	txPool := deserializeTxPool(rawBytes)
	txPool.sizeLimit = txPoolSize
	return txPool
}

func (txPool *TransactionPool) serialize() []byte {

	rawBytes, err := proto.Marshal(txPool.ToProto())
	if err != nil {
		logger.WithError(err).Panic("TxPool: failed to serialize TxPool transactions.")
	}
	return rawBytes
}

func (txPool *TransactionPool) SaveToDatabase(db storage.Storage) error {
	txPool.mutex.Lock()
	defer txPool.mutex.Unlock()
	return db.Put([]byte(TxPoolDbKey), txPool.serialize())
}

//GetMinTipTransaction gets the transactionNode with minimum tip
func (txPool *TransactionPool) GetMaxTipTransaction() *TransactionNode {
	txid := txPool.GetMaxTipTxid()
	if txid == "" {
		return nil
	}
	if txPool.txs[txid] == nil {
		logger.WithFields(logger.Fields{
			"txid" : txid,
		}).Warn("TransactionPool: max tip transaction is not found in pool")
	}
	return txPool.txs[txid]
}

//GetMinTipTransaction gets the transactionNode with minimum tip
func (txPool *TransactionPool) GetMinTipTransaction() *TransactionNode {
	txid := txPool.GetMinTipTxid()
	if txid == "" {
		return nil
	}
	return txPool.txs[txid]
}

//GetMinTipTxid gets the txid of the transaction with minimum tip
func (txPool *TransactionPool) GetMaxTipTxid() string {
	if len(txPool.tipOrder) == 0 {
		logger.Warn("TransactionPool: nothing in the tip order")
		return ""
	}
	return txPool.tipOrder[0]
}

//GetMinTipTxid gets the txid of the transaction with minimum tip
func (txPool *TransactionPool) GetMinTipTxid() string {
	if len(txPool.tipOrder) == 0 {
		return ""
	}
	return txPool.tipOrder[len(txPool.tipOrder)-1]
}

func (txPool *TransactionPool) ToProto() proto.Message {
	txs := make(map[string]*corepb.TransactionNode)
	for key, val := range txPool.txs {
		txs[key] = val.ToProto().(*corepb.TransactionNode)
	}
	return &corepb.TransactionPool{
		Txs:      txs,
		TipOrder: txPool.tipOrder,
		CurrSize: txPool.currSize,
	}
}

func (txPool *TransactionPool) FromProto(pb proto.Message) {
	for key, val := range pb.(*corepb.TransactionPool).Txs {
		txNode := NewTransactionNode(nil)
		txNode.FromProto(val)
		txPool.txs[key] = txNode
	}
	txPool.tipOrder = pb.(*corepb.TransactionPool).TipOrder
	txPool.currSize = pb.(*corepb.TransactionPool).CurrSize
}




