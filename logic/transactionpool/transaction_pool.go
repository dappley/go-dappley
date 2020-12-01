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

package transactionpool

import (
	"bytes"
	"encoding/hex"
	"sort"
	"sync"

	"github.com/dappley/go-dappley/core/transaction"
	transactionpb "github.com/dappley/go-dappley/core/transaction/pb"
	"github.com/dappley/go-dappley/logic/ltransaction"
	"github.com/dappley/go-dappley/logic/lutxo"

	"github.com/asaskevich/EventBus"
	"github.com/dappley/go-dappley/common/pubsub"

	"github.com/dappley/go-dappley/network/networkmodel"
	"github.com/golang-collections/collections/stack"
	"github.com/golang/protobuf/proto"
	logger "github.com/sirupsen/logrus"
)

const (
	NewTransactionTopic   = "NewTransaction"
	EvictTransactionTopic = "EvictTransaction"

	TxPoolDbKey = "txpool"

	BroadcastTx       = "BroadcastTx"
	BroadcastBatchTxs = "BraodcastBatchTxs"
)

var (
	txPoolSubscribedTopics = []string{
		BroadcastTx,
		BroadcastBatchTxs,
	}
)

type TransactionPool struct {
	txs        map[string]*transaction.TransactionNode
	pendingTxs []*transaction.Transaction
	tipOrder   []string
	sizeLimit  uint32
	currSize   uint32
	EventBus   EventBus.Bus
	mutex      sync.RWMutex
	netService NetService
}

func NewTransactionPool(netService NetService, limit uint32) *TransactionPool {
	txPool := &TransactionPool{
		txs:        make(map[string]*transaction.TransactionNode),
		pendingTxs: make([]*transaction.Transaction, 0),
		tipOrder:   make([]string, 0),
		sizeLimit:  limit,
		currSize:   0,
		EventBus:   EventBus.New(),
		mutex:      sync.RWMutex{},
		netService: netService,
	}
	txPool.ListenToNetService()
	return txPool
}

func (txPool *TransactionPool) GetTipOrder() []string { return txPool.tipOrder }

func (txPool *TransactionPool) ListenToNetService() {
	if txPool.netService == nil {
		return
	}

	txPool.netService.Listen(txPool)
}

func (txPool *TransactionPool) GetSubscribedTopics() []string {
	return txPoolSubscribedTopics
}

func (txPool *TransactionPool) GetTopicHandler(topic string) pubsub.TopicHandler {

	switch topic {
	case BroadcastTx:
		return txPool.BroadcastTxHandler
	case BroadcastBatchTxs:
		return txPool.BroadcastBatchTxsHandler
	}
	return nil
}

func (txPool *TransactionPool) DeepCopy() *TransactionPool {
	txPoolCopy := TransactionPool{
		txs:       make(map[string]*transaction.TransactionNode),
		tipOrder:  make([]string, len(txPool.tipOrder)),
		sizeLimit: txPool.sizeLimit,
		currSize:  0,
		EventBus:  EventBus.New(),
		mutex:     sync.RWMutex{},
	}

	copy(txPoolCopy.tipOrder, txPool.tipOrder)

	for key, tx := range txPool.txs {
		newTx := tx.Value.DeepCopy()
		newTxNode := transaction.NewTransactionNode(&newTx)

		for childKey, childTx := range tx.Children {
			newTxNode.Children[childKey] = childTx
		}
		txPoolCopy.txs[key] = newTxNode
	}

	return &txPoolCopy
}

func (txPool *TransactionPool) SetSizeLimit(sizeLimit uint32) {
	txPool.sizeLimit = sizeLimit
}

func (txPool *TransactionPool) GetSizeLimit() uint32 {
	return txPool.sizeLimit
}

func (txPool *TransactionPool) GetTransactions() []*transaction.Transaction {
	txPool.mutex.RLock()
	defer txPool.mutex.RUnlock()
	return txPool.getSortedTransactions()
}

