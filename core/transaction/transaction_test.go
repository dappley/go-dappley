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
	"math/big"
	"testing"

	"github.com/dappley/go-dappley/core/utxo"
	"github.com/dappley/go-dappley/crypto/keystore/secp256k1"
	errval "github.com/dappley/go-dappley/errors"

	transactionpb "github.com/dappley/go-dappley/core/transaction/pb"
	"github.com/dappley/go-dappley/core/transactionbase"

	"github.com/dappley/go-dappley/common"
	"github.com/dappley/go-dappley/core/account"
	"github.com/dappley/go-dappley/util"
	"github.com/golang/protobuf/proto"
	"github.com/stretchr/testify/assert"
)

func getAoB(length int64) []byte {
	return util.GenerateRandomAoB(length)
}

func GenerateFakeTxInputs() []transactionbase.TXInput {
	return []transactionbase.TXInput{
		{getAoB(2), 10, getAoB(2), getAoB(2)},
		{getAoB(2), 5, getAoB(2), getAoB(2)},
	}
}

func GenerateFakeTxOutputs() []transactionbase.TXOutput {
	return []transactionbase.TXOutput{
		{common.NewAmount(1), account.PubKeyHash(getAoB(2)), ""},
		{common.NewAmount(2), account.PubKeyHash(getAoB(2)), ""},
	}
}

func TestTrimmedCopy(t *testing.T) {
	var tx1 = Transaction{
		ID:   util.GenerateRandomAoB(1),
		Vin:  GenerateFakeTxInputs(),
		Vout: GenerateFakeTxOutputs(),
		Tip:  common.NewAmount(2),
	}

	t2 := tx1.TrimmedCopy(false)

	assert.Equal(t, tx1.ID, t2.ID)
	assert.Equal(t, tx1.Tip, t2.Tip)
	assert.Equal(t, tx1.Vout, t2.Vout)
	for index, vin := range t2.Vin {
		assert.Nil(t, vin.Signature)
		assert.Nil(t, vin.PubKey)
		assert.Equal(t, tx1.Vin[index].Txid, vin.Txid)
		assert.Equal(t, tx1.Vin[index].Vout, vin.Vout)
	}
}

func TestTransaction_Proto(t *testing.T) {
	tx1 := Transaction{
		ID:   util.GenerateRandomAoB(1),
		Vin:  GenerateFakeTxInputs(),
		Vout: GenerateFakeTxOutputs(),
		Tip:  common.NewAmount(5),
	}

	pb := tx1.ToProto()
	var i interface{} = pb
	_, correct := i.(proto.Message)
	assert.Equal(t, true, correct)
	mpb, err := proto.Marshal(pb)
	assert.Nil(t, err)

	newpb := &transactionpb.Transaction{}
	err = proto.Unmarshal(mpb, newpb)
	assert.Nil(t, err)

	tx2 := Transaction{}
	tx2.FromProto(newpb)

	assert.Equal(t, tx1, tx2)
}

