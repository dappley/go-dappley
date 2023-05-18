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
package account

import (
	"bytes"

	accountpb "github.com/dappley/go-dappley/core/account/pb"
	"github.com/golang/protobuf/proto"
)

type Account struct {
	key        *KeyPair
	address    Address
	pubKeyHash PubKeyHash
}

func NewAccount() *Account {
	account := &Account{}
	account.key = NewKeyPair()
	account.address = account.key.GenerateAddress()
	account.pubKeyHash = NewUserPubKeyHash(account.key.GetPublicKey())
	return account
}

func NewAccountByKey(key *KeyPair) *Account {
	account := &Account{}
	account.key = key
	account.address = account.key.GenerateAddress()
	account.pubKeyHash = NewUserPubKeyHash(account.key.GetPublicKey())
	return account
}

func NewAccountByPrivateKey(privKey string) *Account {
	kp := GenerateKeyPairByPrivateKey(privKey)
	return NewAccountByKey(kp)
}

func (a Account) GetKeyPair() *KeyPair {
	return a.key
}

func (a Account) GetAddress() Address {
	return a.address
}

func (a *Account) SetAddress(addr string) {
	address := Address{}
	address.address = addr
	a.address = address
	return
}

func (a Account) GetPubKeyHash() PubKeyHash {
	return a.pubKeyHash
}

func (a *Account) ToProto() proto.Message {
	addr := &accountpb.Address{
		Address: a.address.address,
	}
	return &accountpb.Account{
		KeyPair:    a.key.ToProto().(*accountpb.KeyPair),
		Address:    addr,
		PubKeyHash: a.pubKeyHash,
	}
}
func (a *Account) IsValid() bool {
	actualChecksum := a.address.getAddressCheckSum()
	if actualChecksum == nil {
		return false
	}
	targetChecksum := Checksum(a.pubKeyHash)
	if bytes.Compare(actualChecksum, targetChecksum) == 0 {
		return true
	}
	return false
}

func (a *Account) FromProto(pb proto.Message) {
	keyPair := &KeyPair{}
	keyPair.FromProto(pb.(*accountpb.Account).KeyPair)
	a.key = keyPair
	address := Address{}
	address.FromProto(pb.(*accountpb.Account).Address)
	a.address = address
	a.pubKeyHash = pb.(*accountpb.Account).PubKeyHash
}
