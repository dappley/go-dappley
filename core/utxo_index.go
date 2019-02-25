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
	"errors"
	"sync"

	logger "github.com/sirupsen/logrus"

	"github.com/dappley/go-dappley/common"
	"github.com/dappley/go-dappley/storage"
)

const utxoMapKey = "utxo"
const contractUtxoKey = "ContractUtxos"

var (
	ErrUTXONotFound   = errors.New("utxo not found when trying to remove from cache")
	ErrTXInputInvalid = errors.New("txInput refers to non-existing transaction")
)

// UTXOIndex holds all unspent TXOutputs indexed by public key hash.
type UTXOIndex struct {
	index map[string][]*UTXO
	mutex *sync.RWMutex
}

// NewUTXOIndex initializes an UTXOIndex instance
func NewUTXOIndex() *UTXOIndex {
	return &UTXOIndex{make(map[string][]*UTXO), &sync.RWMutex{}}
}

func deserializeUTXOIndex(d []byte) *UTXOIndex {
	utxos := NewUTXOIndex()
	utxos.mutex.Lock()
	defer utxos.mutex.Unlock()
	decoder := gob.NewDecoder(bytes.NewReader(d))
	err := decoder.Decode(&utxos.index)
	if err != nil {
		logger.WithError(err).Panic("UTXOIndex: failed to deserialize UTXOs.")
	}
	return utxos
}

