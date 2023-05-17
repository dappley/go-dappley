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
	"github.com/dappley/go-dappley/core/transaction"
	transactionpb "github.com/dappley/go-dappley/core/transaction/pb"
	"github.com/dappley/go-dappley/core/utxo"
	errval "github.com/dappley/go-dappley/errors"
	"github.com/dappley/go-dappley/logic/ltransaction"
	"github.com/dappley/go-dappley/logic/lutxo"
	"github.com/dappley/go-dappley/storage"
	"sort"
	"sync"

	"github.com/asaskevich/EventBus"
	"github.com/dappley/go-dappley/common/pubsub"

	"github.com/dappley/go-dappley/network/networkmodel"
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
	nonces     map[string]uint64 // map of tx addresses to nonces
	utxoCache  *utxo.UTXOCache
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
		nonces:     make(map[string]uint64),
		utxoCache:  utxo.NewUTXOCache(storage.NewRamStorage()),
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
		nonces:    make(map[string]uint64),
		utxoCache: txPool.utxoCache,
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

	for key, nonce := range txPool.nonces {
		txPoolCopy.nonces[key] = nonce
	}

	return &txPoolCopy
}

func (txPool *TransactionPool) SetUTXOCache(utxoCache *utxo.UTXOCache) {
	txPool.utxoCache = utxoCache
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

// PopTransactionWithMostTips pops the transactions with the most tips
func (txPool *TransactionPool) PopTransactionWithMostTips(utxoIndex *lutxo.UTXOIndex) (*transaction.TransactionNode, error) {
	txPool.mutex.Lock()
	defer txPool.mutex.Unlock()

	txNode := txPool.getMaxTipTransaction()
	if txNode == nil {
		return txNode, errval.TxNode
	}
	//remove the transaction from tip order
	txPool.tipOrder = txPool.tipOrder[1:]

	if err := ltransaction.VerifyTransaction(utxoIndex, txNode.Value, 0); err == nil {
		txPool.insertChildrenIntoSortedWaitlist(txNode)
		txPool.removeTransaction(txNode)
	} else if err == errval.TXInputNotFound {
		// The parent transaction might not have arrived yet, skip the transaction. It will be added back to the tip order if its parent is successfully used.
		logger.WithError(err).Warn("Transaction Pool: Pop max tip transaction failed!")
		return nil, nil
	} else {
		logger.WithError(err).Warn("Transaction Pool: Pop max tip transaction failed! Removing transaction and children from tx pool...")
		txPool.removeTransaction(txNode)
		return nil, nil
	}

	txPool.pendingTxs = append(txPool.pendingTxs, txNode.Value)
	return txNode, nil
}

// Rollback adds a popped transaction back to the transaction pool. The existing transactions in txpool may be dependent on the input transactionbase.
// However, the input transaction should never be dependent on any transaction in the current pool
func (txPool *TransactionPool) Rollback(tx transaction.Transaction, nonce uint64) {
	txPool.mutex.Lock()
	defer txPool.mutex.Unlock()

	rollbackTxNode := transaction.NewTransactionNode(&tx)
	newTipOrder := []string{}

	// this remakes the tipOrder so that it doesn't include the children of the rolled-back tx
	for _, txid := range txPool.tipOrder {
		tipSender := txPool.txs[txid].Value.GetDefaultFromTransactionAccount().GetAddress().String()
		if tipSender != tx.GetDefaultFromTransactionAccount().GetAddress().String() {
			newTipOrder = append(newTipOrder, txid)
		}
	}

	txPool.tipOrder = newTipOrder

	txPool.nonces[hex.EncodeToString(tx.ID)] = nonce
	txPool.addTransaction(rollbackTxNode)
	txPool.insertIntoTipOrder(rollbackTxNode)
}

// updateChildren traverses through all transactions in transaction pool and find the input node's children
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

// Push pushes a new transaction into the pool.
// If the nonce is the same as another tx in the pool, the one with higher tips per byte will be kept.
func (txPool *TransactionPool) Push(tx transaction.Transaction, nonce uint64, utxoIndex *lutxo.UTXOIndex) {
	txPool.mutex.Lock()
	defer txPool.mutex.Unlock()
	if txPool.sizeLimit == 0 {
		logger.Warn("TransactionPool: transaction is not pushed to pool because sizeLimit is set to 0.")
		return
	}

	txNode := transaction.NewTransactionNode(&tx)

	// check for existing entry with the same nonce, and replace it if the new tx has more tips ber byte
	replaceTXID := ""
	senderTxs := txPool.getTransactionsFromAddress(tx.GetDefaultFromTransactionAccount().GetAddress().String())
	for _, senderTx := range senderTxs {
		if txPool.nonces[hex.EncodeToString(senderTx.ID)] == nonce {
			if txNode.GetTipsPerByte().Cmp(txPool.txs[hex.EncodeToString(senderTx.ID)].GetTipsPerByte()) == 1 {
				replaceTXID = hex.EncodeToString(senderTx.ID)
			}
		}
	}

	newSize := txPool.currSize + uint32(txNode.Size)
	if replaceTXID != "" {
		newSize -= uint32(txPool.txs[replaceTXID].Size)
	}
	if txPool.currSize != 0 && newSize >= txPool.sizeLimit {
		logger.WithFields(logger.Fields{
			"sizeLimit": txPool.sizeLimit,
		}).Warn("TransactionPool: is full.")
		return
	}

	if replaceTXID != "" {
		txPool.removeTransaction(txPool.txs[replaceTXID])
	}
	txPool.addTransactionAndSort(txNode, nonce, utxoIndex)
}

// CleanUpMinedTxs updates the transaction pool when a new block is added to the blockchain.
// It removes the packed transactions from the txpool while keeping their children
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

// getSortedTransactions returns the txPool's transactions so that transactions from a single account are always arranged by ascending nonce.
func (txPool *TransactionPool) getSortedTransactions() []*transaction.Transaction {
	txsByAddress := make(map[string][]*transaction.Transaction)
	for _, txNode := range txPool.txs {
		tx := txNode.Value
		address := tx.GetDefaultFromTransactionAccount().GetAddress().String()
		if _, exist := txsByAddress[address]; exist {
			index := sort.Search(len(txsByAddress[address]), func(i int) bool {
				compareNonce, ok := txPool.nonces[hex.EncodeToString(txsByAddress[address][i].ID)]
				if !ok {
					logger.Warnf("could not find nonce for tx %s", hex.EncodeToString(txsByAddress[address][i].ID))
					return false
				}
				txNonce, ok := txPool.nonces[hex.EncodeToString(tx.ID)]
				if !ok {
					logger.Warn("could not find nonce for tx %s", hex.EncodeToString(tx.ID))
					return false
				}
				return txNonce < compareNonce
			})

			txPool.tipOrder = append(txPool.tipOrder, "")
			copy(txPool.tipOrder[index+1:], txPool.tipOrder[index:])
			txPool.tipOrder[index] = hex.EncodeToString(txNode.Value.ID)
		} else {
			txsByAddress[address] = []*transaction.Transaction{tx}
		}
	}

	sortedTxs := make([]*transaction.Transaction, len(txPool.txs))
	for _, txs := range txsByAddress {
		sortedTxs = append(sortedTxs, txs...)
	}
	return sortedTxs
}

// TODO: temporary implementation. This could be optimized by storing a map of sender addresses instead of having to iterate through all transactions every time
// getTransactionsFromAddress returns the transactions from a given sender address, sorted by nonce in ascending order
func (txPool *TransactionPool) getTransactionsFromAddress(address string) []*transaction.Transaction {
	addressTxs := []*transaction.Transaction{}
	for _, txNode := range txPool.txs {
		tx := txNode.Value
		if tx.GetDefaultFromTransactionAccount().GetAddress().String() == address {
			if len(addressTxs) == 0 {
				addressTxs = append(addressTxs, tx)
			} else {
				index := sort.Search(len(addressTxs), func(i int) bool {
					compareNonce, ok := txPool.nonces[hex.EncodeToString(addressTxs[i].ID)]
					if !ok {
						logger.Warnf("could not find nonce for tx %s", hex.EncodeToString(addressTxs[i].ID))
						return false
					}
					txNonce, ok := txPool.nonces[hex.EncodeToString(tx.ID)]
					if !ok {
						logger.Warn("could not find nonce for tx %s", hex.EncodeToString(tx.ID))
						return false
					}
					return txNonce < compareNonce
				})

				addressTxs = append(addressTxs, &transaction.Transaction{})
				copy(addressTxs[index+1:], addressTxs[index:])
				addressTxs[index] = tx
			}
		}
	}
	return addressTxs
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

// removeTransactionNode removes the txNode from tx pool.
// Note: this function does not remove the node from tipOrder!
func (txPool *TransactionPool) removeTransaction(txNode *transaction.TransactionNode) {
	txPool.disconnectFromParent(txNode.Value)
	txPool.EventBus.Publish(EvictTransactionTopic, txNode.Value)
	txPool.currSize -= uint32(txNode.Size)
	MetricsTransactionPoolSize.Dec(1)
	delete(txPool.txs, hex.EncodeToString(txNode.Value.ID))
	delete(txPool.nonces, hex.EncodeToString(txNode.Value.ID))
}

// disconnectFromParent removes itself from its parent's node's children field
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
	txPool.removeTransaction(minTipTx)
	txPool.tipOrder = txPool.tipOrder[:len(txPool.tipOrder)-1]
}

func (txPool *TransactionPool) addTransactionAndSort(txNode *transaction.TransactionNode, nonce uint64, utxoIndex *lutxo.UTXOIndex) {
	txPool.nonces[hex.EncodeToString(txNode.Value.ID)] = nonce
	txPool.addTransaction(txNode)
	txPool.EventBus.Publish(NewTransactionTopic, txNode.Value)

	//if it depends on another tx in txpool, the transaction will be not be included in the sorted list
	lastNonce := utxoIndex.GetLastNonceByPubKeyHash(txNode.Value.GetDefaultFromTransactionAccount().GetPubKeyHash())
	if nonce != lastNonce+1 {
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
	addressTxs := txPool.getTransactionsFromAddress(txNode.Value.GetDefaultFromTransactionAccount().GetAddress().String())
	if len(addressTxs) > 0 {
		currNonce := txPool.nonces[hex.EncodeToString(txNode.Value.ID)]
		nextNonce := txPool.nonces[hex.EncodeToString(addressTxs[0].ID)]
		if nextNonce == currNonce+1 {
			txPool.insertIntoTipOrder(txPool.txs[hex.EncodeToString(addressTxs[0].ID)])
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

// insertIntoTipOrder insert a transaction into tipOrder based on tip.
// If the transaction is a child of another transaction, the transaction will NOT be inserted
func (txPool *TransactionPool) insertIntoTipOrder(txNode *transaction.TransactionNode) {
	index := sort.Search(len(txPool.tipOrder), func(i int) bool {
		if txPool.txs[txPool.tipOrder[i]] == nil {
			logger.WithFields(logger.Fields{
				"txid":             txPool.tipOrder[i],
				"len_of_tip_order": len(txPool.tipOrder),
				"len_of_txs":       len(txPool.txs),
			}).Warn("TransactionPool: the tip order does not exist in txs!")
			return false
		}
		if txPool.txs[txPool.tipOrder[i]].Value == nil {
			logger.WithFields(logger.Fields{
				"txid":             txPool.tipOrder[i],
				"len_of_tip_order": len(txPool.tipOrder),
				"len_of_txs":       len(txPool.txs),
			}).Warn("TransactionPool: the transaction in tip order does not exist in txs!")
			return false
		}
		return txPool.txs[txPool.tipOrder[i]].GetTipsPerByte().Cmp(txNode.GetTipsPerByte()) == -1
	})

	txPool.tipOrder = append(txPool.tipOrder, "")
	copy(txPool.tipOrder[index+1:], txPool.tipOrder[index:])
	txPool.tipOrder[index] = hex.EncodeToString(txNode.Value.ID)
}

// getMinTipTransaction gets the transaction.TransactionNode with minimum tip
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

// getMinTipTransaction gets the transaction.TransactionNode with minimum tip
func (txPool *TransactionPool) getMinTipTransaction() *transaction.TransactionNode {
	txid := txPool.getMinTipTxid()
	if txid == "" {
		return nil
	}
	return txPool.txs[txid]
}

// getMinTipTxid gets the txid of the transaction with minimum tip
func (txPool *TransactionPool) getMaxTipTxid() string {
	if len(txPool.tipOrder) == 0 {
		logger.Warn("TransactionPool: nothing in the tip order")
		return ""
	}
	return txPool.tipOrder[0]
}

// getMinTipTxid gets the txid of the transaction with minimum tip
func (txPool *TransactionPool) getMinTipTxid() string {
	if len(txPool.tipOrder) == 0 {
		return ""
	}
	return txPool.tipOrder[len(txPool.tipOrder)-1]
}

func (txPool *TransactionPool) BroadcastTx(tx *transaction.Transaction, nonce uint64) {
	nonceTx := transaction.NewNonceTransaction(tx, nonce)
	txPool.netService.BroadcastNormalPriorityCommand(BroadcastTx, nonceTx.ToProto())
}

func (txPool *TransactionPool) BroadcastTxHandler(input interface{}) {
	var command *networkmodel.DappRcvdCmdContext
	command = input.(*networkmodel.DappRcvdCmdContext)

	//TODO: Check if the blockchain state is ready
	txpb := &transactionpb.NonceTransaction{}

	if err := proto.Unmarshal(command.GetData(), txpb); err != nil {
		logger.Warn(err)
	}

	nonceTx := &transaction.NonceTransaction{}
	nonceTx.FromProto(txpb)
	tx := nonceTx.GetTransaction()
	//TODO: Check if the transaction is generated from running a smart contract
	//utxoIndex := lutxo.NewUTXOIndex(n.GetBlockchain().GetUtxoCache())
	//if tx.IsFromContract(utxoIndex) {
	//	return
	//}

	tx.CreateTime = -1
	txPool.Push(*tx, nonceTx.GetNonce(), lutxo.NewUTXOIndex(txPool.utxoCache))

	if command.IsBroadcast() {
		//relay the original command
		txPool.netService.Relay(command.GetCommand(), networkmodel.PeerInfo{}, networkmodel.NormalPriorityCommand)
	}
}

func (txPool *TransactionPool) BroadcastBatchTxs(txs []transaction.Transaction, nonces []uint64) {

	if len(txs) == 0 || (len(txs) != len(nonces)) {
		return
	}

	transactions := transaction.NewNonceTransactions(txs, nonces)

	txPool.netService.BroadcastNormalPriorityCommand(BroadcastBatchTxs, transactions.ToProto())
}

func (txPool *TransactionPool) BroadcastBatchTxsHandler(input interface{}) {

	var command *networkmodel.DappRcvdCmdContext
	command = input.(*networkmodel.DappRcvdCmdContext)

	//TODO: Check if the blockchain state is ready
	txspb := &transactionpb.NonceTransactions{}

	if err := proto.Unmarshal(command.GetData(), txspb); err != nil {
		logger.Warn(err)
	}

	nonceTxs := &transaction.NonceTransactions{}
	nonceTxs.FromProto(txspb)

	for _, nonceTx := range nonceTxs.GetTransactions() {
		tx := nonceTx.GetTransaction()
		//TODO: Check if the transaction is generated from running a smart contract
		//utxoIndex := lutxo.NewUTXOIndex(n.GetBlockchain().GetUtxoCache())
		//if tx.IsFromContract(utxoIndex) {
		//	return
		//}
		tx.CreateTime = -1
		txPool.Push(*tx, nonceTx.GetNonce(), lutxo.NewUTXOIndex(txPool.utxoCache))
	}

	if command.IsBroadcast() {
		//relay the original command
		txPool.netService.Relay(command.GetCommand(), networkmodel.PeerInfo{}, networkmodel.NormalPriorityCommand)
	}

}
