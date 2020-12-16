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
	"errors"
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

func (utxoCache *UTXOCache) AddUtxos(utxoTx *UTXOTx, pubkey string) error {
	lastestUtxoKey := utxoCache.getLastUTXOKey(pubkey)
	for key, utxo := range utxoTx.Indices {
		if bytes.Equal(util.Str2bytes(utxo.GetUTXOKey()),lastestUtxoKey){
			return errors.New("Add utxo error: utxo already exist.")
		}
		utxo.NextUtxoKey = lastestUtxoKey
		err := utxoCache.putUTXOToDB(utxo)
		if err != nil {
			return err
		}
		lastestUtxoKey = util.Str2bytes(key)
	}
	err := utxoCache.putLastUTXOKeyToDB(pubkey, lastestUtxoKey)
	if err != nil {
		return err
	}
	return nil
}

func (utxoCache *UTXOCache) RemoveUtxos(utxoTx *UTXOTx, pubkey string) error {
	for key, utxo := range utxoTx.Indices {
		preUTXO, err := utxoCache.GetPreUtxo(pubkey, key)
		if err != nil {
			return err
		}
		if preUTXO == nil { //this utxo is the head utxo
			if bytes.Equal(utxo.NextUtxoKey, []byte{}) {
				err = utxoCache.deleteLastUTXOKeyFromDB(pubkey)
				if err != nil {
					return err
				}
			} else {
				err := utxoCache.putLastUTXOKeyToDB(pubkey, utxo.NextUtxoKey)
				if err != nil {
					return err
				}
			}
		} else {
			preUTXO.NextUtxoKey = utxo.NextUtxoKey
			err = utxoCache.putUTXOToDB(preUTXO)
			if err != nil {
				return err
			}
			//*if this utxo in utxoTx,then need to update
			if _, ok := utxoTx.Indices[preUTXO.GetUTXOKey()]; ok {
				utxoTx.Indices[preUTXO.GetUTXOKey()] = preUTXO
			}
		}
		err = utxoCache.deleteUTXOFromDB(key)
		if err != nil {
			return err
		}
	}
	return nil
}

func (utxoCache *UTXOCache) GetUtxo(utxoKey string) (*UTXO, error) {
	var utxo = &UTXO{}
	utxoData, ok := utxoCache.utxo.Get(utxoKey)
	if ok {
		utxo = utxoData.(*UTXO)
		return utxo, nil
	}
	return utxoCache.getUTXOFromDB(utxoKey)
}

func (utxoCache *UTXOCache) GetUtxoByPubkey(pubKey, targetUtxokey string) (*UTXO, error) {
	utxoKey := utxoCache.getLastUTXOKey(pubKey)
	for !bytes.Equal(utxoKey, []byte{}) {
		utxo, err := utxoCache.GetUtxo(util.Bytes2str(utxoKey))
		if err != nil {
			return nil, err
		}
		if utxo.GetUTXOKey() == targetUtxokey {
			return utxo, nil
		}
		utxoKey = utxo.NextUtxoKey
	}
	return nil, errors.New("utxo not found")
}

func (utxoCache *UTXOCache) GetPreUtxo(pubKey, targetUtxokey string) (*UTXO, error) {
	utxoKey := utxoCache.getLastUTXOKey(pubKey)
	for !bytes.Equal(utxoKey, []byte{}) {
		utxo, err := utxoCache.GetUtxo(util.Bytes2str(utxoKey))
		if err != nil {
			return nil, err
		}
		if bytes.Equal(utxo.NextUtxoKey, util.Str2bytes(targetUtxokey)) {
			return utxo, nil
		}
		utxoKey = utxo.NextUtxoKey
	}
	return nil, nil //preutxo not found, this is a possible situation, not an error
}

func (utxoCache *UTXOCache) getLastUTXOKey(pubKeyHash string) []byte {
	lastUtxoKeyData, ok := utxoCache.lastUtxoKey.Get(pubKeyHash)
	if ok {
		return lastUtxoKeyData.([]byte)
	}

	lastUtxoKey, err := utxoCache.db.Get(util.Str2bytes(pubKeyHash))
	if err != nil {
		return []byte{}
	}
	utxoCache.lastUtxoKey.Add(pubKeyHash, lastUtxoKey)
	return lastUtxoKey
}

func (utxoCache *UTXOCache) IsLastUtxoKeyExist(pubKeyHash string) bool {
	if bytes.Equal(utxoCache.getLastUTXOKey(pubKeyHash), []byte{}) {
		return false
	}
	return true
}

