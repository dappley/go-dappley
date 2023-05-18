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
	"errors"
	"fmt"
	"github.com/dappley/go-dappley/core/transaction"
	"github.com/dappley/go-dappley/core/transactionbase"
	"github.com/dappley/go-dappley/core/utxo"
	"github.com/dappley/go-dappley/storage"
	"reflect"
	"testing"

	"github.com/dappley/go-dappley/core/account"
	"github.com/golang/protobuf/proto"

	"github.com/dappley/go-dappley/common"
	"github.com/dappley/go-dappley/util"
	"github.com/stretchr/testify/assert"
)

func getAoB(length int64) []byte {
	return util.GenerateRandomAoB(length)
}

func findTransaction(txs []*transaction.Transaction, toFind *transaction.Transaction) (int, error) {
	for i, tx := range txs {
		if bytes.Equal(tx.ID, toFind.ID) {
			return i, nil
		}
	}
	return -1, errors.New("transaction not found")
}

func GenerateFakeTxInputs() []transactionbase.TXInput {
	return []transactionbase.TXInput{
		{getAoB(2), 10, getAoB(2), account.NewAccount().GetKeyPair().GetPublicKey()},
		{getAoB(2), 5, getAoB(2), account.NewAccount().GetKeyPair().GetPublicKey()},
	}
}

func GenerateFakeTxOutputs() []transactionbase.TXOutput {
	return []transactionbase.TXOutput{
		{common.NewAmount(1), account.PubKeyHash(getAoB(2)), ""},
		{common.NewAmount(2), account.PubKeyHash(getAoB(2)), ""},
	}
}

var tx1 = transaction.Transaction{
	ID:       util.GenerateRandomAoB(1),
	Vin:      GenerateFakeTxInputs(),
	Vout:     GenerateFakeTxOutputs(),
	Tip:      common.NewAmount(2),
	GasLimit: common.NewAmount(0),
	GasPrice: common.NewAmount(0),
}
var tx2 = transaction.Transaction{
	ID:       util.GenerateRandomAoB(1),
	Vin:      GenerateFakeTxInputs(),
	Vout:     GenerateFakeTxOutputs(),
	Tip:      common.NewAmount(5),
	GasLimit: common.NewAmount(0),
	GasPrice: common.NewAmount(0),
}
var tx3 = transaction.Transaction{
	ID:       util.GenerateRandomAoB(1),
	Vin:      GenerateFakeTxInputs(),
	Vout:     GenerateFakeTxOutputs(),
	Tip:      common.NewAmount(10),
	GasLimit: common.NewAmount(0),
	GasPrice: common.NewAmount(0),
}
var tx4 = transaction.Transaction{
	ID:       util.GenerateRandomAoB(1),
	Vin:      GenerateFakeTxInputs(),
	Vout:     GenerateFakeTxOutputs(),
	Tip:      common.NewAmount(20),
	GasLimit: common.NewAmount(0),
	GasPrice: common.NewAmount(0),
}

var expectPopOrder = []*common.Amount{common.NewAmount(20), common.NewAmount(10), common.NewAmount(5), common.NewAmount(2)}

var popInputOrder = []struct {
	order []*transaction.Transaction
}{
	{[]*transaction.Transaction{&tx4, &tx3, &tx2, &tx1}},
	{[]*transaction.Transaction{&tx1, &tx2, &tx3, &tx4}},
	{[]*transaction.Transaction{&tx2, &tx1, &tx4, &tx3}},
	{[]*transaction.Transaction{&tx4, &tx1, &tx3, &tx2}},
}

func TestTransactionPool_GetTopicHandler(t *testing.T) {
	txPool := NewTransactionPool(nil, 128000)

	broadcastTxExpected := reflect.ValueOf(txPool.BroadcastTxHandler).Pointer()
	broadcastTxActual := reflect.ValueOf(txPool.GetTopicHandler(BroadcastTx)).Pointer()
	assert.Equal(t, broadcastTxExpected, broadcastTxActual)

	broadcastBatchExpected := reflect.ValueOf(txPool.BroadcastBatchTxsHandler).Pointer()
	broadcastBatchActual := reflect.ValueOf(txPool.GetTopicHandler(BroadcastBatchTxs)).Pointer()
	assert.Equal(t, broadcastBatchExpected, broadcastBatchActual)

	assert.Nil(t, txPool.GetTopicHandler("not a topic"))
}

