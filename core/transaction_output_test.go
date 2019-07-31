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
	"testing"

	"github.com/dappley/go-dappley/core/account"
	"github.com/dappley/go-dappley/common"
	corepb "github.com/dappley/go-dappley/core/pb"
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

	newpb := &corepb.TXOutput{}
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
