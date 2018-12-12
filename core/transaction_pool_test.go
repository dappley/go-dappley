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
	"testing"

	"github.com/dappley/go-dappley/util"
	"github.com/stretchr/testify/assert"
)

var tx1 = Transaction{
	ID:   util.GenerateRandomAoB(1),
	Vin:  GenerateFakeTxInputs(),
	Vout: GenerateFakeTxOutputs(),
	Tip:  2,
}
var tx2 = Transaction{
	ID:   util.GenerateRandomAoB(1),
	Vin:  GenerateFakeTxInputs(),
	Vout: GenerateFakeTxOutputs(),
	Tip:  5,
}
var tx3 = Transaction{
	ID:   util.GenerateRandomAoB(1),
	Vin:  GenerateFakeTxInputs(),
	Vout: GenerateFakeTxOutputs(),
	Tip:  10,
}
var tx4 = Transaction{
	ID:   util.GenerateRandomAoB(1),
	Vin:  GenerateFakeTxInputs(),
	Vout: GenerateFakeTxOutputs(),
	Tip:  20,
}

var expectPopOrder = []uint64{20, 10, 5, 2}

var popInputOrder = []struct {
	order []Transaction
}{
	{[]Transaction{tx4, tx3, tx2, tx1}},
	{[]Transaction{tx1, tx2, tx3, tx4}},
	{[]Transaction{tx2, tx1, tx4, tx3}},
	{[]Transaction{tx4, tx1, tx3, tx2}},
}

func TestTransactionPool_Push(t *testing.T) {
	txPool := NewTransactionPool(128)
	txPool.Push(tx1)
	assert.Equal(t, 1, txPool.Transactions.Len())
	txPool.Push(tx2)
	assert.Equal(t, 2, txPool.Transactions.Len())
	txPool.Push(tx3)
	txPool.Push(tx4)
	assert.Equal(t, 4, txPool.Transactions.Len())
}

func TestTransactionPoolLimit(t *testing.T) {
	txPool := NewTransactionPool(0)
	txPool.Push(tx1)
	assert.Equal(t, 0, txPool.Transactions.Len())

	txPool = NewTransactionPool(1)
	txPool.Push(tx1)
	txPool.Push(tx2) // Note: t2 has higher tips and should be kept in pool in place of t1
	assert.Equal(t, 1, txPool.Transactions.Len())
	assert.Equal(t, tx2, txPool.Transactions.Get()[0].(Transaction))

	txPool.Push(tx4) // Note: t4 has higher tips and should be kept in pool in place of t2
	assert.Equal(t, 1, txPool.Transactions.Len())
	assert.Equal(t, tx4, txPool.Transactions.Get()[0].(Transaction))

	txPool.Push(tx3) // Note: t3 has less tips and should be discarded
	assert.Equal(t, 1, txPool.Transactions.Len())
	assert.Equal(t, tx4, txPool.Transactions.Get()[0].(Transaction))
}

func TestTransactionPool_Pop(t *testing.T) {
	for _, tt := range popInputOrder {
		var popOrder []uint64
		txPool := NewTransactionPool(128)
		for _, tx := range tt.order {
			txPool.Transactions.Push(tx)
		}
		for txPool.Transactions.Len() > 0 {
			popOrder = append(popOrder, txPool.Transactions.PopRight().(Transaction).Tip)
		}
		assert.Equal(t, expectPopOrder, popOrder)
	}
}

func TestTransactionPool_RemoveMultipleTransactions(t *testing.T) {
	txPool := NewTransactionPool(128)
	totalTx := 5
	var txs []*Transaction
	for i := 0; i < totalTx; i++ {
		tx := MockTransaction()
		txs = append(txs, tx)
		txPool.Push(*tx)
	}
	txPool.RemoveMultipleTransactions(txs)

	assert.Equal(t, 0, txPool.Transactions.Len())

}
