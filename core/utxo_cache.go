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
	"github.com/dappley/go-dappley/storage"
	"github.com/hashicorp/golang-lru"
)

const UtxoCacheLRUCacheLimit = 1024

// UTXOCache holds temporary UTXOTx data
type UTXOCache struct {
	// key: address, value: UTXOTx
	cache *lru.Cache
	db    storage.Storage
}

func NewUTXOCache(db storage.Storage) *UTXOCache {
	utxoCache := &UTXOCache{}
	utxoCache.cache, _ = lru.New(UtxoCacheLRUCacheLimit)
	return utxoCache
}

// Return value from cache
func (utxoCache UTXOCache) Get(pubKeyHash PubKeyHash) *UTXOTx {
	mapData, ok := utxoCache.cache.Get(pubKeyHash)
	if !ok {
		return nil
	}
	if mapData == nil {
		// load from db
		rawBytes, err := utxoCache.db.Get(pubKeyHash)
		if err != nil {
			return DeserializeUTXOTx(rawBytes)
		}
		return nil
	}
	utxoTx := mapData.(*UTXOTx)
	return utxoTx
}

// Add new data into cache
func (utxoCache UTXOCache) Put(pubKeyHash PubKeyHash, value UTXOTx) {
	if pubKeyHash == nil {
		return
	}
	utxoCache.cache.Add(pubKeyHash, value)
}

func (utxoCache *UTXOCache) GetContractUtxos() *UTXOTx {
	return utxoCache.Get([]byte(contractUtxoKey))
}

func (utxoCache *UTXOCache) UpdateUtxo(tx *Transaction) bool {
	if !tx.IsCoinbase() && !tx.IsRewardTx() {
		for _, txin := range tx.Vin {
			pkh, err := NewUserPubKeyHash(txin.PubKey)
			if err != nil {
				return false
			}
			utxoTx := utxoCache.Get(pkh)
			utxoTx.RemoveUtxo(txin.Txid, txin.Vout)
		}
	}
	for i, txout := range tx.Vout {
		utxoCache.addUTXO(txout, tx.ID, i)
	}
	return true
}

func (utxoCache *UTXOCache) addUTXO(txout TXOutput, txid []byte, vout int) {
	u := newUTXO(txout, txid, vout)
	//if it is a smart contract deployment utxo add it to contract utxos
	isContract, _ := txout.PubKeyHash.IsContract();
	utxoTx := utxoCache.Get(u.PubKeyHash)
	if isContract && len(utxoTx.txIndex) == 0 {
		utxoTxContract := utxoCache.GetContractUtxos()
		utxoTxContract.PutUtxo(u)
	}
	utxoTx.PutUtxo(u)
}