func (txPool *TransactionPool) GetNumOfTxInPool() int {
	txPool.mutex.RLock()
	defer txPool.mutex.RUnlock()

	return len(txPool.txs)
}

func (txPool *TransactionPool) ResetPendingTransactions() {
	txPool.mutex.Lock()
	defer txPool.mutex.Unlock()

	txPool.pendingTxs = make([]*transaction.Transaction, 0)
}

func (txPool *TransactionPool) GetAllTransactions() []*transaction.Transaction {
	txPool.mutex.RLock()
	defer txPool.mutex.RUnlock()

	txs := []*transaction.Transaction{}
	for _, tx := range txPool.pendingTxs {
		txs = append(txs, tx)
	}

	for _, tx := range txPool.getSortedTransactions() {
		txs = append(txs, tx)
	}

	return txs
}

//PopTransactionWithMostTips pops the transactions with the most tips
func (txPool *TransactionPool) PopTransactionWithMostTips(utxoIndex *lutxo.UTXOIndex) *transaction.TransactionNode {
	txPool.mutex.Lock()
	defer txPool.mutex.Unlock()

	txNode := txPool.getMaxTipTransaction()
	if txNode == nil {
		return txNode
	}
	//remove the transaction from tip order
	txPool.tipOrder = txPool.tipOrder[1:]

	if err := ltransaction.VerifyTransaction(utxoIndex, txNode.Value, 0); err == nil {
		txPool.insertChildrenIntoSortedWaitlist(txNode)
		txPool.removeTransaction(txNode)
	} else {
		logger.WithError(err).Warn("Transaction Pool: Pop max tip transaction failed!")
		txPool.removeTransactionNodeAndChildren(txNode.Value)
		return nil
	}

	txPool.pendingTxs = append(txPool.pendingTxs, txNode.Value)
	return txNode
}

//Rollback adds a popped transaction back to the transaction pool. The existing transactions in txpool may be dependent on the input transactionbase. However, the input transaction should never be dependent on any transaction in the current pool
func (txPool *TransactionPool) Rollback(tx transaction.Transaction) {
	txPool.mutex.Lock()
	defer txPool.mutex.Unlock()

	rollbackTxNode := transaction.NewTransactionNode(&tx)
	txPool.updateChildren(rollbackTxNode)
	newTipOrder := []string{}

	for _, txid := range txPool.tipOrder {
		if _, exist := rollbackTxNode.Children[txid]; !exist {
			newTipOrder = append(newTipOrder, txid)
		}
	}

	txPool.tipOrder = newTipOrder

	txPool.addTransaction(rollbackTxNode)
	txPool.insertIntoTipOrder(rollbackTxNode)

}

//updateChildren traverses through all transactions in transaction pool and find the input node's children
func (txPool *TransactionPool) updateChildren(node *transaction.TransactionNode) {
	for txid, txNode := range txPool.txs {
	loop:
		for _, vin := range txNode.Value.Vin {
			if bytes.Compare(vin.Txid, node.Value.ID) == 0 {
				node.Children[txid] = txNode.Value
				break loop
			}
		}
	}
}

//Push pushes a new transaction into the pool
func (txPool *TransactionPool) Push(tx transaction.Transaction) {
	txPool.mutex.Lock()
	defer txPool.mutex.Unlock()
	if txPool.sizeLimit == 0 {
		logger.Warn("TransactionPool: transaction is not pushed to pool because sizeLimit is set to 0.")
		return
	}

	txNode := transaction.NewTransactionNode(&tx)

	if txPool.currSize != 0 && txPool.currSize+uint32(txNode.Size) >= txPool.sizeLimit {
		logger.WithFields(logger.Fields{
			"sizeLimit": txPool.sizeLimit,
		}).Warn("TransactionPool: is full.")

		return
	}

	txPool.addTransactionAndSort(txNode)

}

