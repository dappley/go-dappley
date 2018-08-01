package core

import (
	"testing"
	"github.com/stretchr/testify/assert"
	"github.com/dappley/go-dappley/common/sorted"
)

// Copyright (C) 2017 go-nebulas authors
//
// This file is part of the go-nebulas library.
//
// the go-nebulas library is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// the go-nebulas library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with the go-nebulas library.  If not, see <http://www.gnu.org/licenses/>.
//

//test slice using fake transactions
func TestSlice(t *testing.T) {
	t0 := Transaction{
		[]byte("1"),
		[]TXInput{},
		[]TXOutput{},
		10,
	}
	t1 := Transaction{
		[]byte("2"),
		[]TXInput{},
		[]TXOutput{},
		20,
	}
	t2 := Transaction{
		[]byte("3"),
		[]TXInput{},
		[]TXOutput{},
		30,
	}
	t3 := Transaction{
		[]byte("4"),
		[]TXInput{},
		[]TXOutput{},
		40,
	}
	txnPool := TransactionPoool{}
	txnPool.transactions = *sorted.NewSlice(CompareTransactionTips, txnPool.StructDelete, txnPool.StructPush)
	txnPool.transactions.Push(t2)
	txnPool.transactions.Push(t1)
	txnPool.transactions.Push(t3)

	assert.Equal(t, txnPool.transactions.Left().(Transaction).Tip, t1.Tip)
	assert.Equal(t, txnPool.transactions.Right().(Transaction).Tip, t3.Tip)
	assert.Equal(t, txnPool.transactions.PopLeft().(Transaction).Tip, t1.Tip)
	txnPool.transactions.StructDelete(t3)
	txnPool.transactions.Push(t0)
	assert.Equal(t, txnPool.transactions.Right().(Transaction).Tip, t2.Tip)
	assert.Equal(t, txnPool.transactions.Left().(Transaction).Tip, t0.Tip)
}

func TestTransactionPool_TraverseDoNothing(t *testing.T) {
	txPool := GenerateMockTransactionPool(5)

	txPool.Traverse(func(tx Transaction) bool{
		return true
	})
	assert.Equal(t,5,txPool.Len())
}

func TestTransactionPool_TraverseRemoveAllTx(t *testing.T) {
	txPool := GenerateMockTransactionPool(5)
	assert.Equal(t,5,txPool.Len())
	txPool.Traverse(func(tx Transaction) bool{
		return false
	})
	assert.Equal(t,0,txPool.Len())
}


