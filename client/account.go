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
	accountpb "github.com/dappley/go-dappley/client/pb"
	"github.com/golang/protobuf/proto"
)

type Account struct {
	Key       *KeyPair
	Addresses []Address
}

func NewAccount() *Account {
	account := &Account{}
	account.Key = NewKeyPair()
	account.Addresses = append(account.Addresses, account.Key.GenerateAddress(false))
	return account
}

func (a Account) GetAddress() Address {
	return a.Addresses[0]
}

func (a Account) GetKeyPair() *KeyPair {
	return a.Key
}

func (a Account) GetAddresses() []Address {
	return a.Addresses
}

func (a Account) ContainAddress(address Address) bool {
	for _, value := range a.Addresses {
		if value == address {
			return true
		}
	}
	return false
}

func (a *Account) ToProto() proto.Message {
	addrsProto := []*accountpb.Address{}
	for _, addr := range a.Addresses {
		addrsProto = append(addrsProto, addr.ToProto().(*accountpb.Address))
	}
	return &accountpb.Account{
		KeyPair:   a.Key.ToProto().(*accountpb.KeyPair),
		Addresses: addrsProto,
	}
}

func (a *Account) FromProto(pb proto.Message) {
	addrs := []Address{}
	for _, addrPb := range pb.(*accountpb.Account).Addresses {
		addr := Address{}
		addr.FromProto(addrPb)
		addrs = append(addrs, addr)
	}
	keyPair := &KeyPair{}
	keyPair.FromProto(pb.(*accountpb.Account).KeyPair)
	a.Key = keyPair
	a.Addresses = addrs
}
