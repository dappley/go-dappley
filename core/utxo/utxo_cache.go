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
	"github.com/dappley/go-dappley/common"
	"github.com/dappley/go-dappley/common/hash"
	"github.com/dappley/go-dappley/core/account"
	"github.com/dappley/go-dappley/core/stateLog"
	utxopb "github.com/dappley/go-dappley/core/utxo/pb"
	"github.com/dappley/go-dappley/storage"
	"github.com/dappley/go-dappley/util"
	"github.com/golang/protobuf/proto"
	lru "github.com/hashicorp/golang-lru"
	logger "github.com/sirupsen/logrus"
	"strings"
)

const (
	UtxoCacheLRUCacheLimit    = 1024
	ScStateCacheLRUCacheLimit = 1024
)

// UTXOCache holds temporary data
type UTXOCache struct {
	contractCreateCache *lru.Cache
	cache               *lru.Cache
	utxo                *lru.Cache
	utxoInfo            *lru.Cache
	ScStateCache
	db storage.Storage
}

type ScStateCache struct {
	stateLogCache *lru.Cache
	scStateCache  *lru.Cache
}

func NewUTXOCache(db storage.Storage) *UTXOCache {
	utxoCache := &UTXOCache{
		contractCreateCache: nil,
		cache:               nil,
		utxo:                nil,
		utxoInfo:            nil,
		db:                  db,
	}
	utxoCache.cache, _ = lru.New(UtxoCacheLRUCacheLimit)
	utxoCache.utxo, _ = lru.New(UtxoCacheLRUCacheLimit)
	utxoCache.utxoInfo, _ = lru.New(UtxoCacheLRUCacheLimit)
	utxoCache.contractCreateCache, _ = lru.New(UtxoCacheLRUCacheLimit)
	utxoCache.ScStateCache = NewScStateCache()
	return utxoCache
}

func NewScStateCache() ScStateCache {
	scStateCache := ScStateCache{
		stateLogCache: nil,
		scStateCache:  nil,
	}
	scStateCache.stateLogCache, _ = lru.New(ScStateCacheLRUCacheLimit)
	scStateCache.scStateCache, _ = lru.New(ScStateCacheLRUCacheLimit)
	return scStateCache
}

func (utxoCache *UTXOCache) AddUtxos(utxoTx *UTXOTx, pubkeyHash string) error {
	lastestUtxoKey := utxoCache.getLastUTXOKey(pubkeyHash)
	for key, utxo := range utxoTx.Indices {
		if bytes.Equal(util.Str2bytes(key), lastestUtxoKey) {
			return errors.New("add utxo failed: the utxo is same as the last utxo")
		}

		if !bytes.Equal([]byte{}, lastestUtxoKey) { //this pubkeyHash already has a UTXO
			_, err := utxoCache.UpdateNextUTXO(lastestUtxoKey, key)
			if err != nil {
				return err
			}
		}

		utxo.NextUtxoKey = lastestUtxoKey
		err := utxoCache.putUTXOToDB(utxo)
		if err != nil {
			return err
		}
		lastestUtxoKey = util.Str2bytes(key)

		if utxo.UtxoType == UtxoCreateContract {
			err := utxoCache.putCreateContractUTXOKey(pubkeyHash, util.Str2bytes(key))
			if err != nil {
				return err
			}
		}
		err = utxoCache.saveHardCore(utxo)
		if err != nil {
			return err
		}
	}
	err := utxoCache.putLastUTXOKey(pubkeyHash, lastestUtxoKey)
	if err != nil {
		return err
	}
	return nil
}

