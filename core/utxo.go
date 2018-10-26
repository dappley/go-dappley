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
	"encoding/gob"
	"errors"
	"fmt"
	"sync"

	"github.com/jinzhu/copier"
	logger "github.com/sirupsen/logrus"

	"github.com/dappley/go-dappley/common"
	"github.com/dappley/go-dappley/storage"
)

const utxoMapKey = "utxo"

// UTXOIndex holds all unspent TXOutputs indexed by public key hash.
type UTXOIndex struct {
	index map[string][]*UTXO
	mutex *sync.RWMutex
}

// UTXO contains the meta info of an unspent TXOutput.
type UTXO struct {
	Value      *common.Amount
	PubKeyHash []byte
	Txid       []byte
	TxIndex    int
}

// NewUTXOIndex initializes an UTXOIndex instance
func NewUTXOIndex() UTXOIndex {
	return UTXOIndex{make(map[string][]*UTXO), &sync.RWMutex{}}
}

func deserializeUTXOIndex(d []byte) UTXOIndex {
	utxos := NewUTXOIndex()
	utxos.mutex.Lock()
	defer utxos.mutex.Unlock()
	decoder := gob.NewDecoder(bytes.NewReader(d))
	err := decoder.Decode(&utxos.index)
	if err != nil {
		logger.Panicf("failed to deserialize UTXOs: %v", err)
	}
	return utxos
}

