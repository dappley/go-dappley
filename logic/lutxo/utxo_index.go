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

package lutxo

import (
	"sort"
	"sync"

	"github.com/dappley/go-dappley/common"
	"github.com/dappley/go-dappley/core/account"
	"github.com/dappley/go-dappley/core/block"
	"github.com/dappley/go-dappley/core/transaction"
	"github.com/dappley/go-dappley/core/transactionbase"
	"github.com/dappley/go-dappley/core/utxo"
	errorValues "github.com/dappley/go-dappley/errors"
	"github.com/dappley/go-dappley/storage"
	logger "github.com/sirupsen/logrus"
)

var contractUtxoKey = []byte("contractUtxoKey")

// UTXOIndex holds all unspent TXOutputs indexed by public key hash.
type UTXOIndex struct {
	indexRemove map[string]*utxo.UTXOTx
	indexAdd    map[string]*utxo.UTXOTx
	cache       *utxo.UTXOCache
	mutex       *sync.RWMutex
}

// NewUTXOIndex initializes an UTXOIndex instance
func NewUTXOIndex(cache *utxo.UTXOCache) *UTXOIndex {
	return &UTXOIndex{
		indexRemove: make(map[string]*utxo.UTXOTx),
		indexAdd:    make(map[string]*utxo.UTXOTx),
		cache:       cache,
		mutex:       &sync.RWMutex{},
	}
}

func (utxos *UTXOIndex) SetIndexAdd(indexAdd map[string]*utxo.UTXOTx) {
	utxos.indexAdd = indexAdd
}

func (utxos *UTXOIndex) SetindexRemove(indexRemove map[string]*utxo.UTXOTx) {
	utxos.indexRemove = indexRemove
}

func (utxos *UTXOIndex) IsIndexAddExist(pubKeyHash account.PubKeyHash) bool {
	_, ok := utxos.indexAdd[pubKeyHash.String()]
	return ok
}

func (utxos *UTXOIndex) Save() error {
	utxos.mutex.Lock()
	defer utxos.mutex.Unlock()

	//save utxo to db/cache
	for pubkeyHash, utxoTx := range utxos.indexAdd {
		err := utxos.cache.AddUtxos(utxoTx, pubkeyHash)
		if err != nil {
			return err
		}
	}

	//delete utxo from db/cache which in indexRemove
	for pubkeyHash, utxoTx := range utxos.indexRemove {
		err := utxos.cache.RemoveUtxos(utxoTx, pubkeyHash)
		if err != nil {
			return err
		}
	}
	//clear
	utxos.indexAdd = make(map[string]*utxo.UTXOTx)
	utxos.indexRemove = make(map[string]*utxo.UTXOTx)
	return nil
}

func (utxos *UTXOIndex) IsLastUtxoKeyExist(pubKeyHash account.PubKeyHash) bool {
	utxos.mutex.RLock()
	defer utxos.mutex.RUnlock()
	return utxos.cache.IsLastUtxoKeyExist(pubKeyHash.String())
}

func (utxos *UTXOIndex) GetAllUTXOsByPubKeyHash(pubkeyHash account.PubKeyHash) *utxo.UTXOTx {
	utxos.mutex.RLock()
	defer utxos.mutex.RUnlock()
	utxoTx := utxos.cache.GetUTXOTx(pubkeyHash)
	utxoTxAdd := utxos.indexAdd[pubkeyHash.String()]
	if utxoTxAdd != nil {
		for k, v := range utxoTxAdd.Indices {
			utxoTx.Indices[k] = v
		}
	}

	utxoTxRemove := utxos.indexRemove[pubkeyHash.String()]
	if utxoTxRemove != nil {
		for k := range utxoTxRemove.Indices {
			if _, ok := utxoTx.Indices[k]; ok {
				delete(utxoTx.Indices, k)
			}
		}
	}
	return utxoTx
}