func (utxoCache *UTXOCache) RemoveUtxos(utxoTx *UTXOTx, pubkeyHash string) error {
	for key, utxo := range utxoTx.Indices {
		preUTXO, err := utxoCache.GetPreUtxo(key)
		if err != nil {
			return err
		}
		if preUTXO == nil { //this utxo is the head utxo
			if bytes.Equal(utxo.NextUtxoKey, []byte{}) { //the only utxo
				err = utxoCache.deleteUTXOInfo(pubkeyHash)
				if err != nil {
					return err
				}
			} else { //the first utxo in the chain
				err := utxoCache.putLastUTXOKey(pubkeyHash, utxo.NextUtxoKey)
				if err != nil {
					return err
				}

				nextUTXO, err := utxoCache.UpdateNextUTXO(utxo.NextUtxoKey, "")
				if err != nil {
					return err
				}
				if _, ok := utxoTx.Indices[nextUTXO.GetUTXOKey()]; ok {
					utxoTx.PutUtxo(nextUTXO)
				}
			}
		} else {
			if bytes.Equal(utxo.NextUtxoKey, util.Str2bytes(preUTXO.GetUTXOKey())) {
				return errors.New("remove utxo error: find duplicate utxo in db")
			}
			preUTXO.NextUtxoKey = utxo.NextUtxoKey
			err = utxoCache.putUTXOToDB(preUTXO)
			if err != nil {
				return err
			}
			//*if this utxo in utxoTx,then need to update
			if _, ok := utxoTx.Indices[preUTXO.GetUTXOKey()]; ok {
				utxoTx.PutUtxo(preUTXO)
			}

			if !bytes.Equal(utxo.NextUtxoKey, []byte{}) {
				nextUTXO, err := utxoCache.UpdateNextUTXO(utxo.NextUtxoKey, preUTXO.GetUTXOKey())
				if err != nil {
					return err
				}
				if _, ok := utxoTx.Indices[nextUTXO.GetUTXOKey()]; ok {
					utxoTx.PutUtxo(nextUTXO)
				}
			}

		}
		err = utxoCache.deleteUTXOFromDB(key)
		if err != nil {
			return err
		}
		err = utxoCache.deleteHardCore(utxo);
		if err != nil {
			return err
		}
	}
	return nil
}

func (utxoCache *UTXOCache) GetUtxo(utxoKey string) (*UTXO, error) {
	if utxoData, ok := utxoCache.utxo.Get(utxoKey); ok {
		utxo := utxoData.(*UTXO)
		return utxo, nil
	}
	return utxoCache.getUTXOFromDB(utxoKey)
}

func (utxoCache *UTXOCache) GetPreUtxo(thisUTXOKey string) (*UTXO, error) {
	utxo, err := utxoCache.GetUtxo(thisUTXOKey)
	if err != nil {
		return nil, err
	}
	if bytes.Equal(utxo.PrevUtxoKey, []byte{}) {
		return nil, nil
	}
	return utxoCache.GetUtxo(util.Bytes2str(utxo.PrevUtxoKey))
}

func (utxoCache *UTXOCache) getLastUTXOKey(pubKeyHash string) []byte {
	utxoInfo, err := utxoCache.getUTXOInfo(pubKeyHash)
	if err != nil {
		logger.Warn("getLastUTXOKey error:", err)
		return []byte{}
	}
	return utxoInfo.GetLastUtxoKey()
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
			logger.Error("GetUTXOTx:", err)
			break
		}
		utxoTx.Indices[utxoKey] = utxo
		utxoKey = util.Bytes2str(utxo.NextUtxoKey) //get previous utxo key
	}
	return &utxoTx
}

func (utxoCache *UTXOCache) putUTXOToDB(utxo *UTXO) error {
	utxoBytes, err := proto.Marshal(utxo.ToProto().(*utxopb.Utxo))
	if err != nil {
		return err
	}
	if err = utxoCache.db.Put(util.Str2bytes(utxo.GetUTXOKey()), utxoBytes); err != nil {
		logger.Error("put utxo to db failed！")
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
			logger.Error("Unmarshal utxo failed.")
			return nil, err
		}
		utxo.FromProto(utxoPb)
		utxoCache.utxo.Add(utxoKey, utxo)
		return utxo, nil
	}
	logger.Warn("get utxo from db failed！")
	return nil, err
}

func (utxoCache *UTXOCache) deleteUTXOFromDB(utxoKey string) error {
	if err := utxoCache.db.Del(util.Str2bytes(utxoKey)); err != nil {
		logger.Error("delete utxo from db failed.")
		return err
	}
	utxoCache.utxo.Remove(utxoKey)
	return nil
}

func (utxoCache *UTXOCache) putLastUTXOKey(pubkeyHash string, lastUTXOKey []byte) error {
	utxoInfo, err := utxoCache.getUTXOInfo(pubkeyHash)
	if err != nil {
		logger.Warn("putLastUTXOKey:", err)
	}
	utxoInfo.SetLastUtxoKey(lastUTXOKey)
	if err = utxoCache.putUTXOInfo(pubkeyHash, utxoInfo); err != nil {
		logger.Error("put last utxo key to db failed.")
		return err
	}
	return nil
}