func TestTransactionPool_Push(t *testing.T) {
	db := storage.NewRamStorage()
	defer db.Close()

	txPool := NewTransactionPool(nil, 128000)
	txPool.SetUTXOCache(utxo.NewUTXOCache(db))
	txPool.Push(tx1, 1)

	assert.Equal(t, 1, len(txPool.GetTransactions()))
	txPool.Push(tx2, 1)
	assert.Equal(t, 2, len(txPool.GetTransactions()))
	txPool.Push(tx3, 1)
	txPool.Push(tx4, 1)
	assert.Equal(t, 4, len(txPool.GetTransactions()))

	newTxPool := NewTransactionPool(nil, 128000)
	db2 := storage.NewRamStorage()
	defer db2.Close()
	newTxPool.SetUTXOCache(utxo.NewUTXOCache(db2))
	var txs = []transaction.Transaction{tx1, tx2, tx3, tx4}
	for _, tx := range txs {
		//txPointer := tx.DeepCopy()
		newTxPool.Push(tx, 1) // &txPointer)
	}
	diffTxs := newTxPool.GetTransactions()
	for i := 0; i < 3; i++ {
		assert.NotEqual(t, diffTxs[i].ID, diffTxs[i+1].ID)
	}
	// TODO: test size limit check
	// TODO: test duplicate nonce handling
}

func TestTransactionPool_addTransaction(t *testing.T) {
	db := storage.NewRamStorage()
	defer db.Close()

	txs := generateDependentTxNodes()
	for _, tx := range txs {
		fmt.Println(tx.Size)
	}

	txPool := NewTransactionPool(nil, 128)
	txPool.SetUTXOCache(utxo.NewUTXOCache(db))
	//push the first transactionbase. It should be in stored in txs and tipOrder
	txPool.addTransactionAndSort(txs[0])
	assert.Equal(t, 1, len(txPool.txs))
	assert.Equal(t, 1, len(txPool.tipOrder))
	assert.Equal(t, hex.EncodeToString(txs[0].Value.ID), txPool.tipOrder[0])

	//push ttx1. It should be stored in txs. But it should not be in tipOrder since it is a child of ttx0
	txPool.addTransactionAndSort(txs[1])
	assert.Equal(t, 2, len(txPool.txs))
	assert.Equal(t, 1, len(txPool.tipOrder))
	assert.Equal(t, hex.EncodeToString(txs[0].Value.ID), txPool.tipOrder[0])

	//push ttx2. It should be stored in txs. But it should not be in tipOrder since it is a child of ttx0
	txPool.addTransactionAndSort(txs[2])
	assert.Equal(t, 3, len(txPool.txs))
	assert.Equal(t, 1, len(txPool.tipOrder))
	assert.Equal(t, hex.EncodeToString(txs[0].Value.ID), txPool.tipOrder[0])

	//push ttx3. It should be stored in txs. But it should not be in tipOrder since it is a child of ttx1
	txPool.addTransactionAndSort(txs[3])
	assert.Equal(t, 4, len(txPool.txs))
	assert.Equal(t, 1, len(txPool.tipOrder))
	assert.Equal(t, hex.EncodeToString(txs[0].Value.ID), txPool.tipOrder[0])

	//push ttx4. It should be stored in txs and tipOrder
	txPool.addTransactionAndSort(txs[4])
	assert.Equal(t, 5, len(txPool.txs))
	assert.Equal(t, 2, len(txPool.tipOrder))
	//since ttx4 has a higher tip than ttx0, it should rank position 0 in tipOrder
	assert.Equal(t, hex.EncodeToString(txs[4].Value.ID), txPool.tipOrder[0])
	assert.Equal(t, hex.EncodeToString(txs[0].Value.ID), txPool.tipOrder[1])

	//push ttx5. It should be stored in txs. But it should not be in tipOrder since it is a child of ttx4
	txPool.addTransactionAndSort(txs[5])
	assert.Equal(t, 6, len(txPool.txs))
	assert.Equal(t, 2, len(txPool.tipOrder))
	//since ttx4 has a higher tip than ttx0, it should rank position 0 in tipOrder
	assert.Equal(t, hex.EncodeToString(txs[4].Value.ID), txPool.tipOrder[0])
	assert.Equal(t, hex.EncodeToString(txs[0].Value.ID), txPool.tipOrder[1])

	//push ttx6.  It should be stored in txs and tipOrder
	txPool.addTransactionAndSort(txs[6])
	assert.Equal(t, 7, len(txPool.txs))
	assert.Equal(t, 3, len(txPool.tipOrder))
	//since ttx4 has a higher tip than ttx0, it should rank position 0 in tipOrder
	assert.Equal(t, hex.EncodeToString(txs[6].Value.ID), txPool.tipOrder[0])
	assert.Equal(t, hex.EncodeToString(txs[4].Value.ID), txPool.tipOrder[1])
	assert.Equal(t, hex.EncodeToString(txs[0].Value.ID), txPool.tipOrder[2])
	assert.Equal(t, uint32(975), txPool.currSize)
}

