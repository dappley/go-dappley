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

package main

import (
	"bytes"
	"encoding/gob"
	"encoding/hex"
	"flag"
	"sync"

	"github.com/dappley/go-dappley/core/utxo"
	errval "github.com/dappley/go-dappley/errors"
	"github.com/dappley/go-dappley/logic/lutxo"
	"github.com/dappley/go-dappley/storage"
	logger "github.com/sirupsen/logrus"
)

const utxoMapKeyOld = "utxo"
const contractUtxoKeyOld = "ContractUtxos"
const dbFilePath = "../../bin/node1.db"

// UTXOIndexOld holds all unspent TXOutputs indexed by public key hash.
type UTXOIndexOld struct {
	index map[string][]*utxo.UTXO
	mutex *sync.RWMutex
}

// Convert old utxo_index map data to new utxo_tx data
// user parameter '-f dbPath' to reset db config
func main() {
	logger.Info("Utxo_data_transfer: start...")
	db := getDb()
	defer db.Close()

	convert(db)
}

func convert(db storage.Storage) {
	// read old data
	logger.Info("Utxo_data_transfer: read old data start...")
	utxoIndexOld := readOld(db)
	if utxoIndexOld == nil {
		logger.Info("Utxo_data_transfer: no old data, exit...")
		return
	}
	logger.WithFields(logger.Fields{
		"oldSize": len(utxoIndexOld.index),
	}).Info("Utxo_data_transfer: read old data finished...")

	// data convert
	utxoIndexNew := convertData(db, utxoIndexOld)
	if utxoIndexNew == nil {
		logger.Info("Utxo_data_transfer: convert error, exit...")
		return
	}
	// save new data
	logger.Info("Utxo_data_transfer: save new data start...")
	saveNewData(utxoIndexNew)
	logger.Info("Utxo_data_transfer: save new data finish...")
}

// Returns storage connection
func getDb() storage.Storage {
	var filePath string
	flag.StringVar(&filePath, "f", dbFilePath, "Configuration DB Path. Default to bin/node1.db")
	flag.Parse()
	logger.Info("dbFilePath:" + filePath)
	db := storage.OpenDatabase(filePath)
	return db
}

// Read old utxo_index data
func readOld(db storage.Storage) *UTXOIndexOld {
	return LoadUTXOIndexOld(db)
}

// LoadUTXOIndexOld returns the UTXOIndex fetched from db.
func LoadUTXOIndexOld(db storage.Storage) *UTXOIndexOld {
	utxoBytes, err := db.Get([]byte(utxoMapKeyOld))
	if err != nil && err == errval.InvalidKey || len(utxoBytes) == 0 {
		logger.Error("Utxo_data_transfer: utxo does not exists in database.")
		return nil
	}
	return deserializeUTXOIndexOld(utxoBytes)
}

// NewUTXOIndexOld initializes an UTXOIndex instance
func NewUTXOIndexOld() *UTXOIndexOld {
	return &UTXOIndexOld{make(map[string][]*utxo.UTXO), &sync.RWMutex{}}
}

func deserializeUTXOIndexOld(d []byte) *UTXOIndexOld {
	utxos := NewUTXOIndexOld()
	utxos.mutex.Lock()
	defer utxos.mutex.Unlock()
	decoder := gob.NewDecoder(bytes.NewReader(d))
	err := decoder.Decode(&utxos.index)
	if err != nil {
		logger.WithError(err).Panic("UTXOIndex: failed to deserialize old UTXOs.")
	}
	return utxos
}

func (utxos *UTXOIndexOld) serializeUTXOIndexOld() []byte {
	var encoded bytes.Buffer
	utxos.mutex.Lock()
	defer utxos.mutex.Unlock()
	enc := gob.NewEncoder(&encoded)
	err := enc.Encode(utxos.index)
	if err != nil {
		logger.Panic(err)
	}
	return encoded.Bytes()
}

// Convert old utxoIndex data to new utxoIndex data
func convertData(db storage.Storage, utxoIndexOld *UTXOIndexOld) *lutxo.UTXOIndex {
	utxoIndexNew := lutxo.NewUTXOIndex(utxo.NewUTXOCache(db))
	for address, utxoArray := range utxoIndexOld.index {
		if address == contractUtxoKeyOld {
			continue
		}
		addUtxoArrayToIndex(utxoArray, utxoIndexNew)
	}
	return utxoIndexNew
}

// Add each utxo in utxoArray into new utxoIndex
func addUtxoArrayToIndex(utxoArray []*utxo.UTXO, utxoIndexNew *lutxo.UTXOIndex) {
	if utxoArray == nil {
		return
	}
	for _, utxo := range utxoArray {
		utxoIndexNew.AddUTXO(utxo.TXOutput, utxo.Txid, utxo.TxIndex)
		logger.WithFields(logger.Fields{
			"address": utxo.GetAddress(),
			"Txid":    hex.EncodeToString(utxo.Txid),
			"TxIndex": utxo.TxIndex,
		}).Info("Utxo_data_transfer: add utxo")
	}
}

// Save all new data to storage
func saveNewData(utxoIndexNew *lutxo.UTXOIndex) {
	utxoIndexNew.Save()
	logger.Info("Utxo_data_transfer: new data has been saved into storage...")
}
