// Copyright (C) 2018 go-dappley authors
//
// This file is part of the go-dappley library.
//
// the go-dappley library is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// the go-dappley library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with the go-dappley library.  If not, see <http://www.gnu.org/licenses/>.
//
package client

import (
	"testing"

	accountpb "github.com/dappley/go-dappley/client/pb"
	"github.com/golang/protobuf/proto"
	"github.com/stretchr/testify/assert"
)

func TestAccount_ContainAddress(t *testing.T) {
	account := NewAccount()
	tests := []struct {
		name     string
		input    Address
		expected bool
	}{{"contains address", account.GetKeyPair().GenerateAddress(), true},
		{"does not contain address", Address{}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, account.ContainAddress(tt.input))
		})
	}
}

func TestAccount_Proto(t *testing.T) {
	account := NewAccount()
	rawBytes, err := proto.Marshal(account.ToProto())
	assert.Nil(t, err)
	accountProto := &accountpb.Account{}
	err = proto.Unmarshal(rawBytes, accountProto)
	assert.Nil(t, err)
	account1 := &Account{}
	account1.FromProto(accountProto)
	assert.Equal(t, account, account1)
}