func TestTransactionPool_removeTransaction(t *testing.T) {
	db := storage.NewRamStorage()
	defer db.Close()

	txPool := NewTransactionPool(nil, 128000)
	txPool.SetUTXOCache(utxo.NewUTXOCache(db))
	txs := generateDependentTxNodes()

	for _, tx := range txs {
		txPool.Push(*tx.Value, tx.Nonce)
	}
	expectedCurrSize := txPool.currSize

	key := hex.EncodeToString(txs[1].Value.ID)
	node := txPool.txs[key]
	txPool.removeTransaction(node)

	// txPool.txs and currSize should be updated
	_, ok := txPool.txs[key]
	assert.False(t, ok)
	expectedCurrSize -= uint32(txs[1].Value.GetSize())
	assert.Equal(t, expectedCurrSize, txPool.currSize)
}

func TestTransactionPool_CleanUpMinedTxs(t *testing.T) {
	db := storage.NewRamStorage()
	defer db.Close()
	txs := generateDependentTxNodes()
	txPool := NewTransactionPool(nil, 128)
	txPool.SetUTXOCache(utxo.NewUTXOCache(db))
	for _, tx := range txs {
		txPool.addTransactionAndSort(tx)
	}

	// The tx from the same sender as tx0 with the next nonce will be bumped up into the tip order
	packedTxs := []*transaction.Transaction{txs[0].Value}
	txPool.CleanUpMinedTxs(packedTxs)
	assert.Equal(t, 7, len(txPool.txs))
	assert.Equal(t, 4, len(txPool.tipOrder))
	assert.Equal(t, hex.EncodeToString(txs[6].Value.ID), txPool.tipOrder[0])
	assert.Equal(t, hex.EncodeToString(txs[7].Value.ID), txPool.tipOrder[1])
	assert.Equal(t, hex.EncodeToString(txs[4].Value.ID), txPool.tipOrder[2])
	assert.Equal(t, hex.EncodeToString(txs[1].Value.ID), txPool.tipOrder[3])
}

func TestTransactionPoolLimit(t *testing.T) {
	db := storage.NewRamStorage()
	defer db.Close()
	txPool := NewTransactionPool(nil, 0)
	txPool.SetUTXOCache(utxo.NewUTXOCache(db))
	txPool.Push(tx1, 1)
	assert.Equal(t, 0, len(txPool.GetTransactions()))

	txPool = NewTransactionPool(nil, 1)
	txPool.Push(tx1, 1)
	txPool.Push(tx2, 1) // Note: t2 should be ignored
	assert.Equal(t, 1, len(txPool.GetTransactions()))
	assert.Equal(t, tx1, *(txPool.GetTransactions()[0]))
}

