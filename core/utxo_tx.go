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
	"bytes"
	"encoding/gob"
	"github.com/dappley/go-dappley/storage"
	"strconv"
	"sync"

	logger "github.com/sirupsen/logrus"
)

// UTXOTx holds txid_vout and UTXO pairs
type UTXOTx struct {
	TxIndex map[string]*UTXO
	mutex   *sync.RWMutex
}

func NewUTXOTx() *UTXOTx {
	return &UTXOTx{make(map[string]*UTXO), &sync.RWMutex{}}
}

// Construct with UTXO data
func NewUTXOTxWithData(utxo UTXO) *UTXOTx {
	utxoTx := NewUTXOTx()
	key := string(utxo.Txid) + "_" + strconv.Itoa(utxo.TxIndex)
	utxoTx.TxIndex[key] = &utxo
	return utxoTx
}

func LoadFromMap(m map[string]*UTXO) *UTXOTx {
	if m == nil {
		return nil
	}
	utxoTx := NewUTXOTx()
	utxoTx.TxIndex = m
	return utxoTx
}

func DeserializeUTXOTx(d []byte) *UTXOTx {
	reader := bytes.NewReader(d)
	utxoTx := NewUTXOTx()
	utxoTx.mutex.Lock()
	defer utxoTx.mutex.Unlock()
	decoder := gob.NewDecoder(reader)
	err := decoder.Decode(&utxoTx.TxIndex)
	if err != nil {
		logger.WithError(err).Panic("UTXOTx: failed to deserialize UTXO.")
	}
	return utxoTx
}

func (utxoTx *UTXOTx) serialize() []byte {
	var encoded bytes.Buffer
	utxoTx.mutex.Lock()
	defer utxoTx.mutex.Unlock()
	enc := gob.NewEncoder(&encoded)
	err := enc.Encode(utxoTx.TxIndex)
	if err != nil {
		logger.Panic(err)
	}
	return encoded.Bytes()
}

// Returns utxo info by transaction id and vout index
func (utxoTx *UTXOTx) GetUtxo(txid []byte, vout int) *UTXO {
	key := string(txid) + "_" + strconv.Itoa(vout)
	return utxoTx.TxIndex[key]
}

// Add new utxo to map
func (utxoTx *UTXOTx) PutUtxo(utxo *UTXO) (ok bool) {
	if utxo == nil {
		return false
	}
	key := string(utxo.Txid) + "_" + strconv.Itoa(utxo.TxIndex)
	utxoTx.TxIndex[key] = utxo
	return true
}

// Delete invalid element in map
func (utxoTx *UTXOTx) RemoveUtxo(txid []byte, vout int) {
	key := string(txid) + "_" + strconv.Itoa(vout)
	delete(utxoTx.TxIndex, key)
}

// Save the UTXOTx to db
func (utxoTx UTXOTx) Save(pubKeyHash PubKeyHash, db storage.Storage) error {
	return db.Put(pubKeyHash, utxoTx.serialize())
}

// Copy data
func (utxoTx UTXOTx) DeepCopy() *UTXOTx {
	copy := NewUTXOTx()
	txIndex := utxoTx.TxIndex
	copy.TxIndex = txIndex
	return copy
}
