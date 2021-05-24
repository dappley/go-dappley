package utxo

import (
	"errors"
	"github.com/dappley/go-dappley/core/stateLog"
	"github.com/dappley/go-dappley/storage"
	"github.com/dappley/go-dappley/util"
	"github.com/stretchr/testify/assert"
	"testing"
)

var (
	address = "dGDrVKjCG3sdXtDUgWZ7Fp3Q97tLhqWivf"
	key     = "Account1"
	value   = "99"
	blkHash = []byte{7,7,7,7}
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

	cache.stateLogCache.Add(GetscStateLogKey([]byte{8,8,8,8}) , stLog)
	getLog, err = cache.GetStateLog(GetscStateLogKey([]byte{8,8,8,8}))
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
	assert.Equal(t, errors.New("key is invalid"), err)

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
	val,err:=cache.GetScStates(GetscStateKey(address, key))
	assert.Nil(t,err)
	assert.Equal(t, value,val)

	cache.scStateCache.Add(GetscStateKey(address, "Account2"), value)
	val,err=cache.GetScStates(GetscStateKey(address, "Account2"))
	assert.Nil(t,err)
	assert.Equal(t, value,val)
}

func TestUTXO_DelScStates(t *testing.T) {
	db := storage.NewRamStorage()
	defer db.Close()
	cache := NewUTXOCache(db)
	cache.scStateCache.Add(GetscStateKey(address, key), value)
	assert.Nil(t, cache.db.Put(util.Str2bytes(GetscStateKey(address, key)), util.Str2bytes(value)))

	assert.Nil(t,cache.DelScStates(GetscStateKey(address, key)))

	_, ok := cache.scStateCache.Get(GetscStateKey(address, key))
	assert.Equal(t,false,ok)

	_, err := cache.db.Get(util.Str2bytes(GetscStateKey(address, key)))
	assert.Equal(t, errors.New("key is invalid"), err)
}