func (utxos *UTXOIndex) GetUpdatedUtxo(pubkeyHash account.PubKeyHash, txid []byte, vout int) (*utxo.UTXO, error) {
	utxos.mutex.RLock()
	defer utxos.mutex.RUnlock()
	if _, ok := utxos.indexAdd[pubkeyHash.String()]; ok {
		utxo := utxos.indexAdd[pubkeyHash.String()].GetUtxo(txid, vout)
		if utxo != nil {
			return utxo, nil
		}
	}

	if _, ok := utxos.indexRemove[pubkeyHash.String()]; ok {
		utxo := utxos.indexRemove[pubkeyHash.String()].GetUtxo(txid, vout)
		if utxo != nil {
			return nil, errorValues.ErrUtxoAlreadyRemoved
		}
	}

	utxo, err := utxos.cache.GetUtxo(utxo.GetUTXOKey(txid, vout))
	if err != nil {
		logger.Warn("GetUpdatedUtxo err.")
		return nil, err
	}
	return utxo, nil
}

func (utxos *UTXOIndex) GetContractCreateUTXOByPubKeyHash(pubkeyHash account.PubKeyHash) *utxo.UTXO {
	utxos.mutex.RLock()
	defer utxos.mutex.RUnlock()
	return utxos.cache.GetUtxoCreateContract(pubkeyHash.String())
}

// Returns utxo list of current contract address, except creation utxo
func (utxos *UTXOIndex) GetContractInvokeUTXOsByPubKeyHash(pubkeyHash account.PubKeyHash) []*utxo.UTXO {
	utxoTx := utxos.GetAllUTXOsByPubKeyHash(pubkeyHash)
	if utxoTx == nil {
		return nil
	}
	// Use a sorted key array to make utxo list ordered
	var sortedKeys []string
	for k, u := range utxoTx.Indices {
		if u.UtxoType != utxo.UtxoCreateContract {
			sortedKeys = append(sortedKeys, k)
		}
	}
	sort.Strings(sortedKeys)

	var invokeUTXOs []*utxo.UTXO
	for _, k := range sortedKeys {
		u := utxoTx.Indices[k]
		invokeUTXOs = append(invokeUTXOs, u)
	}
	return invokeUTXOs
}

// GetUTXOsAccordingToAmount returns a number of UTXOs that has a sum more than or equal to the amount
func (utxos *UTXOIndex) GetUTXOsAccordingToAmount(pubkeyHash account.PubKeyHash, amount *common.Amount) ([]*utxo.UTXO, error) {
	utxos.mutex.RLock()
	defer utxos.mutex.RUnlock()

	utxoTxRemove, utxoCache, cacheUTXOAmount, err := utxos.getUTXOsFromCacheUTXO(pubkeyHash, amount)
	if err != nil {
		return nil, err
	}
	if cacheUTXOAmount.Cmp(amount) >= 0 {
		return utxoCache, nil
	}

	leftAmount, err := amount.Sub(cacheUTXOAmount)
	if err != nil {
		return nil, err
	}

	utxoFromdb, err := utxos.cache.GetUTXOsByAmountWithOutRemovedUTXOs(pubkeyHash, leftAmount, utxoTxRemove)
	if err != nil {
		return nil, err
	}
	return append(utxoCache, utxoFromdb...), nil
}

func (utxos *UTXOIndex) getUTXOsFromCacheUTXO(pubkeyHash account.PubKeyHash, amount *common.Amount) (*utxo.UTXOTx, []*utxo.UTXO, *common.Amount, error) {
	utxoTxAdd := utxos.indexAdd[pubkeyHash.String()]
	utxoTxRemove := utxos.indexRemove[pubkeyHash.String()]

	var utxoSlice []*utxo.UTXO
	utxoAmount := common.NewAmount(0)

	if utxoTxAdd != nil {
		for _, u := range utxoTxAdd.Indices {
			if u.UtxoType == utxo.UtxoCreateContract {
				continue
			}
			if utxoTxRemove != nil {
				if _, ok := utxoTxRemove.Indices[u.GetUTXOKey()]; ok {
					continue
				}
			}
			utxoAmount = utxoAmount.Add(u.Value)
			utxoSlice = append(utxoSlice, u)
			if utxoAmount.Cmp(amount) >= 0 {
				break
			}
		}
	}
	return utxoTxRemove, utxoSlice, utxoAmount, nil
}

