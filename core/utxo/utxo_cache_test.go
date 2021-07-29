package utxo

import (
	"testing"

	"github.com/dappley/go-dappley/common"
	"github.com/dappley/go-dappley/core/stateLog"
	"github.com/dappley/go-dappley/core/transactionbase"
	utxopb "github.com/dappley/go-dappley/core/utxo/pb"
	errorValues "github.com/dappley/go-dappley/errors"
	"github.com/dappley/go-dappley/storage"
	"github.com/dappley/go-dappley/util"
	"github.com/golang/protobuf/proto"
	lru "github.com/hashicorp/golang-lru"
	"github.com/stretchr/testify/assert"
)

var (
	address = "dGDrVKjCG3sdXtDUgWZ7Fp3Q97tLhqWivf"
	key     = "Account1"
	value   = "99"
	blkHash = []byte{7, 7, 7, 7}
)

func TestUTXO_AddStateLog(t *testing.T) {
	db := storage.NewRamStorage()
	defer db.Close()
	cache := NewUTXOCache(db)

	stLog := stateLog.NewStateLog()
	stLog.Log = map[string]map[string]string{address: {key: value}}
	assert.Nil(t, cache.AddStateLog(GetscStateLogKey(blkHash), stLog))

	stLogData, _ := cache.stateLogCache.Get(GetscStateLogKey(blkHash))
	assert.Equal(t, stLog, stLogData.(*stateLog.StateLog))

	stLogBytes, err := cache.db.Get(util.Str2bytes(GetscStateLogKey(blkHash)))
	assert.Nil(t, err)
	assert.Equal(t, stLog, stateLog.DeserializeStateLog(stLogBytes))
}

func TestUTXO_GetStateLog(t *testing.T) {
	db := storage.NewRamStorage()
	defer db.Close()
	cache := NewUTXOCache(db)

	stLog := stateLog.NewStateLog()
	stLog.Log = map[string]map[string]string{address: {key: value}}
	assert.Nil(t, db.Put(util.Str2bytes(GetscStateLogKey(blkHash)), stLog.SerializeStateLog()))

	getLog, err := cache.GetStateLog(GetscStateLogKey(blkHash))
	assert.Nil(t, err)
	assert.Equal(t, stLog, getLog)

	cache.stateLogCache.Add(GetscStateLogKey([]byte{8, 8, 8, 8}), stLog)
	getLog, err = cache.GetStateLog(GetscStateLogKey([]byte{8, 8, 8, 8}))
	assert.Nil(t, err)
	assert.Equal(t, stLog, getLog)
}

func TestUTXO_DelStateLog(t *testing.T) {
	db := storage.NewRamStorage()
	defer db.Close()
	cache := NewUTXOCache(db)

	stLog := stateLog.NewStateLog()
	stLog.Log = map[string]map[string]string{address: {key: value}}
	cache.stateLogCache.Add(GetscStateLogKey(blkHash), stLog)
	assert.Nil(t, db.Put(util.Str2bytes(GetscStateLogKey(blkHash)), stLog.SerializeStateLog()))

	assert.Nil(t, cache.DelStateLog(GetscStateLogKey(blkHash)))

	_, ok := cache.stateLogCache.Get(GetscStateLogKey(blkHash))
	assert.Equal(t, false, ok)
	_, err := cache.db.Get(util.Str2bytes(GetscStateLogKey(blkHash)))
	assert.Equal(t, errorValues.InvalidKey, err)

}

func TestUTXO_AddScStates(t *testing.T) {
	db := storage.NewRamStorage()
	defer db.Close()
	cache := NewUTXOCache(db)

	assert.Nil(t, cache.AddScStates(GetscStateKey(address, key), value))

	scStateData, _ := cache.scStateCache.Get(GetscStateKey(address, key))
	assert.Equal(t, value, scStateData.(string))

	valBytes, _ := cache.db.Get(util.Str2bytes(GetscStateKey(address, key)))
	assert.Equal(t, value, util.Bytes2str(valBytes))

}

func TestUTXO_GetScStates(t *testing.T) {
	db := storage.NewRamStorage()
	defer db.Close()
	cache := NewUTXOCache(db)
	assert.Nil(t, cache.db.Put(util.Str2bytes(GetscStateKey(address, key)), util.Str2bytes(value)))
	val, err := cache.GetScStates(GetscStateKey(address, key))
	assert.Nil(t, err)
	assert.Equal(t, value, val)

	cache.scStateCache.Add(GetscStateKey(address, "Account2"), value)
	val, err = cache.GetScStates(GetscStateKey(address, "Account2"))
	assert.Nil(t, err)
	assert.Equal(t, value, val)
}

func TestUTXO_DelScStates(t *testing.T) {
	db := storage.NewRamStorage()
	defer db.Close()
	cache := NewUTXOCache(db)
	cache.scStateCache.Add(GetscStateKey(address, key), value)
	assert.Nil(t, cache.db.Put(util.Str2bytes(GetscStateKey(address, key)), util.Str2bytes(value)))

	assert.Nil(t, cache.DelScStates(GetscStateKey(address, key)))

	_, ok := cache.scStateCache.Get(GetscStateKey(address, key))
	assert.Equal(t, false, ok)

	_, err := cache.db.Get(util.Str2bytes(GetscStateKey(address, key)))
	assert.Equal(t, errorValues.InvalidKey, err)
}

