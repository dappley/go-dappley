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

package utxo

import (
	"github.com/dappley/go-dappley/core/account"
	utxopb "github.com/dappley/go-dappley/core/utxo/pb"
	"github.com/dappley/go-dappley/storage"
	"github.com/dappley/go-dappley/util"
	"github.com/golang/protobuf/proto"
	lru "github.com/hashicorp/golang-lru"
	logger "github.com/sirupsen/logrus"
)

const UtxoCacheLRUCacheLimit = 1024

// UTXOCache holds temporary UTXOTx data
type UTXOCache struct {
	// key: address, value: UTXOTx
	contractCreateCache *lru.Cache
	cache               *lru.Cache
	utxo                *lru.Cache
	db                  storage.Storage
}

func NewUTXOCache(db storage.Storage) *UTXOCache {
	utxoCache := &UTXOCache{
		contractCreateCache: nil,
		cache:               nil,
		utxo:                nil,
		db:                  db,
	}
	utxoCache.cache, _ = lru.New(UtxoCacheLRUCacheLimit)
	utxoCache.utxo, _ = lru.New(UtxoCacheLRUCacheLimit)
	utxoCache.contractCreateCache, _ = lru.New(UtxoCacheLRUCacheLimit)
	return utxoCache
}

func (utxoCache *UTXOCache) AddUtxos(utxoTx *UTXOTx) error {
	for key, utxo := range utxoTx.Indices {
		utxoBytes, err := proto.Marshal(utxo.ToProto().(*utxopb.Utxo))
		if err != nil {
			return err
		}

		err = utxoCache.db.Put(util.Str2bytes(key), utxoBytes)
		if err != nil {
			return err
		}
		utxoCache.utxo.Add(key, utxo)
	}
	return nil
}

func (utxoCache *UTXOCache) RemoveUtxos(utxoTx *UTXOTx) error {
	for key := range utxoTx.Indices {
		err := utxoCache.db.Del(util.Str2bytes(key))
		if err != nil {
			logger.WithFields(logger.Fields{"error": err}).Error("delete utxo from db failed.")
			return err
		}
		utxoCache.utxo.Remove(key)
	}
	return nil
}

func (utxoCache *UTXOCache) DeserializeUTXOTx(d []byte) (UTXOTx, error) {
	utxoTx := NewUTXOTx()

	utxokeyList := &utxopb.UtxoKeyList{}
	err := proto.Unmarshal(d, utxokeyList)
	if err != nil {
		logger.WithFields(logger.Fields{"error": err}).Error("UtxoTx: parse UtxoTx failed.")
		return utxoTx, nil
	}

	//get all utxo from db by using utxokey
	for _, utxoKey := range utxokeyList.UtxoKey {
		var utxo = &UTXO{}
		utxoData, ok := utxoCache.utxo.Get(util.Bytes2str(utxoKey))
		if ok {
			utxo = utxoData.(*UTXO)
		} else {
			rawBytes, err := utxoCache.db.Get(utxoKey)
			if err == nil {
				utxoPb := &utxopb.Utxo{}
				err := proto.Unmarshal(rawBytes, utxoPb)
				if err != nil {
					logger.WithFields(logger.Fields{"error": err}).Error("DeserializeUTXOTx: Unmarshal utxo failed.")
					return utxoTx, err
				}
				utxo.FromProto(utxoPb)
			} else {
				logger.WithFields(logger.Fields{"error": err}).Error("DeserializeUTXOTx: utxo didn't in db！")
				return utxoTx, err
			}
		}
		utxoCache.utxo.Add(util.Bytes2str(utxoKey), utxo)
		utxoTx.Indices[util.Bytes2str(utxoKey)] = utxo
	}
	return utxoTx, nil
}

// Return value from cache
func (utxoCache *UTXOCache) Get(pubKeyHash account.PubKeyHash) *UTXOTx {
	mapData, ok := utxoCache.cache.Get(string(pubKeyHash))
	if ok {
		return mapData.(*UTXOTx)
	}

	rawBytes, err := utxoCache.db.Get(pubKeyHash)

	var utxoTx UTXOTx
	if err == nil {
		utxoTx, err = utxoCache.DeserializeUTXOTx(rawBytes)
		utxoCache.cache.Add(string(pubKeyHash), &utxoTx)
	} else {
		utxoTx = NewUTXOTx()
	}

	for _, u := range utxoTx.Indices {
		if u.UtxoType == UtxoCreateContract {
			utxoCache.contractCreateCache.Add(string(pubKeyHash), u)
		}
	}
	return &utxoTx
}

// Return value from cache
func (utxoCache *UTXOCache) GetContractCreateUtxo(pubKeyHash account.PubKeyHash) *UTXO {
	mapData, ok := utxoCache.contractCreateCache.Get(string(pubKeyHash))
	if !ok {
		utxotx := utxoCache.Get(pubKeyHash)
		if len(utxotx.Indices) > 0 {
			mapData, ok = utxoCache.contractCreateCache.Get(string(pubKeyHash))
			if !ok {
				return nil
			}
		} else {
			return nil
		}
	}

	return mapData.(*UTXO)
}

// Add new data into cache
func (utxoCache *UTXOCache) Put(pubKeyHash account.PubKeyHash, value *UTXOTx) error {
	if pubKeyHash == nil {
		return account.ErrEmptyPublicKeyHash
	}
	err := utxoCache.putUtxoTx(pubKeyHash, value)
	if err != nil {
		return err
	}

	for _, u := range value.Indices {
		if u.UtxoType == UtxoCreateContract {
			utxoCache.contractCreateCache.Add(string(pubKeyHash), u)
		}
	}
	return nil
}

// Add new data into cache
func (utxoCache *UTXOCache) putUtxoTx(pubKeyHash account.PubKeyHash, value *UTXOTx) error {
	savedUtxoTx := value.DeepCopy()
	mapData, ok := utxoCache.cache.Get(string(pubKeyHash))
	utxoCache.cache.Add(string(pubKeyHash), savedUtxoTx)
	err := utxoCache.db.Put(pubKeyHash, value.Serialize())
	if err == nil && ok {
		Free(mapData.(*UTXOTx))
	}
	return err
}

func (utxoCache *UTXOCache) Delete(pubKeyHash account.PubKeyHash) error {
	if pubKeyHash == nil {
		return account.ErrEmptyPublicKeyHash
	}
	return utxoCache.db.Del(pubKeyHash)
}
