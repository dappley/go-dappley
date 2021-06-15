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

package transactionbase

import (
	"testing"

	"github.com/dappley/go-dappley/common"
	"github.com/dappley/go-dappley/core/account"
	transactionbasepb "github.com/dappley/go-dappley/core/transactionbase/pb"
	"github.com/golang/protobuf/proto"
	"github.com/stretchr/testify/assert"
)

func TestTXOutput_Proto(t *testing.T) {
	vout := TXOutput{
		common.NewAmount(1),
		account.PubKeyHash([]byte("PubKeyHash")),
		"contract",
	}

	pb := vout.ToProto()
	var i interface{} = pb
	_, correct := i.(proto.Message)
	assert.Equal(t, true, correct)
	mpb, err := proto.Marshal(pb)
	assert.Nil(t, err)

	newpb := &transactionbasepb.TXOutput{}
	err = proto.Unmarshal(mpb, newpb)
	assert.Nil(t, err)

	vout2 := TXOutput{}
	vout2.FromProto(newpb)

	assert.Equal(t, vout, vout2)
}

func TestTXOutput_IsFoundInRewardStorage(t *testing.T) {

	tests := []struct {
		name          string
		vout          TXOutput
		rewardStorage map[string]string
		expectedRes   bool
	}{
		{"normal",
			TXOutput{
				common.NewAmount(1),
				account.PubKeyHash([]byte{
					0x5a, 0xc9, 0x85, 0x37, 0x92, 0x37, 0x76, 0x80,
					0xb1, 0x31, 0xa1, 0xab, 0xb, 0x5b, 0xa6, 0x49,
					0xe5, 0x27, 0xf0, 0x42, 0x5d}),
				"contract",
			},
			map[string]string{"dXnq2R6SzRNUt7ZANAqyZc2P9ziF6vYekB": "1"},
			true,
		},
		{"NotInStorage",
			TXOutput{
				common.NewAmount(1),
				account.PubKeyHash([]byte{
					0x5a, 0xc9, 0x85, 0x37, 0x92, 0x37, 0x76, 0x80,
					0xb1, 0x31, 0xa1, 0xab, 0xb, 0x5b, 0xa6, 0x49,
					0xe5, 0x27, 0xf0, 0x42, 0x5d}),
				"contract",
			},
			map[string]string{},
			false,
		},
		{"InvalidAmount",
			TXOutput{
				common.NewAmount(1),
				account.PubKeyHash([]byte{
					0x5a, 0xc9, 0x85, 0x37, 0x92, 0x37, 0x76, 0x80,
					0xb1, 0x31, 0xa1, 0xab, 0xb, 0x5b, 0xa6, 0x49,
					0xe5, 0x27, 0xf0, 0x42, 0x5d}),
				"contract",
			},
			map[string]string{"dXnq2R6SzRNUt7ZANAqyZc2P9ziF6vYekB": "1asdf"},
			false,
		},
		{"NilInput",
			TXOutput{
				common.NewAmount(1),
				account.PubKeyHash([]byte{
					0x5a, 0xc9, 0x85, 0x37, 0x92, 0x37, 0x76, 0x80,
					0xb1, 0x31, 0xa1, 0xab, 0xb, 0x5b, 0xa6, 0x49,
					0xe5, 0x27, 0xf0, 0x42, 0x5d}),
				"contract",
			},
			nil,
			false,
		},
		{"WrongValue",
			TXOutput{
				common.NewAmount(1),
				account.PubKeyHash([]byte{
					0x5a, 0xc9, 0x85, 0x37, 0x92, 0x37, 0x76, 0x80,
					0xb1, 0x31, 0xa1, 0xab, 0xb, 0x5b, 0xa6, 0x49,
					0xe5, 0x27, 0xf0, 0x42, 0x5d}),
				"contract",
			},
			map[string]string{"dXnq2R6SzRNUt7ZANAqyZc2P9ziF6vYekB": "3"},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expectedRes, tt.vout.IsFoundInRewardStorage(tt.rewardStorage))
		})
	}
}

func TestNewTxOut(t *testing.T) {
	pubKeyBytes := account.PubKeyHash([]byte{0x5a, 0xb1, 0x34, 0x4c, 0x17, 0x67, 0x4c, 0x18, 0xd1, 0xa2, 0xdc, 0xea, 0x9f, 0x17, 0x16, 0xe0, 0x49, 0xf4, 0xa0, 0x5e, 0x6c})
	transactionAccount := account.NewContractAccountByPubKeyHash(pubKeyBytes)

	expected := &TXOutput{
		Value:      common.NewAmount(10),
		PubKeyHash: transactionAccount.GetPubKeyHash(),
		Contract:   "test",
	}
	assert.Equal(t, expected, NewTxOut(common.NewAmount(10), transactionAccount, "test"))
}

func TestNewTXOutput(t *testing.T) {
	pubKeyBytes := account.PubKeyHash([]byte{0x5a, 0xb1, 0x34, 0x4c, 0x17, 0x67, 0x4c, 0x18, 0xd1, 0xa2, 0xdc, 0xea, 0x9f, 0x17, 0x16, 0xe0, 0x49, 0xf4, 0xa0, 0x5e, 0x6c})
	transactionAccount := account.NewContractAccountByPubKeyHash(pubKeyBytes)

	expected := &TXOutput{
		Value:      common.NewAmount(13),
		PubKeyHash: transactionAccount.GetPubKeyHash(),
		Contract:   "",
	}
	assert.Equal(t, expected, NewTXOutput(common.NewAmount(13), transactionAccount))
}

func TestNewContractTXOutput(t *testing.T) {
	pubKeyBytes := account.PubKeyHash([]byte{0x5a, 0xb1, 0x34, 0x4c, 0x17, 0x67, 0x4c, 0x18, 0xd1, 0xa2, 0xdc, 0xea, 0x9f, 0x17, 0x16, 0xe0, 0x49, 0xf4, 0xa0, 0x5e, 0x6c})
	transactionAccount := account.NewContractAccountByPubKeyHash(pubKeyBytes)

	expected := &TXOutput{
		Value:      common.NewAmount(0),
		PubKeyHash: transactionAccount.GetPubKeyHash(),
		Contract:   "contract",
	}
	assert.Equal(t, expected, NewContractTXOutput(transactionAccount, "contract"))
}

func TestTXOutput_GetAddress(t *testing.T) {
	pubKeyBytes := account.PubKeyHash([]byte{0x5a, 0xb1, 0x34, 0x4c, 0x17, 0x67, 0x4c, 0x18, 0xd1, 0xa2, 0xdc, 0xea, 0x9f, 0x17, 0x16, 0xe0, 0x49, 0xf4, 0xa0, 0x5e, 0x6c})
	transactionAccount := account.NewContractAccountByPubKeyHash(pubKeyBytes)

	txo := &TXOutput{
		Value:      common.NewAmount(0),
		PubKeyHash: transactionAccount.GetPubKeyHash(),
		Contract:   "contract",
	}

	assert.Equal(t, account.NewAddress("dVaFsQL9He4Xn4CEUh1TCNtfEhHNHKX3hs"), txo.GetAddress())
}