func TestTransaction_MatchRewards(t *testing.T) {

	tests := []struct {
		name          string
		tx            *Transaction
		rewardStorage map[string]string
		expectedRes   bool
	}{
		{"normal",
			&Transaction{
				nil,
				[]transactionbase.TXInput{{nil, -1, nil, RewardTxData}},
				[]transactionbase.TXOutput{{
					common.NewAmount(1),
					account.PubKeyHash([]byte{
						0x5a, 0xc9, 0x85, 0x37, 0x92, 0x37, 0x76, 0x80,
						0xb1, 0x31, 0xa1, 0xab, 0xb, 0x5b, 0xa6, 0x49,
						0xe5, 0x27, 0xf0, 0x42, 0x5d}),
					"",
				}},
				common.NewAmount(0),
				common.NewAmount(0),
				common.NewAmount(0),
				0,
				TxTypeReward,
			},
			map[string]string{"dXnq2R6SzRNUt7ZANAqyZc2P9ziF6vYekB": "1"},
			true,
		},
		{"emptyVout",
			&Transaction{
				nil,
				[]transactionbase.TXInput{{nil, -1, nil, RewardTxData}},
				[]transactionbase.TXOutput{},
				common.NewAmount(0),
				common.NewAmount(0),
				common.NewAmount(0),
				0,
				TxTypeReward,
			},
			map[string]string{"dXnq2R6SzRNUt7ZANAqyZc2P9ziF6vYekB": "1"},
			false,
		},
		{"emptyRewardMap",
			&Transaction{
				nil,
				[]transactionbase.TXInput{{nil, -1, nil, RewardTxData}},
				[]transactionbase.TXOutput{{
					common.NewAmount(1),
					account.PubKeyHash([]byte{
						0x5a, 0xc9, 0x85, 0x37, 0x92, 0x37, 0x76, 0x80,
						0xb1, 0x31, 0xa1, 0xab, 0xb, 0x5b, 0xa6, 0x49,
						0xe5, 0x27, 0xf0, 0x42, 0x5d}),
					"",
				}},
				common.NewAmount(0),
				common.NewAmount(0),
				common.NewAmount(0),
				0,
				TxTypeReward,
			},
			nil,
			false,
		},
		{"Wrong address",
			&Transaction{
				nil,
				[]transactionbase.TXInput{{nil, -1, nil, RewardTxData}},
				[]transactionbase.TXOutput{{
					common.NewAmount(1),
					account.PubKeyHash([]byte{
						0x5a, 0xc9, 0x85, 0x37, 0x92, 0x37, 0x76, 0x80,
						0xb1, 0x31, 0xa1, 0xab, 0xb, 0x5b, 0xa6, 0x49,
						0xe5, 0x27, 0xf0, 0x42, 0x5d}),
					"",
				}},
				common.NewAmount(0),
				common.NewAmount(0),
				common.NewAmount(0),
				0,
				TxTypeReward,
			},
			map[string]string{"dXnq2R6SzRNUt7ZsNAqyZc2P9ziF6vYekB": "1"},
			false,
		},
		{"Wrong amount",
			&Transaction{
				nil,
				[]transactionbase.TXInput{{nil, -1, nil, RewardTxData}},
				[]transactionbase.TXOutput{{
					common.NewAmount(3),
					account.PubKeyHash([]byte{
						0x5a, 0xc9, 0x85, 0x37, 0x92, 0x37, 0x76, 0x80,
						0xb1, 0x31, 0xa1, 0xab, 0xb, 0x5b, 0xa6, 0x49,
						0xe5, 0x27, 0xf0, 0x42, 0x5d}),
					"",
				}},
				common.NewAmount(0),
				common.NewAmount(0),
				common.NewAmount(0),
				0,
				TxTypeReward,
			},
			map[string]string{"dXnq2R6SzRNUt7ZsNAqyZc2P9ziF6vYekB": "1"},
			false,
		},
		{"twoAddresses",
			&Transaction{
				nil,
				[]transactionbase.TXInput{{nil, -1, nil, RewardTxData}},
				[]transactionbase.TXOutput{{
					common.NewAmount(1),
					account.PubKeyHash([]byte{
						0x5a, 0xc9, 0x85, 0x37, 0x92, 0x37, 0x76, 0x80,
						0xb1, 0x31, 0xa1, 0xab, 0xb, 0x5b, 0xa6, 0x49,
						0xe5, 0x27, 0xf0, 0x42, 0x5d}),
					"",
				},
					{
						common.NewAmount(4),
						account.PubKeyHash([]byte{
							90, 13, 39, 130, 118, 11, 160, 130, 83, 126, 86, 102, 252, 178, 87,
							218, 57, 174, 123, 244, 229}),
						"",
					}},
				common.NewAmount(0),
				common.NewAmount(0),
				common.NewAmount(0),
				0,
				TxTypeReward,
			},
			map[string]string{
				"dEcqjSgREFi9gTCbAWpEQ3kbPxgsBzzhWS": "4",
				"dXnq2R6SzRNUt7ZANAqyZc2P9ziF6vYekB": "1",
			},
			true,
		},
		{"MoreRewards",
			&Transaction{
				nil,
				[]transactionbase.TXInput{{nil, -1, nil, RewardTxData}},
				[]transactionbase.TXOutput{{
					common.NewAmount(1),
					account.PubKeyHash([]byte{
						0x5a, 0xc9, 0x85, 0x37, 0x92, 0x37, 0x76, 0x80,
						0xb1, 0x31, 0xa1, 0xab, 0xb, 0x5b, 0xa6, 0x49,
						0xe5, 0x27, 0xf0, 0x42, 0x5d}),
					"",
				},
					{
						common.NewAmount(4),
						account.PubKeyHash([]byte{
							90, 13, 39, 130, 118, 11, 160, 130, 83, 126, 86, 102, 252, 178, 87,
							218, 57, 174, 123, 244, 229}),
						"",
					}},
				common.NewAmount(0),
				common.NewAmount(0),
				common.NewAmount(0),
				0,
				TxTypeReward,
			},
			map[string]string{
				"dEcqjSgREFi9gTCbAWpEQ3kbPxgsBzzhWS": "4",
				"dXnq2R6SzRNUt7ZANAqyZc2P9ziF6vYekB": "1",
				"dXnq2R6SzRNUt7ZANAqyZc2P9ziF6fYekB": "3",
			},
			false,
		},
		{"MoreVout",
			&Transaction{
				nil,
				[]transactionbase.TXInput{{nil, -1, nil, RewardTxData}},
				[]transactionbase.TXOutput{{
					common.NewAmount(1),
					account.PubKeyHash([]byte{
						0x5a, 0xc9, 0x85, 0x37, 0x92, 0x37, 0x76, 0x80,
						0xb1, 0x31, 0xa1, 0xab, 0xb, 0x5b, 0xa6, 0x49,
						0xe5, 0x27, 0xf0, 0x42, 0x5d}),
					"",
				},
					{
						common.NewAmount(4),
						account.PubKeyHash([]byte{
							90, 13, 39, 130, 118, 11, 160, 130, 83, 126, 86, 102, 252, 178, 87,
							218, 57, 174, 123, 244, 229}),
						"",
					}},
				common.NewAmount(0),
				common.NewAmount(0),
				common.NewAmount(0),
				0,
				TxTypeReward,
			},
			map[string]string{
				"dEcqjSgREFi9gTCbAWpEQ3kbPxgsBzzhWS": "4",
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expectedRes, tt.tx.MatchRewards(tt.rewardStorage))
		})
	}
}

