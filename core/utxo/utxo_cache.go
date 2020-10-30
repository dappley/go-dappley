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
	"bytes"
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
	lastUtxoKey         *lru.Cache
	db                  storage.Storage
}

func NewUTXOCache(db storage.Storage) *UTXOCache {
	utxoCache := &UTXOCache{
		contractCreateCache: nil,
		cache:               nil,
		utxo:                nil,
		lastUtxoKey:         nil,
		db:                  db,
	}
	utxoCache.cache, _ = lru.New(UtxoCacheLRUCacheLimit)
	utxoCache.utxo, _ = lru.New(UtxoCacheLRUCacheLimit)
	utxoCache.lastUtxoKey, _ = lru.New(UtxoCacheLRUCacheLimit)
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
	err = utxoCache.db.Put(util.Str2bytes(pubkey), lastestUtxoKey) // storage the latest utxokey
	if err != nil {
		return err
	}
	utxoCache.lastUtxoKey.Add(pubkey, lastestUtxoKey)

	//utxoCache.cache.Add(pubkey, indexUtxoTx)

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
		preUtxo := utxoCache.GetPreUtxo(pubkey, key)
		if preUtxo == nil { //this utxo is the head utxo
			if len(utxo.NextUtxoKey) == 0 {
				err := utxoCache.db.Del(util.Str2bytes(pubkey))
				if err != nil {
					logger.WithFields(logger.Fields{"error": err}).Error("delete utxo from db failed.")
					return err
				}
				utxoCache.lastUtxoKey.Remove(pubkey)
			} else {
				err := utxoCache.db.Put(util.Str2bytes(pubkey), utxo.NextUtxoKey)
				if err != nil {
					return err
				}
				utxoCache.lastUtxoKey.Add(pubkey, utxo.NextUtxoKey)
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
	//utxoCache.cache.Add(pubkey, indexUtxoTx)
	return nil
}

func (utxoCache *UTXOCache) GetUtxo(utxoKey string) *UTXO {
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
				logger.WithFields(logger.Fields{"error": err}).Error("GetUtxoByPubkey: Unmarshal utxo failed.")
				return nil
			}
			utxo.FromProto(utxoPb)
		} else {
			logger.WithFields(logger.Fields{"error": err}).Error("GetUtxoByPubkey: utxo didn't in db！")
			return nil
		}
	}
	utxoCache.utxo.Add(utxoKey, utxo)
	return utxo
}

func (utxoCache *UTXOCache) GetUtxoByPubkey(pubKey, targetUtxokey string) *UTXO {
	lastUtxokey, err := utxoCache.getLastUTXOKey(pubKey)
	if err != nil {
		return nil
	}
	utxoKey := util.Bytes2str(lastUtxokey)

	for utxoKey != "" {
		utxo := utxoCache.GetUtxo(utxoKey)
		utxokey := string(utxo.Txid) + "_" + strconv.Itoa(utxo.TxIndex)
		if utxokey == targetUtxokey {
			return utxo
		}
		utxoKey = util.Bytes2str(utxo.NextUtxoKey) //get previous utxo key
	}
	return nil
}

func (utxoCache *UTXOCache) GetPreUtxo(pubKey, targetUtxokey string) *UTXO {
	lastUtxokey, err := utxoCache.getLastUTXOKey(pubKey)
	if err != nil {
		return nil
	}
	utxoKey := util.Bytes2str(lastUtxokey)

	for utxoKey != "" {
		utxo := utxoCache.GetUtxo(utxoKey)
		if bytes.Equal(utxo.NextUtxoKey, util.Str2bytes(targetUtxokey)) {
			return utxo
		}
		utxoKey = util.Bytes2str(utxo.NextUtxoKey) //get previous utxo key
	}
	return nil
}

func (utxoCache *UTXOCache) getLastUTXOKey(pubKeyHash string) ([]byte, error) {
	lastUtxoKeyData, ok := utxoCache.lastUtxoKey.Get(pubKeyHash)
	if ok {
		return lastUtxoKeyData.([]byte), nil
	}

	lastUtxoKey, err := utxoCache.db.Get(util.Str2bytes(pubKeyHash))
	if err == nil {
		utxoCache.lastUtxoKey.Add(pubKeyHash, lastUtxoKey)
		return lastUtxoKey, nil
	}

	return []byte{}, err
}

func (utxoCache *UTXOCache) IsLastUtxoKeyExist(pubKeyHash account.PubKeyHash) bool {
	_, err := utxoCache.getLastUTXOKey(pubKeyHash.String())
	if err == nil {
		return true
	}
	return false
}

func (utxoCache *UTXOCache) GetUTXOTx(pubKeyHash account.PubKeyHash) *UTXOTx {
	lastUtxokey, err := utxoCache.getLastUTXOKey(pubKeyHash.String())
	utxoTx := NewUTXOTx()
	if err == nil {
		utxoKey := util.Bytes2str(lastUtxokey)
		for utxoKey != "" {
			utxo := utxoCache.GetUtxo(utxoKey)
			utxoTx.Indices[utxoKey] = utxo
			utxoKey = util.Bytes2str(utxo.NextUtxoKey) //get previous utxo key
		}
		//utxoCache.cache.Add(pubKeyHash.String(), &utxoTx)
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
		utxotx := utxoCache.GetUTXOTx(pubKeyHash)
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
