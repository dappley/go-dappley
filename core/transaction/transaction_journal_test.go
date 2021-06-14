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
package transaction

import (
	"testing"

	"github.com/dappley/go-dappley/common"
	"github.com/dappley/go-dappley/core/transactionbase"
	"github.com/dappley/go-dappley/util"

	"github.com/dappley/go-dappley/storage"
	"github.com/stretchr/testify/assert"

	transactionpb "github.com/dappley/go-dappley/core/transaction/pb"
	transactionbasepb "github.com/dappley/go-dappley/core/transactionbase/pb"
)

var tx1 = Transaction{
	ID:       util.GenerateRandomAoB(1),
	Vin:      transactionbase.GenerateFakeTxInputs(),
	Vout:     transactionbase.GenerateFakeTxOutputs(),
	Tip:      common.NewAmount(5),
	GasLimit: common.NewAmount(0),
	GasPrice: common.NewAmount(0),
}

func TestJournalPutAndGet(t *testing.T) {
	db := storage.NewRamStorage()
	vin := transactionbase.TXInput{tx1.ID, 1, nil, nil}
	err := PutTxJournal(tx1, db)
	assert.Nil(t, err)
	vout, err := GetTxOutput(vin, db)
	// Expect transaction logs have been successfully saved
	assert.Nil(t, err)
	assert.Equal(t, vout.PubKeyHash, tx1.Vout[1].PubKeyHash)
}

func TestJournalToProto(t *testing.T) {
	journal := &TxJournal{Txid: tx1.ID, Vout: tx1.Vout}
	var voutArray []*transactionbasepb.TXOutput
	for _, txout := range tx1.Vout {
		voutArray = append(voutArray, txout.ToProto().(*transactionbasepb.TXOutput))
	}

	expected := &transactionpb.TransactionJournal{Vout: voutArray}
	assert.Equal(t, expected, journal.toProto())
}

/* TODO: fix test
func TestJournalFromProto(t *testing.T) {
	journal := &TxJournal{}
	var voutArray []*transactionbasepb.TXOutput
	for _, txout := range tx1.Vout {
		voutArray = append(voutArray, txout.ToProto().(*transactionbasepb.TXOutput))
	}
	journalProto := &transactionpb.TransactionJournal{Vout: voutArray}
	journal.fromProto(journalProto)

	assert.Equal(t, tx1.Vout, journal.Vout)
}
 */

func TestNewTxJournal(t *testing.T) {
	journal := NewTxJournal(tx1.ID, tx1.Vout)
	expected := &TxJournal{Txid: tx1.ID, Vout: tx1.Vout}
	assert.Equal(t, expected, journal)
}

func TestGetStorageKey(t *testing.T) {
	assert.Equal(t, []byte{0x74, 0x78, 0x5f, 0x6a, 0x6f, 0x75, 0x72, 0x6e, 0x61, 0x6c, 0x5f}, getStorageKey(nil))
	assert.Equal(t, []byte{0x74, 0x78, 0x5f, 0x6a, 0x6f, 0x75, 0x72, 0x6e, 0x61, 0x6c, 0x5f, 0x88, 0x77}, getStorageKey([]byte{0x88, 0x77}))
}

func TestTxJournal_SerializeJournal(t *testing.T) {
	journal := &TxJournal{
		Txid: []byte{0x88},
		Vout: []transactionbase.TXOutput{
			{
				Value: common.NewAmount(10),
				PubKeyHash: []byte{0xc6, 0x49},
				Contract: "test1",
			},
			{
				Value: common.NewAmount(5),
				PubKeyHash: []byte{0xc7, 0x4a},
				Contract: "test2",
			},
		},
	}
	expected := []byte{0xa, 0xe, 0xa, 0x1, 0xa, 0x12, 0x2, 0xc6, 0x49, 0x1a, 0x5, 0x74, 0x65, 0x73, 0x74, 0x31, 0xa, 0xe, 0xa, 0x1, 0x5, 0x12, 0x2, 0xc7, 0x4a, 0x1a, 0x5, 0x74, 0x65, 0x73, 0x74, 0x32}
	result, err := journal.SerializeJournal()

	assert.Nil(t, err)
	assert.Equal(t, expected, result)
}

func TestDeserializeJournal(t *testing.T) {
	serializedBytes := []byte{0xa, 0xe, 0xa, 0x1, 0xa, 0x12, 0x2, 0xc6, 0x49, 0x1a, 0x5, 0x74, 0x65, 0x73, 0x74, 0x31, 0xa, 0xe, 0xa, 0x1, 0x5, 0x12, 0x2, 0xc7, 0x4a, 0x1a, 0x5, 0x74, 0x65, 0x73, 0x74, 0x32}
	expected := &TxJournal{
		Txid: []byte{0x88},
		Vout: []transactionbase.TXOutput{
			{
				Value: common.NewAmount(10),
				PubKeyHash: []byte{0xc6, 0x49},
				Contract: "test1",
			},
			{
				Value: common.NewAmount(5),
				PubKeyHash: []byte{0xc7, 0x4a},
				Contract: "test2",
			},
		},
	}
	result, err := DeserializeJournal(serializedBytes)

	assert.Nil(t, err)
	assert.Equal(t, expected, result)
}

func TestTxJournal_Save(t *testing.T) {
	db := storage.NewRamStorage()
	journal := &TxJournal{
		Txid: []byte{0x88},
		Vout: []transactionbase.TXOutput{
			{
				Value: common.NewAmount(10),
				PubKeyHash: []byte{0xc6, 0x49},
				Contract: "test1",
			},
			{
				Value: common.NewAmount(5),
				PubKeyHash: []byte{0xc7, 0x4a},
				Contract: "test2",
			},
		},
	}
	journal.Save(db)
	result, err := db.Get([]byte{0x74, 0x78, 0x5f, 0x6a, 0x6f, 0x75, 0x72, 0x6e, 0x61, 0x6c, 0x5f, 0x88})
	expected := []byte{0xa, 0xe, 0xa, 0x1, 0xa, 0x12, 0x2, 0xc6, 0x49, 0x1a, 0x5, 0x74, 0x65, 0x73, 0x74, 0x31, 0xa, 0xe, 0xa, 0x1, 0x5, 0x12, 0x2, 0xc7, 0x4a, 0x1a, 0x5, 0x74, 0x65, 0x73, 0x74, 0x32}
	assert.Nil(t, err)
	assert.Equal(t, expected, result)
}