func TestTransaction_Sign(t *testing.T) {
	privKey, _ := secp256k1.NewECDSAPrivateKey()
	tx := &Transaction{
		ID: []byte{0x66},
		Vin: []transactionbase.TXInput{
			{Txid: []byte{0xc7, 0x4d}, Vout: 10, Signature: nil, PubKey: []byte{0x7c, 0x4d}},
			{Txid: []byte{0xc8, 0x4e}, Vout: 10, Signature: nil, PubKey: []byte{0x7d, 0x4e}},
		},
		Vout: []transactionbase.TXOutput{
			{Value: common.NewAmount(1), PubKeyHash: account.PubKeyHash([]byte{0xc6, 0x49}), Contract: "test"},
			{Value: common.NewAmount(2), PubKeyHash: account.PubKeyHash([]byte{0xc7, 0x4a}), Contract: "test"},
		},
		Tip:      common.NewAmount(5),
		GasLimit: common.NewAmount(1024),
		GasPrice: common.NewAmount(1),
		Type:     TxTypeNormal,
	}
	utxos := []*utxo.UTXO{
		{
			TXOutput: transactionbase.TXOutput{Value: common.NewAmount(10), PubKeyHash: []byte{0xde, 0x4d}, Contract: ""},
			Txid:     []byte{0x20, 0x21},
			TxIndex:  0,
			UtxoType: 0,
		},
		{
			TXOutput: transactionbase.TXOutput{Value: common.NewAmount(5), PubKeyHash: []byte{0xbe, 0xef}, Contract: ""},
			Txid:     []byte{0x13, 0x30},
			TxIndex:  1,
			UtxoType: 0,
		},
	}
	privKeyBytes, _ := secp256k1.FromECDSAPrivateKey(privKey)
	bytesToSign := [][]byte{
		[]byte{0xe6, 0xe6, 0xda, 0xc5, 0xf1, 0xc, 0xb, 0xb0, 0x85, 0x44, 0xc2, 0xb1, 0xdc, 0xe2, 0x19, 0x5a, 0x59, 0xf4, 0x4c, 0xad, 0xf8, 0x50, 0x68, 0x93, 0xe0, 0x1f, 0xdb, 0x72, 0x76, 0xdc, 0xa0, 0xa5},
		[]byte{0xc4, 0x3, 0x9, 0xbb, 0xa6, 0xfa, 0x9e, 0xe6, 0x1, 0xc6, 0xe4, 0x5f, 0x7e, 0x73, 0xc9, 0x3b, 0xc1, 0x2a, 0x8e, 0x35, 0xd2, 0xf, 0x74, 0x99, 0x42, 0x3b, 0x53, 0xb7, 0xac, 0x67, 0xe0, 0x4a},
	}
	err := tx.Sign(*privKey, utxos)
	assert.Nil(t, err)
	for i, vin := range tx.Vin {
		expectedSignature, _ := secp256k1.Sign(bytesToSign[i], privKeyBytes)
		assert.Equal(t, expectedSignature, vin.Signature)
	}
}

func TestTransaction_GetToHashBytes(t *testing.T) {
	tests := []struct {
		name        string
		tx          *Transaction
		expectedRes []byte
	}{
		{
			name: "noTipOrGas",
			tx: &Transaction{
				ID: []byte{0x58},
				Vin: []transactionbase.TXInput{
					{Txid: []byte{0x18, 0x30}, Vout: 10, Signature: nil, PubKey: []byte{0x20, 0x40}},
				},
				Vout: []transactionbase.TXOutput{
					{Value: common.NewAmount(1), PubKeyHash: account.PubKeyHash([]byte{0x40, 0x80}), Contract: "test"},
				},
			},
			expectedRes: []byte{0x18, 0x30, 0x0, 0x0, 0x0, 0xa, 0x20, 0x40, 0x1, 0x40, 0x80, 0x74, 0x65, 0x73, 0x74},
		},
		{
			name: "withTipAndGas",
			tx: &Transaction{
				ID: []byte{0x66},
				Vin: []transactionbase.TXInput{
					{Txid: []byte{0xc7, 0x4d}, Vout: 10, Signature: nil, PubKey: []byte{0x7c, 0x4d}},
					{Txid: []byte{0xc8, 0x4e}, Vout: 10, Signature: nil, PubKey: []byte{0x7d, 0x4e}},
				},
				Vout: []transactionbase.TXOutput{
					{Value: common.NewAmount(1), PubKeyHash: account.PubKeyHash([]byte{0xc6, 0x49}), Contract: "test"},
					{Value: common.NewAmount(2), PubKeyHash: account.PubKeyHash([]byte{0xc7, 0x4a}), Contract: "test"},
				},
				Tip:      common.NewAmount(5),
				GasLimit: common.NewAmount(1024),
				GasPrice: common.NewAmount(1),
				Type:     TxTypeNormal,
			},
			expectedRes: []byte{0xc7, 0x4d, 0x0, 0x0, 0x0, 0xa, 0x7c, 0x4d, 0xc8, 0x4e, 0x0, 0x0, 0x0, 0xa, 0x7d, 0x4e, 0x1, 0xc6, 0x49, 0x74, 0x65, 0x73, 0x74, 0x2, 0xc7, 0x4a, 0x74, 0x65, 0x73, 0x74, 0x5, 0x4, 0x0, 0x1, 0x0, 0x0, 0x0, 0x1},
		},
		{
			name: "emptyVinVout",
			tx: &Transaction{
				ID:   []byte{0x0},
				Vin:  []transactionbase.TXInput{},
				Vout: []transactionbase.TXOutput{},
			},
			expectedRes: []byte(nil),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expectedRes, tt.tx.GetToHashBytes())
		})
	}
}

