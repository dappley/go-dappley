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
	"time"

	"github.com/dappley/go-dappley/core/transaction"

	"github.com/dappley/go-dappley/common"
	"github.com/dappley/go-dappley/core/account"
	"github.com/dappley/go-dappley/core/block"
	"github.com/dappley/go-dappley/core/transactionbase"
	"github.com/dappley/go-dappley/core/utxo"
	"github.com/dappley/go-dappley/util"
)

func GenerateMockBlock() *block.Block {
	t1 := MockTransaction()
	t2 := MockTransaction()

	return block.NewBlockWithRawInfo(
		[]byte("hash"),
		[]byte("prevhash"),
		1,
		time.Now().Unix(),
		0,
		[]*transaction.Transaction{t1, t2},
	)
}

func MockTransaction() *transaction.Transaction {
	return &transaction.Transaction{
		ID:       util.GenerateRandomAoB(1),
		Vin:      MockTxInputs(),
		Vout:     MockTxOutputs(),
		Tip:      common.NewAmount(5),
		GasLimit: common.NewAmount(0),
		GasPrice: common.NewAmount(0),
	}
}

func MockTxInputs() []transactionbase.TXInput {
	return []transactionbase.TXInput{
		{util.GenerateRandomAoB(2),
			6,
			util.GenerateRandomAoB(2),
			[]byte("12345678901234567890123456789013")},
		{util.GenerateRandomAoB(2),
			2,
			util.GenerateRandomAoB(2),
			[]byte("12345678901234567890123456789014")},
	}
}

func MockTxInputsWithPubkey(pubkey []byte) []transactionbase.TXInput {
	return []transactionbase.TXInput{
		{util.GenerateRandomAoB(2),
			6,
			util.GenerateRandomAoB(2),
			pubkey},
		{util.GenerateRandomAoB(2),
			2,
			util.GenerateRandomAoB(2),
			pubkey},
	}
}

func MockUtxos(inputs []transactionbase.TXInput) []*utxo.UTXO {
	utxos := make([]*utxo.UTXO, len(inputs))

	for index, input := range inputs {
		ta := account.NewTransactionAccountByPubKey(input.PubKey)
		utxos[index] = &utxo.UTXO{
			TXOutput: transactionbase.TXOutput{Value: common.NewAmount(10), PubKeyHash: ta.GetPubKeyHash(), Contract: ""},
			Txid:     input.Txid,
			TxIndex:  0,
		}
	}

	return utxos
}

func MockTxOutputs() []transactionbase.TXOutput {
	ta := account.NewTransactionAccountByPubKey(util.GenerateRandomAoB(2))
	return []transactionbase.TXOutput{
		{common.NewAmount(5), ta.GetPubKeyHash(), ""},
		{common.NewAmount(7), ta.GetPubKeyHash(), ""},
	}
}

func GenerateUtxoMockBlockWithoutInputs() *block.Block {

	t1 := MockUtxoTransactionWithoutInputs()
	return block.NewBlockWithRawInfo(
		[]byte("hash"),
		nil,
		1,
		time.Now().Unix(),
		0,
		[]*transaction.Transaction{t1},
	)
}

func GenerateUtxoMockBlockWithInputs() *block.Block {

	t1 := MockUtxoTransactionWithInputs()
	return block.NewBlockWithRawInfo(
		[]byte("hash1"),
		[]byte("hash"),
		1,
		time.Now().Unix(),
		1,
		[]*transaction.Transaction{t1},
	)

}

func MockUtxoTransactionWithoutInputs() *transaction.Transaction {
	return &transaction.Transaction{
		ID:   []byte("tx1"),
		Vin:  []transactionbase.TXInput{},
		Vout: MockUtxoOutputsWithoutInputs(),
		Tip:  common.NewAmount(5),
	}
}

func MockUtxoTransactionWithInputs() *transaction.Transaction {
	return &transaction.Transaction{
		ID:   []byte("tx2"),
		Vin:  MockUtxoInputs(),
		Vout: MockUtxoOutputsWithInputs(),
		Tip:  common.NewAmount(5),
	}
}

// Padding Address to 32 Byte
var address1Bytes = []byte("address1000000000000000000000000")
var address2Bytes = []byte("address2000000000000000000000000")
var ta1 = account.NewTransactionAccountByPubKey(address1Bytes)
var ta2 = account.NewTransactionAccountByPubKey(address2Bytes)

func MockUtxoInputs() []transactionbase.TXInput {
	return []transactionbase.TXInput{
		{
			[]byte("tx1"),
			0,
			util.GenerateRandomAoB(2),
			address1Bytes},
		{
			[]byte("tx1"),
			1,
			util.GenerateRandomAoB(2),
			address1Bytes},
	}
}

func MockUtxoOutputsWithoutInputs() []transactionbase.TXOutput {
	return []transactionbase.TXOutput{
		{common.NewAmount(5), ta1.GetPubKeyHash(), ""},
		{common.NewAmount(7), ta1.GetPubKeyHash(), ""},
	}
}

func MockUtxoOutputsWithInputs() []transactionbase.TXOutput {
	return []transactionbase.TXOutput{
		{common.NewAmount(4), ta1.GetPubKeyHash(), ""},
		{common.NewAmount(5), ta2.GetPubKeyHash(), ""},
		{common.NewAmount(3), ta2.GetPubKeyHash(), ""},
	}
}
