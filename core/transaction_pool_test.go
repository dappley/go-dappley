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
	"github.com/dappley/go-dappley/common"
)

var tx1 = Transaction{
	ID:   util.GenerateRandomAoB(1),
	Vin:  GenerateFakeTxInputs(),
	Vout: GenerateFakeTxOutputs(),
	Tip:  common.NewAmount(2),
}
var tx2 = Transaction{
	ID:   util.GenerateRandomAoB(1),
	Vin:  GenerateFakeTxInputs(),
	Vout: GenerateFakeTxOutputs(),
	Tip:  common.NewAmount(5),
}
var tx3 = Transaction{
	ID:   util.GenerateRandomAoB(1),
	Vin:  GenerateFakeTxInputs(),
	Vout: GenerateFakeTxOutputs(),
	Tip:  common.NewAmount(10),
}
var tx4 = Transaction{
	ID:   util.GenerateRandomAoB(1),
	Vin:  GenerateFakeTxInputs(),
	Vout: GenerateFakeTxOutputs(),
	Tip:  common.NewAmount(20),
}

var expectPopOrder = []*common.Amount{common.NewAmount(20), common.NewAmount(10), common.NewAmount(5), common.NewAmount(2)}

var popInputOrder = []struct {
	order []*Transaction
}{
	{[]*Transaction{&tx4, &tx3, &tx2, &tx1}},
	{[]*Transaction{&tx1, &tx2, &tx3, &tx4}},
	{[]*Transaction{&tx2, &tx1, &tx4, &tx3}},
	{[]*Transaction{&tx4, &tx1, &tx3, &tx2}},
}

func TestTransactionPool_Push(t *testing.T) {
	txPool := NewTransactionPool(128)
	txPool.Push(&tx1)
	assert.Equal(t, 1, len(txPool.GetTransactions()))
	txPool.Push(&tx2)
	assert.Equal(t, 2, len(txPool.GetTransactions()))
	txPool.Push(&tx3)
	txPool.Push(&tx4)
	assert.Equal(t, 4, len(txPool.GetTransactions()))
}

func TestTransactionPoolLimit(t *testing.T) {
	txPool := NewTransactionPool(0)
	txPool.Push(&tx1)
	assert.Equal(t, 0, len(txPool.GetTransactions()))

	txPool = NewTransactionPool(1)
	txPool.Push(&tx1)
	txPool.Push(&tx2) // Note: t2 has higher tips and should be kept in pool in place of t1
	assert.Equal(t, 1, len(txPool.GetTransactions()))
	assert.Equal(t, tx2, *(txPool.GetTransactions()[0]))

	txPool.Push(&tx4) // Note: t4 has higher tips and should be kept in pool in place of t2
	assert.Equal(t, 1, len(txPool.GetTransactions()))
	assert.Equal(t, tx4, *(txPool.GetTransactions()[0]))

	txPool.Push(&tx3) // Note: t3 has less tips and should be discarded
	assert.Equal(t, 1, len(txPool.GetTransactions()))
	assert.Equal(t, tx4, *(txPool.GetTransactions()[0]))
}

func TestTransactionPool_Pop(t *testing.T) {
	for _, tt := range popInputOrder {
		var popOrder []*common.Amount
		txPool := NewTransactionPool(128)
		for _, tx := range tt.order {
			txPool.Push(tx)
		}

		txs := txPool.GetAndResetTransactions()

		for _, tx := range txs {
			popOrder = append(popOrder, tx.Tip)
		}
		//assert.Equal(t, expectPopOrder, popOrder)
	}
}

func TestTransactionPool_RemoveMultipleTransactions(t *testing.T) {
	txPool := NewTransactionPool(128)
	totalTx := 5
	var txs []*Transaction
	for i := 0; i < totalTx; i++ {
		tx := MockTransaction()
		txs = append(txs, tx)
		txPool.Push(tx)
	}
	txPool.CheckAndRemoveTransactions(txs)

	assert.Equal(t, 0, len(txPool.GetTransactions()))

}

func TestTransactionPool_GetTransactions(t *testing.T) {
	var prikey1 = "bb23d2ff19f5b16955e8a24dca34dd520980fe3bddca2b3e1b56663f0ec1aa71"
	var pubkey1 = GetKeyPairByString(prikey1).PublicKey
	var contractPubkeyHash = NewContractPubKeyHash()

	var deploymentTx = Transaction{
		ID: nil,
		Vin: []TXInput{
			{tx1.ID, 1, nil, pubkey1},
		},
		Vout: []TXOutput{
			{common.NewAmount(5), contractPubkeyHash, "dapp_schedule"},
		},
		Tip: common.NewAmount(1),
	}
	deploymentTx.ID = deploymentTx.Hash()

	var executionTx = Transaction{
		ID: nil,
		Vin: GenerateFakeTxInputs(),
		Vout: []TXOutput{
			{common.NewAmount(5), contractPubkeyHash, "execution"},
		},
		Tip: common.NewAmount(2),
	}
	executionTx.ID = executionTx.Hash()

	txPool := NewTransactionPool(10)
	txPool.Push(&executionTx)
	txPool.Push(&deploymentTx)

	// deployment transaction should be ahead of execution transaction
	txs := txPool.GetTransactions()
	assert.Equal(t, &deploymentTx, txs[0])
	assert.Equal(t, &executionTx, txs[1])
}
