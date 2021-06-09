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
	"errors"
	"testing"

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

func TestTransaction_GetToHashBytes(t *testing.T) {
	tx1 := Transaction{
		ID: []byte{88},
		Vin: []transactionbase.TXInput{
			{
				Txid: []byte{24, 48},
				Vout: 10,
				Signature: nil,
				PubKey: []byte{32, 64},
			},
		},
		Vout: []transactionbase.TXOutput{
			{
				Value: common.NewAmount(1),
				PubKeyHash: account.PubKeyHash([]byte{64, 128}),
				Contract: "test",
			},
		},
	}
	tx2 := Transaction{
		ID: []byte{102},
		Vin: []transactionbase.TXInput{
			{
				Txid: []byte{199, 77},
				Vout: 10,
				Signature: nil,
				PubKey: []byte{124, 77},
			},
			{
				Txid: []byte{200, 78},
				Vout: 10,
				Signature: nil,
				PubKey: []byte{125, 78},
			},
		},
		Vout: []transactionbase.TXOutput{
			{
				Value: common.NewAmount(1),
				PubKeyHash: account.PubKeyHash([]byte{198, 73}),
				Contract: "test",
			},
			{
				Value: common.NewAmount(2),
				PubKeyHash: account.PubKeyHash([]byte{199, 74}),
				Contract: "test",
			},
		},
		Tip: common.NewAmount(5),
		GasLimit: common.NewAmount(1024),
		GasPrice: common.NewAmount(1),
		Type: TxTypeNormal,
	}
	tx3 := Transaction{
		ID: []byte{0},
		Vin: []transactionbase.TXInput{},
		Vout: []transactionbase.TXOutput{},
	}
	expected1 := []byte{0x18, 0x30, 0x0, 0x0, 0x0, 0xa, 0x20, 0x40, 0x1, 0x40, 0x80, 0x74, 0x65, 0x73, 0x74}
	expected2 := []byte{0xc7, 0x4d, 0x0, 0x0, 0x0, 0xa, 0x7c, 0x4d, 0xc8, 0x4e, 0x0, 0x0, 0x0, 0xa, 0x7d, 0x4e, 0x1, 0xc6, 0x49, 0x74, 0x65, 0x73, 0x74, 0x2, 0xc7, 0x4a, 0x74, 0x65, 0x73, 0x74, 0x5, 0x4, 0x0, 0x1, 0x0, 0x0, 0x0, 0x1}
	assert.Equal(t, expected1, tx1.GetToHashBytes())
	assert.Equal(t, expected2, tx2.GetToHashBytes())
	assert.Equal(t, []byte(nil), tx3.GetToHashBytes())
}

func TestTransaction_Hash(t *testing.T) {
	tx1 := Transaction{
		ID: []byte{102},
		Vin: []transactionbase.TXInput{
			{
				Txid: []byte{199, 77},
				Vout: 10,
				Signature: nil,
				PubKey: []byte{124, 77},
			},
			{
				Txid: []byte{200, 78},
				Vout: 10,
				Signature: nil,
				PubKey: []byte{125, 78},
			},
		},
		Vout: []transactionbase.TXOutput{
			{
				Value: common.NewAmount(1),
				PubKeyHash: account.PubKeyHash([]byte{198, 73}),
				Contract: "test",
			},
			{
				Value: common.NewAmount(2),
				PubKeyHash: account.PubKeyHash([]byte{199, 74}),
				Contract: "test",
			},
		},
		Tip: common.NewAmount(5),
		GasLimit: common.NewAmount(1024),
		GasPrice: common.NewAmount(1),
		Type: TxTypeNormal,
	}
	tx2 := Transaction{
		ID: []byte{0},
		Vin: []transactionbase.TXInput{},
		Vout: []transactionbase.TXOutput{},
	}
	expected1 := []byte{0x4f, 0xda, 0x27, 0xf8, 0x2c, 0xa, 0x49, 0x9d, 0x8c, 0x19, 0x37, 0x46, 0x2c, 0x19, 0x9, 0xc6, 0x96, 0x54, 0x31, 0x99, 0x9f, 0x1f, 0xc6, 0x84, 0xf5, 0xc0, 0xc5, 0x6b, 0xbd, 0xbd, 0xe, 0xc8}
	expected2 := []byte{0xe3, 0xb0, 0xc4, 0x42, 0x98, 0xfc, 0x1c, 0x14, 0x9a, 0xfb, 0xf4, 0xc8, 0x99, 0x6f, 0xb9, 0x24, 0x27, 0xae, 0x41, 0xe4, 0x64, 0x9b, 0x93, 0x4c, 0xa4, 0x95, 0x99, 0x1b, 0x78, 0x52, 0xb8, 0x55}
	assert.Equal(t, expected1, tx1.Hash())
	assert.Equal(t, expected2, tx2.Hash())
}