//CleanUpMinedTxs updates the transaction pool when a new block is added to the blockchain.
//It removes the packed transactions from the txpool while keeping their children
func (txPool *TransactionPool) CleanUpMinedTxs(minedTxs []*transaction.Transaction) {
	txPool.mutex.Lock()
	defer txPool.mutex.Unlock()

	for _, tx := range minedTxs {

		txNode, ok := txPool.txs[hex.EncodeToString(tx.ID)]
		if !ok {
			continue
		}
		txPool.insertChildrenIntoSortedWaitlist(txNode)
		txPool.removeTransaction(txNode)
		txPool.removeFromTipOrder(tx.ID)
	}
}

func (txPool *TransactionPool) removeFromTipOrder(txID []byte) {
	key := hex.EncodeToString(txID)

	for index, value := range txPool.tipOrder {
		if value == key {
			txPool.tipOrder = append(txPool.tipOrder[:index], txPool.tipOrder[index+1:]...)
			return
		}
	}

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

func (txPool *TransactionPool) getSortedTransactions() []*transaction.Transaction {

	nodes := make(map[string]*transaction.TransactionNode)
	scDeploymentTxExists := make(map[string]bool)

	for key, node := range txPool.txs {
		nodes[key] = node
		ctx := ltransaction.NewTxContract(node.Value)
		if ctx != nil && !ctx.IsScheduleContract() {
			scDeploymentTxExists[ctx.GetContractPubKeyHash().GenerateAddress().String()] = true
		}
	}

	var sortedTxs []*transaction.Transaction
	for len(nodes) > 0 {
		for key, node := range nodes {
			if !checkDependTxInMap(node.Value, nodes) {
				ctx := ltransaction.NewTxContract(node.Value)
				if ctx != nil {
					ctxPkhStr := ctx.GetContractPubKeyHash().GenerateAddress().String()
					if ctx.IsScheduleContract() {
						if !scDeploymentTxExists[ctxPkhStr] {
							sortedTxs = append(sortedTxs, node.Value)
							delete(nodes, key)
						}
					} else {
						sortedTxs = append(sortedTxs, node.Value)
						delete(nodes, key)
						scDeploymentTxExists[ctxPkhStr] = false
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

func checkDependTxInMap(tx *transaction.Transaction, existTxs map[string]*transaction.TransactionNode) bool {
	for _, vin := range tx.Vin {
		if _, exist := existTxs[hex.EncodeToString(vin.Txid)]; exist {
			return true
		}
	}
	return false
}

func (txPool *TransactionPool) GetTransactionById(txid []byte) *transaction.Transaction {
	txPool.mutex.RLock()
	defer txPool.mutex.RUnlock()
	txNode, ok := txPool.txs[hex.EncodeToString(txid)]
	if !ok {
		return nil
	}
	return txNode.Value
}

func (txPool *TransactionPool) getDependentTxs(txNode *transaction.TransactionNode) map[string]*transaction.TransactionNode {

	toRemoveTxs := make(map[string]*transaction.TransactionNode)
	toCheckTxs := []*transaction.TransactionNode{txNode}

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

// The param toRemoveTxs must be calculated by function getDependentTxs
func (txPool *TransactionPool) removeSelectedTransactions(toRemoveTxs map[string]*transaction.TransactionNode) {
	for _, txNode := range toRemoveTxs {
		txPool.removeTransactionNodeAndChildren(txNode.Value)
	}
}

//removeTransactionNodeAndChildren removes the txNode from tx pool and all its children.
//Note: this function does not remove the node from tipOrder!
func (txPool *TransactionPool) removeTransactionNodeAndChildren(tx *transaction.Transaction) {

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
		txPool.removeTransaction(currTxNode)
	}
}

//removeTransactionNodeAndChildren removes the txNode from tx pool.
//Note: this function does not remove the node from tipOrder!
func (txPool *TransactionPool) removeTransaction(txNode *transaction.TransactionNode) {
	txPool.disconnectFromParent(txNode.Value)
	txPool.EventBus.Publish(EvictTransactionTopic, txNode.Value)
	txPool.currSize -= uint32(txNode.Size)
	MetricsTransactionPoolSize.Dec(1)
	delete(txPool.txs, hex.EncodeToString(txNode.Value.ID))
}

//disconnectFromParent removes itself from its parent's node's children field
func (txPool *TransactionPool) disconnectFromParent(tx *transaction.Transaction) {
	for _, vin := range tx.Vin {
		if parentTx, exist := txPool.txs[hex.EncodeToString(vin.Txid)]; exist {
			delete(parentTx.Children, hex.EncodeToString(tx.ID))
		}
	}
}

func (txPool *TransactionPool) removeMinTipTx() {
	minTipTx := txPool.getMinTipTransaction()
	if minTipTx == nil {
		return
	}
	txPool.removeTransactionNodeAndChildren(minTipTx.Value)
	txPool.tipOrder = txPool.tipOrder[:len(txPool.tipOrder)-1]
}

func (txPool *TransactionPool) addTransactionAndSort(txNode *transaction.TransactionNode) {
	isDependentOnParent := false
	for _, vin := range txNode.Value.Vin {
		parentTx, exist := txPool.txs[hex.EncodeToString(vin.Txid)]
		if exist {
			parentTx.Children[hex.EncodeToString(txNode.Value.ID)] = txNode.Value
			isDependentOnParent = true
		}
	}

	txPool.addTransaction(txNode)

	txPool.EventBus.Publish(NewTransactionTopic, txNode.Value)

	//if it depends on another tx in txpool, the transaction will be not be included in the sorted list
	if isDependentOnParent {
		return
	}

	txPool.insertIntoTipOrder(txNode)
}

func (txPool *TransactionPool) addTransaction(txNode *transaction.TransactionNode) {
	txPool.txs[hex.EncodeToString(txNode.Value.ID)] = txNode
	txPool.currSize += uint32(txNode.Size)
	MetricsTransactionPoolSize.Inc(1)
}

func (txPool *TransactionPool) insertChildrenIntoSortedWaitlist(txNode *transaction.TransactionNode) {
	for _, child := range txNode.Children {
		parentTxidsInTxPool := txPool.GetParentTxidsInTxPool(child)
		if len(parentTxidsInTxPool) == 1 {
			txPool.insertIntoTipOrder(txPool.txs[hex.EncodeToString(child.ID)])
		}
	}
}

func (txPool *TransactionPool) GetParentTxidsInTxPool(tx *transaction.Transaction) []string {
	txids := []string{}
	for _, vin := range tx.Vin {
		txidStr := hex.EncodeToString(vin.Txid)
		if _, exist := txPool.txs[txidStr]; exist {
			txids = append(txids, txidStr)
		}
	}
	return txids
}

//insertIntoTipOrder insert a transaction into txSort based on tip.
//If the transaction is a child of another transaction, the transaction will NOT be inserted
func (txPool *TransactionPool) insertIntoTipOrder(txNode *transaction.TransactionNode) {
	index := sort.Search(len(txPool.tipOrder), func(i int) bool {
		if txPool.txs[txPool.tipOrder[i]] == nil {
			logger.WithFields(logger.Fields{
				"txid":             txPool.tipOrder[i],
				"len_of_tip_order": len(txPool.tipOrder),
				"len_of_txs":       len(txPool.txs),
			}).Warn("TransactionPool: the transaction in tip order does not exist in txs!")
			return false
		}
		if txPool.txs[txPool.tipOrder[i]].Value == nil {
			logger.WithFields(logger.Fields{
				"txid":             txPool.tipOrder[i],
				"len_of_tip_order": len(txPool.tipOrder),
				"len_of_txs":       len(txPool.txs),
			}).Warn("TransactionPool: the transaction in tip order does not exist in txs!")
		}
		return txPool.txs[txPool.tipOrder[i]].GetTipsPerByte().Cmp(txNode.GetTipsPerByte()) == -1
	})

	txPool.tipOrder = append(txPool.tipOrder, "")
	copy(txPool.tipOrder[index+1:], txPool.tipOrder[index:])
	txPool.tipOrder[index] = hex.EncodeToString(txNode.Value.ID)
}


//getMinTipTransaction gets the transaction.TransactionNode with minimum tip
func (txPool *TransactionPool) getMaxTipTransaction() *transaction.TransactionNode {
	txid := txPool.getMaxTipTxid()
	if txid == "" {
		return nil
	}
	for txPool.txs[txid] == nil {
		logger.WithFields(logger.Fields{
			"txid": txid,
		}).Warn("TransactionPool: max tip transaction is not found in pool")
		txPool.tipOrder = txPool.tipOrder[1:]
		txid = txPool.getMaxTipTxid()
		if txid == "" {
			return nil
		}
	}
	return txPool.txs[txid]
}

//getMinTipTransaction gets the transaction.TransactionNode with minimum tip
func (txPool *TransactionPool) getMinTipTransaction() *transaction.TransactionNode {
	txid := txPool.getMinTipTxid()
	if txid == "" {
		return nil
	}
	return txPool.txs[txid]
}

//getMinTipTxid gets the txid of the transaction with minimum tip
func (txPool *TransactionPool) getMaxTipTxid() string {
	if len(txPool.tipOrder) == 0 {
		logger.Warn("TransactionPool: nothing in the tip order")
		return ""
	}
	return txPool.tipOrder[0]
}

//getMinTipTxid gets the txid of the transaction with minimum tip
func (txPool *TransactionPool) getMinTipTxid() string {
	if len(txPool.tipOrder) == 0 {
		return ""
	}
	return txPool.tipOrder[len(txPool.tipOrder)-1]
}


func (txPool *TransactionPool) BroadcastTx(tx *transaction.Transaction) {
	txPool.netService.BroadcastNormalPriorityCommand(BroadcastTx, tx.ToProto())
}

func (txPool *TransactionPool) BroadcastTxHandler(input interface{}) {

	var command *networkmodel.DappRcvdCmdContext
	command = input.(*networkmodel.DappRcvdCmdContext)

	//TODO: Check if the blockchain state is ready
	txpb := &transactionpb.Transaction{}

	if err := proto.Unmarshal(command.GetData(), txpb); err != nil {
		logger.Warn(err)
	}

	tx := &transaction.Transaction{}
	tx.FromProto(txpb)
	//TODO: Check if the transaction is generated from running a smart contract
	//utxoIndex := lutxo.NewUTXOIndex(n.GetBlockchain().GetUtxoCache())
	//if tx.IsFromContract(utxoIndex) {
	//	return
	//}

	tx.CreateTime = -1
	txPool.Push(*tx)

	if command.IsBroadcast() {
		//relay the original command
		txPool.netService.Relay(command.GetCommand(), networkmodel.PeerInfo{}, networkmodel.NormalPriorityCommand)
	}
}

func (txPool *TransactionPool) BroadcastBatchTxs(txs []transaction.Transaction) {

	if len(txs) == 0 {
		return
	}

	transactions := transaction.NewTransactions(txs)

	txPool.netService.BroadcastNormalPriorityCommand(BroadcastBatchTxs, transactions.ToProto())
}

func (txPool *TransactionPool) BroadcastBatchTxsHandler(input interface{}) {

	var command *networkmodel.DappRcvdCmdContext
	command = input.(*networkmodel.DappRcvdCmdContext)

	//TODO: Check if the blockchain state is ready
	txspb := &transactionpb.Transactions{}

	if err := proto.Unmarshal(command.GetData(), txspb); err != nil {
		logger.Warn(err)
	}

	txs := &transaction.Transactions{}

	//load the tx with proto
	txs.FromProto(txspb)

	for _, tx := range txs.GetTransactions() {
		//TODO: Check if the transaction is generated from running a smart contract
		//utxoIndex := lutxo.NewUTXOIndex(n.GetBlockchain().GetUtxoCache())
		//if tx.IsFromContract(utxoIndex) {
		//	return
		//}
		tx.CreateTime = -1
		txPool.Push(tx)
	}

	if command.IsBroadcast() {
		//relay the original command
		txPool.netService.Relay(command.GetCommand(), networkmodel.PeerInfo{}, networkmodel.NormalPriorityCommand)
	}

}