func (utxos UTXOIndex) serialize() []byte {
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

// LoadUTXOIndex returns the UTXOIndex fetched from db.
func LoadUTXOIndex(db storage.Storage) UTXOIndex {
	utxoBytes, err := db.Get([]byte(utxoMapKey))

	if err != nil && err.Error() == storage.ErrKeyInvalid.Error() || len(utxoBytes) == 0 {
		return NewUTXOIndex()
	}
	return deserializeUTXOIndex(utxoBytes)
}

// Save stores the index to db
func (utxos UTXOIndex) Save(mapkey string, db storage.Storage) error {
	return db.Put([]byte(mapkey), utxos.serialize())
}

// FindUTXO returns the UTXO instance of the corresponding TXOutput in the transaction (identified by txid and vout)
// if the TXOutput is unspent. Otherwise, it returns nil.
func (utxos UTXOIndex) FindUTXO(txid []byte, vout int) *UTXO {
	utxos.mutex.RLock()
	defer utxos.mutex.RUnlock()
	for _, utxoArray := range utxos.index {
		for _, u := range utxoArray {
			if bytes.Compare(u.Txid, txid) == 0 && u.TxIndex == vout {
				return u
			}
		}
	}
	return nil
}

// GetAllUTXOsByPubKeyHash returns all current UTXOs identified by pubkey.
func (utxos UTXOIndex) GetAllUTXOsByPubKeyHash(pubkey []byte) []*UTXO {
	utxos.mutex.RLock()
	defer utxos.mutex.RUnlock()
	return utxos.index[string(pubkey)]

}

// GetUTXOsByAmount returns a number of UTXOs that has a sum more than or equal to the amount
func (utxos UTXOIndex) GetUTXOsByAmount(pubkey []byte, amount *common.Amount) ([]*UTXO, error) {

	allUtxos := utxos.GetAllUTXOsByPubKeyHash(pubkey)

	var retUtxos []*UTXO
	sum := common.NewAmount(0)
	for _, u := range allUtxos {
		sum = sum.Add(u.Value)
		retUtxos = append(retUtxos, u)
		if sum.Cmp(amount) >= 0 {
			break
		}
	}

	if sum.Cmp(amount) < 0 {
		return nil, ErrInsufficientFund
	}

	return retUtxos, nil
}

// FindUTXOByVin returns the UTXO instance identified by pubkeyHash, txid and vout
func (utxos UTXOIndex) FindUTXOByVin(pubkeyHash []byte, txid []byte, vout int) *UTXO {
	utxosOfKey := utxos.GetAllUTXOsByPubKeyHash(pubkeyHash)
	for _, utxo := range utxosOfKey {
		if bytes.Compare(utxo.Txid, txid) == 0 && utxo.TxIndex == vout {
			return utxo
		}
	}
	return nil
}

// Update removes the UTXOs spent in the transactions in newBlk from the index and adds UTXOs generated in the
// transactions to the index. The index will be saved to db as a result. If saving failed, index won't be updated.
func (utxos *UTXOIndex) BuildForkUtxoIndex(newBlk *Block, db storage.Storage) error {
	// Create a copy of the index so operations below are only temporal
	tempIndex := utxos.deepCopy()

	for _, tx := range newBlk.GetTransactions() {
		if !tx.IsCoinbase() {
			for _, txin := range tx.Vin {
				err := tempIndex.removeUTXO(txin.Txid, txin.Vout)
				if err != nil {
					logger.Warn(err)
				}
			}
		}
		for i, txout := range tx.Vout {
			tempIndex.addUTXO(txout, tx.ID, i)
		}
	}

	// Save to database
	err := tempIndex.Save(utxoMapKey, db)

	// Assign the temporal copy to the original receiver index ONLY after it is successfully saved to db
	if err == nil {
		*utxos = tempIndex
	} else {
		logger.Error(fmt.Errorf("failed to update utxo index: %v", err))
	}

	return err
}

// newUTXO returns an UTXO instance constructed from a TXOutput.
func newUTXO(txout TXOutput, txid []byte, vout int) *UTXO {
	return &UTXO{txout.Value, txout.PubKeyHash, txid, vout}
}

// undoTxsInBlock compute the (previous) UTXOIndex resulted from undoing the transactions in given blk.
// Note that the operation does not save the index to db.
func (utxos UTXOIndex) undoTxsInBlock(blk *Block, bc *Blockchain, db storage.Storage) {

	for _, tx := range blk.GetTransactions() {
		err := utxos.excludeVoutsInTx(tx, db)
		if err != nil {
			logger.Panic(err)
		}
		if tx.IsCoinbase() {
			continue
		}
		err = utxos.unspendVinsInTx(tx, bc)
		if err != nil {
			logger.Panic(err)
		}
	}
}

// excludeVoutsInTx undoes the spending of UTXO in a transaction.
func (utxos UTXOIndex) excludeVoutsInTx(tx *Transaction, db storage.Storage) error {
	for i := range tx.Vout {
		err := utxos.removeUTXO(tx.ID, i)
		if err != nil {
			return err
		}
	}
	return nil
}

// unspendVinsInTx includes UTXO the UTXOIndex as a result of undoing the spending of UTXO in a transaction.
func (utxos UTXOIndex) unspendVinsInTx(tx *Transaction, bc *Blockchain) error {
	for _, vin := range tx.Vin {
		vout, voutIndex, err := getTXOutputSpent(vin, bc)
		if err != nil {
			return err
		}
		utxos.addUTXO(vout, tx.ID, voutIndex)
	}
	return nil
}

// addUTXO adds an unspent TXOutput to index
func (utxos UTXOIndex) addUTXO(txout TXOutput, txid []byte, vout int) {
	u := newUTXO(txout, txid, vout)
	utxos.mutex.Lock()
	defer utxos.mutex.Unlock()
	utxos.index[string(u.PubKeyHash)] = append(utxos.index[string(u.PubKeyHash)], u)

}

// removeUTXO finds and removes a UTXO from UTXOIndex
func (utxos UTXOIndex) removeUTXO(txid []byte, vout int) error {
	utxos.mutex.Lock()
	defer utxos.mutex.Unlock()

	for _, utxoArray := range utxos.index {
		for i, u := range utxoArray {
			if bytes.Compare(u.Txid, txid) == 0 && u.TxIndex == vout {
				userUTXOs := utxos.index[string(u.PubKeyHash)]
				utxos.index[string(u.PubKeyHash)] = append(userUTXOs[:i], userUTXOs[i+1:]...)
				return nil
			}
		}
	}
	return errors.New("UTXO: utxo not found when trying to remove from cache")
}

func getTXOutputSpent(in TXInput, bc *Blockchain) (TXOutput, int, error) {
	tx, err := bc.FindTransaction(in.Txid)
	if err != nil {
		return TXOutput{}, 0, errors.New("txInput refers to non-existing transaction")
	}
	return tx.Vout[in.Vout], in.Vout, nil
}

func (utxos UTXOIndex) deepCopy() UTXOIndex {
	utxos.mutex.RLock()
	defer utxos.mutex.RUnlock()
	utxocopy := NewUTXOIndex()
	copier.Copy(&utxocopy, &utxos)
	if len(utxocopy.index) == 0 {
		utxocopy = NewUTXOIndex()
	}
	return utxocopy
}

// GetUTXOIndexAtBlockHash returns the previous snapshot of UTXOIndex when the block of given hash was the tail block.
func GetUTXOIndexAtBlockHash(db storage.Storage, bc *Blockchain, hash Hash) (UTXOIndex, error) {
	index := LoadUTXOIndex(db)
	deepCopy := index.deepCopy()
	bci := bc.Iterator()

	// Start from the tail of blockchain, compute the previous UTXOIndex by undoing transactions
	// in the block, until the block hash matches.
	for {
		block, err := bci.Next()

		if bytes.Compare(block.GetHash(), hash) == 0 {
			break
		}

		if err != nil {
			return NewUTXOIndex(), err
		}

		if len(block.GetPrevHash()) == 0 {
			return NewUTXOIndex(), ErrBlockDoesNotExist
		}

		deepCopy.undoTxsInBlock(block, bc, db)
	}

	return deepCopy, nil
}
