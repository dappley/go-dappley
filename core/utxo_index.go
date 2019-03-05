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
	"errors"
	"sync"

	"github.com/dappley/go-dappley/common"
	"github.com/dappley/go-dappley/storage"
	logger "github.com/sirupsen/logrus"
)

var contractUtxoKey = []byte("contractUtxoKey")

var (
	ErrUTXONotFound   = errors.New("utxo not found when trying to remove from cache")
	ErrTXInputInvalid = errors.New("txInput refers to non-existing transaction")
)

// UTXOIndex holds all unspent TXOutputs indexed by public key hash.
type UTXOIndex struct {
	index map[string]*UTXOTx
	cache *UTXOCache
	mutex *sync.RWMutex
}

// NewUTXOIndex initializes an UTXOIndex instance
func NewUTXOIndex(cache *UTXOCache) *UTXOIndex {
	return &UTXOIndex{
		index: make(map[string]*UTXOTx),
		cache: cache,
		mutex: &sync.RWMutex{},
	}
}

func (utxos *UTXOIndex) Save() error {
	for key, utxoTx := range utxos.index {
		err := utxos.cache.Put(PubKeyHash(key), utxoTx)
		if err != nil {
			return err
		}
	}
	return nil
}

// GetAllUTXOsByPubKeyHash returns all current UTXOs identified by pubkey.
func (utxos *UTXOIndex) GetAllUTXOsByPubKeyHash(pubkeyHash []byte) *UTXOTx {
	key := string(pubkeyHash)
	utxos.mutex.RLock()
	utxoTx, ok := utxos.index[key]
	utxos.mutex.RUnlock()

	if ok {
		return utxoTx
	}

	utxoTx = utxos.cache.Get(pubkeyHash)
	utxos.mutex.Lock()
	utxos.index[key] = utxoTx
	utxos.mutex.Unlock()
	return utxoTx
}

//SplitContractUtxo
func (utxos *UTXOIndex) SplitContractUtxo(pubkeyHash []byte) (*UTXO, []*UTXO) {
	if ok, _ := PubKeyHash(pubkeyHash).IsContract(); !ok {
		return nil, nil
	}

	utxoTx := utxos.GetAllUTXOsByPubKeyHash(pubkeyHash)

	var invokeContractUtxos []*UTXO
	var createContractUtxo *UTXO

	_, utxo, nextUtxoTx := utxoTx.Iterator()
	for utxo != nil {
		if utxo.UtxoType == UtxoCreateContract {
			createContractUtxo = utxo
		} else {
			invokeContractUtxos = append(invokeContractUtxos, utxo)
		}
		_, utxo, nextUtxoTx = nextUtxoTx.Iterator()
	}
	return createContractUtxo, invokeContractUtxos
}

// GetUTXOsByAmount returns a number of UTXOs that has a sum more than or equal to the amount
func (utxos *UTXOIndex) GetUTXOsByAmount(pubkeyHash []byte, amount *common.Amount) ([]*UTXO, error) {
	allUtxos := utxos.GetAllUTXOsByPubKeyHash(pubkeyHash)
	retUtxos, ok := allUtxos.PrepareUtxos(amount)
	if !ok {
		return nil, ErrInsufficientFund
	}

	return retUtxos, nil
}

// FindUTXOByVin returns the UTXO instance identified by pubkeyHash, txid and vout
func (utxos *UTXOIndex) FindUTXOByVin(pubkeyHash []byte, txid []byte, vout int) *UTXO {
	utxosOfKey := utxos.GetAllUTXOsByPubKeyHash(pubkeyHash)
	return utxosOfKey.GetUtxo(txid, vout)
}

