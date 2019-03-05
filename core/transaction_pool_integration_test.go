// +build integration

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
	"github.com/dappley/go-dappley/common"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestTransactionPool_PopTransactionsWithMostTipsNoDependency(t *testing.T) {
	txPool := NewTransactionPool(1280000)
	utxoIndex := NewUTXOIndex()
	var kps []*KeyPair
	var pkhs []PubKeyHash
	var addrs []Address
	var prevUTXOs [][]*UTXO
	var txs []*Transaction

	for i:=0;i<5;i++{
		kps = append(kps, NewKeyPair())
		pkh, _ := NewUserPubKeyHash(kps[i].PublicKey)
		pkhs = append(pkhs, pkh)
		addrs = append(addrs, pkh.GenerateAddress())
		cbtx := NewCoinbaseTX(addrs[i], "", 1, common.NewAmount(0))
		utxoIndex.UpdateUtxo(&cbtx)
		prevUTXO := utxoIndex.GetAllUTXOsByPubKeyHash(pkhs[i])
		prevUTXOs = append(prevUTXOs, prevUTXO)
	}

	//Create 4 transactions that can pass the transaction verification
	for i:=0;i<4;i++{
		tx,err := NewUTXOTransaction(prevUTXOs[i], addrs[i], addrs[i+1], common.NewAmount(1), kps[i], common.NewAmount(uint64(i)), "")
		assert.Nil(t, err)
		txPool.Push(tx)
		txs = append(txs, &tx)
	}

	//pop out the transactions with most tips
	poppedTxs := txPool.PopTransactionsWithMostTips(utxoIndex, 660)
	assert.Equal(t, txs[3], poppedTxs[0])
	assert.Equal(t, txs[2], poppedTxs[1])
}

func TestTransactionPool_PopTransactionsWithMostTipsWithDependency(t *testing.T) {
	txPool := NewTransactionPool(1280000)
	utxoIndex := NewUTXOIndex()
	var kps []*KeyPair
	var pkhs []PubKeyHash
	var addrs []Address
	var txs []*Transaction

	for i:=0;i<5;i++{
		kps = append(kps, NewKeyPair())
		pkh, _ := NewUserPubKeyHash(kps[i].PublicKey)
		pkhs = append(pkhs, pkh)
		addrs = append(addrs, pkh.GenerateAddress())
		if i==0 {
			cbtx := NewCoinbaseTX(addrs[i], "", 1, common.NewAmount(0))
			utxoIndex.UpdateUtxo(&cbtx)
		}
	}
	tempUtxoIndex := utxoIndex.DeepCopy()
	//Create 4 transactions that can pass the transaction verification
	for i:=0;i<4;i++{
		prevUTXO := tempUtxoIndex.GetAllUTXOsByPubKeyHash(pkhs[i])
		tx,err := NewUTXOTransaction(prevUTXO, addrs[i], addrs[i+1], common.NewAmount(uint64(100-i*4)), kps[i], common.NewAmount(uint64(i)), "")
		assert.Nil(t, err)
		tempUtxoIndex.UpdateUtxo(&tx)
		txPool.Push(tx)
		txs = append(txs, &tx)
	}
	//pop out the transactions with most tips. Each tx is about 263 bytes
	poppedTxs := txPool.PopTransactionsWithMostTips(utxoIndex, 600)

	//tx 0 should be popped first since it is the parent of all other transactions
	assert.Equal(t, txs[0], poppedTxs[0])
	assert.Equal(t, txs[1], poppedTxs[1])
}