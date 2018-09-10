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

package core

import (
	"bytes"

	"github.com/gogo/protobuf/proto"
	"github.com/dappley/go-dappley/common"
	"github.com/dappley/go-dappley/util"
	"github.com/dappley/go-dappley/core/pb"
)

type TXOutput struct {
	Value      *common.Amount
	PubKeyHash []byte
}

func (out *TXOutput) Lock(address []byte) {
	out.PubKeyHash = HashAddress(address)
}

func HashAddress(address []byte) []byte{
	pubKeyHash := util.Base58Decode(address)
	return pubKeyHash[1 : len(pubKeyHash)-4]
}

func (out *TXOutput) IsLockedWithKey(pubKeyHash []byte) bool {
	return bytes.Compare(out.PubKeyHash, pubKeyHash) == 0
}

func NewTXOutput(value *common.Amount, address string) *TXOutput {
	txo := &TXOutput{value, nil}
	txo.Lock([]byte(address))
	return txo
}

func (out *TXOutput) ToProto() (proto.Message){
	return &corepb.TXOutput{
		Value:		out.Value.Bytes(),
		PubKeyHash:	out.PubKeyHash,
	}
}

func (out *TXOutput) FromProto(pb proto.Message){
	out.Value = common.NewAmountFromBytes(pb.(*corepb.TXOutput).Value)
	out.PubKeyHash = pb.(*corepb.TXOutput).PubKeyHash
}