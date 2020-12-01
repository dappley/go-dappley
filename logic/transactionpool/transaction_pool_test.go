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
	"encoding/hex"
	"testing"

	"github.com/dappley/go-dappley/core/transaction"
	"github.com/dappley/go-dappley/core/transactionbase"

	"github.com/dappley/go-dappley/core/account"
	"github.com/golang/protobuf/proto"

	"github.com/dappley/go-dappley/common"
	"github.com/dappley/go-dappley/util"
	"github.com/stretchr/testify/assert"
)

func getAoB(length int64) []byte {
	return util.GenerateRandomAoB(length)
}

func GenerateFakeTxInputs() []transactionbase.TXInput {
	return []transactionbase.TXInput{
		{getAoB(2), 10, getAoB(2), getAoB(2)},
		{getAoB(2), 5, getAoB(2), getAoB(2)},
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

func TestTransactionPool_Push(t *testing.T) {

	txPool := NewTransactionPool(nil, 128000)
	txPool.Push(tx1)

	assert.Equal(t, 1, len(txPool.GetTransactions()))
	txPool.Push(tx2)
	assert.Equal(t, 2, len(txPool.GetTransactions()))
	txPool.Push(tx3)
	txPool.Push(tx4)
	assert.Equal(t, 4, len(txPool.GetTransactions()))

	newTxPool := NewTransactionPool(nil, 128000)
	var txs = []transaction.Transaction{tx1, tx2, tx3, tx4}
	for _, tx := range txs {
		//txPointer := tx.DeepCopy()
		newTxPool.Push(tx) // &txPointer)
	}
	diffTxs := newTxPool.GetTransactions()
	for i := 0; i < 3; i++ {
		assert.NotEqual(t, diffTxs[i].ID, diffTxs[i+1].ID)
	}
}

func TestTransactionPool_addTransaction(t *testing.T) {

	txs := generateDependentTxs()

	txPool := NewTransactionPool(nil, 128)
	//push the first transactionbase. It should be in stored in txs and tipOrder
	txPool.addTransactionAndSort(transaction.NewTransactionNode(txs[0]))
	assert.Equal(t, 1, len(txPool.txs))
	assert.Equal(t, 1, len(txPool.tipOrder))
	assert.Equal(t, hex.EncodeToString(txs[0].ID), txPool.tipOrder[0])

	//push ttx1. It should be stored in txs. But it should not be in tipOrder since it is a child of ttx0
	txPool.addTransactionAndSort(transaction.NewTransactionNode(txs[1]))
	assert.Equal(t, 2, len(txPool.txs))
	assert.Equal(t, 1, len(txPool.tipOrder))
	assert.Equal(t, hex.EncodeToString(txs[0].ID), txPool.tipOrder[0])

	//push ttx2. It should be stored in txs. But it should not be in tipOrder since it is a child of ttx0
	txPool.addTransactionAndSort(transaction.NewTransactionNode(txs[2]))
	assert.Equal(t, 3, len(txPool.txs))
	assert.Equal(t, 1, len(txPool.tipOrder))
	assert.Equal(t, hex.EncodeToString(txs[0].ID), txPool.tipOrder[0])

	//push ttx3. It should be stored in txs. But it should not be in tipOrder since it is a child of ttx1
	txPool.addTransactionAndSort(transaction.NewTransactionNode(txs[3]))
	assert.Equal(t, 4, len(txPool.txs))
	assert.Equal(t, 1, len(txPool.tipOrder))
	assert.Equal(t, hex.EncodeToString(txs[0].ID), txPool.tipOrder[0])

	//push ttx4. It should be stored in txs and tipOrder
	txPool.addTransactionAndSort(transaction.NewTransactionNode(txs[4]))
	assert.Equal(t, 5, len(txPool.txs))
	assert.Equal(t, 2, len(txPool.tipOrder))
	//since ttx4 has a higher tip than ttx0, it should rank position 0 in tipOrder
	assert.Equal(t, hex.EncodeToString(txs[4].ID), txPool.tipOrder[0])
	assert.Equal(t, hex.EncodeToString(txs[0].ID), txPool.tipOrder[1])

	//push ttx5. It should be stored in txs. But it should not be in tipOrder since it is a child of ttx4
	txPool.addTransactionAndSort(transaction.NewTransactionNode(txs[5]))
	assert.Equal(t, 6, len(txPool.txs))
	assert.Equal(t, 2, len(txPool.tipOrder))
	//since ttx4 has a higher tip than ttx0, it should rank position 0 in tipOrder
	assert.Equal(t, hex.EncodeToString(txs[4].ID), txPool.tipOrder[0])
	assert.Equal(t, hex.EncodeToString(txs[0].ID), txPool.tipOrder[1])

	//push ttx6.  It should be stored in txs and tipOrder
	txPool.addTransactionAndSort(transaction.NewTransactionNode(txs[6]))
	assert.Equal(t, 7, len(txPool.txs))
	assert.Equal(t, 3, len(txPool.tipOrder))
	//since ttx4 has a higher tip than ttx0, it should rank position 0 in tipOrder
	assert.Equal(t, hex.EncodeToString(txs[6].ID), txPool.tipOrder[0])
	assert.Equal(t, hex.EncodeToString(txs[4].ID), txPool.tipOrder[1])
	assert.Equal(t, hex.EncodeToString(txs[0].ID), txPool.tipOrder[2])
	assert.Equal(t, uint32(339), txPool.currSize)
}

func TestTransactionPool_RemoveTransactionNodeAndChildren(t *testing.T) {
	txs := generateDependentTxs()
	txPool := NewTransactionPool(nil, 128)
	for _, tx := range txs {
		txPool.addTransactionAndSort(transaction.NewTransactionNode(tx))
	}
	//Since tx2 has no children, only tx2 will be removed
	txPool.removeTransactionNodeAndChildren(txs[2])
	assert.Equal(t, 7, len(txPool.txs))
	assert.Equal(t, uint32(437), txPool.currSize)
	//Since tx0 is the root, all txs wlil be removed
	txPool.removeTransactionNodeAndChildren(txs[0])
	assert.Equal(t, 4, len(txPool.txs))
	assert.Equal(t, uint32(300), txPool.currSize)
}

func TestTransactionPool_removeMinTipTx(t *testing.T) {
	txs := generateDependentTxs()
	txPool := NewTransactionPool(nil, 128)
	for _, tx := range txs {
		txPool.addTransactionAndSort(transaction.NewTransactionNode(tx))
	}
	//Since tx0 is the minimum tip, all children will be removed
	txPool.removeMinTipTx()
	assert.Equal(t, 4, len(txPool.txs))
	assert.Equal(t, hex.EncodeToString(txs[6].ID), txPool.tipOrder[0])
	assert.Equal(t, hex.EncodeToString(txs[4].ID), txPool.tipOrder[1])
	assert.Equal(t, hex.EncodeToString(txs[7].ID), txPool.tipOrder[2])
}

func TestTransactionPool_Update(t *testing.T) {
	txs := generateDependentTxs()
	txPool := NewTransactionPool(nil, 128)
	for _, tx := range txs {
		txPool.addTransactionAndSort(transaction.NewTransactionNode(tx))
	}

	//Since tx0 is the root, its children will be bumped up into the sorted list
	packedTxs := []*transaction.Transaction{txs[0]}
	txPool.CleanUpMinedTxs(packedTxs)
	assert.Equal(t, 7, len(txPool.txs))
	assert.Equal(t, 5, len(txPool.tipOrder))
	assert.Equal(t, hex.EncodeToString(txs[6].ID), txPool.tipOrder[0])
	assert.Equal(t, hex.EncodeToString(txs[4].ID), txPool.tipOrder[1])
	assert.Equal(t, hex.EncodeToString(txs[1].ID), txPool.tipOrder[2])
	assert.Equal(t, hex.EncodeToString(txs[7].ID), txPool.tipOrder[3])
	assert.Equal(t, hex.EncodeToString(txs[2].ID), txPool.tipOrder[4])
}

func TestTransactionPoolLimit(t *testing.T) {
	txPool := NewTransactionPool(nil, 0)
	txPool.Push(tx1)
	assert.Equal(t, 0, len(txPool.GetTransactions()))

	txPool = NewTransactionPool(nil, 1)
	txPool.Push(tx1)
	txPool.Push(tx2) // Note: t2 should be ignore
	assert.Equal(t, 1, len(txPool.GetTransactions()))
	assert.Equal(t, tx1, *(txPool.GetTransactions()[0]))

}

func TestTransactionPool_GetTransactions(t *testing.T) {
	var prikey1 = "bb23d2ff19f5b16955e8a24dca34dd520980fe3bddca2b3e1b56663f0ec1aa71"
	var pubkey1 = account.GenerateKeyPairByPrivateKey(prikey1).GetPublicKey()
	var contractAccount = account.NewContractTransactionAccount()

	var deploymentTx = transaction.Transaction{
		ID: nil,
		Vin: []transactionbase.TXInput{
			{tx1.ID, 1, nil, pubkey1},
		},
		Vout: []transactionbase.TXOutput{
			{common.NewAmount(5), contractAccount.GetPubKeyHash(), "dapp_schedule"},
		},
		Tip:      common.NewAmount(1),
		GasLimit: common.NewAmount(0),
		GasPrice: common.NewAmount(0),
		Type:     transaction.TxTypeContract,
	}
	deploymentTx.ID = deploymentTx.Hash()

	var executionTx = transaction.Transaction{
		ID:  nil,
		Vin: GenerateFakeTxInputs(),
		Vout: []transactionbase.TXOutput{
			{common.NewAmount(5), contractAccount.GetPubKeyHash(), "execution"},
		},
		Tip:      common.NewAmount(2),
		GasLimit: common.NewAmount(0),
		GasPrice: common.NewAmount(0),
		Type:     transaction.TxTypeContract,
	}
	executionTx.ID = executionTx.Hash()

	txPool := NewTransactionPool(nil, 100000)
	txPool.Push(executionTx)
	txPool.Push(deploymentTx)

	// deployment transaction should be ahead of execution transaction
	txs := txPool.GetTransactions()
	assert.Equal(t, &deploymentTx, txs[0])
	assert.Equal(t, &executionTx, txs[1])
}

func TestTransactionPool_Rollback(t *testing.T) {
	txs := generateDependentTxs()

	txPool := NewTransactionPool(nil, 128000)

	//only push tx1, tx2 and tx3
	for i := 1; i < 4; i++ {
		txPool.Push(*txs[i])
	}
	//the current structure in txpool should be:
	/*
		  tx1 tx2
		  /
		tx3
	*/
	//rollback tx0 into the txpool
	txPool.Rollback(*txs[0])
	//the current structure in txpool should be:
	/*
		        tx0
		        / \
			  tx1 tx2
			  /
			tx3
	*/

	assert.Equal(t, 4, len(txPool.txs))
	assert.Equal(t, 1, len(txPool.tipOrder))
	tx0Id := hex.EncodeToString(txs[0].ID)
	assert.Equal(t, tx0Id, txPool.tipOrder[0])
	assert.Equal(t, 2, len(txPool.txs[tx0Id].Children))
}

func generateDependentTxs() []*transaction.Transaction {

	//generate 7 txs that has dependency relationships like the graph below
	/*
				tx0         tx4      tx6     tx7
				/ \         /
		      tx1 tx2     tx5
		      /
		    tx3
	*/

	//size 60
	ttx0 := &transaction.Transaction{
		ID:   util.GenerateRandomAoB(5),
		Vin:  GenerateFakeTxInputs(),
		Vout: GenerateFakeTxOutputs(),
		Tip:  common.NewAmount(3000),
	}

	//size 37
	ttx1 := &transaction.Transaction{
		ID:   util.GenerateRandomAoB(5),
		Vin:  []transactionbase.TXInput{{Txid: ttx0.ID}},
		Vout: GenerateFakeTxOutputs(),
		Tip:  common.NewAmount(2000),
	}

	//size 37
	ttx2 := &transaction.Transaction{
		ID:   util.GenerateRandomAoB(5),
		Vin:  []transactionbase.TXInput{{Txid: ttx0.ID}},
		Vout: GenerateFakeTxOutputs(),
		Tip:  common.NewAmount(1000),
	}

	//size 37
	ttx3 := &transaction.Transaction{
		ID:   util.GenerateRandomAoB(5),
		Vin:  []transactionbase.TXInput{{Txid: ttx1.ID}},
		Vout: GenerateFakeTxOutputs(),
		Tip:  common.NewAmount(2000),
	}

	//size 61
	ttx4 := &transaction.Transaction{
		ID:   util.GenerateRandomAoB(6),
		Vin:  GenerateFakeTxInputs(),
		Vout: GenerateFakeTxOutputs(),
		Tip:  common.NewAmount(4000),
	}

	//size 38
	ttx5 := &transaction.Transaction{
		ID:   util.GenerateRandomAoB(5),
		Vin:  []transactionbase.TXInput{{Txid: ttx4.ID}},
		Vout: GenerateFakeTxOutputs(),
		Tip:  common.NewAmount(5000),
	}

	//size 62
	ttx6 := &transaction.Transaction{
		ID:   util.GenerateRandomAoB(7),
		Vin:  GenerateFakeTxInputs(),
		Vout: GenerateFakeTxOutputs(),
		Tip:  common.NewAmount(6000),
	}

	//size 135
	ttx7 := &transaction.Transaction{
		ID:   util.GenerateRandomAoB(80),
		Vin:  GenerateFakeTxInputs(),
		Vout: GenerateFakeTxOutputs(),
		Tip:  common.NewAmount(7000),
	}
	return []*transaction.Transaction{ttx0, ttx1, ttx2, ttx3, ttx4, ttx5, ttx6, ttx7}
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

	txNode := transaction.NewTransactionNode(ttx1)
	assert.Equal(t, ttx1, txNode.Value)
	assert.Equal(t, 0, len(txNode.Children))
	assert.Equal(t, len(rawBytes), txNode.Size)
}
