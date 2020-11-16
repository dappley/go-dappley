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
	"testing"

	"github.com/dappley/go-dappley/core/transaction"
	"github.com/dappley/go-dappley/core/transactionbase"
	"github.com/dappley/go-dappley/core/utxo"
	"github.com/dappley/go-dappley/logic/ltransaction"
	"github.com/dappley/go-dappley/logic/lutxo"

	"github.com/dappley/go-dappley/common"
	"github.com/dappley/go-dappley/core/account"
	"github.com/dappley/go-dappley/storage"
	"github.com/stretchr/testify/assert"
)

func TestTransactionPool_VerifyDependentTransactions(t *testing.T) {
	var prikey1 = "bb23d2ff19f5b16955e8a24dca34dd520980fe3bddca2b3e1b56663f0ec1aa71"
	var ta1 = account.NewAccountByPrivateKey(prikey1)
	var prikey2 = "bb23d2ff19f5b16955e8a24dca34dd520980fe3bddca2b3e1b56663f0ec1aa72"
	var ta2 = account.NewAccountByPrivateKey(prikey2)
	var prikey3 = "bb23d2ff19f5b16955e8a24dca34dd520980fe3bddca2b3e1b56663f0ec1aa73"
	var ta3 = account.NewAccountByPrivateKey(prikey3)
	var prikey4 = "bb23d2ff19f5b16955e8a24dca34dd520980fe3bddca2b3e1b56663f0ec1aa74"
	var ta4 = account.NewAccountByPrivateKey(prikey4)
	var prikey5 = "bb23d2ff19f5b16955e8a24dca34dd520980fe3bddca2b3e1b56663f0ec1aa75"
	var ta5 = account.NewAccountByPrivateKey(prikey5)

	var dependentTx1 = &transaction.Transaction{
		ID: nil,
		Vin: []transactionbase.TXInput{
			{tx1.ID, 1, nil, ta1.GetKeyPair().GetPublicKey()},
		},
		Vout: []transactionbase.TXOutput{
			{common.NewAmount(5), ta1.GetPubKeyHash(), ""},
			{common.NewAmount(10), ta2.GetPubKeyHash(), ""},
		},
		Tip:  common.NewAmount(3),
		Type: transaction.TxTypeNormal,
	}
	dependentTx1.ID = dependentTx1.Hash()

	var dependentTx2 = &transaction.Transaction{
		ID: nil,
		Vin: []transactionbase.TXInput{
			{dependentTx1.ID, 1, nil, ta2.GetKeyPair().GetPublicKey()},
		},
		Vout: []transactionbase.TXOutput{
			{common.NewAmount(5), ta3.GetPubKeyHash(), ""},
			{common.NewAmount(3), ta4.GetPubKeyHash(), ""},
		},
		Tip: common.NewAmount(2),
		Type: transaction.TxTypeNormal,
	}
	dependentTx2.ID = dependentTx2.Hash()

	var dependentTx3 = &transaction.Transaction{
		ID: nil,
		Vin: []transactionbase.TXInput{
			{dependentTx2.ID, 0, nil, ta3.GetKeyPair().GetPublicKey()},
		},
		Vout: []transactionbase.TXOutput{
			{common.NewAmount(1), ta4.GetPubKeyHash(), ""},
		},
		Tip: common.NewAmount(4),
		Type: transaction.TxTypeNormal,
	}
	dependentTx3.ID = dependentTx3.Hash()

	var dependentTx4 = &transaction.Transaction{
		ID: nil,
		Vin: []transactionbase.TXInput{
			{dependentTx2.ID, 1, nil, ta4.GetKeyPair().GetPublicKey()},
			{dependentTx3.ID, 0, nil, ta4.GetKeyPair().GetPublicKey()},
		},
		Vout: []transactionbase.TXOutput{
			{common.NewAmount(3), ta1.GetPubKeyHash(), ""},
		},
		Tip: common.NewAmount(1),
		Type: transaction.TxTypeNormal,
	}
	dependentTx4.ID = dependentTx4.Hash()

	var dependentTx5 = &transaction.Transaction{
		ID: nil,
		Vin: []transactionbase.TXInput{
			{dependentTx1.ID, 0, nil, ta1.GetKeyPair().GetPublicKey()},
			{dependentTx4.ID, 0, nil, ta1.GetKeyPair().GetPublicKey()},
		},
		Vout: []transactionbase.TXOutput{
			{common.NewAmount(4), ta5.GetPubKeyHash(), ""},
		},
		Tip: common.NewAmount(4),
		Type: transaction.TxTypeNormal,
	}
	dependentTx5.ID = dependentTx5.Hash()

	utxoIndex := lutxo.NewUTXOIndex(utxo.NewUTXOCache(storage.NewRamStorage()))

	utxoTx2 := utxo.NewUTXOTx()
	utxoTx2.PutUtxo(&utxo.UTXO{dependentTx1.Vout[1], dependentTx1.ID, 1, utxo.UtxoNormal,[]byte{}})

	utxoTx1 := utxo.NewUTXOTx()
	utxoTx1.PutUtxo(&utxo.UTXO{dependentTx1.Vout[0], dependentTx1.ID, 0, utxo.UtxoNormal,[]byte{}})

	utxoIndex.SetIndexAdd(map[string]*utxo.UTXOTx{
		ta2.GetPubKeyHash().String(): &utxoTx2,
		ta1.GetPubKeyHash().String(): &utxoTx1,
	})

	tx2Utxo1 := utxo.UTXO{dependentTx2.Vout[0], dependentTx2.ID, 0, utxo.UtxoNormal,[]byte{}}
	tx2Utxo2 := utxo.UTXO{dependentTx2.Vout[1], dependentTx2.ID, 1, utxo.UtxoNormal,[]byte{}}
	tx2Utxo3 := utxo.UTXO{dependentTx3.Vout[0], dependentTx3.ID, 0, utxo.UtxoNormal,[]byte{}}
	tx2Utxo4 := utxo.UTXO{dependentTx1.Vout[0], dependentTx1.ID, 0, utxo.UtxoNormal,[]byte{}}
	tx2Utxo5 := utxo.UTXO{dependentTx4.Vout[0], dependentTx4.ID, 0, utxo.UtxoNormal,[]byte{}}
	ltransaction.NewTxDecorator(dependentTx2).Sign(account.GenerateKeyPairByPrivateKey(prikey2).GetPrivateKey(), utxoIndex.GetAllUTXOsByPubKeyHash(ta2.GetPubKeyHash()).GetAllUtxos())
	ltransaction.NewTxDecorator(dependentTx3).Sign(account.GenerateKeyPairByPrivateKey(prikey3).GetPrivateKey(), []*utxo.UTXO{&tx2Utxo1})
	ltransaction.NewTxDecorator(dependentTx4).Sign(account.GenerateKeyPairByPrivateKey(prikey4).GetPrivateKey(), []*utxo.UTXO{&tx2Utxo2, &tx2Utxo3})
	ltransaction.NewTxDecorator(dependentTx5).Sign(account.GenerateKeyPairByPrivateKey(prikey1).GetPrivateKey(), []*utxo.UTXO{&tx2Utxo4, &tx2Utxo5})

	txPool := NewTransactionPool(nil, 6000000)
	// verify dependent txs 2,3,4,5 with relation:
	//tx1 (UtxoIndex)
	//|     \
	//tx2    \
	//|  \    \
	//tx3-tx4-tx5

	// test a transaction whose Vin is from UtxoIndex
	err1 := ltransaction.VerifyTransaction(utxoIndex, dependentTx2, 0)
	assert.Nil(t, err1)
	txPool.Push(*dependentTx2)

	// test a transaction whose Vin is from another transaction in transaction pool
	utxoIndex2 := *utxoIndex.DeepCopy()
	utxoIndex2.UpdateUtxos(txPool.GetTransactions())
	err2 := ltransaction.VerifyTransaction(&utxoIndex2, dependentTx3, 0)
	assert.Nil(t, err2)
	txPool.Push(*dependentTx3)

	// test a transaction whose Vin is from another two transactions in transaction pool
	utxoIndex3 := *utxoIndex.DeepCopy()
	utxoIndex3.UpdateUtxos(txPool.GetTransactions())
	err3 := ltransaction.VerifyTransaction(&utxoIndex3, dependentTx4, 0)
	assert.Nil(t, err3)
	txPool.Push(*dependentTx4)

	// test a transaction whose Vin is from another transaction in transaction pool and UtxoIndex
	utxoIndex4 := *utxoIndex.DeepCopy()
	utxoIndex4.UpdateUtxos(txPool.GetTransactions())
	err4 := ltransaction.VerifyTransaction(&utxoIndex4, dependentTx5, 0)
	assert.Nil(t, err4)
	txPool.Push(*dependentTx5)

	// test UTXOs not found for parent transactions
	err5 := ltransaction.VerifyTransaction(lutxo.NewUTXOIndex(utxo.NewUTXOCache(storage.NewRamStorage())), dependentTx3, 0)
	assert.NotNil(t, err5)

	// test a standalone transaction
	txPool.Push(tx1)
	err6 := ltransaction.VerifyTransaction(utxoIndex, &tx1, 0)
	assert.NotNil(t, err6)
}

