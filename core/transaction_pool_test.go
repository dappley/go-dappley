// Copyright (C) 2018 go-dappley authors
//
// This file is part of the go-dappley library.
//
// the go-dappley library is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
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

var t1 = Transaction{
	ID:   util.GenerateRandomAoB(1),
	Vin:  GenerateFakeTxInputs(),
	Vout: GenerateFakeTxOutputs(),
	Tip:  2,
}
var t2 = Transaction{
	ID:   util.GenerateRandomAoB(1),
	Vin:  GenerateFakeTxInputs(),
	Vout: GenerateFakeTxOutputs(),
	Tip:  5,
}
var t3 = Transaction{
	ID:   util.GenerateRandomAoB(1),
	Vin:  GenerateFakeTxInputs(),
	Vout: GenerateFakeTxOutputs(),
	Tip:  10,
}
var t4 = Transaction{
	ID:   util.GenerateRandomAoB(1),
	Vin:  GenerateFakeTxInputs(),
	Vout: GenerateFakeTxOutputs(),
	Tip:  20,
}

var expectPopOrder = []uint64{20, 10, 5, 2}

var popInputOrder = []struct {
	order []Transaction
}{
	{[]Transaction{t4, t3, t2, t1}},
	{[]Transaction{t1, t2, t3, t4}},
	{[]Transaction{t2, t1, t4, t3}},
	{[]Transaction{t4, t1, t3, t2}},
}

//transaction pool push function
func TestTxPoolPush(t *testing.T) {
	txPool := NewTransactionPool()
	txPool.Push(t1)
	assert.Equal(t, 1, txPool.Transactions.Len())
	txPool.Push(t2)
	assert.Equal(t, 2, txPool.Transactions.Len())
	txPool.Push(t3)
	txPool.Push(t4)
	assert.Equal(t, 4, txPool.Transactions.Len())
}

func TestTranstionPoolPop(t *testing.T) {
	for _, tt := range popInputOrder {
		var popOrder = []uint64{}
		txPool := NewTransactionPool()
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
	txPool := NewTransactionPool()
	totalTx:=5
	var txs []*Transaction
	for i:=0; i < totalTx; i++ {
		tx := MockTransaction()
		txs = append(txs, tx)
		txPool.Push(*tx)
	}
	txPool.RemoveMultipleTransactions(txs)

	assert.Equal(t,0, txPool.Transactions.Len())

}