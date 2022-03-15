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
	"testing"

	"github.com/dappley/go-dappley/crypto/keystore/secp256k1"
	logger "github.com/sirupsen/logrus"

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
			PublicKey:  account.key.publicKey,
		},
		Address:    &accountpb.Address{Address: account.address.address},
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

func TestAccount_SetAddress(t *testing.T) {
	account := NewAccount()
	account.SetAddress("testValue")
	assert.Equal(t, "testValue", account.address.address)
}

func TestAccount_NewAccountByKey(t *testing.T) {
	testKey := "bb23d2ff19f5b16955e8a24dca34dd520980fe3bddca2b3e1b56663f0ec1aa7e"
	testKeyPair := GenerateKeyPairByPrivateKey(testKey)
	account := NewAccountByKey(testKeyPair)
	assert.Equal(t, testKeyPair, account.key)
	assert.Equal(t, "dQEooMsqp23RkPsvZXj3XbsRh9BUyGz2S9", account.address.address)
	assert.Equal(t, "5a76ae00ceb16dbc3ec303553cc9fb7249e7e5f0aa", account.pubKeyHash.String())

	defer func() {
		if r := recover(); r == nil {
			t.Errorf("The code did not panic")
		}
	}()
	_ = NewAccountByKey(nil)
}

func TestAccount_NewAccountByPrivateKey(t *testing.T) {
	testKey := "bb23d2ff19f5b16955e8a24dca34dd520980fe3bddca2b3e1b56663f0ec1aa7e"
	testKeyPair := GenerateKeyPairByPrivateKey(testKey)
	account := NewAccountByPrivateKey(testKey)
	assert.Equal(t, testKeyPair, account.key)
	assert.Equal(t, "dQEooMsqp23RkPsvZXj3XbsRh9BUyGz2S9", account.address.address)
	assert.Equal(t, "5a76ae00ceb16dbc3ec303553cc9fb7249e7e5f0aa", account.pubKeyHash.String())

	defer func() {
		if r := recover(); r == nil {
			t.Errorf("The code did not panic")
		}
	}()
	account = NewAccountByPrivateKey("")
}