func (utxoCache *UTXOCache) getUTXOInfo(pubkeyHash string) (*UTXOInfo, error) {
	utxoInfoData, ok := utxoCache.utxoInfo.Get(pubkeyHash)
	if ok {
		return utxoInfoData.(*UTXOInfo), nil
	}

	utxoInfo := NewUTXOInfo()
	rawBytes, err := utxoCache.db.Get(util.Str2bytes(pubkeyHash))
	if err != nil {
		logger.Warn("utxoInfo not found in db")
		return utxoInfo, err
	}

	utxoInfoPb := &utxopb.UtxoInfo{}
	err = proto.Unmarshal(rawBytes, utxoInfoPb)
	if err != nil {
		logger.Error("Unmarshal utxo info failed.")
		return utxoInfo, err
	}
	utxoInfo.FromProto(utxoInfoPb)

	utxoCache.utxoInfo.Add(pubkeyHash, utxoInfo)
	return utxoInfo, nil
}

func (utxoCache *UTXOCache) putUTXOInfo(pubkeyHash string, utxoInfo *UTXOInfo) error {
	utxoBytes, err := proto.Marshal(utxoInfo.ToProto().(*utxopb.UtxoInfo))
	if err != nil {
		return err
	}
	err = utxoCache.db.Put(util.Str2bytes(pubkeyHash), utxoBytes)
	if err != nil {
		logger.Error("put utxoInfo to db failed.")
		return err
	}
	utxoCache.utxoInfo.Add(pubkeyHash, utxoInfo)
	return nil
}

func (utxoCache *UTXOCache) deleteUTXOInfo(pubkeyHash string) error {
	if err := utxoCache.db.Del(util.Str2bytes(pubkeyHash)); err != nil {
		logger.Error("deleteUTXOInfo: delete utxoInfo from db failed.")
		return err
	}
	utxoCache.utxoInfo.Remove(pubkeyHash)
	return nil
}

func (utxoCache *UTXOCache) putCreateContractUTXOKey(pubkeyHash string, createContractUTXOKey []byte) error {
	if _, err := utxoCache.db.Get(util.Str2bytes(pubkeyHash)); err == nil {
		return errors.New("this utxoInfo already exists")
	}

	utxoInfo := NewUTXOInfo()
	utxoInfo.SetCreateContractUTXOKey(createContractUTXOKey)
	if err := utxoCache.putUTXOInfo(pubkeyHash, utxoInfo); err != nil {
		logger.Error("put utxoCreateContractKey to db failed.")
		return err
	}
	return nil
}

func (utxoCache *UTXOCache) GetUtxoCreateContract(pubKeyHash string) *UTXO {
	utxoInfo, err := utxoCache.getUTXOInfo(pubKeyHash)
	if err != nil || utxoInfo.GetCreateContractUTXOKey() == nil {
		return nil
	}
	utxo, err := utxoCache.GetUtxo(util.Bytes2str(utxoInfo.GetCreateContractUTXOKey()))
	if err != nil {
		logger.Error("Get UtxoCreateContract failed.")
		return nil
	}
	return utxo
}

func (utxoCache *UTXOCache) UpdateNextUTXO(nextUTXOKey []byte, preUTXOKey string) (*UTXO, error) {
	nextUTXO, err := utxoCache.GetUtxo(util.Bytes2str(nextUTXOKey))
	if err != nil {
		return nil, err
	}
	nextUTXO.PrevUtxoKey = util.Str2bytes(preUTXOKey)
	if err = utxoCache.putUTXOToDB(nextUTXO); err != nil {
		return nil, err
	}
	return nextUTXO, nil
}

func (utxoCache *UTXOCache) AddScStates(scStateKey, value string) error {
	err := utxoCache.db.Put(util.Str2bytes(scStateKey), util.Str2bytes(value))
	if err != nil {
		return err
	}
	utxoCache.scStateCache.Add(scStateKey, value)
	return nil
}

func (utxoCache *UTXOCache) GetScStates(scStateKey string) (string, error) {
	scStateData, ok := utxoCache.scStateCache.Get(scStateKey)
	if ok {
		return scStateData.(string), nil
	}

	valBytes, err := utxoCache.db.Get(util.Str2bytes(scStateKey))
	if err != nil {
		return "", err
	}
	return util.Bytes2str(valBytes), nil
}