func TestTransactionPool_GetTransactions(t *testing.T) {
	db := storage.NewRamStorage()
	defer db.Close()
	txPool := NewTransactionPool(nil, 100000)
	txPool.SetUTXOCache(utxo.NewUTXOCache(db))

	txs := generateDependentTxNodes()
	for _, tx := range txs {
		txPool.Push(*tx.Value, tx.Nonce)
	}

	result := txPool.GetTransactions()
	assert.Equal(t, len(generateDependentTxNodes()), len(result))

	// all child transactions must come after their parent transactions
	txIndex0, err := findTransaction(result, txs[0].Value)
	assert.Nil(t, err)
	txIndex1, err := findTransaction(result, txs[1].Value)
	assert.Nil(t, err)
	assert.Greater(t, txIndex1, txIndex0)
	txIndex2, err := findTransaction(result, txs[2].Value)
	assert.Nil(t, err)
	assert.Greater(t, txIndex2, txIndex1)
	txIndex3, err := findTransaction(result, txs[3].Value)
	assert.Nil(t, err)
	assert.Greater(t, txIndex3, txIndex2)

	txIndex4, err := findTransaction(result, txs[4].Value)
	assert.Nil(t, err)
	txIndex5, err := findTransaction(result, txs[5].Value)
	assert.Nil(t, err)
	assert.Greater(t, txIndex5, txIndex4)
}

func TestTransactionPool_GetAllTransactions(t *testing.T) {
	db := storage.NewRamStorage()
	defer db.Close()
	txPool := NewTransactionPool(nil, 100000)
	txPool.SetUTXOCache(utxo.NewUTXOCache(db))

	txs := generateDependentTxNodes()
	for i := 0; i < 4; i++ {
		txPool.Push(*txs[i].Value, txs[i].Nonce)
	}
	for i := 4; i < 8; i++ {
		txPool.pendingTxs = append(txPool.pendingTxs, txs[i].Value)
	}

	result := txPool.GetAllTransactions()
	// pendingTxs were added first
	assert.Equal(t, txs[4].Value, result[0])
	assert.Equal(t, txs[5].Value, result[1])
	assert.Equal(t, txs[6].Value, result[2])
	assert.Equal(t, txs[7].Value, result[3])

	// 0 is the parent
	txIndex0, err := findTransaction(result, txs[0].Value)
	assert.Nil(t, err)
	assert.Equal(t, 4, txIndex0)
	// txs[3] must come after txs[1]
	txIndex1, err := findTransaction(result, txs[1].Value)
	assert.Nil(t, err)
	txIndex3, err := findTransaction(result, txs[3].Value)
	assert.Nil(t, err)
	assert.Greater(t, txIndex3, txIndex1)
}

func TestTransactionPool_Rollback(t *testing.T) {
	txs := generateDependentTxNodes()

	db := storage.NewRamStorage()
	defer db.Close()
	txPool := NewTransactionPool(nil, 128000)
	txPool.SetUTXOCache(utxo.NewUTXOCache(db))

	// only push tx1, tx2 and tx3
	for i := 1; i < 4; i++ {
		txPool.Push(*txs[i].Value, txs[i].Nonce)
	}
	// the current structure in txpool should be:
	/*
		tx1 -> tx2 -> tx3
	*/
	// rollback tx0 into the txpool
	txPool.Rollback(*txs[0].Value, txs[0].Nonce)
	// the current structure in txpool should be:
	/*
		tx0 -> tx1 -> tx2 -> tx3
	*/

	assert.Equal(t, 4, len(txPool.txs))
	assert.Equal(t, 1, len(txPool.tipOrder))
	tx0Id := hex.EncodeToString(txs[0].Value.ID)
	assert.Equal(t, tx0Id, txPool.tipOrder[0])
}

func TestTransactionPool_GetTransactionById(t *testing.T) {
	db := storage.NewRamStorage()
	defer db.Close()
	txPool := NewTransactionPool(nil, 128000)
	txPool.SetUTXOCache(utxo.NewUTXOCache(db))
	txPool.Push(tx1, 1)
	txPool.Push(tx2, 1)

	result := txPool.GetTransactionById(tx1.ID)
	assert.Equal(t, &tx1, result)

	result = txPool.GetTransactionById(tx2.ID)
	assert.Equal(t, &tx2, result)

	result = txPool.GetTransactionById([]byte("invalid"))
	assert.Nil(t, result)
}

