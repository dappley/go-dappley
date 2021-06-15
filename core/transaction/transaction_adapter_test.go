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
					ID:         []byte{0x67},
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
					ID:         []byte{0x67},
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
					ID:         []byte{0x67},
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
					ID:         []byte{0x67},
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
					ID:         []byte{0x67},
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
					ID:         []byte{0x67},
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
					ID:         []byte{0x67},
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
					ID:         []byte{0x67},
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
					ID:         []byte{0x67},
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
					ID:         []byte{0x67},
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
					ID:         []byte{0x67},
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

func TestTxAdapter_isRewardTx(t *testing.T) {
	tests := []struct {
		name          string
		txAdapter     *TxAdapter
		expectedRes   bool
	} {
		{
			name: "rewardTxTrue",
			txAdapter: &TxAdapter{
				&Transaction{
					ID:         []byte{0x67},
					Vin:        []transactionbase.TXInput{
						{Txid: []byte{}, Vout: -1, Signature: nil, PubKey: RewardTxData},
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
					ID:         []byte{0x67},
					Vin:        []transactionbase.TXInput{
						{Txid: []byte{0x1a}, Vout: 0, Signature: nil, PubKey: RewardTxData},
					},
					Vout:       []transactionbase.TXOutput{
						{Value: nil, PubKeyHash: nil, Contract: ""},
					},
				},
			},
			expectedRes: false,
		},
		{
			name: "vinPubKeyNotRewardTx",
			txAdapter: &TxAdapter{
				&Transaction{
					ID:         []byte{0x67},
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
			assert.Equal(t, tt.expectedRes, tt.txAdapter.isRewardTx())
		})
	}
}

func TestTxAdapter_isGasRewardTx(t *testing.T) {
	tests := []struct {
		name          string
		txAdapter     *TxAdapter
		expectedRes   bool
	} {
		{
			name: "rewardTxTrue",
			txAdapter: &TxAdapter{
				&Transaction{
					ID:         []byte{0x67},
					Vin:        []transactionbase.TXInput{
						{Txid: []byte{}, Vout: -1, Signature: nil, PubKey: GasRewardData},
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
					ID:         []byte{0x67},
					Vin:        []transactionbase.TXInput{
						{Txid: []byte{0x1a}, Vout: 0, Signature: nil, PubKey: GasRewardData},
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
					ID:         []byte{0x67},
					Vin:        []transactionbase.TXInput{
						{Txid: []byte{}, Vout: -1, Signature: nil, PubKey: GasRewardData},
					},
					Vout:       []transactionbase.TXOutput{},
				},
			},
			expectedRes: false,
		},
		{
			name: "vinPubKeyNotGasReward",
			txAdapter: &TxAdapter{
				&Transaction{
					ID:         []byte{0x67},
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
			assert.Equal(t, tt.expectedRes, tt.txAdapter.isGasRewardTx())
		})
	}
}

func TestTxAdapter_isChangeProducter(t *testing.T) {
	tests := []struct {
		name          string
		txAdapter     *TxAdapter
		expectedRes   bool
	} {
		{
			name: "changeProducterTrue",
			txAdapter: &TxAdapter{
				&Transaction{
					ID:         []byte{0x67},
					Vin:        []transactionbase.TXInput{
						{Txid: []byte{}, Vout: -1, Signature: nil, PubKey: []byte{0xde, 0xad, 0xbe, 0xef}},
					},
					Vout:       []transactionbase.TXOutput{
						{Value: nil, PubKeyHash: []byte{0xde, 0xad, 0xbe, 0xef}, Contract: "test1"},
						{Value: nil, PubKeyHash: []byte{0xde, 0xad, 0xbe, 0xef}, Contract: "test2"},
					},
				},
			},
			expectedRes: true,
		},
		{
			name: "emptyVin",
			txAdapter: &TxAdapter{
				&Transaction{
					ID:         []byte{0x67},
					Vin:        []transactionbase.TXInput{},
					Vout:       []transactionbase.TXOutput{
						{Value: nil, PubKeyHash: nil, Contract: ""},
					},
				},
			},
			expectedRes: false,
		},
		{
			name: "emptyVout",
			txAdapter: &TxAdapter{
				&Transaction{
					ID:         []byte{0x67},
					Vin:        []transactionbase.TXInput{
						{Txid: []byte{}, Vout: -1, Signature: nil, PubKey: GasRewardData},
					},
					Vout:       []transactionbase.TXOutput{},
				},
			},
			expectedRes: false,
		},
		{
			name: "incorrectHashes",
			txAdapter: &TxAdapter{
				&Transaction{
					ID:         []byte{0x67},
					Vin:        []transactionbase.TXInput{
						{Txid: []byte{}, Vout: -1, Signature: nil, PubKey: []byte{0xde, 0xad, 0xbe, 0xef}},
					},
					Vout:       []transactionbase.TXOutput{
						{Value: nil, PubKeyHash: []byte{0xff}, Contract: ""},
						{Value: nil, PubKeyHash: []byte{0xde, 0xad, 0xbe, 0xff}, Contract: ""},
					},
				},
			},
			expectedRes: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expectedRes, tt.txAdapter.isChangeProducter())
		})
	}
}

func TestTxAdapter_isGasChangeTx(t *testing.T) {
	tests := []struct {
		name          string
		txAdapter     *TxAdapter
		expectedRes   bool
	} {
		{
			name: "gasChangeTxTrue",
			txAdapter: &TxAdapter{
				&Transaction{
					ID:         []byte{0x67},
					Vin:        []transactionbase.TXInput{
						{Txid: []byte{}, Vout: -1, Signature: nil, PubKey: GasChangeData},
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
					ID:         []byte{0x67},
					Vin:        []transactionbase.TXInput{
						{Txid: []byte{0x1a}, Vout: 0, Signature: nil, PubKey: GasChangeData},
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
					ID:         []byte{0x67},
					Vin:        []transactionbase.TXInput{
						{Txid: []byte{}, Vout: -1, Signature: nil, PubKey: GasChangeData},
					},
					Vout:       []transactionbase.TXOutput{},
				},
			},
			expectedRes: false,
		},
		{
			name: "vinPubKeyNotGasChange",
			txAdapter: &TxAdapter{
				&Transaction{
					ID:         []byte{0x67},
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
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expectedRes, tt.txAdapter.isGasChangeTx())
		})
	}
}

func TestTxAdapter_IsContract(t *testing.T) {
	tests := []struct {
		name          string
		txAdapter     *TxAdapter
		expectedRes   bool
	}{
		{
			name: "contractTrue",
			txAdapter: &TxAdapter{
				&Transaction{
					ID: []byte{0x67},
					Vin: []transactionbase.TXInput{
						{Txid: []byte{}, Vout: -1, Signature: nil, PubKey: nil},
					},
					Vout: []transactionbase.TXOutput{
						{
							Value: nil,
							PubKeyHash: []byte{0x58, 0xb1, 0x34, 0x4c, 0x17, 0x67, 0x4c, 0x18, 0xd1, 0xa2, 0xdc, 0xea, 0x9f, 0x17, 0x16, 0xe0, 0x49, 0xf4, 0xa0, 0x5e, 0x6c},
							Contract: "test",
						},
					},
				},
			},
			expectedRes: true,
		},
		{
			name: "emptyVout",
			txAdapter: &TxAdapter{
				&Transaction{
					ID: []byte{0x67},
					Vin: []transactionbase.TXInput{
						{Txid: []byte{}, Vout: -1, Signature: nil, PubKey: nil},
					},
					Vout: []transactionbase.TXOutput{},
				},
			},
			expectedRes: false,
		},
		{
			name: "voutPubKeyHashIsNotContract",
			txAdapter: &TxAdapter{
				&Transaction{
					ID: []byte{0x67},
					Vin: []transactionbase.TXInput{
						{Txid: []byte{}, Vout: -1, Signature: nil, PubKey: nil},
					},
					Vout: []transactionbase.TXOutput{
						{
							Value: nil,
							PubKeyHash: []byte{0x5a, 0xb1, 0x34, 0x4c, 0x17, 0x67, 0x4c, 0x18, 0xd1, 0xa2, 0xdc, 0xea, 0x9f, 0x17, 0x16, 0xe0, 0x49, 0xf4, 0xa0, 0x5e, 0x6c},
							Contract: "test",
						},
					},
				},
			},
			expectedRes: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expectedRes, tt.txAdapter.IsContract())
		})
	}
}

func TestTxAdapter_isContractSendTx(t *testing.T) {
	tests := []struct {
		name          string
		txAdapter     *TxAdapter
		expectedRes   bool
	}{
		{
			name: "contractSendTxTrue",
			txAdapter: &TxAdapter{
				&Transaction{
					ID: []byte{0x67},
					Vin: []transactionbase.TXInput{
						{Txid: []byte{}, Vout: -1, Signature: nil, PubKey: []byte{0x58, 0xb1, 0x34, 0x4c, 0x17, 0x67, 0x4c, 0x18, 0xd1, 0xa2, 0xdc, 0xea, 0x9f, 0x17, 0x16, 0xe0, 0x49, 0xf4, 0xa0, 0x5e, 0x6c}},
						{Txid: []byte{}, Vout: -1, Signature: nil, PubKey: []byte{0x58, 0x98, 0xb4, 0xc, 0xd1, 0xb5, 0xff, 0xc1, 0xd9, 0x42, 0x61, 0x63, 0xac, 0xbc, 0x8c, 0x58, 0x62, 0xd4, 0xf1, 0x27, 0x5b}},
					},
					Vout: []transactionbase.TXOutput{},
				},
			},
			expectedRes: true,
		},
		{
			name: "emptyVin",
			txAdapter: &TxAdapter{
				&Transaction{
					ID: []byte{0x67},
					Vin: []transactionbase.TXInput{},
					Vout: []transactionbase.TXOutput{},
				},
			},
			expectedRes: false,
		},
		{
			name: "pubKeyHashIsNotContract",
			txAdapter: &TxAdapter{
				&Transaction{
					ID: []byte{0x67},
					Vin: []transactionbase.TXInput{
						{Txid: []byte{}, Vout: -1, Signature: nil, PubKey: []byte{0x58, 0xb1, 0x34, 0x4c, 0x17, 0x67, 0x4c, 0x18, 0xd1, 0xa2, 0xdc, 0xea, 0x9f, 0x17, 0x16, 0xe0, 0x49, 0xf4, 0xa0, 0x5e, 0x6c}},
						{Txid: []byte{}, Vout: -1, Signature: nil, PubKey: []byte{0x5a, 0x98, 0xb4, 0xc, 0xd1, 0xb5, 0xff, 0xc1, 0xd9, 0x42, 0x61, 0x63, 0xac, 0xbc, 0x8c, 0x58, 0x62, 0xd4, 0xf1, 0x27, 0x5b}},
					},
					Vout: []transactionbase.TXOutput{},
				},
			},
			expectedRes: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expectedRes, tt.txAdapter.isContractSendTx())
		})
	}
}

func TestTxAdapter_fillTypeAndNewTxAdapter(t *testing.T) {
	tests := []struct {
		name          string
		txAdapter     *TxAdapter
		expectedRes   TxType
	}{
		{
			name: "preExistingTxType",
			txAdapter: &TxAdapter{
				&Transaction{
					ID:         []byte{0x67},
					Vin:        []transactionbase.TXInput{},
					Vout:       []transactionbase.TXOutput{},
					Type: TxTypeContract,
				},
			},
			expectedRes: TxTypeContract,
		},
		{
			name: "TxTypeContract",
			txAdapter: &TxAdapter{
				&Transaction{
					ID: []byte{0x67},
					Vin: []transactionbase.TXInput{
						{Txid: []byte{}, Vout: -1, Signature: nil, PubKey: nil},
					},
					Vout: []transactionbase.TXOutput{
						{
							Value: nil,
							PubKeyHash: []byte{0x58, 0xb1, 0x34, 0x4c, 0x17, 0x67, 0x4c, 0x18, 0xd1, 0xa2, 0xdc, 0xea, 0x9f, 0x17, 0x16, 0xe0, 0x49, 0xf4, 0xa0, 0x5e, 0x6c},
							Contract: "test",
						},
					},
				},
			},
			expectedRes: TxTypeContract,
		},
		{
			name: "TxTypeCoinbase",
			txAdapter: &TxAdapter{
				&Transaction{
					ID:         []byte{0x67},
					Vin:        []transactionbase.TXInput{
						{Txid: []byte{}, Vout: -1, Signature: nil, PubKey: []byte{0x6c}},
					},
					Vout:       []transactionbase.TXOutput{
						{Value: nil, PubKeyHash: nil, Contract: ""},
					},
				},
			},
			expectedRes: TxTypeCoinbase,
		},
		{
			name: "TxTypeGasReward",
			txAdapter: &TxAdapter{
				&Transaction{
					ID:         []byte{0x67},
					Vin:        []transactionbase.TXInput{
						{Txid: []byte{}, Vout: -1, Signature: nil, PubKey: GasRewardData},
					},
					Vout:       []transactionbase.TXOutput{
						{Value: nil, PubKeyHash: nil, Contract: ""},
					},
				},
			},
			expectedRes: TxTypeGasReward,
		},
		{
			name: "TxTypeGasChange",
			txAdapter: &TxAdapter{
				&Transaction{
					ID:         []byte{0x67},
					Vin:        []transactionbase.TXInput{
						{Txid: []byte{}, Vout: -1, Signature: nil, PubKey: GasChangeData},
					},
					Vout:       []transactionbase.TXOutput{
						{Value: nil, PubKeyHash: nil, Contract: ""},
					},
				},
			},
			expectedRes: TxTypeGasChange,
		},
		{
			name: "TxTypeReward",
			txAdapter: &TxAdapter{
				&Transaction{
					ID:         []byte{0x67},
					Vin:        []transactionbase.TXInput{
						{Txid: []byte{}, Vout: -1, Signature: nil, PubKey: RewardTxData},
					},
					Vout:       []transactionbase.TXOutput{
						{Value: nil, PubKeyHash: nil, Contract: ""},
					},
				},
			},
			expectedRes: TxTypeReward,
		},
		{
			name: "TxTypeContractSend",
			txAdapter: &TxAdapter{
				&Transaction{
					ID: []byte{0x67},
					Vin: []transactionbase.TXInput{
						{Txid: []byte{}, Vout: -1, Signature: nil, PubKey: []byte{0x58, 0xb1, 0x34, 0x4c, 0x17, 0x67, 0x4c, 0x18, 0xd1, 0xa2, 0xdc, 0xea, 0x9f, 0x17, 0x16, 0xe0, 0x49, 0xf4, 0xa0, 0x5e, 0x6c}},
						{Txid: []byte{}, Vout: -1, Signature: nil, PubKey: []byte{0x58, 0x98, 0xb4, 0xc, 0xd1, 0xb5, 0xff, 0xc1, 0xd9, 0x42, 0x61, 0x63, 0xac, 0xbc, 0x8c, 0x58, 0x62, 0xd4, 0xf1, 0x27, 0x5b}},
					},
					Vout: []transactionbase.TXOutput{},
				},
			},
			expectedRes: TxTypeContractSend,
		},
		{
			name: "TxTypeProducerChange",
			txAdapter: &TxAdapter{
				&Transaction{
					ID:         []byte{0x67},
					Vin:        []transactionbase.TXInput{
						{Txid: []byte{}, Vout: -1, Signature: nil, PubKey: []byte{0xde, 0xad, 0xbe, 0xef}},
					},
					Vout:       []transactionbase.TXOutput{
						{Value: nil, PubKeyHash: []byte{0xde, 0xad, 0xbe, 0xef}, Contract: "test1"},
						{Value: nil, PubKeyHash: []byte{0xde, 0xad, 0xbe, 0xef}, Contract: "test2"},
					},
				},
			},
			expectedRes: TxTypeProducerChange,
		},
		{
			name: "TxTypeNormal",
			txAdapter: &TxAdapter{
				&Transaction{
					ID:         []byte{0x67},
					Vin:        []transactionbase.TXInput{},
					Vout:       []transactionbase.TXOutput{},
				},
			},
			expectedRes: TxTypeNormal,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.txAdapter.fillType()
			assert.Equal(t, tt.expectedRes, tt.txAdapter.Type)

			tx := tt.txAdapter.Transaction
			tx.Type = tt.expectedRes
			assert.Equal(t, TxAdapter{tx}, NewTxAdapter(tx))
		})
	}
}
