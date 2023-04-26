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
	nonce                 uint64
}

func NewUTXOInfo() *UTXOInfo {
	return &UTXOInfo{[]byte{}, []byte{}, 0}
}

func (utxoHead *UTXOInfo) ToProto() proto.Message {
	return &utxopb.UtxoInfo{
		LastUtxoKey:           utxoHead.lastUTXOKey,
		UtxoCreateContractKey: utxoHead.createContractUTXOKey,
		Nonce:                 utxoHead.nonce,
	}
}

func (utxoHead *UTXOInfo) FromProto(pb proto.Message) {
	utxoHeadpb := pb.(*utxopb.UtxoInfo)
	utxoHead.lastUTXOKey = utxoHeadpb.LastUtxoKey
	utxoHead.createContractUTXOKey = utxoHeadpb.UtxoCreateContractKey
	utxoHead.nonce = utxoHeadpb.Nonce
}

func (utxoHead *UTXOInfo) GetLastUtxoKey() []byte {
	return utxoHead.lastUTXOKey
}

func (utxoHead *UTXOInfo) SetLastUtxoKey(lastUTXOKey []byte) {
	utxoHead.lastUTXOKey = lastUTXOKey
}

func (utxoHead *UTXOInfo) GetCreateContractUTXOKey() []byte {
	return utxoHead.createContractUTXOKey
}

func (utxoHead *UTXOInfo) SetCreateContractUTXOKey(createContractUTXOKey []byte) {
	utxoHead.createContractUTXOKey = createContractUTXOKey
}

func (utxoHead *UTXOInfo) GetNonce() uint64 {
	return utxoHead.nonce
}

func (utxoHead *UTXOInfo) SetNonce(nonce uint64) {
	utxoHead.nonce = nonce
}