func TestTransactionPool_insertChildrenIntoSortedWaitlist(t *testing.T) {
	db := storage.NewRamStorage()
	defer db.Close()
	txPool := NewTransactionPool(nil, 128000)
	txPool.SetUTXOCache(utxo.NewUTXOCache(db))
	txs := generateDependentTxNodes()

	for _, tx := range txs {
		txPool.Push(*tx.Value, tx.Nonce)
	}
	// only nodes that aren't children are added
	expected := []string{
		hex.EncodeToString(txs[6].Value.ID),
		hex.EncodeToString(txs[7].Value.ID),
		hex.EncodeToString(txs[4].Value.ID),
		hex.EncodeToString(txs[0].Value.ID),
	}
	assert.Equal(t, expected, txPool.tipOrder)

	txPool.insertChildrenIntoSortedWaitlist(txPool.txs[hex.EncodeToString(txs[0].Value.ID)])
	expected = []string{
		hex.EncodeToString(txs[6].Value.ID),
		hex.EncodeToString(txs[7].Value.ID),
		hex.EncodeToString(txs[4].Value.ID),
		hex.EncodeToString(txs[1].Value.ID), // child of txs[0] inserted
		hex.EncodeToString(txs[0].Value.ID),
	}
	assert.Equal(t, expected, txPool.tipOrder)
}

func TestTransactionPool_insertIntoTipOrder(t *testing.T) {
	db := storage.NewRamStorage()
	defer db.Close()
	txPool := NewTransactionPool(nil, 128000)
	txPool.SetUTXOCache(utxo.NewUTXOCache(db))
	txs := generateDependentTxNodes()

	// add while txPool.txs is empty
	for _, tx := range txs {
		txPool.insertIntoTipOrder(tx)
	}
	// tipOrder matches the order in which they were added (fallback case in search algorithm)
	for i, tx := range txs {
		assert.Equal(t, hex.EncodeToString(tx.Value.ID), txPool.tipOrder[i])
	}

	txPool.tipOrder = []string{}
	txsToAdd := []*transaction.TransactionNode{txs[0], txs[4], txs[6], txs[7]}
	for _, tx := range txsToAdd {
		txPool.addTransaction(tx)
		txPool.insertIntoTipOrder(tx)
	}
	// sorted in order of descending tips per byte
	expected := []string{
		hex.EncodeToString(txs[6].Value.ID),
		hex.EncodeToString(txs[7].Value.ID),
		hex.EncodeToString(txs[4].Value.ID),
		hex.EncodeToString(txs[0].Value.ID),
	}
	assert.Equal(t, expected, txPool.tipOrder)
}

func TestTransactionPool_removeFromTipOrder(t *testing.T) {
	db := storage.NewRamStorage()
	defer db.Close()
	txPool := NewTransactionPool(nil, 128000)
	txPool.SetUTXOCache(utxo.NewUTXOCache(db))
	txs := generateDependentTxNodes()

	for _, tx := range txs {
		txPool.Push(*tx.Value, tx.Nonce)
	}
	expected := []string{
		hex.EncodeToString(txs[6].Value.ID),
		hex.EncodeToString(txs[7].Value.ID),
		hex.EncodeToString(txs[4].Value.ID),
		hex.EncodeToString(txs[0].Value.ID),
	}
	assert.Equal(t, expected, txPool.tipOrder)

	txPool.removeFromTipOrder([]byte("nonexistent"))
	assert.Equal(t, expected, txPool.tipOrder)

	txPool.removeFromTipOrder(txs[7].Value.ID)
	expected = []string{
		hex.EncodeToString(txs[6].Value.ID),
		hex.EncodeToString(txs[4].Value.ID),
		hex.EncodeToString(txs[0].Value.ID),
	}
	assert.Equal(t, expected, txPool.tipOrder)

	txPool.removeFromTipOrder(txs[6].Value.ID)
	expected = []string{
		hex.EncodeToString(txs[4].Value.ID),
		hex.EncodeToString(txs[0].Value.ID),
	}
	assert.Equal(t, expected, txPool.tipOrder)

	txPool.removeFromTipOrder(txs[0].Value.ID)
	expected = []string{
		hex.EncodeToString(txs[4].Value.ID),
	}
	assert.Equal(t, expected, txPool.tipOrder)

	txPool.removeFromTipOrder(txs[4].Value.ID)
	assert.Equal(t, []string{}, txPool.tipOrder)
}