func (utxoCache *UTXOCache) GetUTXOTx(pubKeyHash account.PubKeyHash) *UTXOTx {
	lastUtxokey := utxoCache.getLastUTXOKey(pubKeyHash.String())
	utxoTx := NewUTXOTx()
	utxoKey := util.Bytes2str(lastUtxokey)
	for utxoKey != "" {
		utxo, err := utxoCache.GetUtxo(utxoKey)
		if err != nil {
			//todo: return err
			logger.Error(err)
		}
		utxoTx.Indices[utxoKey] = utxo
		utxoKey = util.Bytes2str(utxo.NextUtxoKey) //get previous utxo key
	}
	return &utxoTx
}

// Return value from cache
func (utxoCache *UTXOCache) GetContractCreateUtxo(pubKeyHash account.PubKeyHash) *UTXO {
	mapData, ok := utxoCache.contractCreateCache.Get(pubKeyHash.String())
	if ok{
		return  mapData.(*UTXO)
	}
	return utxoCache.getContractCreateUtxoFromDB(pubKeyHash.String())
}

func (utxoCache *UTXOCache) Delete(pubKeyHash account.PubKeyHash) error {
	if pubKeyHash == nil {
		return account.ErrEmptyPublicKeyHash
	}
	return utxoCache.db.Del(pubKeyHash)
}

func (utxoCache *UTXOCache) putUTXOToDB(utxo *UTXO) error {
	utxoBytes, err := proto.Marshal(utxo.ToProto().(*utxopb.Utxo))
	if err != nil {
		return err
	}
	err = utxoCache.db.Put(util.Str2bytes(utxo.GetUTXOKey()), utxoBytes)
	if err != nil {
		logger.WithFields(logger.Fields{"error": err}).Error("put utxo to db failed！")
		return err
	}
	utxoCache.utxo.Add(utxo.GetUTXOKey(), utxo)
	return nil
}

func (utxoCache *UTXOCache) getUTXOFromDB(utxoKey string) (*UTXO, error) {
	var utxo = &UTXO{}
	rawBytes, err := utxoCache.db.Get(util.Str2bytes(utxoKey))
	if err == nil {
		utxoPb := &utxopb.Utxo{}
		err := proto.Unmarshal(rawBytes, utxoPb)
		if err != nil {
			logger.WithFields(logger.Fields{"error": err}).Error("Unmarshal utxo failed.")
			return nil, err
		}
		utxo.FromProto(utxoPb)
	} else {
		logger.WithFields(logger.Fields{"error": err}).Error("get utxo from db failed！")
		return nil, err
	}
	utxoCache.utxo.Add(utxoKey, utxo)
	return utxo, nil
}

func (utxoCache *UTXOCache) deleteUTXOFromDB(utxoKey string) error {
	err := utxoCache.db.Del(util.Str2bytes(utxoKey))
	if err != nil {
		logger.WithFields(logger.Fields{"error": err}).Error("delete utxo from db failed.")
		return err
	}
	utxoCache.utxo.Remove(utxoKey)
	return nil
}

func (utxoCache *UTXOCache) putLastUTXOKeyToDB(pubkey string, lastestUtxoKey []byte) error {
	err := utxoCache.db.Put(util.Str2bytes(pubkey), lastestUtxoKey)
	if err != nil {
		logger.WithFields(logger.Fields{"error": err}).Error("put last utxo key to db failed.")
		return err
	}
	utxoCache.lastUtxoKey.Add(pubkey, lastestUtxoKey)
	return nil
}

func (utxoCache *UTXOCache) deleteLastUTXOKeyFromDB(pubkey string) error {
	err := utxoCache.db.Del(util.Str2bytes(pubkey))
	if err != nil {
		logger.WithFields(logger.Fields{"error": err}).Error("delete last utxo key from db failed.")
		return err
	}
	utxoCache.lastUtxoKey.Remove(pubkey)
	return nil
}

func (utxoCache *UTXOCache) getContractCreateUtxoFromDB(pubKey string) *UTXO {
	utxoKey := utxoCache.getLastUTXOKey(pubKey)
	for !bytes.Equal(utxoKey, []byte{}) {
		utxo, err := utxoCache.GetUtxo(util.Bytes2str(utxoKey))
		if err != nil {
			return nil
		}
		if utxo.UtxoType == UtxoCreateContract {
			utxoCache.contractCreateCache.Add(pubKey, utxo)
			return utxo
		}
		utxoKey = utxo.NextUtxoKey
	}
	return  nil
}