func (utxoCache *UTXOCache) DelScStates(scStateKey string) error {
	err := utxoCache.db.Del(util.Str2bytes(scStateKey))
	if err != nil {
		return err
	}
	utxoCache.scStateCache.Remove(scStateKey)
	return nil
}

func (utxoCache *UTXOCache) AddStateLog(scStateLogKey string, stLog *stateLog.StateLog) error {
	utxoCache.stateLogCache.Add(scStateLogKey, stLog)

	err := utxoCache.db.Put(util.Str2bytes(scStateLogKey), stLog.SerializeStateLog())
	if err != nil {
		return err
	}
	return nil
}

func (utxoCache *UTXOCache) GetStateLog(scStateLogKey string) (*stateLog.StateLog, error) {
	stLogData, ok := utxoCache.stateLogCache.Get(scStateLogKey)
	if ok {
		return stLogData.(*stateLog.StateLog), nil
	}

	stLogBytes, err := utxoCache.db.Get(util.Str2bytes(scStateLogKey))
	if err != nil {
		return nil, err
	}
	return stateLog.DeserializeStateLog(stLogBytes), nil
}

func (utxoCache *UTXOCache) DelStateLog(scStateLogKey string) error {
	err := utxoCache.db.Del(util.Str2bytes(scStateLogKey))
	if err != nil {
		return err
	}
	utxoCache.stateLogCache.Remove(scStateLogKey)
	return nil
}

func (utxoCache *UTXOCache) GetUTXOsByAmountWithOutRemovedUTXOs(pubKeyHash account.PubKeyHash,amount *common.Amount, utxoTxRemove *UTXOTx) ([]*UTXO,error) {
	lastUtxokey := utxoCache.getLastUTXOKey(pubKeyHash.String())
	var utxoSlice []*UTXO
	utxoAmount := common.NewAmount(0)
	utxoKey := util.Bytes2str(lastUtxokey)

	for utxoKey != "" {
		utxo, err := utxoCache.GetUtxo(utxoKey)
		if err != nil {
			logger.Warn( err)
		}
		if utxo.UtxoType == UtxoCreateContract {
			continue
		}
		if utxoTxRemove != nil {
			if _, ok := utxoTxRemove.Indices[utxo.GetUTXOKey()]; ok {
				continue
			}
		}
		utxoAmount = utxoAmount.Add(utxo.Value)
		utxoSlice = append(utxoSlice, utxo)
		if utxoAmount.Cmp(amount) >= 0 {
			return utxoSlice, nil
		}

		utxoKey = util.Bytes2str(utxo.NextUtxoKey) //get previous utxo key
	}
	return nil, errors.New("transaction: insufficient balance")
}

func GetscStateKey(address, key string) string {
	return "scState" + address + key
}
func GetscStateLogKey(blockHash hash.Hash) string {
	return "scLog" + util.Bytes2str(blockHash)
}

func (utxoCache *UTXOCache) saveHardCore(utxo *UTXO) error {
	isContract, err := utxo.PubKeyHash.IsContract()
	if err != nil {
		return err
	}
	if isContract || utxo.Contract == "" {
		return nil
	}
	scStateKey, value := getScKeyValue(utxo.Contract, utxo.PubKeyHash)
	if err := utxoCache.db.Put(scStateKey, value); err != nil {
		return err
	}
	return nil
}

func (utxoCache *UTXOCache) deleteHardCore(utxo *UTXO) error {
	isContract, err := utxo.PubKeyHash.IsContract()
	if err != nil {
		return err
	}
	if isContract || utxo.Contract == "" {
		return nil
	}

	scStateKey, _ := getScKeyValue(utxo.Contract, utxo.PubKeyHash)
	if err := utxoCache.db.Del(scStateKey); err != nil {
		return err
	}
	return nil
}

func getScKeyValue(data string, pubkeyHash account.PubKeyHash) ([]byte, []byte){
	address := pubkeyHash.GenerateAddress().String()
	separator := strings.Index(data, ":")
	key := data[0:separator]
	value := data[separator+1:]
	scStateKey := GetscStateKey(address, key)
	return util.Str2bytes(scStateKey), util.Str2bytes(value)
}