func TestTransactionPool_getMaxTipTxid(t *testing.T) {
	db := storage.NewRamStorage()
	defer db.Close()
	txPool := NewTransactionPool(nil, 128000)
	txPool.SetUTXOCache(utxo.NewUTXOCache(db))
	txs := generateDependentTxNodes()

	assert.Equal(t, "", txPool.getMaxTipTxid())

	for _, tx := range txs {
		txPool.Push(*tx.Value, tx.Nonce)
	}
	// txs[6] has the highest tips per byte
	assert.Equal(t, hex.EncodeToString(txs[6].Value.ID), txPool.getMaxTipTxid())

	txPool.removeFromTipOrder(txs[6].Value.ID)
	// txs[4] has the next highest tips per byte
	assert.Equal(t, hex.EncodeToString(txs[7].Value.ID), txPool.getMaxTipTxid())
}

func TestTransactionPool_getMaxTipTransaction(t *testing.T) {
	db := storage.NewRamStorage()
	defer db.Close()
	txPool := NewTransactionPool(nil, 128000)
	txPool.SetUTXOCache(utxo.NewUTXOCache(db))
	txs := generateDependentTxNodes()

	assert.Nil(t, txPool.getMaxTipTransaction())

	for _, tx := range txs {
		txPool.Push(*tx.Value, tx.Nonce)
	}

	assert.Equal(t, txs[6], txPool.getMaxTipTransaction())

	// ignore txs that are in txPool.tipOrder but not in txPool.txs
	txPool.removeTransaction(txs[6])
	txPool.removeTransaction(txs[4])
	assert.Equal(t, txs[7], txPool.getMaxTipTransaction())

	txPool.txs = make(map[string]*transaction.TransactionNode)
	assert.Nil(t, txPool.getMaxTipTransaction())
}

func generateDependentTxNodes() []*transaction.TransactionNode {

	//generate 8 txs that have nonces ordered as below
	/*
		sender1: tx0 -> tx1 -> tx2 -> tx3
		sender2: tx4 -> tx5
		sender3: tx6
		sender4: tx7
	*/

	//size 185, tips/byte 1621621
	ttx0 := &transaction.Transaction{
		ID:   util.GenerateRandomAoB(5),
		Vin:  GenerateFakeTxInputs(),
		Vout: GenerateFakeTxOutputs(),
		Tip:  common.NewAmount(3000),
	}

	//size 104, tips/byte 1923076
	ttx1 := &transaction.Transaction{
		ID:   util.GenerateRandomAoB(5),
		Vin:  []transactionbase.TXInput{{Txid: ttx0.ID, PubKey: ttx0.Vin[0].PubKey}},
		Vout: GenerateFakeTxOutputs(),
		Tip:  common.NewAmount(2000),
	}

	//size 104, tips/byte 961538
	ttx2 := &transaction.Transaction{
		ID:   util.GenerateRandomAoB(5),
		Vin:  []transactionbase.TXInput{{Txid: ttx0.ID, PubKey: ttx1.Vin[0].PubKey}},
		Vout: GenerateFakeTxOutputs(),
		Tip:  common.NewAmount(1000),
	}

	//size 104, tips/byte 1923076
	ttx3 := &transaction.Transaction{
		ID:   util.GenerateRandomAoB(5),
		Vin:  []transactionbase.TXInput{{Txid: ttx1.ID, PubKey: ttx2.Vin[0].PubKey}},
		Vout: GenerateFakeTxOutputs(),
		Tip:  common.NewAmount(2000),
	}

	//size 186, tips/byte 2150537
	ttx4 := &transaction.Transaction{
		ID:   util.GenerateRandomAoB(6),
		Vin:  GenerateFakeTxInputs(),
		Vout: GenerateFakeTxOutputs(),
		Tip:  common.NewAmount(4000),
	}

	//size 105, tips/byte 4761904
	ttx5 := &transaction.Transaction{
		ID:   util.GenerateRandomAoB(5),
		Vin:  []transactionbase.TXInput{{Txid: ttx4.ID, PubKey: ttx4.Vin[0].PubKey}},
		Vout: GenerateFakeTxOutputs(),
		Tip:  common.NewAmount(5000),
	}

	//size 187, tips/byte 3208556
	ttx6 := &transaction.Transaction{
		ID:   util.GenerateRandomAoB(7),
		Vin:  GenerateFakeTxInputs(),
		Vout: GenerateFakeTxOutputs(),
		Tip:  common.NewAmount(6000),
	}

	//size 260, tips/byte 2692307
	ttx7 := &transaction.Transaction{
		ID:   util.GenerateRandomAoB(80),
		Vin:  GenerateFakeTxInputs(),
		Vout: GenerateFakeTxOutputs(),
		Tip:  common.NewAmount(7000),
	}
	return []*transaction.TransactionNode{
		transaction.NewTransactionNode(ttx0, 1),
		transaction.NewTransactionNode(ttx1, 2),
		transaction.NewTransactionNode(ttx2, 3),
		transaction.NewTransactionNode(ttx3, 4),
		transaction.NewTransactionNode(ttx4, 1),
		transaction.NewTransactionNode(ttx5, 2),
		transaction.NewTransactionNode(ttx6, 1),
		transaction.NewTransactionNode(ttx7, 1)}
}

