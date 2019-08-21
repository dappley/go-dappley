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
	accountpb "github.com/dappley/go-dappley/core/account/pb"
	"github.com/golang/protobuf/proto"
)

type Account struct {
	key     *KeyPair
	subKeys []*KeyPair
}

func NewAccount() *Account {
	account := &Account{}
	account.key = NewKeyPair()
	account.subKeys = []*KeyPair{}
	return account
}

func NewAccountByKey(key *KeyPair) *Account {
	account := &Account{}
	account.key = key
	account.subKeys = []*KeyPair{}
	return account
}

func (a Account) GetSubKeys() []*KeyPair {
	return a.subKeys
}

func (a Account) GetKeyPair() *KeyPair {
	return a.key
}

func (a Account) GetAllKeys() []*KeyPair {
	keys := append(a.subKeys, a.key)
	return keys
}

func (a *Account) ToProto() proto.Message {
	keysProto := []*accountpb.KeyPair{}
	for _, key := range a.subKeys {
		keysProto = append(keysProto, key.ToProto().(*accountpb.KeyPair))
	}
	return &accountpb.Account{
		KeyPair: a.key.ToProto().(*accountpb.KeyPair),
		SubKeys: keysProto,
	}
}

func (a *Account) FromProto(pb proto.Message) {
	keys := []*KeyPair{}
	for _, keyPb := range pb.(*accountpb.Account).SubKeys {
		key := &KeyPair{}
		key.FromProto(keyPb)
		keys = append(keys, key)
	}
	keyPair := &KeyPair{}
	keyPair.FromProto(pb.(*accountpb.Account).KeyPair)
	a.key = keyPair
	a.subKeys = keys
}
