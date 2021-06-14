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
	"github.com/dappley/go-dappley/common"
	"github.com/dappley/go-dappley/core/account"
	"github.com/dappley/go-dappley/core/transactionbase"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNewTransactionNode(t *testing.T) {
	normalTx := &Transaction{
		ID: []byte{0x4f, 0xda, 0x27, 0xf8, 0x2c, 0xa, 0x49, 0x9d, 0x8c, 0x19, 0x37, 0x46, 0x2c, 0x19, 0x9, 0xc6, 0x96, 0x54, 0x31, 0x99, 0x9f, 0x1f, 0xc6, 0x84, 0xf5, 0xc0, 0xc5, 0x6b, 0xbd, 0xbd, 0xe, 0xc8},
		Vin: []transactionbase.TXInput{
			{Txid: []byte{0xc7, 0x4d}, Vout: 10, Signature: nil, PubKey: []byte{0x7c, 0x4d}},
		},
		Vout: []transactionbase.TXOutput{
			{Value: common.NewAmount(1), PubKeyHash: account.PubKeyHash([]byte{0xc6, 0x49}), Contract: "test"},
		},
		Tip: common.NewAmount(5),
		GasLimit: common.NewAmount(1024),
		GasPrice: common.NewAmount(1),
		Type: TxTypeNormal,
	}

	txNodeNormal := NewTransactionNode(normalTx)
	expectedTxNodeNormal := &TransactionNode{
		Children: make(map[string]*Transaction),
		Value:    normalTx,
		Size:     73,
	}
	assert.Equal(t, expectedTxNodeNormal, txNodeNormal)

	expectedTxNodeEmpty := &TransactionNode{Children: make(map[string]*Transaction)}
	assert.Equal(t, expectedTxNodeEmpty, NewTransactionNode(nil))
	// The following line causes a segfault if Tip is not specified. Is this intended? (check Transaction.ToProto)
	assert.Equal(t, expectedTxNodeEmpty, NewTransactionNode(&Transaction{Tip: common.NewAmount(0)}))
}