func TestTransaction_Hash(t *testing.T) {
	tests := []struct {
		name        string
		tx          *Transaction
		expectedRes []byte
	}{
		{
			name: "normal",
			tx: &Transaction{
				ID: []byte{0x66},
				Vin: []transactionbase.TXInput{
					{Txid: []byte{0xc7, 0x4d}, Vout: 10, Signature: nil, PubKey: []byte{0x7c, 0x4d}},
					{Txid: []byte{0xc8, 0x4e}, Vout: 10, Signature: nil, PubKey: []byte{0x7d, 0x4e}},
				},
				Vout: []transactionbase.TXOutput{
					{Value: common.NewAmount(1), PubKeyHash: account.PubKeyHash([]byte{0xc6, 0x49}), Contract: "test"},
					{Value: common.NewAmount(2), PubKeyHash: account.PubKeyHash([]byte{0xc7, 0x4a}), Contract: "test"},
				},
				Tip:      common.NewAmount(5),
				GasLimit: common.NewAmount(1024),
				GasPrice: common.NewAmount(1),
				Type:     TxTypeNormal,
			},
			expectedRes: []byte{0x4f, 0xda, 0x27, 0xf8, 0x2c, 0xa, 0x49, 0x9d, 0x8c, 0x19, 0x37, 0x46, 0x2c, 0x19, 0x9, 0xc6, 0x96, 0x54, 0x31, 0x99, 0x9f, 0x1f, 0xc6, 0x84, 0xf5, 0xc0, 0xc5, 0x6b, 0xbd, 0xbd, 0xe, 0xc8},
		},
		{
			name: "emptyVinVout",
			tx: &Transaction{
				ID:   []byte{0},
				Vin:  []transactionbase.TXInput{},
				Vout: []transactionbase.TXOutput{},
			},
			expectedRes: []byte{0xe3, 0xb0, 0xc4, 0x42, 0x98, 0xfc, 0x1c, 0x14, 0x9a, 0xfb, 0xf4, 0xc8, 0x99, 0x6f, 0xb9, 0x24, 0x27, 0xae, 0x41, 0xe4, 0x64, 0x9b, 0x93, 0x4c, 0xa4, 0x95, 0x99, 0x1b, 0x78, 0x52, 0xb8, 0x55},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expectedRes, tt.tx.Hash())
		})
	}
}

func TestTransaction_DeepCopy(t *testing.T) {
	original := &Transaction{
		ID:         util.GenerateRandomAoB(1),
		Vin:        GenerateFakeTxInputs(),
		Vout:       GenerateFakeTxOutputs(),
		Tip:        common.NewAmount(5),
		GasLimit:   common.NewAmount(1024),
		GasPrice:   common.NewAmount(1),
		CreateTime: 99,
		Type:       TxTypeNormal,
	}
	copy := original.DeepCopy()
	for i, vin := range copy.Vin {
		assert.Equal(t, original.Vin[i].Txid, vin.Txid)
		assert.Equal(t, original.Vin[i].Vout, vin.Vout)
		assert.Equal(t, original.Vin[i].Signature, vin.Signature)
		assert.Equal(t, original.Vin[i].PubKey, vin.PubKey)
	}
	for i, vout := range copy.Vout {
		assert.Equal(t, original.Vout[i].Value, vout.Value)
		assert.Equal(t, original.Vout[i].PubKeyHash, vout.PubKeyHash)
		assert.Equal(t, original.Vout[i].Contract, vout.Contract)
	}
	assert.Equal(t, original.ID, copy.ID)
	assert.Equal(t, original.Tip, copy.Tip)
	assert.Equal(t, common.NewAmount(1024), copy.GasLimit)
	assert.Equal(t, common.NewAmount(1), copy.GasPrice)
	assert.Equal(t, int64(99), copy.CreateTime)
	assert.Equal(t, TxTypeNormal, copy.Type)
}

func TestTransaction_verifyID(t *testing.T) {
	normalTx := Transaction{
		ID: []byte{0x4f, 0xda, 0x27, 0xf8, 0x2c, 0xa, 0x49, 0x9d, 0x8c, 0x19, 0x37, 0x46, 0x2c, 0x19, 0x9, 0xc6, 0x96, 0x54, 0x31, 0x99, 0x9f, 0x1f, 0xc6, 0x84, 0xf5, 0xc0, 0xc5, 0x6b, 0xbd, 0xbd, 0xe, 0xc8},
		Vin: []transactionbase.TXInput{
			{Txid: []byte{0xc7, 0x4d}, Vout: 10, Signature: nil, PubKey: []byte{0x7c, 0x4d}},
			{Txid: []byte{0xc8, 0x4e}, Vout: 10, Signature: nil, PubKey: []byte{0x7d, 0x4e}},
		},
		Vout: []transactionbase.TXOutput{
			{Value: common.NewAmount(1), PubKeyHash: account.PubKeyHash([]byte{0xc6, 0x49}), Contract: "test"},
			{Value: common.NewAmount(2), PubKeyHash: account.PubKeyHash([]byte{0xc7, 0x4a}), Contract: "test"},
		},
		Tip:      common.NewAmount(5),
		GasLimit: common.NewAmount(1024),
		GasPrice: common.NewAmount(1),
		Type:     TxTypeNormal,
	}
	emptyVinVoutTx := Transaction{
		ID:   []byte{0xe3, 0xb0, 0xc4, 0x42, 0x98, 0xfc, 0x1c, 0x14, 0x9a, 0xfb, 0xf4, 0xc8, 0x99, 0x6f, 0xb9, 0x24, 0x27, 0xae, 0x41, 0xe4, 0x64, 0x9b, 0x93, 0x4c, 0xa4, 0x95, 0x99, 0x1b, 0x78, 0x52, 0xb8, 0x55},
		Vin:  []transactionbase.TXInput{},
		Vout: []transactionbase.TXOutput{},
	}

	verified, err := normalTx.verifyID()
	assert.True(t, verified)
	assert.Nil(t, err)

	verified, err = emptyVinVoutTx.verifyID()
	assert.True(t, verified)
	assert.Nil(t, err)

	normalTx.ID = []byte{0x0}
	verified, err = normalTx.verifyID()
	assert.False(t, verified)
	assert.Equal(t, errval.TransactionIDInvalid, err)
}

