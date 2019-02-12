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

package core

import (
	"github.com/golang/protobuf/proto"

	"github.com/dappley/go-dappley/core/pb"
)

type TXInput struct {
	Txid      []byte
	Vout      int
	Signature []byte
	PubKey    []byte
}

func (in *TXInput) ToProto() proto.Message {
	return &corepb.TXInput{
		Txid:      in.Txid,
		Vout:      int32(in.Vout),
		Signature: in.Signature,
		PublicKey: in.PubKey,
	}
}

func (in *TXInput) FromProto(pb proto.Message) {
	in.Txid = pb.(*corepb.TXInput).GetTxid()
	in.Vout = int(pb.(*corepb.TXInput).GetVout())
	in.Signature = pb.(*corepb.TXInput).GetSignature()
	in.PubKey = pb.(*corepb.TXInput).GetPublicKey()
}
