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
	"github.com/dappley/go-dappley/client/pb"
	"github.com/dappley/go-dappley/core"
	"github.com/dappley/go-dappley/core/pb"
	"github.com/golang/protobuf/proto"
)

type Wallet struct {
	Key       *core.KeyPair
	Addresses []core.Address
}

func NewWallet() *Wallet {
	wallet := &Wallet{}
	wallet.Key = core.NewKeyPair()
	wallet.Addresses = append(wallet.Addresses, wallet.Key.GenerateAddress(false))
	return wallet
}

func (w Wallet) GetAddress() core.Address {
	return w.Addresses[0]
}

func (w Wallet) GetKeyPair() *core.KeyPair {
	return w.Key
}

func (w Wallet) GetAddresses() []core.Address {
	return w.Addresses
}

func (w Wallet) ContainAddress(address core.Address) bool {
	for _, value := range w.Addresses {
		if value == address {
			return true
		}
	}
	return false
}

func (w *Wallet) ToProto() proto.Message {
	addrsProto := []*corepb.Address{}
	for _, addr := range w.Addresses {
		addrsProto = append(addrsProto, addr.ToProto().(*corepb.Address))
	}
	return &walletpb.Wallet{
		KeyPair:   w.Key.ToProto().(*corepb.KeyPair),
		Addresses: addrsProto,
	}
}

func (w *Wallet) FromProto(pb proto.Message) {
	addrs := []core.Address{}
	for _, addrPb := range pb.(*walletpb.Wallet).Addresses {
		addr := core.Address{}
		addr.FromProto(addrPb)
		addrs = append(addrs, addr)
	}
	keyPair := &core.KeyPair{}
	keyPair.FromProto(pb.(*walletpb.Wallet).KeyPair)
	w.Key = keyPair
	w.Addresses = addrs
}