func TestTransactionPool_PopTransactionsWithMostTipsNoDependency(t *testing.T) {
	txPool := NewTransactionPool(nil, 1280000)
	utxoIndex := lutxo.NewUTXOIndex(utxo.NewUTXOCache(storage.NewRamStorage()))
	var prevUTXOs []*utxo.UTXOTx
	var txs []*transaction.Transaction
	var accounts []*account.Account
	for i := 0; i < 5; i++ {
		acc := account.NewAccount()
		accounts = append(accounts, acc)
		cbtx := ltransaction.NewCoinbaseTX(acc.GetAddress(), "", 1, common.NewAmount(0))
		utxoIndex.UpdateUtxo(&cbtx)
		prevUTXO := utxoIndex.GetAllUTXOsByPubKeyHash(acc.GetPubKeyHash())
		prevUTXOs = append(prevUTXOs, prevUTXO)
	}

	//Create 4 transactions that can pass the transaction verification
	for i := 0; i < 4; i++ {
		sendTxParam := transaction.NewSendTxParam(accounts[i].GetAddress(), accounts[i].GetKeyPair(), accounts[i+1].GetAddress(), common.NewAmount(1), common.NewAmount(uint64(i)), common.NewAmount(0), common.NewAmount(0), "")
		tx, err := ltransaction.NewUTXOTransaction(prevUTXOs[i].GetAllUtxos(), sendTxParam)
		assert.Nil(t, err)
		txPool.Push(tx)
		txs = append(txs, &tx)
	}

	//pop out the transactions with most tips
	poppedTx := txPool.PopTransactionWithMostTips(utxoIndex)
	assert.Equal(t, txs[3], poppedTx.Value)
}

