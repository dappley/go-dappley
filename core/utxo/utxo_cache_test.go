package utxo

import (
	"github.com/dappley/go-dappley/core/stateLog"
	"github.com/dappley/go-dappley/storage"
	"github.com/dappley/go-dappley/util"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestUTXO_GetStateLog(t *testing.T) {
	db := storage.NewRamStorage()
	defer db.Close()
	cache := NewUTXOCache(db)

	stLog := stateLog.NewStateLog()
	stLog.Log = map[string]map[string]string{"dGDrVKjCG3sdXtDUgWZ7Fp3Q97tLhqWivf": {"Account1": "99"}}
	assert.Nil(t, db.Put(util.Str2bytes(ScStateLogKey+"blkHash"), stLog.SerializeStateLog()))

	getLog, err := cache.GetStateLog("blkHash")
	assert.Nil(t, err)
	assert.Equal(t, stLog, getLog)

	cache.stateLogCache.Add(ScStateLogKey+"blkHash2", stLog)
	getLog, err = cache.GetStateLog("blkHash2")
	assert.Nil(t, err)
	assert.Equal(t, stLog, getLog)
}
