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

package transaction_pool

import (
	"testing"

	"github.com/dappley/go-dappley/core/transaction"
	"github.com/dappley/go-dappley/core/transaction_base"
	"github.com/dappley/go-dappley/core/utxo"
	"github.com/dappley/go-dappley/logic/transaction_logic"
	"github.com/dappley/go-dappley/logic/utxo_logic"

	"github.com/dappley/go-dappley/common"
	"github.com/dappley/go-dappley/core/account"
	"github.com/dappley/go-dappley/storage"
	"github.com/stretchr/testify/assert"
)

func TestTransactionPool_VerifyDependentTransactions(t *testing.T) {
	var prikey1 = "bb23d2ff19f5b16955e8a24dca34dd520980fe3bddca2b3e1b56663f0ec1aa71"
	var pubkey1 = account.GenerateKeyPairByPrivateKey(prikey1).GetPublicKey()
	var pkHash1, _ = account.NewUserPubKeyHash(pubkey1)
	var prikey2 = "bb23d2ff19f5b16955e8a24dca34dd520980fe3bddca2b3e1b56663f0ec1aa72"
	var pubkey2 = account.GenerateKeyPairByPrivateKey(prikey2).GetPublicKey()
	var pkHash2, _ = account.NewUserPubKeyHash(pubkey2)
	var prikey3 = "bb23d2ff19f5b16955e8a24dca34dd520980fe3bddca2b3e1b56663f0ec1aa73"
	var pubkey3 = account.GenerateKeyPairByPrivateKey(prikey3).GetPublicKey()
	var pkHash3, _ = account.NewUserPubKeyHash(pubkey3)
	var prikey4 = "bb23d2ff19f5b16955e8a24dca34dd520980fe3bddca2b3e1b56663f0ec1aa74"
	var pubkey4 = account.GenerateKeyPairByPrivateKey(prikey4).GetPublicKey()
	var pkHash4, _ = account.NewUserPubKeyHash(pubkey4)
	var prikey5 = "bb23d2ff19f5b16955e8a24dca34dd520980fe3bddca2b3e1b56663f0ec1aa75"
	var pubkey5 = account.GenerateKeyPairByPrivateKey(prikey5).GetPublicKey()
	var pkHash5, _ = account.NewUserPubKeyHash(pubkey5)

	var dependentTx1 = transaction.Transaction{
		ID: nil,
		Vin: []transaction_base.TXInput{
			{tx1.ID, 1, nil, pubkey1},
		},
		Vout: []transaction_base.TXOutput{
			{common.NewAmount(5), pkHash1, ""},
			{common.NewAmount(10), pkHash2, ""},
		},
		Tip: common.NewAmount(3),
	}
	dependentTx1.ID = dependentTx1.Hash()

	var dependentTx2 = transaction.Transaction{
		ID: nil,
		Vin: []transaction_base.TXInput{
			{dependentTx1.ID, 1, nil, pubkey2},
		},
		Vout: []transaction_base.TXOutput{
			{common.NewAmount(5), pkHash3, ""},
			{common.NewAmount(3), pkHash4, ""},
		},
		Tip: common.NewAmount(2),
	}
	dependentTx2.ID = dependentTx2.Hash()

	var dependentTx3 = transaction.Transaction{
		ID: nil,
		Vin: []transaction_base.TXInput{
			{dependentTx2.ID, 0, nil, pubkey3},
		},
		Vout: []transaction_base.TXOutput{
			{common.NewAmount(1), pkHash4, ""},
		},
		Tip: common.NewAmount(4),
	}
	dependentTx3.ID = dependentTx3.Hash()

	var dependentTx4 = transaction.Transaction{
		ID: nil,
		Vin: []transaction_base.TXInput{
			{dependentTx2.ID, 1, nil, pubkey4},
			{dependentTx3.ID, 0, nil, pubkey4},
		},
		Vout: []transaction_base.TXOutput{
			{common.NewAmount(3), pkHash1, ""},
		},
		Tip: common.NewAmount(1),
	}
	dependentTx4.ID = dependentTx4.Hash()

	var dependentTx5 = transaction.Transaction{
		ID: nil,
		Vin: []transaction_base.TXInput{
			{dependentTx1.ID, 0, nil, pubkey1},
			{dependentTx4.ID, 0, nil, pubkey1},
		},
		Vout: []transaction_base.TXOutput{
			{common.NewAmount(4), pkHash5, ""},
		},
		Tip: common.NewAmount(4),
	}
	dependentTx5.ID = dependentTx5.Hash()

	utxoIndex := utxo_logic.NewUTXOIndex(utxo.NewUTXOCache(storage.NewRamStorage()))

	utxoTx2 := utxo.NewUTXOTx()
	utxoTx2.PutUtxo(&utxo.UTXO{dependentTx1.Vout[1], dependentTx1.ID, 1, utxo.UtxoNormal})

	utxoTx1 := utxo.NewUTXOTx()
	utxoTx1.PutUtxo(&utxo.UTXO{dependentTx1.Vout[0], dependentTx1.ID, 0, utxo.UtxoNormal})

	utxoIndex.SetIndex(map[string]*utxo.UTXOTx{
		pkHash2.String(): &utxoTx2,
		pkHash1.String(): &utxoTx1,
	})

	tx2Utxo1 := utxo.UTXO{dependentTx2.Vout[0], dependentTx2.ID, 0, utxo.UtxoNormal}
	tx2Utxo2 := utxo.UTXO{dependentTx2.Vout[1], dependentTx2.ID, 1, utxo.UtxoNormal}
	tx2Utxo3 := utxo.UTXO{dependentTx3.Vout[0], dependentTx3.ID, 0, utxo.UtxoNormal}
	tx2Utxo4 := utxo.UTXO{dependentTx1.Vout[0], dependentTx1.ID, 0, utxo.UtxoNormal}
	tx2Utxo5 := utxo.UTXO{dependentTx4.Vout[0], dependentTx4.ID, 0, utxo.UtxoNormal}
	Sign(account.GenerateKeyPairByPrivateKey(prikey2).GetPrivateKey(), utxoIndex.GetAllUTXOsByPubKeyHash(pkHash2).GetAllUtxos(), &dependentTx2)
	Sign(account.GenerateKeyPairByPrivateKey(prikey3).GetPrivateKey(), []*utxo.UTXO{&tx2Utxo1}, &dependentTx3)
	Sign(account.GenerateKeyPairByPrivateKey(prikey4).GetPrivateKey(), []*utxo.UTXO{&tx2Utxo2, &tx2Utxo3}, &dependentTx4)
	Sign(account.GenerateKeyPairByPrivateKey(prikey1).GetPrivateKey(), []*utxo.UTXO{&tx2Utxo4, &tx2Utxo5}, &dependentTx5)

	txPool := transaction_pool.NewTransactionPool(nil, 6000000)
	// verify dependent txs 2,3,4,5 with relation:
	//tx1 (UtxoIndex)
	//|     \
	//tx2    \
	//|  \    \
	//tx3-tx4-tx5

	// test a transaction whose Vin is from UtxoIndex
	_, err1 := VerifyTransaction(utxoIndex, &dependentTx2, 0)
	assert.Nil(t, err1)
	txPool.Push(dependentTx2)

	// test a transaction whose Vin is from another transaction in transaction pool
	utxoIndex2 := *utxoIndex.DeepCopy()
	utxoIndex2.UpdateUtxoState(txPool.GetTransactions())
	_, err2 := VerifyTransaction(&utxoIndex2, &dependentTx3, 0)
	assert.Nil(t, err2)
	txPool.Push(dependentTx3)

	// test a transaction whose Vin is from another two transactions in transaction pool
	utxoIndex3 := *utxoIndex.DeepCopy()
	utxoIndex3.UpdateUtxoState(txPool.GetTransactions())
	_, err3 := VerifyTransaction(&utxoIndex3, &dependentTx4, 0)
	assert.Nil(t, err3)
	txPool.Push(dependentTx4)

	// test a transaction whose Vin is from another transaction in transaction pool and UtxoIndex
	utxoIndex4 := *utxoIndex.DeepCopy()
	utxoIndex4.UpdateUtxoState(txPool.GetTransactions())
	_, err4 := VerifyTransaction(&utxoIndex4, &dependentTx5, 0)
	assert.Nil(t, err4)
	txPool.Push(dependentTx5)

	// test UTXOs not found for parent transactions
	_, err5 := VerifyTransaction(utxo_logic.NewUTXOIndex(utxo.NewUTXOCache(storage.NewRamStorage())), &dependentTx3, 0)
	assert.NotNil(t, err5)

	// test a standalone transaction
	txPool.Push(tx1)
	_, err6 := VerifyTransaction(utxoIndex, &tx1, 0)
	assert.NotNil(t, err6)
}