func (utxos *UTXOIndex) UpdateUtxo(tx *Transaction) bool {
	if !tx.IsCoinbase() && !tx.IsRewardTx() {
		for _, txin := range tx.Vin {
			//TODO spent contract utxo
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
		utxos.AddUTXO(txout, tx.ID, i)
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
		utxos.AddUTXO(vout, vin.Txid, voutIndex)
	}
	return nil
}

// AddUTXO adds an unspent TXOutput to index
func (utxos *UTXOIndex) AddUTXO(txout TXOutput, txid []byte, vout int) {
	originalUtxos := utxos.GetAllUTXOsByPubKeyHash(txout.PubKeyHash)

	var utxo *UTXO
	//if it is a smart contract deployment utxo add it to contract utxos
	if isContract, _ := txout.PubKeyHash.IsContract(); isContract {
		if originalUtxos.Size() == 0 {
			utxo = newUTXO(txout, txid, vout, UtxoCreateContract)
			contractUtxos := utxos.GetAllUTXOsByPubKeyHash(contractUtxoKey)
			newContractUtxos := contractUtxos.PutUtxo(utxo)
			utxos.mutex.Lock()
			utxos.index[string(contractUtxoKey)] = &newContractUtxos
			utxos.mutex.Unlock()
		} else {
			utxo = newUTXO(txout, txid, vout, UtxoInvokeContract)
		}
	} else {
		utxo = newUTXO(txout, txid, vout, UtxoNormal)
	}

	newUtxos := originalUtxos.PutUtxo(utxo)
	utxos.mutex.Lock()
	utxos.index[string(txout.PubKeyHash)] = &newUtxos
	utxos.mutex.Unlock()
}

func (utxos *UTXOIndex) GetContractUtxos() []*UTXO {
	utxoTx := utxos.GetAllUTXOsByPubKeyHash(contractUtxoKey)

	var contractUtxos []*UTXO
	_, utxo, nextUtxoTx := utxoTx.Iterator()
	for utxo != nil {
		contractUtxos = append(contractUtxos, utxo)
		_, utxo, nextUtxoTx = nextUtxoTx.Iterator()
	}
	return contractUtxos
}

// removeUTXO finds and removes a UTXO from UTXOIndex
func (utxos *UTXOIndex) removeUTXO(pkh PubKeyHash, txid []byte, vout int) error {
	originalUtxos := utxos.GetAllUTXOsByPubKeyHash(pkh)
	utxos.mutex.Lock()
	defer utxos.mutex.Unlock()

	utxo := originalUtxos.GetUtxo(txid, vout)
	if utxo == nil {
		return ErrUTXONotFound
	}

	newUtxos := originalUtxos.RemoveUtxo(txid, vout)
	utxos.index[string(pkh)] = &newUtxos

	if utxo.UtxoType != UtxoCreateContract {
		return nil
	}
	// remove contract utxos
	isContract, _ := pkh.IsContract()
	if isContract {
		contractUtxos := utxos.GetAllUTXOsByPubKeyHash(contractUtxoKey)
		if contractUtxos == nil {
			return ErrUTXONotFound
		}
		newContractUtxos := contractUtxos.RemoveUtxo(txid, vout)
		utxos.index[string(contractUtxoKey)] = &newContractUtxos
	}
	return nil
}

//creates a deepcopy of the receiver object
func (utxos *UTXOIndex) DeepCopy() *UTXOIndex {
	utxos.mutex.RLock()
	defer utxos.mutex.RUnlock()

	utxocopy := NewUTXOIndex(utxos.cache)
	for pkh, utxoTx := range utxos.index {
		utxocopy.index[pkh] = utxoTx
	}
	return utxocopy
}

// GetUTXOIndexAtBlockHash returns the previous snapshot of UTXOIndex when the block of given hash was the tail block.
func GetUTXOIndexAtBlockHash(db storage.Storage, bc *Blockchain, hash Hash) (*UTXOIndex, error) {
	index := NewUTXOIndex(bc.GetUtxoCache())
	bci := bc.Iterator()

	// Start from the tail of blockchain, compute the previous UTXOIndex by undoing transactions
	// in the block, until the block hash matches.
	for {
		block, err := bci.Next()

		if bytes.Compare(block.GetHash(), hash) == 0 {
			break
		}

		if err != nil {
			return nil, err
		}

		if len(block.GetPrevHash()) == 0 {
			return nil, ErrBlockDoesNotExist
		}

		err = index.undoTxsInBlock(block, bc, db)
		if err != nil {
			logger.WithError(err).WithFields(logger.Fields{
				"hash": block.GetHash(),
			}).Warn("UTXOIndex: failed to calculate previous state of UTXO index for the block")
			return nil, err
		}
	}

	return index, nil
}
