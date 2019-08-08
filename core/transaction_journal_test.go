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
	"github.com/dappley/go-dappley/common/hash"
	"github.com/dappley/go-dappley/core/blockchain"
	"github.com/dappley/go-dappley/core/transaction"
	"github.com/dappley/go-dappley/core/transaction_base"
	"github.com/dappley/go-dappley/core/utxo"
	"github.com/dappley/go-dappley/logic/blockchain_logic"
	"sync"
	"testing"

	"github.com/dappley/go-dappley/common"
	"github.com/dappley/go-dappley/core/account"
	"github.com/dappley/go-dappley/storage"
	"github.com/dappley/go-dappley/util"
	"github.com/stretchr/testify/assert"
)

func getAoB(length int64) []byte {
	return util.GenerateRandomAoB(length)
}

func GenerateFakeTxInputs() []transaction_base.TXInput {
	return []transaction_base.TXInput{
		{getAoB(2), 10, getAoB(2), getAoB(2)},
		{getAoB(2), 5, getAoB(2), getAoB(2)},
	}
}

func GenerateFakeTxOutputs() []transaction_base.TXOutput {
	return []transaction_base.TXOutput{
		{common.NewAmount(1), account.PubKeyHash(getAoB(2)), ""},
		{common.NewAmount(2), account.PubKeyHash(getAoB(2)), ""},
	}
}

func TestJournalPutAndGet(t *testing.T) {
	db := storage.NewRamStorage()

	// Create a blockchain for testing
	addr := account.NewAddress("dGDrVKjCG3sdXtDUgWZ7Fp3Q97tLhqWivf")
	bc := &blockchain_logic.Blockchain{blockchain.NewBlockchain(hash.Hash{}, hash.Hash{}), db, utxo.NewUTXOCache(db), nil, NewTransactionPool(nil, 128), nil, nil, 1000000, &sync.Mutex{}}

	// Add genesis block
	genesis := blockchain_logic.NewGenesisBlock(addr, transaction.Subsidy)

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
	txs := genesis.GetTransactions()
	txs = append(txs, &tx1)
	txs = append(txs, &tx2)
	genesis.SetTransactions(txs)

	err := bc.AddBlockContextToTail(PrepareBlockContext(bc, genesis))
	// Expect no error when adding genesis block
	assert.Nil(t, err)

	vin := transaction_base.TXInput{tx1.ID, 1, nil, nil}
	vout, err := GetTxOutput(vin, db)
	// Expect transaction logs have been successfully saved
	assert.Equal(t, vout.PubKeyHash, tx1.Vout[1].PubKeyHash)
}