func TestTransactionPool_PopTransactionsWithMostTipsWithDependency(t *testing.T) {
	txPool := NewTransactionPool(nil, 1280000)
	utxoIndex := lutxo.NewUTXOIndex(utxo.NewUTXOCache(storage.NewRamStorage()))
	var accounts []*account.Account
	var txs []*transaction.Transaction

	for i := 0; i < 5; i++ {
		acc := account.NewAccount()
		accounts = append(accounts, acc)
		if i == 0 {
			cbtx := ltransaction.NewCoinbaseTX(acc.GetAddress(), "", 1, common.NewAmount(0))
			utxoIndex.UpdateUtxo(&cbtx)
		}
	}
	tempUtxoIndex := utxoIndex.DeepCopy()
	//Create 4 transactions that can pass the transaction verification
	for i := 0; i < 4; i++ {
		prevUTXO := tempUtxoIndex.GetAllUTXOsByPubKeyHash(accounts[i].GetPubKeyHash())
		sendTxParam := transaction.NewSendTxParam(accounts[i].GetAddress(), accounts[i].GetKeyPair(), accounts[i+1].GetAddress(), common.NewAmount(uint64(100-i*4)), common.NewAmount(uint64(i)), common.NewAmount(0), common.NewAmount(0), "")
		tx, err := ltransaction.NewUTXOTransaction(prevUTXO.GetAllUtxos(), sendTxParam)
		assert.Nil(t, err)
		tempUtxoIndex.UpdateUtxo(&tx)
		txPool.Push(tx)
		txs = append(txs, &tx)
	}
	//pop out the transactions with most tips. Each tx is about 263 bytes
	poppedTx := txPool.PopTransactionWithMostTips(utxoIndex)

	//tx 0 should be popped first since it is the parent of all other transactions
	assert.Equal(t, txs[0], poppedTx.Value)
}