func TestTransaction_verifyAmount(t *testing.T) {
	prevUtxosSum := common.NewAmount(24)

	tx := Transaction{
		ID:       util.GenerateRandomAoB(1),
		Vin:      GenerateFakeTxInputs(),  //15
		Vout:     GenerateFakeTxOutputs(), //3
		Tip:      common.NewAmount(5),
		GasLimit: common.NewAmount(8),
		GasPrice: common.NewAmount(2),
	}

	txTotalVoutValue := common.NewAmount(0)
	for _, vout := range tx.Vout {
		txTotalVoutValue = txTotalVoutValue.Add(vout.Value)
	}

	verified, err := tx.verifyAmount(prevUtxosSum, txTotalVoutValue)
	assert.True(t, verified)
	assert.Nil(t, err)

	prevUtxosSum = common.NewAmount(2)
	verified, err = tx.verifyAmount(prevUtxosSum, txTotalVoutValue)
	assert.False(t, verified)
	assert.Equal(t, errval.TransactionAmountInvalid, err)

	prevUtxosSum = common.NewAmount(18)
	verified, err = tx.verifyAmount(prevUtxosSum, txTotalVoutValue)
	assert.False(t, verified)
	assert.Equal(t, errval.TransactionGasLimitInvalid, err)

	prevUtxosSum = common.NewAmount(23)
	verified, err = tx.verifyAmount(prevUtxosSum, txTotalVoutValue)
	assert.False(t, verified)
	assert.Equal(t, errval.TransactionTipInvalid, err)
}

func TestTransaction_CalculateTotalVoutValue(t *testing.T) {
	tx := Transaction{
		ID:  []byte{0x66},
		Vin: []transactionbase.TXInput{},
		Vout: []transactionbase.TXOutput{
			{Value: common.NewAmount(20), PubKeyHash: account.PubKeyHash([]byte{0xc6, 0x49}), Contract: "test"},
			{Value: common.NewAmount(44), PubKeyHash: account.PubKeyHash([]byte{0xc7, 0x4a}), Contract: "test"},
		},
	}

	amount, success := tx.CalculateTotalVoutValue()
	assert.Equal(t, common.NewAmount(64), amount)
	assert.True(t, success)

	tx.Vout[0].Value = nil
	amount, success = tx.CalculateTotalVoutValue()
	assert.Nil(t, amount)
	assert.False(t, success)

	tx.Vout[0].Value = &common.Amount{*new(big.Int).SetInt64(-1)}
	amount, success = tx.CalculateTotalVoutValue()
	assert.Nil(t, amount)
	assert.False(t, success)

	tx.Vout = []transactionbase.TXOutput{}
	amount, success = tx.CalculateTotalVoutValue()
	assert.Equal(t, common.NewAmount(0), amount)
	assert.True(t, success)
}

func TestTransaction_String(t *testing.T) {
	tx := Transaction{
		ID: []byte{0x66},
		Vin: []transactionbase.TXInput{
			{Txid: []byte{0xc7, 0x4d}, Vout: 10, Signature: nil, PubKey: []byte{0x7c, 0x4d}},
			{Txid: []byte{0xc8, 0x4e}, Vout: 10, Signature: nil, PubKey: []byte{0x7d, 0x4e}},
		},
		Vout: []transactionbase.TXOutput{
			{Value: common.NewAmount(1), PubKeyHash: account.PubKeyHash([]byte{0xc6, 0x49}), Contract: "test"},
			{Value: common.NewAmount(2), PubKeyHash: account.PubKeyHash([]byte{0xc7, 0x4a}), Contract: "test"},
		},
		Tip:      common.NewAmount(5),
		GasLimit: common.NewAmount(1024),
		GasPrice: common.NewAmount(1),
		Type:     TxTypeNormal,
	}

	expectedString := "\n--- Transaction: 66\n" +
		"     Input 0:\n" +
		"       TXID:      c74d\n" +
		"       Out:       10\n" +
		"       Signature: \n" +
		"       PubKey:    7c4d\n" +
		"     Input 1:\n" +
		"       TXID:      c84e\n" +
		"       Out:       10\n" +
		"       Signature: \n" +
		"       PubKey:    7d4e\n" +
		"     Output: 0\n" +
		"       Value:  1\n" +
		"       Script: c649\n" +
		"       Contract: test\n" +
		"     Output: 1\n" +
		"       Value:  2\n" +
		"       Script: c74a\n" +
		"       Contract: test\n" +
		"     GasLimit: 1024\n" +
		"     GasPrice: 1\n" +
		"     Type: 1\n\n"

	assert.Equal(t, expectedString, tx.String())
}

func TestTransaction_GetSize(t *testing.T) {
	tx := Transaction{
		ID:       util.GenerateRandomAoB(1),
		Vin:      GenerateFakeTxInputs(),
		Vout:     GenerateFakeTxOutputs(),
		Tip:      common.NewAmount(2),
		GasLimit: common.NewAmount(3),
		GasPrice: common.NewAmount(1),
	}
	assert.Equal(t, 62, tx.GetSize())
}

