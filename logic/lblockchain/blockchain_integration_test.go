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

package lblockchain

import (
	"github.com/dappley/go-dappley/logic/ltransaction"
	"testing"

	"github.com/dappley/go-dappley/core/block"
	"github.com/dappley/go-dappley/core/scState"
	"github.com/dappley/go-dappley/core/transaction"
	"github.com/dappley/go-dappley/logic/lblock"
	"github.com/dappley/go-dappley/logic/lutxo"
	"github.com/dappley/go-dappley/logic/transactionpool"

	"github.com/dappley/go-dappley/common"
	"github.com/dappley/go-dappley/core/account"
	"github.com/dappley/go-dappley/storage"
	logger "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestBlockchain_RollbackToABlockWithTransactions(t *testing.T) {
	//create a mock blockchain with max height of 5
	//create a new block chain
	s := storage.NewRamStorage()
	defer s.Close()
	coinbaseAccount := account.NewAccount()
	coinbaseAddr := coinbaseAccount.GetAddress()
	bc := CreateBlockchain(coinbaseAddr, s, nil, transactionpool.NewTransactionPool(nil, 128000), nil, 100000)

	for i := 0; i < 3; i++ {
		tailBlk, _ := bc.GetTailBlock()
		cbtx := ltransaction.NewCoinbaseTX(coinbaseAddr, "", bc.GetMaxHeight(), common.NewAmount(0))
		b := block.NewBlock([]*transaction.Transaction{&cbtx}, tailBlk, coinbaseAddr.String())
		b.SetHash(lblock.CalculateHash(b))
		bc.AddBlockContextToTail(PrepareBlockContext(bc, b))
	}

	//generate 5 txs that has dependency relationships like the graph below
	/*
		tx0 - tx1 -tx2 - tx3 -tx4 -
	*/

	utxoIndex := lutxo.NewUTXOIndex(bc.utxoCache)
	txs := fakeDependentTxs(utxoIndex, coinbaseAccount, 5)

	//tx0 is in blk 4 and tx1 is in blk5. all other transactions are still in transaction pool
	//The current transactions in transaction pool should look like
	/*
		tx2 - tx3 - tx4
	*/
	for i := 2; i < len(txs); i++ {
		bc.txPool.Push(txs[i])
	}

	assert.Equal(t, 3, len(bc.txPool.GetAllTransactions()))
	assert.Equal(t, 1, len(bc.txPool.GetTipOrder()))

	//add block 4 with tx0
	tailBlk, _ := bc.GetTailBlock()
	cbtx := ltransaction.NewCoinbaseTX(coinbaseAddr, "", bc.GetMaxHeight(), common.NewAmount(0))
	b := block.NewBlock([]*transaction.Transaction{&cbtx, &txs[0]}, tailBlk, "16PencPNnF8CiSx2EBGEd1axhf7vuHCouj")
	b.SetHash(lblock.CalculateHash(b))
	bc.AddBlockContextToTail(PrepareBlockContext(bc, b))

	//add block 5 with tx1
	tailBlk, _ = bc.GetTailBlock()
	b = block.NewBlock([]*transaction.Transaction{&cbtx, &txs[1]}, tailBlk, "16PencPNnF8CiSx2EBGEd1axhf7vuHCouj")
	b.SetHash(lblock.CalculateHash(b))
	bc.AddBlockContextToTail(PrepareBlockContext(bc, b))

	//find the hash at height 3
	blk, err := bc.GetBlockByHeight(3)
	assert.Nil(t, err)

	//rollback to height 3
	bc.Rollback(blk.GetHash(), scState.NewScState())

	//the height 3 block should be the new tail block
	newTailBlk, err := bc.GetTailBlock()
	assert.Nil(t, err)
	assert.Equal(t, blk.GetHash(), newTailBlk.GetHash())

	//The current transactions in transaction pool should look like
	/*
		tx0 - tx1 - tx2 - tx3 - tx4
	*/
	assert.Equal(t, 5, len(bc.txPool.GetAllTransactions()))
	assert.Equal(t, 1, len(bc.txPool.GetTipOrder()))

}

func fakeDependentTxs(utxoIndex *lutxo.UTXOIndex, fundAccount *account.Account, numOfTx int) []transaction.Transaction {
	var txs []transaction.Transaction

	fundAddr := fundAccount.GetAddress()
	fundKeyPair := fundAccount.GetKeyPair()

	account1 := account.NewAccount()
	keyPair1 := account1.GetKeyPair()
	addr1 := account1.GetAddress()

	account2 := account.NewAccount()
	keyPair2 := account2.GetKeyPair()
	addr2 := account2.GetAddress()

	//first transaction's vin is from fund addr
	params := transaction.SendTxParam{
		fundAddr,
		fundKeyPair,
		addr1,
		common.NewAmount(5),
		common.NewAmount(0),
		common.NewAmount(0),
		common.NewAmount(0),
		"",
	}

	newTx, err := createTransaction(utxoIndex, params)
	if err == nil {
		txs = append(txs, newTx)
	}

	for i := 0; i < numOfTx-1; i++ {
		params := transaction.SendTxParam{
			addr1,
			keyPair1,
			addr2,
			common.NewAmount(5),
			common.NewAmount(0),
			common.NewAmount(0),
			common.NewAmount(0),
			"",
		}
		if i%2 == 1 {
			params.SenderKeyPair = keyPair2
			params.From = addr2
			params.To = addr1
		}
		newTx, err := createTransaction(utxoIndex, params)
		if err == nil {
			txs = append(txs, newTx)
		}
	}

	return txs
}

func createTransaction(utxoIndex *lutxo.UTXOIndex, params transaction.SendTxParam) (transaction.Transaction, error) {
	ta := account.NewAccountByKey(params.SenderKeyPair)
	utxos, _ := utxoIndex.GetUTXOsByAmount(ta.GetPubKeyHash(), params.TotalCost())
	tx, err := ltransaction.NewUTXOTransaction(utxos, params)
	if err != nil {
		logger.WithError(err).Error("CreateTransaction failed")
		return tx, err
	}
	utxoIndex.UpdateUtxo(&tx)
	return tx, nil
}
