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
	"hash/fnv"
	"strconv"

	"github.com/dappley/go-dappley/common"
	"github.com/dappley/go-dappley/core/pb"
	"github.com/golang/protobuf/proto"
	"github.com/raviqqe/hamt"
	logger "github.com/sirupsen/logrus"
)

// UTXOTx holds txid_vout and UTXO pairs
type UTXOTx hamt.Map

type StringEntry string

func (key *StringEntry) Hash() uint32 {
	h := fnv.New32a()
	h.Write([]byte(string(*key)))
	return h.Sum32()
}

func (key *StringEntry) Equal(other hamt.Entry) bool {
	otherStr, ok := other.(*StringEntry)
	if !ok {
		return false
	}

	return string(*key) == string(*otherStr)
}

func NewUTXOTx() UTXOTx {
	return UTXOTx(hamt.NewMap())
}

// Construct with UTXO data
func NewUTXOTxWithData(utxo *UTXO) UTXOTx {
	key := StringEntry(string(utxo.Txid) + "_" + strconv.Itoa(utxo.TxIndex))
	return UTXOTx(hamt.NewMap().Insert(&key, utxo))
}

func DeserializeUTXOTx(d []byte) UTXOTx {
	utxoTx := NewUTXOTx()

	utxoList := &corepb.UtxoList{}
	err := proto.Unmarshal(d, utxoList)
	if err != nil {
		logger.WithFields(logger.Fields{"error": err}).Error("UtxoTx: parse UtxoTx failed.")
		return utxoTx
	}

	for _, utxoPb := range utxoList.Utxos {
		var utxo = &UTXO{}
		utxo.FromProto(utxoPb)
		utxoTx = utxoTx.PutUtxo(utxo)
	}

	return utxoTx
}

func (utxoTx UTXOTx) Serialize() []byte {
	utxoList := &corepb.UtxoList{}

	_, utxo, nextUtxoTx := utxoTx.Iterator()
	for utxo != nil {
		utxoList.Utxos = append(utxoList.Utxos, utxo.ToProto().(*corepb.Utxo))
		_, utxo, nextUtxoTx = nextUtxoTx.Iterator()
	}

	bytes, err := proto.Marshal(utxoList)
	if err != nil {
		logger.WithFields(logger.Fields{"error": err}).Error("UtxoTx: serialize UtxoTx failed.")
		return nil
	}
	return bytes
}

// Returns utxo info by transaction id and vout index
func (utxoTx UTXOTx) GetUtxo(txid []byte, vout int) *UTXO {
	key := StringEntry(string(txid) + "_" + strconv.Itoa(vout))
	value := hamt.Map(utxoTx).Find(&key)
	utxo, ok := value.(*UTXO)
	if !ok {
		return nil
	}

	return utxo
}

// Add new utxo to map
func (utxoTx UTXOTx) PutUtxo(utxo *UTXO) UTXOTx {
	key := StringEntry(string(utxo.Txid) + "_" + strconv.Itoa(utxo.TxIndex))
	return UTXOTx(hamt.Map(utxoTx).Insert(&key, utxo))
}

// Delete invalid element in map
func (utxoTx *UTXOTx) RemoveUtxo(txid []byte, vout int) UTXOTx {
	key := StringEntry(string(txid) + "_" + strconv.Itoa(vout))
	newMap := UTXOTx(hamt.Map(*utxoTx).Delete(&key))
	return newMap
}

func (utxoTx UTXOTx) Iterator() (string, *UTXO, UTXOTx) {
	key, value, nextMap := hamt.Map(utxoTx).FirstRest()
	if key == nil {
		return "", nil, NewUTXOTx()
	}

	keyStr, ok := key.(*StringEntry)
	if !ok {
		logger.Panic("UtxoTx: invalid key type")
	}

	utxo, ok := value.(*UTXO)
	if !ok {
		logger.Panic("UtxoTx: invalid value type")
	}

	return string(*keyStr), utxo, UTXOTx(nextMap)
}

func (utxoTx UTXOTx) Size() int {
	return hamt.Map(utxoTx).Size()
}

func (utxoTx UTXOTx) GetAllUtxos() []*UTXO {
	var utxos []*UTXO
	_, utxo, nextUtxoTx := utxoTx.Iterator()
	for utxo != nil {
		utxos = append(utxos, utxo)
		_, utxo, nextUtxoTx = nextUtxoTx.Iterator()
	}

	return utxos
}

func (utxoTx UTXOTx) PrepareUtxos(amount *common.Amount) ([]*UTXO, bool) {
	sum := common.NewAmount(0)

	if utxoTx.Size() < 1 {
		return nil, false
	}

	var utxos []*UTXO
	_, utxo, nextUtxoTx := utxoTx.Iterator()
	for utxo != nil {
		if utxo.UtxoType == UtxoCreateContract {
			_, utxo, nextUtxoTx = nextUtxoTx.Iterator()
			continue
		}

		sum = sum.Add(utxo.Value)
		utxos = append(utxos, utxo)
		if sum.Cmp(amount) >= 0 {
			return utxos, true
		}
		_, utxo, nextUtxoTx = nextUtxoTx.Iterator()
	}

	return nil, false
}