func TestNewTransactionNode(t *testing.T) {
	ttx1 := &transaction.Transaction{
		ID:   util.GenerateRandomAoB(5),
		Vin:  GenerateFakeTxInputs(),
		Vout: GenerateFakeTxOutputs(),
		Tip:  common.NewAmount(7),
	}

	rawBytes, err := proto.Marshal(ttx1.ToProto())
	assert.Nil(t, err)

	txNode := transaction.NewTransactionNode(ttx1, 1)
	assert.Equal(t, ttx1, txNode.Value)
	assert.Equal(t, 0, len(txNode.Children))
	assert.Equal(t, len(rawBytes), txNode.Size)
	assert.Equal(t, uint64(1), txNode.Nonce)
}

/* TODO: benchmark different conditions: all txs same sender, all txs different sender
func BenchmarkTransactionPool_GetTransactions(b *testing.B) {
	generateTxPool := func(n int) *TransactionPool {
		db := storage.NewRamStorage()
		defer db.Close()
		txPool := NewTransactionPool(nil, 128000000)
		txPool.SetUTXOCache(utxo.NewUTXOCache(db))
		var prevTxId []byte
		// generate a chain of dependent txs
		for i := 0; i < n; i++ {
			tx := transaction.Transaction{
				ID:   util.GenerateRandomAoB(5),
				Vin:  []transactionbase.TXInput{{Txid: prevTxId}},
				Vout: GenerateFakeTxOutputs(),
				Tip:  common.NewAmount(1)}
			txPool.Push(tx)
			prevTxId = tx.ID
		}
		return txPool
	}

	benchData := map[string]struct {
		n int
	}{
		"with 100 txs":       {n: 100},
		"with 1,000 txs":     {n: 1000},
		"with 10,000 txs":    {n: 10000},
		"with 50,000 txs":    {n: 50000},
		"with 100,000 txs":   {n: 100000},
		"with 1,000,000 txs": {n: 1000000},
	}
	b.ResetTimer()
	for benchName, data := range benchData {
		b.StopTimer()
		txPool := generateTxPool(data.n)

		b.StartTimer()
		b.Run(benchName, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				txPool.GetTransactions()
			}
		})
	}
}

func BenchmarkTransactionPool_Rollback(b *testing.B) {
	generateTxsToRollback := func(n int) []transaction.Transaction {
		txs := make([]transaction.Transaction, 0, n)
		var prevTxId []byte

		// generate a chain of dependent txs
		for i := 0; i < n; i++ {
			tx := transaction.Transaction{
				ID:   util.GenerateRandomAoB(5),
				Vin:  []transactionbase.TXInput{{Txid: prevTxId}},
				Vout: GenerateFakeTxOutputs(),
				Tip:  common.NewAmount(1)}
			txs = append(txs, tx)
			prevTxId = tx.ID
		}
		return txs
	}

	benchData := map[string]struct {
		n int
	}{
		"with 100 txs":    {n: 100},
		"with 1,000 txs":  {n: 1000},
		"with 10,000 txs": {n: 10000},
		//"with 50,000 txs": {n: 50000},
		//"with 100,000 txs":   {n: 100000},
		//"with 1,000,000 txs": {n: 1000000},
	}
	b.ResetTimer()
	for benchName, data := range benchData {
		b.StopTimer()
		txs := generateTxsToRollback(data.n)

		b.StartTimer()
		b.Run(benchName, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				txPool := NewTransactionPool(nil, 128000000)
				for j := len(txs) - 1; j >= 0; j-- {
					txPool.Rollback(txs[j])
				}
			}
		})
	}
}
*/
