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

func TestNewRewardTx(t *testing.T) {
	rewards := map[string]string{
		"dXnq2R6SzRNUt7ZANAqyZc2P9ziF6vYekB": "8",
		"dastXXWLe5pxbRYFhcyUq8T3wb5srWkHKa": "9",
	}
	tx := NewRewardTx(5, rewards)

	values := []*common.Amount{tx.Vout[0].Value, tx.Vout[1].Value}
	assert.Contains(t, values, common.NewAmount(8))
	assert.Contains(t, values, common.NewAmount(9))
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

func TestTransaction_GetContractAddress(t *testing.T) {

	tests := []struct {
		name        string
		addr        string
		expectedRes string
	}{
		{
			name:        "ContainsContractAddress",
			addr:        "cavQdWxvUQU1HhBg1d7zJFwhf31SUaQwop",
			expectedRes: "cavQdWxvUQU1HhBg1d7zJFwhf31SUaQwop",
		},
		{
			name:        "ContainsUserAddress",
			addr:        "dGDrVKjCG3sdXtDUgWZ7Fp3Q97tLhqWivf",
			expectedRes: "",
		},
		{
			name:        "EmptyInput",
			addr:        "",
			expectedRes: "",
		},
		{
			name:        "InvalidAddress",
			addr:        "dsdGDrVKjCG3sdXtDUgWZ7Fp3Q97tLhqWivf",
			expectedRes: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			acc := account.NewContractAccountByAddress(account.NewAddress(tt.addr))
			tx := &Transaction{
				nil,
				nil,
				[]transactionbase.TXOutput{
					{nil,
						acc.GetPubKeyHash(),
						"",
					},
				},
				common.NewAmount(0),
				common.NewAmount(0),
				common.NewAmount(0),
				0,
				TxTypeContract,
			}
			ctx := NewTxContract(tx)
			if ctx != nil {
				assert.Equal(t, account.NewAddress(tt.expectedRes), ctx.GetContractAddress())
			}
		})
	}
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