func TestTransaction_GetDefaultFromTransactionAccount(t *testing.T) {
	// nil Vin
	tx := Transaction{
		ID:   util.GenerateRandomAoB(1),
		Vin:  nil,
		Vout: nil,
	}
	result := tx.GetDefaultFromTransactionAccount()
	assert.Equal(t, result.GetPubKeyHash().GenerateAddress(), result.GetAddress())

	// empty Vin
	tx.Vin = []transactionbase.TXInput{}
	result = tx.GetDefaultFromTransactionAccount()
	assert.Equal(t, result.GetPubKeyHash().GenerateAddress(), result.GetAddress())

	// normal Vin
	tx.Vin = []transactionbase.TXInput{
		{Txid: []byte{0xc7, 0x4d}, Vout: 10, Signature: nil, PubKey: []byte{0x7c, 0x4d}},
		{Txid: []byte{0xc8, 0x4e}, Vout: 10, Signature: nil, PubKey: []byte{0x7d, 0x4e}},
	}
	result = tx.GetDefaultFromTransactionAccount()
	assert.Equal(t, result.GetPubKeyHash().GenerateAddress(), result.GetAddress())

	// Vin with valid PubKey
	tx.Vin[0].PubKey = []byte{0xff, 0x62, 0x80, 0x2b, 0xec, 0xac, 0x6f, 0x6c, 0x16, 0xda, 0xde, 0x6e, 0xa9, 0x3b, 0x87, 0x8a, 0x17, 0xc7, 0x9c, 0x2e, 0x2e, 0x4c, 0x2f, 0xb9, 0x64, 0xda, 0x12, 0x60, 0x91, 0x82, 0x9a, 0x64, 0x73, 0xd7, 0xd3, 0x4b, 0x51, 0x81, 0x9e, 0xd2, 0x2e, 0xb9, 0x42, 0x1, 0xce, 0xe0, 0x19, 0x97, 0xa0, 0x8e, 0xea, 0x80, 0xb, 0x18, 0x64, 0x8b, 0xf4, 0xd4, 0xd, 0xdc, 0x91, 0x40, 0x37, 0x75}
	result = tx.GetDefaultFromTransactionAccount()
	expectedAddr := account.NewAddress("dG6HhzSdA5m7KqvJNszVSf8i5f4neAteSs")
	expectedPkh := account.PubKeyHash{0x5a, 0x1d, 0x50, 0x8c, 0x96, 0x62, 0x43, 0x85, 0xcd, 0x80, 0x1, 0xd3, 0xc0, 0x29, 0x29, 0xa5, 0x25, 0xad, 0xe, 0xea, 0x47}

	assert.Equal(t, expectedAddr, result.GetAddress())
	assert.Equal(t, expectedPkh, result.GetPubKeyHash())
}

func TestTransaction_VerifySignatures(t *testing.T) {
	keyPair := account.NewAccount().GetKeyPair()
	tx := &Transaction{
		ID: []byte{0x66},
		Vin: []transactionbase.TXInput{
			{Txid: []byte{0x12, 0x34}, Vout: 0, Signature: nil, PubKey: keyPair.GetPublicKey()},
			{Txid: []byte{0x56, 0x78}, Vout: 1, Signature: nil, PubKey: keyPair.GetPublicKey()},
		},
		Vout:     []transactionbase.TXOutput{},
		Tip:      common.NewAmount(5),
		GasLimit: common.NewAmount(1024),
		GasPrice: common.NewAmount(1),
		Type:     TxTypeNormal,
	}
	utxos := []*utxo.UTXO{
		{
			TXOutput: transactionbase.TXOutput{Value: common.NewAmount(10), PubKeyHash: []byte{}, Contract: ""},
			Txid:     []byte{0x20, 0x21},
			TxIndex:  0,
			UtxoType: 0,
		},
		{
			TXOutput: transactionbase.TXOutput{Value: common.NewAmount(20), PubKeyHash: []byte{}, Contract: ""},
			Txid:     []byte{0x22, 0x23},
			TxIndex:  1,
			UtxoType: 0,
		},
	}
	err := tx.Sign(keyPair.GetPrivateKey(), utxos)
	assert.Nil(t, err)
	success, err := tx.VerifySignatures(utxos)
	assert.True(t, success)
	assert.Nil(t, err)

	tx.Vin[0].Signature = nil
	success, err = tx.VerifySignatures(utxos)
	assert.False(t, success)
	assert.Equal(t, errval.SignaturesEmpty, err)

	tx.Vin[0].Signature = []byte("invalid")
	success, err = tx.VerifySignatures(utxos)
	assert.False(t, success)
	assert.Equal(t, errval.SignaturesInvalid, err)
}

