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
	"encoding/hex"
	"github.com/dappley/go-dappley/core/account"
	utxopb "github.com/dappley/go-dappley/core/utxo/pb"
	"github.com/dappley/go-dappley/storage"
	"github.com/dappley/go-dappley/util"
	"github.com/golang/protobuf/proto"
	lru "github.com/hashicorp/golang-lru"
	logger "github.com/sirupsen/logrus"
	"strconv"
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

func (utxoCache *UTXOCache) AddUtxos(utxoTx *UTXOTx, pubkey string, indexUtxoTx *UTXOTx) error {
	lastestUtxoKey, err := utxoCache.db.Get(util.Str2bytes(pubkey))
	for key, utxo := range utxoTx.Indices {
		utxo.NextUtxoKey = lastestUtxoKey
		lastestUtxoKey = util.Str2bytes(key)
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
	err = utxoCache.db.Put(util.Str2bytes(pubkey), lastestUtxoKey) // storge the latest utxokey
	if err != nil {
		return err
	}

	utxoCache.cache.Add(pubkey, indexUtxoTx)

	//contract
	pubKeyHash, err := hex.DecodeString(pubkey)
	if err != nil {
		return err
	}
	err = utxoCache.Put(pubKeyHash, utxoTx)
	if err != nil {
		return err
	}

	return nil
}

func (utxoCache *UTXOCache) RemoveUtxos(utxoTx *UTXOTx, pubkey string, indexUtxoTx *UTXOTx) error {
	for key, utxo := range utxoTx.Indices {
		preUtxo := indexUtxoTx.GetPerUtxoByKey(util.Str2bytes(key))
		if preUtxo == nil {//this utxo is the head utxo
			if len(utxo.NextUtxoKey) == 0 {
				err := utxoCache.db.Del(util.Str2bytes(pubkey))
				if err != nil {
					logger.WithFields(logger.Fields{"error": err}).Error("delete utxo from db failed.")
					return err
				}
			} else {
				err := utxoCache.db.Put(util.Str2bytes(pubkey), utxo.NextUtxoKey)
				if err != nil {
					return err
				}
			}
		} else {
			preUtxo.NextUtxoKey = utxo.NextUtxoKey
			utxoBytes, err := proto.Marshal(preUtxo.ToProto().(*utxopb.Utxo))
			if err != nil {
				return err
			}
			preUtxokey := string(preUtxo.Txid) + "_" + strconv.Itoa(preUtxo.TxIndex)
			err = utxoCache.db.Put(util.Str2bytes(preUtxokey), utxoBytes)
			if err != nil {
				return err
			}
			utxoCache.utxo.Add(preUtxokey, preUtxo)
		}
		err := utxoCache.db.Del(util.Str2bytes(key))
		if err != nil {
			logger.WithFields(logger.Fields{"error": err}).Error("delete utxo from db failed.")
			return err
		}
		utxoCache.utxo.Remove(key)
	}
	utxoCache.cache.Add(pubkey, indexUtxoTx)
	return nil
}

func (utxoCache *UTXOCache) DeserializeUTXOTx(utxokey string) (UTXOTx, error) {
	utxoTx := NewUTXOTx()
	utxoKey := utxokey

	for utxoKey != "" {
		var utxo = &UTXO{}
		utxoData, ok := utxoCache.utxo.Get(utxoKey)
		if ok {
			utxo = utxoData.(*UTXO)
		} else {
			rawBytes, err := utxoCache.db.Get(util.Str2bytes(utxoKey))
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
		utxoCache.utxo.Add(utxoKey, utxo)
		utxoTx.Indices[utxoKey] = utxo

		utxoKey = util.Bytes2str(utxo.NextUtxoKey) //get previous utxo key
	}
	return utxoTx, nil
}

// Return value from cache
func (utxoCache *UTXOCache) Get(pubKeyHash account.PubKeyHash) *UTXOTx {
	mapData, ok := utxoCache.cache.Get(pubKeyHash.String())
	if ok {
		return mapData.(*UTXOTx)
	}

	lastUtxokey, err := utxoCache.db.Get(util.Str2bytes(pubKeyHash.String()))
	var utxoTx UTXOTx
	if err == nil {
		utxoTx, err = utxoCache.DeserializeUTXOTx(util.Bytes2str(lastUtxokey))
		utxoCache.cache.Add(pubKeyHash.String(), &utxoTx)
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

	for _, u := range value.Indices {
		if u.UtxoType == UtxoCreateContract {
			utxoCache.contractCreateCache.Add(string(pubKeyHash), u)
		}
	}
	return nil
}

func (utxoCache *UTXOCache) Delete(pubKeyHash account.PubKeyHash) error {
	if pubKeyHash == nil {
		return account.ErrEmptyPublicKeyHash
	}
	return utxoCache.db.Del(pubKeyHash)
}
