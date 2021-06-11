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
	"github.com/dappley/go-dappley/core/transactionbase"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestTxAdapter_isVinCoinbase(t *testing.T) {
	tests := []struct {
		name          string
		txAdapter     *TxAdapter
		expectedRes   bool
	} {
		{
			name: "vinCoinbaseTrue",
			txAdapter: &TxAdapter{
				&Transaction{
					ID:         getAoB(1),
					Vin:        []transactionbase.TXInput{
						{Txid: []byte{}, Vout: -1, Signature: nil, PubKey: nil},
					},
					Vout:       []transactionbase.TXOutput{},
				},
			},
			expectedRes: true,
		},
		{
			name: "nonEmptyVinTxid",
			txAdapter: &TxAdapter{
				&Transaction{
					ID:         getAoB(1),
					Vin:        []transactionbase.TXInput{
						{Txid: []byte{0x12, 0x34}, Vout: -1, Signature: nil, PubKey: nil},
					},
					Vout:       []transactionbase.TXOutput{},
				},
			},
			expectedRes: false,
		},
		{
			name: "wrongVinVoutValue",
			txAdapter: &TxAdapter{
				&Transaction{
					ID:         getAoB(1),
					Vin:        []transactionbase.TXInput{
						{Txid: []byte{}, Vout: 0, Signature: nil, PubKey: nil},
					},
					Vout:       []transactionbase.TXOutput{},
				},
			},
			expectedRes: false,
		},
		{
			name: "tooManyVin",
			txAdapter: &TxAdapter{
				&Transaction{
					ID:         getAoB(1),
					Vin:        []transactionbase.TXInput{
						{Txid: []byte{}, Vout: -1, Signature: nil, PubKey: nil},
						{Txid: []byte{}, Vout: -1, Signature: nil, PubKey: nil},
					},
					Vout:       []transactionbase.TXOutput{},
				},
			},
			expectedRes: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expectedRes, tt.txAdapter.isVinCoinbase())
		})
	}
}

func TestTxAdapter_isCoinbase(t *testing.T) {
	tests := []struct {
		name          string
		txAdapter     *TxAdapter
		expectedRes   bool
	} {
		{
			name: "coinbaseTrue",
			txAdapter: &TxAdapter{
				&Transaction{
					ID:         getAoB(1),
					Vin:        []transactionbase.TXInput{
						{Txid: []byte{}, Vout: -1, Signature: nil, PubKey: []byte{0x6c}},
					},
					Vout:       []transactionbase.TXOutput{
						{Value: nil, PubKeyHash: nil, Contract: ""},
					},
				},
			},
			expectedRes: true,
		},
		{
			name: "vinCoinbaseFalse",
			txAdapter: &TxAdapter{
				&Transaction{
					ID:         getAoB(1),
					Vin:        []transactionbase.TXInput{
						{Txid: []byte{0x1a}, Vout: 0, Signature: nil, PubKey: []byte{0x6c}},
					},
					Vout:       []transactionbase.TXOutput{
						{Value: nil, PubKeyHash: nil, Contract: ""},
					},
				},
			},
			expectedRes: false,
		},
		{
			name: "voutLengthInvalid",
			txAdapter: &TxAdapter{
				&Transaction{
					ID:         getAoB(1),
					Vin:        []transactionbase.TXInput{
						{Txid: []byte{}, Vout: -1, Signature: nil, PubKey: []byte{0x6c}},
					},
					Vout:       []transactionbase.TXOutput{},
				},
			},
			expectedRes: false,
		},
		{
			name: "vinPubKeyEmpty",
			txAdapter: &TxAdapter{
				&Transaction{
					ID:         getAoB(1),
					Vin:        []transactionbase.TXInput{
						{Txid: []byte{}, Vout: -1, Signature: nil, PubKey: []byte{}},
					},
					Vout:       []transactionbase.TXOutput{
						{Value: nil, PubKeyHash: nil, Contract: ""},
					},
				},
			},
			expectedRes: false,
		},
		{
			name: "vinPubKeyRewardTxData",
			txAdapter: &TxAdapter{
				&Transaction{
					ID:         getAoB(1),
					Vin:        []transactionbase.TXInput{
						{Txid: []byte{}, Vout: -1, Signature: nil, PubKey: RewardTxData},
					},
					Vout:       []transactionbase.TXOutput{
						{Value: nil, PubKeyHash: nil, Contract: ""},
					},
				},
			},
			expectedRes: false,
		},
		{
			name: "vinPubKeyGasRewardData",
			txAdapter: &TxAdapter{
				&Transaction{
					ID:         getAoB(1),
					Vin:        []transactionbase.TXInput{
						{Txid: []byte{}, Vout: -1, Signature: nil, PubKey: GasRewardData},
					},
					Vout:       []transactionbase.TXOutput{
						{Value: nil, PubKeyHash: nil, Contract: ""},
					},
				},
			},
			expectedRes: false,
		},
		{
			name: "vinPubKeyGasChangeData",
			txAdapter: &TxAdapter{
				&Transaction{
					ID:         getAoB(1),
					Vin:        []transactionbase.TXInput{
						{Txid: []byte{}, Vout: -1, Signature: nil, PubKey: GasChangeData},
					},
					Vout:       []transactionbase.TXOutput{
						{Value: nil, PubKeyHash: nil, Contract: ""},
					},
				},
			},
			expectedRes: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expectedRes, tt.txAdapter.isCoinbase())
		})
	}
}
