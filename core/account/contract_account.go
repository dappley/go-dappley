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

type ContractAccount struct {
	address    Address
	pubKeyHash PubKeyHash
}

func NewContractAccount() *ContractAccount {
	account := &ContractAccount{}
	account.pubKeyHash = NewContractPubKeyHash()
	account.address = account.pubKeyHash.GenerateAddress()
	return account
}

func NewContractAccountByPubKeyHash(pubKeyHash PubKeyHash) *ContractAccount {
	account := &ContractAccount{}
	account.pubKeyHash = pubKeyHash
	account.address = account.pubKeyHash.GenerateAddress()
	return account
}

func NewContractAccountByAddress(address Address) *ContractAccount {
	account := &ContractAccount{}
	account.address = address
	account.pubKeyHash, _ = GeneratePubKeyHashByAddress(address)
	return account
}

func (ca ContractAccount) GetAddress() Address {
	return ca.address
}

func (ca ContractAccount) GetPubKeyHash() PubKeyHash {
	return ca.pubKeyHash
}

func (ca *ContractAccount) ToProto() proto.Message {
	addr := &accountpb.Address{
		Address: ca.address.address,
	}
	return &accountpb.ContractAccount{
		Address:    addr,
		PubKeyHash: ca.pubKeyHash,
	}
}

func (ca *ContractAccount) FromProto(pb proto.Message) {
	address := Address{}
	address.FromProto(pb.(*accountpb.ContractAccount).Address)
	ca.address = address
	ca.pubKeyHash = pb.(*accountpb.ContractAccount).PubKeyHash
}
