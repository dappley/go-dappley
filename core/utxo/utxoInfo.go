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
	lastUTXOKey           []byte
	createContractUTXOKey []byte
}

func NewUTXOInfo() *UTXOInfo {
	return &UTXOInfo{[]byte{}, []byte{}}
}

func (utxoHead *UTXOInfo) ToProto() proto.Message {
	return &utxopb.UtxoInfo{
		LastUtxoKey:           utxoHead.lastUTXOKey,
		UtxoCreateContractKey: utxoHead.createContractUTXOKey,
	}
}

func (utxoHead *UTXOInfo) FromProto(pb proto.Message) {
	utxoHeadpb := pb.(*utxopb.UtxoInfo)
	utxoHead.lastUTXOKey = utxoHeadpb.LastUtxoKey
	utxoHead.createContractUTXOKey = utxoHeadpb.UtxoCreateContractKey
}

func (utxoHead *UTXOInfo) GetLastUtxoKey  () []byte {
	return utxoHead.lastUTXOKey
}

func (utxoHead *UTXOInfo) SetLastUtxoKey(lastUTXOKey []byte) {
	utxoHead.lastUTXOKey = lastUTXOKey
}

func (utxoHead *UTXOInfo) GetCreateContractUTXOKey  () []byte {
	return utxoHead.createContractUTXOKey
}

func (utxoHead *UTXOInfo) SetCreateContractUTXOKey(createContractUTXOKey []byte) {
	utxoHead.createContractUTXOKey = createContractUTXOKey
}