func (utxos *UTXOIndex) serialize() []byte {
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
func LoadUTXOIndex(db storage.Storage) *UTXOIndex {
	utxoBytes, err := db.Get([]byte(utxoMapKey))

	if err != nil && err.Error() == storage.ErrKeyInvalid.Error() || len(utxoBytes) == 0 {
		logger.Debug("UTXOIndex: does not exists in database. Creating a new one.")
		return NewUTXOIndex()
	}
	return deserializeUTXOIndex(utxoBytes)
}

// Save stores the index to db
func (utxos *UTXOIndex) Save(db storage.Storage) error {
	return db.Put([]byte(utxoMapKey), utxos.serialize())
}

// FindUTXO returns the UTXO instance of the corresponding TXOutput in the transaction (identified by txid and vout)
// if the TXOutput is unspent. Otherwise, it returns nil.
//func (utxos *UTXOIndex) FindUTXO(txid []byte, vout int) *UTXO {
//	utxos.mutex.RLock()
//	defer utxos.mutex.RUnlock()
//	for _, utxoArray := range utxos.index {
//		for _, u := range utxoArray {
//			if bytes.Compare(u.Txid, txid) == 0 && u.TxIndex == vout {
//				return u
//			}
//		}
//	}
//	return nil
//}

// GetAllUTXOsByPubKeyHash returns all current UTXOs identified by pubkey.
func (utxos *UTXOIndex) GetAllUTXOsByPubKeyHash(pubkeyHash []byte) []*UTXO {
	utxos.mutex.RLock()
	defer utxos.mutex.RUnlock()
	return utxos.index[string(pubkeyHash)]

}

// GetUTXOsByAmount returns a number of UTXOs that has a sum more than or equal to the amount
func (utxos *UTXOIndex) GetUTXOsByAmount(pubkeyHash []byte, amount *common.Amount) ([]*UTXO, error) {

	allUtxos := utxos.GetAllUTXOsByPubKeyHash(pubkeyHash)

	retUtxos, ok := PrepareUTXOs(allUtxos, amount)
	if !ok {
		return nil, ErrInsufficientFund
	}

	return retUtxos, nil
}

// FindUTXOByVin returns the UTXO instance identified by pubkeyHash, txid and vout
func (utxos *UTXOIndex) FindUTXOByVin(pubkeyHash []byte, txid []byte, vout int) *UTXO {
	utxosOfKey := utxos.GetAllUTXOsByPubKeyHash(pubkeyHash)
	for _, utxo := range utxosOfKey {
		if bytes.Compare(utxo.Txid, txid) == 0 && utxo.TxIndex == vout {
			return utxo
		}
	}
	return nil
}

func (utxos *UTXOIndex) UpdateUtxo(tx *Transaction) bool {
	if !tx.IsCoinbase() && !tx.IsRewardTx() {
		for _, txin := range tx.Vin {
			pkh, err := NewUserPubKeyHash(txin.PubKey)
			if err != nil {
				return false
			}
			err = utxos.removeUTXO(pkh, txin.Txid, txin.Vout)
			if err != nil {
				return false
			}
		}
	}
	for i, txout := range tx.Vout {
		utxos.addUTXO(txout, tx.ID, i)
	}
	return true
}

// Update removes the UTXOs spent in the transactions in newBlk from the index and adds UTXOs generated in the
// transactions to the index. The index will be saved to db as a result. If saving failed, index won't be updated.
func (utxos *UTXOIndex) UpdateUtxoState(txs []*Transaction) {
	// Create a copy of the index so operations below are only temporal
	for _, tx := range txs {
		utxos.UpdateUtxo(tx)
	}
}

// undoTxsInBlock compute the (previous) UTXOIndex resulted from undoing the transactions in given blk.
// Note that the operation does not save the index to db.
func (utxos *UTXOIndex) undoTxsInBlock(blk *Block, bc *Blockchain, db storage.Storage) error {

	for i := len(blk.GetTransactions()) - 1; i >= 0; i-- {
		tx := blk.GetTransactions()[i]
		err := utxos.excludeVoutsInTx(tx, db)
		if err != nil {
			return err
		}
		if tx.IsCoinbase() || tx.IsRewardTx() {
			continue
		}
		err = utxos.unspendVinsInTx(tx, bc)
		if err != nil {
			return err
		}
	}
	return nil
}

// excludeVoutsInTx removes the UTXOs generated in a transaction from the UTXOIndex.
func (utxos *UTXOIndex) excludeVoutsInTx(tx *Transaction, db storage.Storage) error {
	for i, vout := range tx.Vout {
		err := utxos.removeUTXO(vout.PubKeyHash, tx.ID, i)
		if err != nil {
			return err
		}
	}
	return nil
}

// unspendVinsInTx adds UTXOs back to the UTXOIndex as a result of undoing the spending of the UTXOs in a transaction.
func (utxos *UTXOIndex) unspendVinsInTx(tx *Transaction, bc *Blockchain) error {
	for _, vin := range tx.Vin {
		vout, voutIndex, err := getTXOutputSpent(vin, bc)
		if err != nil {
			return err
		}
		utxos.addUTXO(vout, vin.Txid, voutIndex)
	}
	return nil
}

// addUTXO adds an unspent TXOutput to index
func (utxos *UTXOIndex) addUTXO(txout TXOutput, txid []byte, vout int) {
	u := newUTXO(txout, txid, vout)
	utxos.mutex.Lock()
	defer utxos.mutex.Unlock()
	//if it is a smart contract deployment utxo add it to contract utxos
	if isContract, _ := txout.PubKeyHash.IsContract(); isContract &&
		len(utxos.index[string(u.PubKeyHash)]) == 0 {
		utxos.index[contractUtxoKey] = append(utxos.index[contractUtxoKey], u)
	}
	utxos.index[string(u.PubKeyHash)] = append(utxos.index[string(u.PubKeyHash)], u)
}

//func (utxos *UTXOIndex) GetContractUtxos() []*UTXO {
//	return utxos.index[contractUtxoKey]
//}

// removeUTXO finds and removes a UTXO from UTXOIndex
func (utxos *UTXOIndex) removeUTXO(pkh PubKeyHash, txid []byte, vout int) error {
	originalUtxos := utxos.GetAllUTXOsByPubKeyHash(pkh)
	utxos.mutex.Lock()
	defer utxos.mutex.Unlock()

	for i, utxo := range originalUtxos {
		if bytes.Compare(utxo.Txid, txid) == 0 && utxo.TxIndex == vout {
			utxos.index[string(pkh)] = append(originalUtxos[:i], originalUtxos[i+1:]...)
			return nil
		}
	}

	return ErrUTXONotFound
}

//creates a deepcopy of the receiver object
func (utxos *UTXOIndex) DeepCopy() *UTXOIndex {
	utxos.mutex.RLock()
	defer utxos.mutex.RUnlock()

	utxocopy := NewUTXOIndex()
	for pkh := range utxos.index {
		utxocopy.index[pkh] = make([]*UTXO, 0)
		for _, utxo := range utxos.index[pkh] {
			utxocopy.index[pkh] = append(utxocopy.index[pkh], utxo)
		}
	}
	return utxocopy
}

// GetUTXOIndexAtBlockHash returns the previous snapshot of UTXOIndex when the block of given hash was the tail block.
func GetUTXOIndexAtBlockHash(db storage.Storage, bc *Blockchain, hash Hash) (*UTXOIndex, error) {
	index := LoadUTXOIndex(db)
	deepCopy := index.DeepCopy()
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

		err = deepCopy.undoTxsInBlock(block, bc, db)
		if err != nil {
			logger.WithError(err).WithFields(logger.Fields{
				"hash": block.GetHash(),
			}).Warn("UTXOIndex: failed to calculate previous state of UTXO index for the block")
			return NewUTXOIndex(), err
		}
	}

	return deepCopy, nil
}