func TestTransaction_VerifyPublicKeyHash(t *testing.T) {
	acc := account.NewAccount()
	acc2 := account.NewAccount()
	contractAcc := account.NewContractTransactionAccount()

	tests := []struct {
		name        string
		tx          *Transaction
		utxos       []*utxo.UTXO
		expectedRes bool
		expectedErr error
	}{
		{
			name: "successful",
			tx: &Transaction{
				ID: []byte{0x66},
				Vin: []transactionbase.TXInput{
					{Txid: []byte{0x12, 0x34}, Vout: 0, Signature: nil, PubKey: acc.GetKeyPair().GetPublicKey()},
					{Txid: []byte{0x56, 0x78}, Vout: 1, Signature: nil, PubKey: acc.GetKeyPair().GetPublicKey()},
				},
				Vout:     []transactionbase.TXOutput{},
				Tip:      common.NewAmount(5),
				GasLimit: common.NewAmount(1024),
				GasPrice: common.NewAmount(1),
				Type:     TxTypeNormal,
			},
			utxos: []*utxo.UTXO{
				{
					TXOutput: transactionbase.TXOutput{Value: common.NewAmount(10), PubKeyHash: acc.GetPubKeyHash(), Contract: ""},
					Txid:     []byte{0x20, 0x21},
					TxIndex:  0,
					UtxoType: 0,
				},
				{
					TXOutput: transactionbase.TXOutput{Value: common.NewAmount(20), PubKeyHash: acc.GetPubKeyHash(), Contract: ""},
					Txid:     []byte{0x22, 0x23},
					TxIndex:  1,
					UtxoType: 0,
				},
			},
			expectedRes: true,
			expectedErr: nil,
		},
		{
			name: "nil utxo PubKeyHash",
			tx: &Transaction{
				ID: []byte{0x66},
				Vin: []transactionbase.TXInput{
					{Txid: []byte{0x12, 0x34}, Vout: 0, Signature: nil, PubKey: acc.GetKeyPair().GetPublicKey()},
				},
				Vout:     []transactionbase.TXOutput{},
				Tip:      common.NewAmount(5),
				GasLimit: common.NewAmount(1024),
				GasPrice: common.NewAmount(1),
				Type:     TxTypeNormal,
			},
			utxos: []*utxo.UTXO{
				{
					TXOutput: transactionbase.TXOutput{Value: common.NewAmount(10), PubKeyHash: nil, Contract: ""},
					Txid:     []byte{0x20, 0x21},
					TxIndex:  0,
					UtxoType: 0,
				},
			},
			expectedRes: false,
			expectedErr: errval.PrevUtxosMissing,
		},
		{
			name: "invalid tx.Vin PubKey",
			tx: &Transaction{
				ID: []byte{0x66},
				Vin: []transactionbase.TXInput{
					{Txid: []byte{0x12, 0x34}, Vout: 0, Signature: nil, PubKey: []byte("invalid")},
				},
				Vout:     []transactionbase.TXOutput{},
				Tip:      common.NewAmount(5),
				GasLimit: common.NewAmount(1024),
				GasPrice: common.NewAmount(1),
				Type:     TxTypeNormal,
			},
			utxos: []*utxo.UTXO{
				{
					TXOutput: transactionbase.TXOutput{Value: common.NewAmount(10), PubKeyHash: acc.GetPubKeyHash(), Contract: ""},
					Txid:     []byte{0x20, 0x21},
					TxIndex:  0,
					UtxoType: 0,
				},
			},
			expectedRes: false,
			expectedErr: errval.IncorrectPublicKey,
		},
		{
			name: "tx vin PubKey does not match utxo PubKeyHash",
			tx: &Transaction{
				ID: []byte{0x66},
				Vin: []transactionbase.TXInput{
					{Txid: []byte{0x12, 0x34}, Vout: 0, Signature: nil, PubKey: acc2.GetKeyPair().GetPublicKey()},
				},
				Vout:     []transactionbase.TXOutput{},
				Tip:      common.NewAmount(5),
				GasLimit: common.NewAmount(1024),
				GasPrice: common.NewAmount(1),
				Type:     TxTypeNormal,
			},
			utxos: []*utxo.UTXO{
				{
					TXOutput: transactionbase.TXOutput{Value: common.NewAmount(10), PubKeyHash: acc.GetPubKeyHash(), Contract: ""},
					Txid:     []byte{0x20, 0x21},
					TxIndex:  0,
					UtxoType: 0,
				},
			},
			expectedRes: false,
			expectedErr: errval.PublicKeyHashDoesNotMatch,
		},
		{
			name: "contract",
			tx: &Transaction{
				ID: []byte{0x66},
				Vin: []transactionbase.TXInput{
					{Txid: []byte{0x12, 0x34}, Vout: 0, Signature: nil, PubKey: nil},
				},
				Vout:     []transactionbase.TXOutput{},
				Tip:      common.NewAmount(5),
				GasLimit: common.NewAmount(1024),
				GasPrice: common.NewAmount(1),
				Type:     TxTypeNormal,
			},
			utxos: []*utxo.UTXO{
				{
					TXOutput: transactionbase.TXOutput{Value: common.NewAmount(10), PubKeyHash: contractAcc.GetPubKeyHash(), Contract: ""},
					Txid:     []byte{0x20, 0x21},
					TxIndex:  0,
					UtxoType: 0,
				},
			},
			expectedRes: true,
			expectedErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			success, err := tt.tx.VerifyPublicKeyHash(tt.utxos)
			assert.Equal(t, tt.expectedRes, success)
			if tt.expectedErr != nil {
				assert.Equal(t, tt.expectedErr, err)
			} else {
				assert.Nil(t, err)
			}
		})
	}
}

func TestTransaction_Verify(t *testing.T) {
	acc := account.NewAccount()
	tx := &Transaction{
		ID: []byte{},
		Vin: []transactionbase.TXInput{
			{Txid: []byte{}, Vout: 0, Signature: nil, PubKey: nil},
			{Txid: []byte{}, Vout: 1, Signature: nil, PubKey: acc.GetKeyPair().GetPublicKey()},
		},
		Vout: []transactionbase.TXOutput{
			{Value: common.NewAmount(10), PubKeyHash: acc.GetPubKeyHash(), Contract: ""},
			{Value: common.NewAmount(20), PubKeyHash: acc.GetPubKeyHash(), Contract: ""},
		},
		Tip:      common.NewAmount(1),
		GasLimit: common.NewAmount(3000),
		GasPrice: common.NewAmount(2),
		Type:     TxTypeNormal,
	}
	utxos := []*utxo.UTXO{
		{
			TXOutput: transactionbase.TXOutput{Value: common.NewAmount(3010), PubKeyHash: acc.GetPubKeyHash(), Contract: ""},
			Txid:     []byte{0x20, 0x21},
			TxIndex:  0,
			UtxoType: 0,
		},
		{
			TXOutput: transactionbase.TXOutput{Value: common.NewAmount(3021), PubKeyHash: acc.GetPubKeyHash(), Contract: ""},
			Txid:     []byte{0x22, 0x23},
			TxIndex:  1,
			UtxoType: 0,
		},
	}
	// nil prevUtxos
	err := tx.Verify(nil)

	// invalid tx ID
	err = tx.Verify(utxos)
	assert.Equal(t, errval.TransactionIDInvalid, err)
	txCopy := tx.TrimmedCopy(true)
	tx.ID = (&txCopy).Hash()

	// invalid PubKey/PubKeyHash
	err = tx.Verify(utxos)
	assert.Equal(t, errval.IncorrectPublicKey, err)
	tx.Vin[0].PubKey = acc.GetKeyPair().GetPublicKey()
	txCopy = tx.TrimmedCopy(true)
	tx.ID = (&txCopy).Hash()

	// invalid tx vout value
	tx.Vout[0].Value = &common.Amount{*new(big.Int).SetInt64(-1)}
	txCopy = tx.TrimmedCopy(true)
	tx.ID = (&txCopy).Hash()
	err = tx.Verify(utxos)
	assert.Equal(t, errval.VoutInvalid, err)

	// invalid utxo amount
	tx.Vout[0].Value = common.NewAmount(999)
	txCopy = tx.TrimmedCopy(true)
	tx.ID = (&txCopy).Hash()
	err = tx.Verify(utxos)
	assert.Equal(t, errval.TransactionGasLimitInvalid, err)
	tx.Vout[0].Value = common.NewAmount(10)

	// invalid signature
	txCopy = tx.TrimmedCopy(true)
	tx.ID = (&txCopy).Hash()
	err = tx.Verify(utxos)
	assert.Equal(t, errval.SignaturesEmpty, err)

	// successfully verified
	err = tx.Sign(acc.GetKeyPair().GetPrivateKey(), utxos)
	assert.Nil(t, err)
	err = tx.Verify(utxos)
	assert.Nil(t, err)
}

