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

package utxo

import (
	utxopb "github.com/dappley/go-dappley/core/utxo/pb"
	"github.com/golang/protobuf/proto"
)

type UTXOInfo struct {
	LastUtxoKey           []byte
	UtxoCreateContractKey []byte
}

func (utxoHead *UTXOInfo) ToProto() proto.Message {
	return &utxopb.UtxoInfo{
		LastUtxoKey:           utxoHead.LastUtxoKey,
		UtxoCreateContractKey: utxoHead.UtxoCreateContractKey,
	}
}

func (utxoHead *UTXOInfo) FromProto(pb proto.Message) {
	utxoHeadpb := pb.(*utxopb.UtxoInfo)
	utxoHead.LastUtxoKey = utxoHeadpb.LastUtxoKey
	utxoHead.UtxoCreateContractKey = utxoHeadpb.UtxoCreateContractKey
}