func TestTransactionPool_PopTransactionsWithMostTipsNoDependency(t *testing.T) {
	txPool := NewTransactionPool(nil, 1280000)
	utxoIndex := utxo_logic.NewUTXOIndex(utxo.NewUTXOCache(storage.NewRamStorage()))
	var kps []*account.KeyPair
	var pkhs []account.PubKeyHash
	var addrs []account.Address
	var prevUTXOs []*utxo.UTXOTx
	var txs []*transaction.Transaction

	for i := 0; i < 5; i++ {
		kps = append(kps, account.NewKeyPair())
		pkh, _ := account.NewUserPubKeyHash(kps[i].GetPublicKey())
		pkhs = append(pkhs, pkh)
		addrs = append(addrs, pkh.GenerateAddress())
		cbtx := transaction_logic.NewCoinbaseTX(addrs[i], "", 1, common.NewAmount(0))
		utxoIndex.UpdateUtxo(&cbtx)
		prevUTXO := utxoIndex.GetAllUTXOsByPubKeyHash(pkhs[i])
		prevUTXOs = append(prevUTXOs, prevUTXO)
	}

	//Create 4 transactions that can pass the transaction verification
	for i := 0; i < 4; i++ {
		sendTxParam := transaction.NewSendTxParam(addrs[i], kps[i], addrs[i+1], common.NewAmount(1), common.NewAmount(uint64(i)), common.NewAmount(0), common.NewAmount(0), "")
		tx, err := transaction_logic.NewUTXOTransaction(prevUTXOs[i].GetAllUtxos(), sendTxParam)
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
	utxoIndex := utxo_logic.NewUTXOIndex(utxo.NewUTXOCache(storage.NewRamStorage()))
	var kps []*account.KeyPair
	var pkhs []account.PubKeyHash
	var addrs []account.Address
	var txs []*transaction.Transaction

	for i := 0; i < 5; i++ {
		kps = append(kps, account.NewKeyPair())
		pkh, _ := account.NewUserPubKeyHash(kps[i].GetPublicKey())
		pkhs = append(pkhs, pkh)
		addrs = append(addrs, pkh.GenerateAddress())
		if i == 0 {
			cbtx := transaction_logic.NewCoinbaseTX(addrs[i], "", 1, common.NewAmount(0))
			utxoIndex.UpdateUtxo(&cbtx)
		}
	}
	tempUtxoIndex := utxoIndex.DeepCopy()
	//Create 4 transactions that can pass the transaction verification
	for i := 0; i < 4; i++ {
		prevUTXO := tempUtxoIndex.GetAllUTXOsByPubKeyHash(pkhs[i])
		sendTxParam := transaction.NewSendTxParam(addrs[i], kps[i], addrs[i+1], common.NewAmount(uint64(100-i*4)), common.NewAmount(uint64(i)), common.NewAmount(0), common.NewAmount(0), "")
		tx, err := transaction_logic.NewUTXOTransaction(prevUTXO.GetAllUtxos(), sendTxParam)
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
