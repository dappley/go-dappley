package utxo

import (
	"errors"
	"github.com/dappley/go-dappley/common"
	"github.com/dappley/go-dappley/core/account"
	"github.com/dappley/go-dappley/core/stateLog"
	"github.com/dappley/go-dappley/core/transactionbase"
	utxopb "github.com/dappley/go-dappley/core/utxo/pb"
	"github.com/dappley/go-dappley/storage"
	"github.com/dappley/go-dappley/util"
	"github.com/golang/protobuf/proto"
	lru "github.com/hashicorp/golang-lru"
	"github.com/stretchr/testify/assert"
	"testing"
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
	assert.Equal(t, errors.New("key is invalid"), err)
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
	err := cache.db.Put(util.Str2bytes(utxo.GetUTXOKey()), utxoBytes)
	assert.Nil(t, err)

	result, err := cache.getUTXOFromDB(utxo.GetUTXOKey())
	assert.Nil(t, err)
	assert.Equal(t, utxo, result)
	cacheUtxo, _ := cache.utxo.Get(utxo.GetUTXOKey())
	assert.Equal(t, utxo, cacheUtxo)

	result, err = cache.getUTXOFromDB("invalid key")
	assert.Nil(t, result)
	assert.Equal(t, errors.New("key is invalid"), err)
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
	assert.Equal(t, errors.New("key is invalid"), err)

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

func TestUTXOCache_GetPreUtxo(t *testing.T) {
	db := storage.NewRamStorage()
	defer db.Close()
	cache := NewUTXOCache(db)

	utxo1 := &UTXO{
		TXOutput: transactionbase.TXOutput{
			Value:      common.NewAmount(10),
			PubKeyHash: []byte{0x5a, 0xb1, 0x34, 0x4c, 0x17, 0x67, 0x4c, 0x18, 0xd1, 0xa2, 0xdc, 0xea, 0x9f, 0x17, 0x16, 0xe0, 0x49, 0xf4, 0xa0, 0x5e, 0x6c},
			Contract:   "contract",
		},
		Txid:        []byte{0x74, 0x65, 0x73, 0x74},
		TxIndex:     0,
		UtxoType:    UtxoNormal,
		PrevUtxoKey: nil,
		NextUtxoKey: []byte{0x74, 0x65, 0x73, 0x74, 0x5f, 0x31},
	}

	utxo2 := &UTXO{
		TXOutput: transactionbase.TXOutput{
			Value:      common.NewAmount(10),
			PubKeyHash: []byte{0x5a, 0xb1, 0x34, 0x4c, 0x17, 0x67, 0x4c, 0x18, 0xd1, 0xa2, 0xdc, 0xea, 0x9f, 0x17, 0x16, 0xe0, 0x49, 0xf4, 0xa0, 0x5e, 0x6c},
			Contract:   "contract",
		},
		Txid:        []byte{0x74, 0x65, 0x73, 0x74},
		TxIndex:     1,
		UtxoType:    UtxoNormal,
		PrevUtxoKey: []byte{0x74, 0x65, 0x73, 0x74, 0x5f, 0x30},
		NextUtxoKey: nil,
	}

	cache.putUTXOToDB(utxo1)
	cache.putUTXOToDB(utxo2)
	result, err := cache.GetPreUtxo("invalid")
	assert.Nil(t, result)
	assert.Equal(t, errors.New("key is invalid"), err)

	result, err = cache.GetPreUtxo(utxo1.GetUTXOKey())
	assert.Nil(t, result)
	assert.Nil(t, err)

	result, err = cache.GetPreUtxo(utxo2.GetUTXOKey())
	assert.Equal(t, utxo1, result)
	assert.Nil(t, err)
}

func TestUTXOCache_putUTXOInfo(t *testing.T) {
	db := storage.NewRamStorage()
	defer db.Close()
	cache := NewUTXOCache(db)

	utxoInfo := &UTXOInfo{
		lastUTXOKey:           []byte{0x74, 0x65, 0x73, 0x74, 0x5f, 0x30},
		createContractUTXOKey: []byte{0x74, 0x65, 0x73, 0x74, 0x5f, 0x31},
	}

	pubKeyHashString := "5ab1344c17674c18d1a2dcea9f1716e049f4a05e6c"
	err := cache.putUTXOInfo(pubKeyHashString, utxoInfo)
	assert.Nil(t, err)
	cacheUtxoInfo, _ := cache.utxoInfo.Get(pubKeyHashString)
	assert.Equal(t, utxoInfo, cacheUtxoInfo)

	dbUtxoInfoBytes, _ := cache.db.Get(util.Str2bytes(pubKeyHashString))
	expectedUtxoInfoBytes, _ := proto.Marshal(utxoInfo.ToProto().(*utxopb.UtxoInfo))
	assert.Equal(t, expectedUtxoInfoBytes, dbUtxoInfoBytes)
}

func TestUTXOCache_getUTXOInfo(t *testing.T) {
	db := storage.NewRamStorage()
	defer db.Close()
	cache := NewUTXOCache(db)

	utxoInfo := &UTXOInfo{
		lastUTXOKey:           []byte{0x74, 0x65, 0x73, 0x74, 0x5f, 0x30},
		createContractUTXOKey: []byte{0x74, 0x65, 0x73, 0x74, 0x5f, 0x31},
	}
	pubKeyHash := "5ab1344c17674c18d1a2dcea9f1716e049f4a05e6c"

	utxoInfoBytes, _ := proto.Marshal(utxoInfo.ToProto().(*utxopb.UtxoInfo))
	err := cache.db.Put(util.Str2bytes(pubKeyHash), utxoInfoBytes)
	assert.Nil(t, err)

	result, err := cache.getUTXOInfo(pubKeyHash)
	assert.Nil(t, err)
	assert.Equal(t, utxoInfo, result)
	cacheUtxo, _ := cache.utxoInfo.Get(pubKeyHash)
	assert.Equal(t, utxoInfo, cacheUtxo)

	result, err = cache.getUTXOInfo("invalid key")
	assert.Equal(t, &UTXOInfo{lastUTXOKey: []uint8{}, createContractUTXOKey: []uint8{}}, result)
	assert.Equal(t, errors.New("key is invalid"), err)
}

func TestUTXOCache_deleteUTXOInfo(t *testing.T) {
	db := storage.NewRamStorage()
	defer db.Close()
	cache := NewUTXOCache(db)

	utxoInfo := &UTXOInfo{
		lastUTXOKey:           []byte{0x74, 0x65, 0x73, 0x74, 0x5f, 0x30},
		createContractUTXOKey: []byte{0x74, 0x65, 0x73, 0x74, 0x5f, 0x31},
	}
	pubKeyHash := "5ab1344c17674c18d1a2dcea9f1716e049f4a05e6c"

	cache.putUTXOInfo(pubKeyHash, utxoInfo)
	result, err := cache.getUTXOInfo(pubKeyHash)
	assert.Equal(t, utxoInfo, result)
	assert.Nil(t, err)
	err = cache.deleteUTXOInfo(pubKeyHash)
	assert.Nil(t, err)
	result, err = cache.getUTXOInfo(pubKeyHash)
	assert.Equal(t, &UTXOInfo{lastUTXOKey: []uint8{}, createContractUTXOKey: []uint8{}}, result)
	assert.Equal(t, errors.New("key is invalid"), err)
}

func TestUTXOCache_deleteUTXOFromDB(t *testing.T) {
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

	cache.putUTXOToDB(utxo)
	result, err := cache.getUTXOFromDB(utxo.GetUTXOKey())
	assert.Equal(t, utxo, result)
	assert.Nil(t, err)
	err = cache.deleteUTXOFromDB(utxo.GetUTXOKey())
	assert.Nil(t, err)
	result, err = cache.getUTXOFromDB(utxo.GetUTXOKey())
	assert.Nil(t, result)
	assert.Equal(t, errors.New("key is invalid"), err)
}

func TestUTXOCache_putLastUTXOKey(t *testing.T) {
	db := storage.NewRamStorage()
	defer db.Close()
	cache := NewUTXOCache(db)

	pubKeyHash := "5ab1344c17674c18d1a2dcea9f1716e049f4a05e6c"

	expected1 := &UTXOInfo{
		lastUTXOKey:           []byte{0x74, 0x65, 0x73, 0x74, 0x5f, 0x30},
		createContractUTXOKey: []byte{},
	}
	expected2 := &UTXOInfo{
		lastUTXOKey:           []byte{0x74, 0x65, 0x73, 0x74, 0x5f, 0x31},
		createContractUTXOKey: []byte{},
	}

	// use putLastUTXOKey on UTXOInfo that isn't in cache yet
	err := cache.putLastUTXOKey(pubKeyHash, []byte{0x74, 0x65, 0x73, 0x74, 0x5f, 0x30})
	assert.Nil(t, err)
	result, ok := cache.utxoInfo.Get(pubKeyHash)
	assert.True(t, ok)
	assert.Equal(t, expected1, result)
	// update the existing UTXOInfo
	err = cache.putLastUTXOKey(pubKeyHash, []byte{0x74, 0x65, 0x73, 0x74, 0x5f, 0x31})
	assert.Nil(t, err)
	result, ok = cache.utxoInfo.Get(pubKeyHash)
	assert.True(t, ok)
	assert.Equal(t, expected2, result)
}

func TestUTXOCache_getLastUTXOKey(t *testing.T) {
	db := storage.NewRamStorage()
	defer db.Close()
	cache := NewUTXOCache(db)

	pubKeyHash := "5ab1344c17674c18d1a2dcea9f1716e049f4a05e6c"

	result := cache.getLastUTXOKey(pubKeyHash)
	assert.Equal(t, []byte{}, result)

	err := cache.putLastUTXOKey(pubKeyHash, []byte{0x74, 0x65, 0x73, 0x74, 0x5f, 0x31})
	assert.Nil(t, err)
	result = cache.getLastUTXOKey(pubKeyHash)
	assert.Equal(t, []byte{0x74, 0x65, 0x73, 0x74, 0x5f, 0x31}, result)
}

func TestUTXOCache_IsLastUtxoKeyExist(t *testing.T) {
	db := storage.NewRamStorage()
	defer db.Close()
	cache := NewUTXOCache(db)

	pubKeyHash := "5ab1344c17674c18d1a2dcea9f1716e049f4a05e6c"
	// lastUTXOKey not set yet
	assert.False(t, cache.IsLastUtxoKeyExist(pubKeyHash))
	// set lastUTXOKey
	err := cache.putLastUTXOKey(pubKeyHash, []byte{0x74, 0x65, 0x73, 0x74, 0x5f, 0x30})
	assert.Nil(t, err)
	assert.True(t, cache.IsLastUtxoKeyExist(pubKeyHash))
	// reset lastUTXOKey to empty
	err = cache.putLastUTXOKey(pubKeyHash, []byte{})
	assert.Nil(t, err)
	assert.False(t, cache.IsLastUtxoKeyExist(pubKeyHash))
}

func TestUTXOCache_putCreateContractUTXOKey(t *testing.T) {
	db := storage.NewRamStorage()
	defer db.Close()
	cache := NewUTXOCache(db)

	pubKeyHash := "5ab1344c17674c18d1a2dcea9f1716e049f4a05e6c"
	err := cache.putCreateContractUTXOKey(pubKeyHash, []byte{0x74, 0x65, 0x73, 0x74, 0x5f, 0x31})
	assert.Nil(t, err)

	// attempt to put to existing UTXOInfo
	err = cache.putCreateContractUTXOKey(pubKeyHash, []byte{0x74, 0x65, 0x73, 0x74, 0x5f, 0x32})
	assert.Equal(t, errors.New("this utxoInfo already exists"), err)
}

func TestUTXOCache_GetUtxoCreateContract(t *testing.T) {
	db := storage.NewRamStorage()
	defer db.Close()
	cache := NewUTXOCache(db)

	pubKeyHash := "5ab1344c17674c18d1a2dcea9f1716e049f4a05e6c"
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

	// utxoInfo and utxo are not in cache
	result := cache.GetUtxoCreateContract(pubKeyHash)
	assert.Nil(t, result)
	// utxoInfo in cache, but utxo is not
	err := cache.putCreateContractUTXOKey(pubKeyHash, []byte{0x74, 0x65, 0x73, 0x74, 0x5f, 0x31})
	assert.Nil(t, err)
	assert.Nil(t, cache.GetUtxoCreateContract(pubKeyHash))

	// utxoInfo and utxo in cache
	err = cache.putUTXOToDB(utxo)
	assert.Nil(t, err)
	assert.Equal(t, utxo, cache.GetUtxoCreateContract(pubKeyHash))
}

func TestUTXOCache_GetUTXOTx(t *testing.T) {
	db := storage.NewRamStorage()
	defer db.Close()
	cache := NewUTXOCache(db)

	pubKeyHash := "5ab1344c17674c18d1a2dcea9f1716e049f4a05e6c"
	utxo1 := &UTXO{
		TXOutput: transactionbase.TXOutput{
			Value:      common.NewAmount(10),
			PubKeyHash: []byte{0x5a, 0xb1, 0x34, 0x4c, 0x17, 0x67, 0x4c, 0x18, 0xd1, 0xa2, 0xdc, 0xea, 0x9f, 0x17, 0x16, 0xe0, 0x49, 0xf4, 0xa0, 0x5e, 0x6c},
			Contract:   "contract",
		},
		Txid:        []byte{0x74, 0x65, 0x73, 0x74},
		TxIndex:     0,
		UtxoType:    UtxoNormal,
		PrevUtxoKey: nil,
		NextUtxoKey: []byte{0x74, 0x65, 0x73, 0x74, 0x5f, 0x31},
	}
	utxo2 := &UTXO{
		TXOutput: transactionbase.TXOutput{
			Value:      common.NewAmount(10),
			PubKeyHash: []byte{0x5a, 0xb1, 0x34, 0x4c, 0x17, 0x67, 0x4c, 0x18, 0xd1, 0xa2, 0xdc, 0xea, 0x9f, 0x17, 0x16, 0xe0, 0x49, 0xf4, 0xa0, 0x5e, 0x6d},
			Contract:   "contract",
		},
		Txid:        []byte{0x74, 0x65, 0x73, 0x74},
		TxIndex:     1,
		UtxoType:    UtxoNormal,
		PrevUtxoKey: []byte{0x74, 0x65, 0x73, 0x74, 0x5f, 0x30},
		NextUtxoKey: nil,
	}
	expected := &UTXOTx{map[string]*UTXO{"test_0": utxo1, "test_1": utxo2}}

	// no stored LastUTXOKey for pubKeyHash
	result := cache.GetUTXOTx(utxo1.PubKeyHash)
	assert.Equal(t, &UTXOTx{map[string]*UTXO{}}, result)

	// no utxo in db corresponding to utxo key
	err := cache.putLastUTXOKey(pubKeyHash, []byte{0x74, 0x65, 0x73, 0x74, 0x5f, 0x30})
	assert.Nil(t, err)
	result = cache.GetUTXOTx(utxo1.PubKeyHash)
	assert.Equal(t, &UTXOTx{map[string]*UTXO{}}, result)

	cache.putUTXOToDB(utxo1)
	cache.putUTXOToDB(utxo2)
	result = cache.GetUTXOTx(utxo1.PubKeyHash)
	assert.Equal(t, expected, result)
}

func TestUTXOCache_UpdateNextUTXO(t *testing.T) {
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

	// utxo not in db
	result, err := cache.UpdateNextUTXO(util.Str2bytes(utxo.GetUTXOKey()), "test_0")
	assert.Nil(t, result)
	assert.Equal(t, errors.New("key is invalid"), err)

	// successful update
	err = cache.putUTXOToDB(utxo)
	assert.Nil(t, err)
	result, err = cache.UpdateNextUTXO(util.Str2bytes(utxo.GetUTXOKey()), "test_0")
	assert.Nil(t, err)
	assert.Equal(t, utxo, result)
	assert.Equal(t, []byte{0x74, 0x65, 0x73, 0x74, 0x5f, 0x30}, utxo.PrevUtxoKey)
}

func TestUTXOCache_GetUTXOsByAmountWithOutRemovedUTXOs(t *testing.T) {
	utxo1 := &UTXO{
		TXOutput: transactionbase.TXOutput{
			Value:      common.NewAmount(10),
			PubKeyHash: []byte{0x5a, 0xb1, 0x34, 0x4c, 0x17, 0x67, 0x4c, 0x18, 0xd1, 0xa2, 0xdc, 0xea, 0x9f, 0x17, 0x16, 0xe0, 0x49, 0xf4, 0xa0, 0x5e, 0x6c},
			Contract:   "contract",
		},
		Txid:        []byte{0x74, 0x65, 0x73, 0x74},
		TxIndex:     0,
		UtxoType:    UtxoNormal,
		PrevUtxoKey: []byte{},
		NextUtxoKey: []byte{0x74, 0x65, 0x73, 0x74, 0x5f, 0x31},
	}
	utxo2 := &UTXO{
		TXOutput: transactionbase.TXOutput{
			Value:      common.NewAmount(10),
			PubKeyHash: []byte{0x5a, 0xb1, 0x34, 0x4c, 0x17, 0x67, 0x4c, 0x18, 0xd1, 0xa2, 0xdc, 0xea, 0x9f, 0x17, 0x16, 0xe0, 0x49, 0xf4, 0xa0, 0x5e, 0x6d},
			Contract:   "contract",
		},
		Txid:        []byte{0x74, 0x65, 0x73, 0x74},
		TxIndex:     1,
		UtxoType:    UtxoNormal,
		PrevUtxoKey: []byte{0x74, 0x65, 0x73, 0x74, 0x5f, 0x30},
		NextUtxoKey: []byte{},
	}
	utxo3 := &UTXO{
		TXOutput: transactionbase.TXOutput{
			Value:      common.NewAmount(10),
			PubKeyHash: []byte{0x5a, 0xb1, 0x34, 0x4c, 0x17, 0x67, 0x4c, 0x18, 0xd1, 0xa2, 0xdc, 0xea, 0x9f, 0x17, 0x16, 0xe0, 0x49, 0xf4, 0xa0, 0x5e, 0x6e},
			Contract:   "contract",
		},
		Txid:        []byte{0x74, 0x65, 0x73, 0x74},
		TxIndex:     2,
		UtxoType:    UtxoNormal,
		PrevUtxoKey: []byte{},
		NextUtxoKey: []byte{},
	}

	tests := []struct {
		name           string
		pubKeyHash     account.PubKeyHash
		amount         *common.Amount
		utxoTxRemove   *UTXOTx
		expectedResult []*UTXO
		expectedErr    error
	}{
		{
			name:           "successful operation (nil utxoTxRemove)",
			pubKeyHash:     utxo1.PubKeyHash,
			amount:         common.NewAmount(20),
			utxoTxRemove:   nil,
			expectedResult: []*UTXO{utxo1, utxo2},
			expectedErr:    nil,
		},
		{
			name:           "successful operation (non-nil utxoTxRemove)",
			pubKeyHash:     utxo1.PubKeyHash,
			amount:         common.NewAmount(20),
			utxoTxRemove:   &UTXOTx{map[string]*UTXO{"test_2": utxo3}},
			expectedResult: []*UTXO{utxo1, utxo2},
			expectedErr:    nil,
		},
		{
			name:           "successful operation (small amount)",
			pubKeyHash:     utxo1.PubKeyHash,
			amount:         common.NewAmount(1),
			utxoTxRemove:   nil,
			expectedResult: []*UTXO{utxo1},
			expectedErr:    nil,
		},
		{
			name:           "amount too high",
			pubKeyHash:     utxo1.PubKeyHash,
			amount:         common.NewAmount(21),
			utxoTxRemove:   nil,
			expectedResult: nil,
			expectedErr:    errors.New("transaction: insufficient balance"),
		},
	}

	db := storage.NewRamStorage()
	defer db.Close()
	cache := NewUTXOCache(db)

	err := cache.putUTXOToDB(utxo1)
	assert.Nil(t, err)
	err = cache.putUTXOToDB(utxo2)
	assert.Nil(t, err)
	err = cache.putLastUTXOKey(utxo1.PubKeyHash.String(), util.Str2bytes(utxo1.GetUTXOKey()))
	assert.Nil(t, err)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := cache.GetUTXOsByAmountWithOutRemovedUTXOs(tt.pubKeyHash, tt.amount, tt.utxoTxRemove)
			if tt.expectedErr != nil {
				assert.Nil(t, result)
				assert.Equal(t, tt.expectedErr, err)
			} else {
				assert.Equal(t, tt.expectedResult, result)
				assert.Nil(t, err)
			}
		})
	}
}
