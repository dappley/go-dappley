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
package account

import (
	"github.com/dappley/go-dappley/crypto/keystore/secp256k1"
	logger "github.com/sirupsen/logrus"
	"testing"

	accountpb "github.com/dappley/go-dappley/core/account/pb"
	"github.com/stretchr/testify/assert"
)

func TestAccount_ToProto(t *testing.T) {
	account := NewAccount()
	privateKey, err := secp256k1.FromECDSAPrivateKey(&account.key.privateKey)
	if err != nil {
		logger.Error("Keypair: ToProto: Can not convert private key to bytes")
	}
	expected := &accountpb.Account{
		KeyPair: &accountpb.KeyPair{
			PrivateKey: privateKey,
			PublicKey: account.key.publicKey,
		},
		Address: &accountpb.Address{Address: account.address.address},
		PubKeyHash: account.pubKeyHash,
	}
	assert.Equal(t, expected, account.ToProto())
}

func TestAccount_FromProto(t *testing.T) {
	expected := NewAccount()
	account := &Account{}
	privateKey, err := secp256k1.FromECDSAPrivateKey(&expected.key.privateKey)
	if err != nil {
		logger.Error("Keypair: ToProto: Can not convert private key to bytes")
	}
	accountProto := &accountpb.Account{
		KeyPair: &accountpb.KeyPair{
			PrivateKey: privateKey,
			PublicKey:  expected.key.publicKey,
		},
		Address:    &accountpb.Address{Address: expected.address.address},
		PubKeyHash: expected.pubKeyHash,
	}
	account.FromProto(accountProto)
	assert.Equal(t, expected, account)
}

func TestAccount_IsValid(t *testing.T) {
	account := NewAccount()
	assert.True(t, account.IsValid())

	account.address.address = "address000000000000000000000000011"
	assert.False(t, account.IsValid())
}
