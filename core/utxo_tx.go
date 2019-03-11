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
//type UTXOTx hamt.Map

type UTXOTx struct {
	Indices map[string]*UTXO
}

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
	return UTXOTx{Indices: map[string]*UTXO{}}
}

// Construct with UTXO data
func NewUTXOTxWithData(utxo *UTXO) UTXOTx {
	//key := StringEntry(string(utxo.Txid) + "_" + strconv.Itoa(utxo.TxIndex))
	key := string(utxo.Txid) + "_" + strconv.Itoa(utxo.TxIndex)
	return UTXOTx{Indices: map[string]*UTXO{key: utxo}}
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

	//_, utxo, nextUtxoTx := utxoTx.Iterator()
	for _, utxo := range utxoTx.Indices {
		utxoList.Utxos = append(utxoList.Utxos, utxo.ToProto().(*corepb.Utxo))
		//_, utxo, nextUtxoTx = nextUtxoTx.Iterator()
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
	//key := StringEntry(string(txid) + "_" + strconv.Itoa(vout))
	//value := hamt.Map(utxoTx).Find(&key)
	key := string(txid) + "_" + strconv.Itoa(vout)
	utxo, ok := utxoTx.Indices[key]
	//utxo, ok := value.(*UTXO)
	if !ok {
		return nil
	}

	return utxo
}

// Add new utxo to map
func (utxoTx UTXOTx) PutUtxo(utxo *UTXO) UTXOTx {
	key := string(utxo.Txid) + "_" + strconv.Itoa(utxo.TxIndex)
	utxoTx.Indices[key] = utxo
	return utxoTx
}

// Delete invalid element in map
func (utxoTx UTXOTx) RemoveUtxo(txid []byte, vout int) UTXOTx {
	key := string(txid) + "_" + strconv.Itoa(vout)
	delete(utxoTx.Indices, key)
	return utxoTx
}

//func (utxoTx UTXOTx) Iterator() (string, *UTXO, UTXOTx) {
//	key, value, nextMap := utxoTx.Indices.
//	if key == nil {
//		return "", nil, NewUTXOTx()
//	}
//
//	keyStr, ok := key.(*StringEntry)
//	if !ok {
//		logger.Panic("UtxoTx: invalid key type")
//	}
//
//	utxo, ok := value.(*UTXO)
//	if !ok {
//		logger.Panic("UtxoTx: invalid value type")
//	}
//
//	return string(*keyStr), utxo, UTXOTx(nextMap)
//}

func (utxoTx UTXOTx) Size() int {
	return len(utxoTx.Indices)
}

func (utxoTx UTXOTx) GetAllUtxos() []*UTXO {
	var utxos []*UTXO
	//_, utxo, nextUtxoTx := utxoTx.Iterator()
	for _, utxo := range utxoTx.Indices {
		utxos = append(utxos, utxo)
		//_, utxo, nextUtxoTx = nextUtxoTx.Iterator()
	}

	return utxos
}

func (utxoTx UTXOTx) PrepareUtxos(amount *common.Amount) ([]*UTXO, bool) {
	sum := common.NewAmount(0)

	if utxoTx.Size() < 1 {
		return nil, false
	}

	var utxos []*UTXO
	//_, utxo, nextUtxoTx := utxoTx.Iterator()
	for _, utxo := range utxoTx.Indices {
		if utxo.UtxoType == UtxoCreateContract {
			//_, utxo, nextUtxoTx = nextUtxoTx.Iterator()
			continue
		}

		sum = sum.Add(utxo.Value)
		utxos = append(utxos, utxo)
		if sum.Cmp(amount) >= 0 {
			return utxos, true
		}
		//_, utxo, nextUtxoTx = nextUtxoTx.Iterator()
	}

	return nil, false
}

func (utxoTx UTXOTx) DeepCopy() UTXOTx {

	newUtxoTx := NewUTXOTx()

	for key, utxo := range utxoTx.Indices {
		newUtxoTx.Indices[key] = utxo
		//_, utxo, nextUtxoTx = nextUtxoTx.Iterator()
	}
	return newUtxoTx
}