func TestNewScStateCache(t *testing.T) {
	scStateCache := NewScStateCache()
	expectedMemberCache, _ := lru.New(ScStateCacheLRUCacheLimit)

	assert.Equal(t, expectedMemberCache, scStateCache.stateLogCache)
	assert.Equal(t, expectedMemberCache, scStateCache.scStateCache)
}

func TestUTXOCache_putUTXOToDB(t *testing.T) {
	db := storage.NewRamStorage()
	defer db.Close()
	cache := NewUTXOCache(db)

	utxo := &UTXO{
		TXOutput: transactionbase.TXOutput{
			Value:      common.NewAmount(10),
			PubKeyHash: []byte{0x5a, 0xb1, 0x34, 0x4c, 0x17, 0x67, 0x4c, 0x18, 0xd1, 0xa2, 0xdc, 0xea, 0x9f, 0x17, 0x16, 0xe0, 0x49, 0xf4, 0xa0, 0x5e, 0x6c},
			Contract:   "contract",
		},
		Txid:     []byte{0x74, 0x65, 0x73, 0x74},
		TxIndex:  1,
		UtxoType: UtxoNormal,
	}

	err := cache.putUTXOToDB(utxo)
	assert.Nil(t, err)

	cacheUtxo, _ := cache.utxo.Get(utxo.GetUTXOKey())
	assert.Equal(t, utxo, cacheUtxo)
	dbUtxoBytes, _ := cache.db.Get(util.Str2bytes(utxo.GetUTXOKey()))
	expectedUtxoBytes, _ := proto.Marshal(utxo.ToProto().(*utxopb.Utxo))
	assert.Equal(t, expectedUtxoBytes, dbUtxoBytes)
}

func TestUTXOCache_getUTXOFromDB(t *testing.T) {
	db := storage.NewRamStorage()
	defer db.Close()
	cache := NewUTXOCache(db)

	utxo := &UTXO{
		TXOutput: transactionbase.TXOutput{
			Value:      common.NewAmount(10),
			PubKeyHash: []byte{0x5a, 0xb1, 0x34, 0x4c, 0x17, 0x67, 0x4c, 0x18, 0xd1, 0xa2, 0xdc, 0xea, 0x9f, 0x17, 0x16, 0xe0, 0x49, 0xf4, 0xa0, 0x5e, 0x6c},
			Contract:   "contract",
		},
		Txid:     []byte{0x74, 0x65, 0x73, 0x74},
		TxIndex:  1,
		UtxoType: UtxoNormal,
	}
	utxoBytes, _ := proto.Marshal(utxo.ToProto().(*utxopb.Utxo))
	assert.Nil(t, cache.db.Put(util.Str2bytes(utxo.GetUTXOKey()), utxoBytes))

	result, err := cache.getUTXOFromDB(utxo.GetUTXOKey())
	assert.Nil(t, err)
	assert.Equal(t, utxo, result)
	cacheUtxo, _ := cache.utxo.Get(utxo.GetUTXOKey())
	assert.Equal(t, utxo, cacheUtxo)

	result, err = cache.getUTXOFromDB("invalid key")
	assert.Nil(t, result)
	assert.Equal(t, errorValues.InvalidKey, err)
}

func TestUTXOCache_GetUtxo(t *testing.T) {
	db := storage.NewRamStorage()
	defer db.Close()
	cache := NewUTXOCache(db)

	utxo1 := &UTXO{
		TXOutput: transactionbase.TXOutput{
			Value:      common.NewAmount(10),
			PubKeyHash: []byte{0x5a, 0xb1, 0x34, 0x4c, 0x17, 0x67, 0x4c, 0x18, 0xd1, 0xa2, 0xdc, 0xea, 0x9f, 0x17, 0x16, 0xe0, 0x49, 0xf4, 0xa0, 0x5e, 0x6c},
			Contract:   "contract",
		},
		Txid:     []byte{0x74, 0x65, 0x73, 0x74},
		TxIndex:  0,
		UtxoType: UtxoNormal,
	}
	utxo2 := &UTXO{
		TXOutput: transactionbase.TXOutput{
			Value:      common.NewAmount(15),
			PubKeyHash: []byte{0x5a, 0xb2, 0x34, 0x4c, 0x17, 0x67, 0x4c, 0x18, 0xd1, 0xa2, 0xdc, 0xea, 0x9f, 0x17, 0x16, 0xe0, 0x49, 0xf4, 0xa0, 0x5e, 0x6c},
			Contract:   "contract",
		},
		Txid:     []byte{0x75, 0x66, 0x74, 0x75},
		TxIndex:  1,
		UtxoType: UtxoNormal,
	}

	result, err := cache.GetUtxo(utxo1.GetUTXOKey())
	assert.Nil(t, result)
	assert.Equal(t, errorValues.InvalidKey, err)

	utxoBytes, err := proto.Marshal(utxo1.ToProto().(*utxopb.Utxo))
	assert.Nil(t, err)
	cache.utxo.Add(utxo1.GetUTXOKey(), utxo1)
	result, err = cache.GetUtxo(utxo1.GetUTXOKey())
	assert.Nil(t, err)
	assert.Equal(t, utxo1, result)

	utxoBytes, err = proto.Marshal(utxo2.ToProto().(*utxopb.Utxo))
	assert.Nil(t, err)
	assert.Nil(t, cache.db.Put(util.Str2bytes(utxo2.GetUTXOKey()), utxoBytes))
	result, err = cache.GetUtxo(utxo2.GetUTXOKey())
	assert.Nil(t, err)
	assert.Equal(t, utxo2, result)
}
