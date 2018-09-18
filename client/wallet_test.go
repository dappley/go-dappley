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
	"github.com/stretchr/testify/assert"
	"github.com/dappley/go-dappley/core"
)

func TestWallet_ContainAddress(t *testing.T) {
	wallet := NewWallet()
	tests := []struct{
		name 	 string
		input    core.Address
		expected bool
	}{{"contains address",wallet.GetAddress(), true},
	  {"does not contain address",core.Address{},false},
	}

	for _,tt := range tests{
		t.Run(tt.name,func(t *testing.T){

			assert.Equal(t,tt.expected, wallet.ContainAddress(tt.input))
		})
	}
}

func TestWallet_ContainAddress_with_passphrase(t *testing.T) {
	wallet := NewWalletWithPassphrase("password")
	tests := []struct{
		name 	 string
		input    core.Address
		expected bool
	}{{"contains address",wallet.GetAddress(), true},
		{"does not contain address",core.Address{},false},
	}

	for _,tt := range tests{
		t.Run(tt.name,func(t *testing.T){

			assert.Equal(t,tt.expected, wallet.ContainAddress(tt.input))
		})
	}
}
