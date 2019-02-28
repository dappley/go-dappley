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
	utxoCache := &UTXOCache{
		cache: nil,
		db:    db,
	}
	utxoCache.cache, _ = lru.New(UtxoCacheLRUCacheLimit)
	return utxoCache
}

// Return value from cache
func (utxoCache *UTXOCache) Get(pubKeyHash PubKeyHash) *UTXOTx {
	mapData, ok := utxoCache.cache.Get(string(pubKeyHash))
	if ok {
		return mapData.(*UTXOTx)
	}

	rawBytes, err := utxoCache.db.Get(pubKeyHash)

	var utxoTx UTXOTx
	if err == nil {
		utxoTx = DeserializeUTXOTx(rawBytes)
		utxoCache.cache.Add(string(pubKeyHash), &utxoTx)
	} else {
		utxoTx = NewUTXOTx()
	}
	return &utxoTx
}

// Add new data into cache
func (utxoCache *UTXOCache) Put(pubKeyHash PubKeyHash, value *UTXOTx) error {
	if pubKeyHash == nil {
		return ErrEmptyPublicKeyHash
	}
	utxoCache.cache.Add(string(pubKeyHash), value)
	return utxoCache.db.Put(pubKeyHash, value.Serialize())
}