func TestTransaction_CheckVinNum(t *testing.T) {
	tx := Transaction{
		ID:   util.GenerateRandomAoB(1),
		Vin:  []transactionbase.TXInput{},
		Vout: nil,
	}
	assert.Nil(t, tx.CheckVinNum())

	for i := 0; i < 50; i++ {
		tx.Vin = append(tx.Vin, transactionbase.TXInput{})
	}
	assert.Nil(t, tx.CheckVinNum())

	tx.Vin = append(tx.Vin, transactionbase.TXInput{})
	assert.Equal(t, errval.TransactionTooManyVin, tx.CheckVinNum())
}

func TestSendTxParam_TotalCost(t *testing.T) {
	params := SendTxParam{
		From:          account.Address{},
		SenderKeyPair: nil,
		To:            account.Address{},
		Amount:        common.NewAmount(20),
		Contract:      "",
	}

	assert.Equal(t, common.NewAmount(20), params.TotalCost())

	params.Tip = common.NewAmount(2)
	assert.Equal(t, common.NewAmount(22), params.TotalCost())

	params.GasLimit = common.NewAmount(3)
	params.GasPrice = common.NewAmount(2)
	assert.Equal(t, common.NewAmount(28), params.TotalCost())
}

func TestSetSubsidy(t *testing.T) {
	SetSubsidy(0)
	assert.Equal(t, common.NewAmount(0), Subsidy)
	SetSubsidy(10000000000)
	assert.Equal(t, common.NewAmount(10000000000), Subsidy)
}

func TestNewSendTxParam(t *testing.T) {
	fromAddress := account.Address{}
	senderKeyPair := account.NewKeyPair()
	toAddress := account.Address{}
	amount := common.NewAmount(50)
	tip := common.NewAmount(5)
	gasLimit := common.NewAmount(10)
	gasPrice := common.NewAmount(1)
	contract := "test contract"

	sendTxParam := NewSendTxParam(fromAddress, senderKeyPair, toAddress, amount, tip, gasLimit, gasPrice, contract)

	assert.Equal(t, fromAddress, sendTxParam.From)
	assert.Equal(t, senderKeyPair, sendTxParam.SenderKeyPair)
	assert.Equal(t, toAddress, sendTxParam.To)
	assert.Equal(t, amount, sendTxParam.Amount)
	assert.Equal(t, tip, sendTxParam.Tip)
	assert.Equal(t, gasLimit, sendTxParam.GasLimit)
	assert.Equal(t, gasPrice, sendTxParam.GasPrice)
	assert.Equal(t, contract, sendTxParam.Contract)
}

func TestCalculateUtxoSum(t *testing.T) {
	var utxos []*utxo.UTXO
	for i := 0; i < 10; i++ {
		newUTXO := &utxo.UTXO{
			TXOutput: transactionbase.TXOutput{
				Value:      common.NewAmount(uint64(i + 1)),
				PubKeyHash: getAoB(2),
				Contract:   "",
			},
			Txid:     getAoB(2),
			TxIndex:  0,
			UtxoType: 0,
		}
		utxos = append(utxos, newUTXO)
	}
	assert.Equal(t, common.NewAmount(55), CalculateUtxoSum(utxos))
}

func TestCalculateChange(t *testing.T) {
	tests := []struct {
		name           string
		input          *common.Amount
		amount         *common.Amount
		tip            *common.Amount
		gasLimit       *common.Amount
		gasPrice       *common.Amount
		expectedChange *common.Amount
		expectedErr    error
	}{
		{
			name:           "successful change calculation",
			input:          common.NewAmount(100),
			amount:         common.NewAmount(25),
			tip:            common.NewAmount(10),
			gasLimit:       common.NewAmount(3),
			gasPrice:       common.NewAmount(2),
			expectedChange: common.NewAmount(59),
			expectedErr:    nil,
		},
		{
			name:           "insufficient input for amount",
			input:          common.NewAmount(0),
			amount:         common.NewAmount(25),
			tip:            common.NewAmount(0),
			gasLimit:       common.NewAmount(0),
			gasPrice:       common.NewAmount(0),
			expectedChange: nil,
			expectedErr:    errval.InsufficientFund,
		},
		{
			name:           "insufficient input for tip",
			input:          common.NewAmount(10),
			amount:         common.NewAmount(10),
			tip:            common.NewAmount(1),
			gasLimit:       common.NewAmount(0),
			gasPrice:       common.NewAmount(0),
			expectedChange: nil,
			expectedErr:    errval.InsufficientFund,
		},
		{
			name:           "insufficient input for gas",
			input:          common.NewAmount(10),
			amount:         common.NewAmount(5),
			tip:            common.NewAmount(0),
			gasLimit:       common.NewAmount(3),
			gasPrice:       common.NewAmount(2),
			expectedChange: nil,
			expectedErr:    errval.InsufficientFund,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := CalculateChange(tt.input, tt.amount, tt.tip, tt.gasLimit, tt.gasPrice)
			assert.Equal(t, tt.expectedChange, res)
			assert.Equal(t, tt.expectedErr, err)
		})
	}
}
