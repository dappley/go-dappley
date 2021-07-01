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

package utxo

import (
	"github.com/dappley/go-dappley/common"
	"github.com/dappley/go-dappley/core/account"
	"github.com/dappley/go-dappley/core/transactionbase"
	utxopb "github.com/dappley/go-dappley/core/utxo/pb"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNewUTXO(t *testing.T) {
	pubKeyBytes := account.PubKeyHash([]byte{0x5a, 0xb1, 0x34, 0x4c, 0x17, 0x67, 0x4c, 0x18, 0xd1, 0xa2, 0xdc, 0xea, 0x9f, 0x17, 0x16, 0xe0, 0x49, 0xf4, 0xa0, 0x5e, 0x6c})
	transactionAccount := account.NewContractAccountByPubKeyHash(pubKeyBytes)

	txo := transactionbase.TXOutput{
		Value:      common.NewAmount(0),
		PubKeyHash: transactionAccount.GetPubKeyHash(),
		Contract:   "contract",
	}

	expected := &UTXO{
		TXOutput:    txo,
		Txid:        []byte{0x67},
		TxIndex:     1,
		UtxoType:    UtxoNormal,
		PrevUtxoKey: []byte{},
		NextUtxoKey: []byte{},
	}

	assert.Equal(t, expected, NewUTXO(txo, []byte{0x67}, 1, UtxoNormal,""))
}

func TestGetUTXOKey(t *testing.T) {
	assert.Equal(t, "test_1", GetUTXOKey([]byte{0x74, 0x65, 0x73, 0x74}, 1))
	assert.Equal(t, "g_0", GetUTXOKey([]byte{0x67}, 0))
}

func TestUTXO_GetUTXOKey(t *testing.T) {
	pubKeyBytes := account.PubKeyHash([]byte{0x5a, 0xb1, 0x34, 0x4c, 0x17, 0x67, 0x4c, 0x18, 0xd1, 0xa2, 0xdc, 0xea, 0x9f, 0x17, 0x16, 0xe0, 0x49, 0xf4, 0xa0, 0x5e, 0x6c})
	transactionAccount := account.NewContractAccountByPubKeyHash(pubKeyBytes)

	txo := transactionbase.TXOutput{
		Value:      common.NewAmount(0),
		PubKeyHash: transactionAccount.GetPubKeyHash(),
		Contract:   "contract",
	}

	utxo := &UTXO{
		TXOutput:    txo,
		Txid:        []byte{0x74, 0x65, 0x73, 0x74},
		TxIndex:     1,
		UtxoType:    UtxoNormal,
		PrevUtxoKey: []byte{},
		NextUtxoKey: []byte{},
	}

	assert.Equal(t, "test_1", utxo.GetUTXOKey())
}

func TestUTXO_ToProto(t *testing.T) {
	pubKeyBytes := account.PubKeyHash([]byte{0x5a, 0xb1, 0x34, 0x4c, 0x17, 0x67, 0x4c, 0x18, 0xd1, 0xa2, 0xdc, 0xea, 0x9f, 0x17, 0x16, 0xe0, 0x49, 0xf4, 0xa0, 0x5e, 0x6c})
	transactionAccount := account.NewContractAccountByPubKeyHash(pubKeyBytes)

	txo := transactionbase.TXOutput{
		Value:      common.NewAmount(10),
		PubKeyHash: transactionAccount.GetPubKeyHash(),
		Contract:   "contract",
	}

	utxo := &UTXO{
		TXOutput:    txo,
		Txid:        []byte{0x74, 0x65, 0x73, 0x74},
		TxIndex:     1,
		UtxoType:    UtxoNormal,
		PrevUtxoKey: []byte{0x6d, 0x6e, 0x6a, 0x6d},
		NextUtxoKey: []byte{0x6b, 0x6e, 0x6e, 0x6f},
	}

	expected := &utxopb.Utxo{
		Amount:        []byte{0xa},
		PublicKeyHash: []byte{0x5a, 0xb1, 0x34, 0x4c, 0x17, 0x67, 0x4c, 0x18, 0xd1, 0xa2, 0xdc, 0xea, 0x9f, 0x17, 0x16, 0xe0, 0x49, 0xf4, 0xa0, 0x5e, 0x6c},
		Txid:          []byte{0x74, 0x65, 0x73, 0x74},
		TxIndex:       uint32(1),
		UtxoType:      uint32(0),
		Contract:      "contract",
		PrevUtxoKey:   []byte{0x6d, 0x6e, 0x6a, 0x6d},
		NextUtxoKey:   []byte{0x6b, 0x6e, 0x6e, 0x6f},
	}

	assert.Equal(t, expected, utxo.ToProto())
}

func TestUTXO_FromProto(t *testing.T) {
	utxoProto := &utxopb.Utxo{
		Amount:        []byte{0xa},
		PublicKeyHash: []byte{0x5a, 0xb1, 0x34, 0x4c, 0x17, 0x67, 0x4c, 0x18, 0xd1, 0xa2, 0xdc, 0xea, 0x9f, 0x17, 0x16, 0xe0, 0x49, 0xf4, 0xa0, 0x5e, 0x6c},
		Txid:          []byte{0x74, 0x65, 0x73, 0x74},
		TxIndex:       uint32(1),
		UtxoType:      uint32(0),
		Contract:      "contract",
		PrevUtxoKey:   []byte{0x6d, 0x6e, 0x6a, 0x6d},
		NextUtxoKey:   []byte{0x6b, 0x6e, 0x6e, 0x6f},
	}

	txo := transactionbase.TXOutput{
		Value:      common.NewAmount(10),
		PubKeyHash: []byte{0x5a, 0xb1, 0x34, 0x4c, 0x17, 0x67, 0x4c, 0x18, 0xd1, 0xa2, 0xdc, 0xea, 0x9f, 0x17, 0x16, 0xe0, 0x49, 0xf4, 0xa0, 0x5e, 0x6c},
		Contract:   "contract",
	}
	expected := &UTXO{
		TXOutput:    txo,
		Txid:        []byte{0x74, 0x65, 0x73, 0x74},
		TxIndex:     1,
		UtxoType:    UtxoNormal,
		PrevUtxoKey: []byte{0x6d, 0x6e, 0x6a, 0x6d},
		NextUtxoKey: []byte{0x6b, 0x6e, 0x6e, 0x6f},
	}

	utxo := &UTXO{}
	utxo.FromProto(utxoProto)
	assert.Equal(t, expected, utxo)
}