func TestTransaction_DeepCopy(t *testing.T) {
	original := Transaction{
		ID: util.GenerateRandomAoB(1),
		Vin: GenerateFakeTxInputs(),
		Vout: GenerateFakeTxOutputs(),
		Tip: common.NewAmount(5),
		GasLimit: common.NewAmount(1024),
		GasPrice: common.NewAmount(1),
		CreateTime: 99,
		Type: TxTypeNormal,
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
	tx1 := Transaction{
		ID: []byte{102},
		Vin: []transactionbase.TXInput{
			{
				Txid: []byte{199, 77},
				Vout: 10,
				Signature: nil,
				PubKey: []byte{124, 77},
			},
			{
				Txid: []byte{200, 78},
				Vout: 10,
				Signature: nil,
				PubKey: []byte{125, 78},
			},
		},
		Vout: []transactionbase.TXOutput{
			{
				Value: common.NewAmount(1),
				PubKeyHash: account.PubKeyHash([]byte{198, 73}),
				Contract: "test",
			},
			{
				Value: common.NewAmount(2),
				PubKeyHash: account.PubKeyHash([]byte{199, 74}),
				Contract: "test",
			},
		},
		Tip: common.NewAmount(5),
		GasLimit: common.NewAmount(1024),
		GasPrice: common.NewAmount(1),
		Type: TxTypeNormal,
	}
	tx2 := Transaction {
		ID: []byte{0xe3, 0xb0, 0xc4, 0x42, 0x98, 0xfc, 0x1c, 0x14, 0x9a, 0xfb, 0xf4, 0xc8, 0x99, 0x6f, 0xb9, 0x24, 0x27, 0xae, 0x41, 0xe4, 0x64, 0x9b, 0x93, 0x4c, 0xa4, 0x95, 0x99, 0x1b, 0x78, 0x52, 0xb8, 0x55},
		Vin: []transactionbase.TXInput{},
		Vout: []transactionbase.TXOutput{},
	}
	verified, err := tx1.verifyID()
	assert.False(t, verified)
	assert.Equal(t, errors.New("Transaction: ID is invalid"), err)

	tx1.ID = tx1.Hash()
	verified, err = tx1.verifyID()
	assert.True(t, verified)
	assert.Nil(t, err)

	verified, err = tx2.verifyID()
	assert.True(t, verified)
	assert.Nil(t, err)
}

func TestTransaction_verifyAmount(t *testing.T) {
	tx1 := Transaction{
		ID:        	util.GenerateRandomAoB(1),
		Vin:        GenerateFakeTxInputs(),
		Vout:       GenerateFakeTxOutputs(),
		Tip:        common.NewAmount(2),
		GasLimit:   common.NewAmount(3),
		GasPrice:   common.NewAmount(1),
	}

	verified, err := tx1.verifyAmount(common.NewAmount(20), common.NewAmount(15))
	assert.True(t, verified)
	assert.Nil(t, err)

	verified, err = tx1.verifyAmount(common.NewAmount(1), common.NewAmount(10))
	assert.False(t, verified)
	assert.Equal(t, errors.New("Transaction: amount is invalid"), err)
}
