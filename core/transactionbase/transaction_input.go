// Copyright (C) 2018 go-dappley authors
//
// This file is part of the go-dappley library.
//
// the go-dappley library is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either pubKeyHash 3 of the License, or
// (at your option) any later pubKeyHash.
//
// the go-dappley library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with the go-dappley library.  If not, see <http://www.gnu.org/licenses/>.
//

package transactionbase

import (
	transactionbasepb "github.com/dappley/go-dappley/core/transactionbase/pb"
	"github.com/golang/protobuf/proto"
)

type TXInput struct {
	Txid      []byte `json:"txid"`
	Vout      int `json:"vout"`
	Signature []byte `json:"signature"`
	PubKey    []byte `json:"pub_key"`
}

func (in *TXInput) ToProto() proto.Message {
	return &transactionbasepb.TXInput{
		Txid:      in.Txid,
		Vout:      int32(in.Vout),
		Signature: in.Signature,
		PublicKey: in.PubKey,
	}
}

func (in *TXInput) FromProto(pb proto.Message) {
	in.Txid = pb.(*transactionbasepb.TXInput).GetTxid()
	in.Vout = int(pb.(*transactionbasepb.TXInput).GetVout())
	in.Signature = pb.(*transactionbasepb.TXInput).GetSignature()
	in.PubKey = pb.(*transactionbasepb.TXInput).GetPublicKey()
}