func (utxos *UTXOIndex) UpdateUtxo(tx *transaction.Transaction) bool {
	adaptedTx := transaction.NewTxAdapter(tx)
	if adaptedTx.IsNormal() || adaptedTx.IsContract() || adaptedTx.IsContractSend() {
		for _, txin := range tx.Vin {
			isContract, _ := account.PubKeyHash(txin.PubKey).IsContract()
			// spent contract utxo
			pubKeyHash := txin.PubKey
			if !isContract {
				// spent normal utxo
				ta := account.NewTransactionAccountByPubKey(txin.PubKey)
				_, err := account.IsValidPubKey(txin.PubKey)
				if err != nil {
					logger.WithError(err).Warn("UTXOIndex: txin.pubKey error, discard update in utxo.")
					return false
				}
				pubKeyHash = ta.GetPubKeyHash()
			}

			err := utxos.removeUTXO(pubKeyHash, txin.Txid, txin.Vout)
			if err != nil {
				logger.WithError(err).Warn("UTXOIndex: removeUTXO error, discard update in utxo.")
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
func (utxos *UTXOIndex) UpdateUtxos(txs []*transaction.Transaction) bool {
	// Create a copy of the index so operations below are only temporal
	errFlag := true
	for _, tx := range txs {
		if !utxos.UpdateUtxo(tx) {
			errFlag = false
		}
	}
	return errFlag
}

// UndoTxsInBlock compute the (previous) UTXOIndex resulted from undoing the transactions in given blk.
// Note that the operation does not save the index to db.
func (utxos *UTXOIndex) UndoTxsInBlock(blk *block.Block, db storage.Storage) error {

	for i := len(blk.GetTransactions()) - 1; i >= 0; i-- {
		tx := blk.GetTransactions()[i]
		err := utxos.excludeVoutsInTx(tx, db)
		if err != nil {
			return err
		}
		adaptedTx := transaction.NewTxAdapter(tx)
		if adaptedTx.IsCoinbase() || adaptedTx.IsRewardTx() || adaptedTx.IsGasRewardTx() || adaptedTx.IsGasChangeTx() {
			continue
		}
		err = utxos.unspendVinsInTx(tx, db)
		if err != nil {
			return err
		}
	}
	return nil
}

// excludeVoutsInTx removes the UTXOs generated in a transaction from the UTXOIndex.
func (utxos *UTXOIndex) excludeVoutsInTx(tx *transaction.Transaction, db storage.Storage) error {
	for i, vout := range tx.Vout {
		err := utxos.removeUTXO(vout.PubKeyHash, tx.ID, i)
		if err != nil {
			logger.Warn("excludeVoutsInTx error")
			return err
		}
	}
	return nil
}

func getTXOutputSpent(in transactionbase.TXInput, db storage.Storage) (transactionbase.TXOutput, int, error) {

	vout, err := transaction.GetTxOutput(in, db)

	if err != nil {
		return transactionbase.TXOutput{}, 0, errorValues.ErrTXInputInvalid
	}
	return vout, in.Vout, nil
}

// unspendVinsInTx adds UTXOs back to the UTXOIndex as a result of undoing the spending of the UTXOs in a transaction.
func (utxos *UTXOIndex) unspendVinsInTx(tx *transaction.Transaction, db storage.Storage) error {
	for _, vin := range tx.Vin {
		vout, voutIndex, err := getTXOutputSpent(vin, db)
		if err != nil {
			return err
		}
		utxos.AddUTXO(vout, vin.Txid, voutIndex)
	}
	return nil
}

// AddUTXO adds an unspent TXOutput to index
func (utxos *UTXOIndex) AddUTXO(txout transactionbase.TXOutput, txid []byte, vout int) {
	var u *utxo.UTXO
	//if it is a smart contract deployment utxo add it to contract utxos
	if isContract, _ := txout.PubKeyHash.IsContract(); isContract {
		if !utxos.IsIndexAddExist(txout.PubKeyHash) &&
			!utxos.IsLastUtxoKeyExist(txout.PubKeyHash) {
			u = utxo.NewUTXO(txout, txid, vout, utxo.UtxoCreateContract)
		} else {
			u = utxo.NewUTXO(txout, txid, vout, utxo.UtxoInvokeContract)
		}
	} else {
		u = utxo.NewUTXO(txout, txid, vout, utxo.UtxoNormal)
	}

	utxoTx, ok := utxos.indexAdd[txout.PubKeyHash.String()]
	utxos.mutex.Lock()
	defer utxos.mutex.Unlock()
	if !ok {
		utxoTx := utxo.NewUTXOTx()
		utxoTx.PutUtxo(u)
		utxos.indexAdd[txout.PubKeyHash.String()] = &utxoTx
	} else {
		utxoTx.PutUtxo(u)
	}
}

// removeUTXO finds and removes a UTXO from UTXOIndex
func (utxos *UTXOIndex) removeUTXO(pkh account.PubKeyHash, txid []byte, vout int) error {
	utxoKey := utxo.GetUTXOKey(txid, vout)
	//update indexRemove
	ok := false
	utxos.mutex.Lock()
	defer utxos.mutex.Unlock()
	if _, ok = utxos.indexAdd[pkh.String()]; ok {
		_, ok = utxos.indexAdd[pkh.String()].Indices[utxoKey]
	}
	if ok {
		delete(utxos.indexAdd[pkh.String()].Indices, utxoKey)
	} else {
		u, err := utxos.cache.GetUtxo(utxoKey)
		if err != nil {
			logger.Error("removeUTXO err")
			return errorValues.ErrUTXONotFound
		}
		utxoTx, ok := utxos.indexRemove[pkh.String()]
		if !ok {
			utxoTx := utxo.NewUTXOTx()
			utxoTx.PutUtxo(u)
			utxos.indexRemove[pkh.String()] = &utxoTx
		} else {
			utxoTx.PutUtxo(u)
		}
	}
	return nil
}

//creates a deepcopy of the receiver object
func (utxos *UTXOIndex) DeepCopy() *UTXOIndex {
	utxos.mutex.RLock()
	defer utxos.mutex.RUnlock()

	utxocopy := NewUTXOIndex(utxos.cache)
	for pkh, utxoTx := range utxos.indexAdd {
		newUtxoTx := utxoTx.DeepCopy()
		utxocopy.indexAdd[pkh] = newUtxoTx
	}

	for pkh, utxoTx := range utxos.indexRemove {
		newUtxoTx := utxoTx.DeepCopy()
		utxocopy.indexRemove[pkh] = newUtxoTx
	}
	return utxocopy
}

func (utxos *UTXOIndex) SelfCheckingUTXO() {
	utxos.mutex.Lock()
	defer utxos.mutex.Unlock()
	logger.Info("start utxo self checking...")
	//remove utxo from addUTXO list which already has been added to db
	for key, utxoTx := range utxos.indexAdd {
		for _, utxo := range utxoTx.Indices {
			if _, err := utxos.cache.GetUtxo(utxo.GetUTXOKey()); err == nil {
				delete(utxoTx.Indices, utxo.GetUTXOKey())
			}
			if len(utxoTx.Indices) == 0 {
				delete(utxos.indexAdd, key)
			}
		}
	}
	//remove utxo from removeUTXO list which already has been deleted from db
	for key, utxoTx := range utxos.indexRemove {
		for _, utxo := range utxoTx.Indices {
			if _, err := utxos.cache.GetUtxo(utxo.GetUTXOKey()); err != nil {
				delete(utxoTx.Indices, utxo.GetUTXOKey())
			}
			if len(utxoTx.Indices) == 0 {
				delete(utxos.indexRemove, key)
			}
		}
	}
	logger.Info("self checking complete")
}
