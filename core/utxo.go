package core

import (
	"bytes"
	"encoding/gob"
	"log"
	"github.com/dappley/go-dappley/storage"
)

var utxoKey = []byte("2")

// An Transactiondb_cache is a max-heap of Transactions.
type UTXOCache []Transaction

func (db_cache UTXOCache) Len() int { return len(db_cache) }
//Compares Transaction Tips
func (db_cache UTXOCache) Less(i, j int) bool { return db_cache[i].Tip > db_cache[j].Tip }
func (db_cache UTXOCache) Swap(i, j int)      { db_cache[i], db_cache[j] = db_cache[j], db_cache[i] }

func (db_cache *UTXOCache) Push(x interface{}) {
	// Push and Pop use pointer receivers because they modify the slice's length,
	// not just its contents.
	*db_cache = append(*db_cache, x.(Transaction))
}

func (db_cache *UTXOCache) Pop() interface{} {
	old := *db_cache
	length := len(old)
	last := old[length-1]
	*db_cache = old[0 : length-1]
	return last
}

func (db_cache *UTXOCache) GetDatabaseUTXO(db storage.LevelDB) []byte {
	utxoArrayOfBytes, err := db.Get(utxoKey)
	if err != nil {
		log.Panic(err)
	}
	return utxoArrayOfBytes
}

func (db_cache *UTXOCache) Deserialize(d []byte) *UTXOCache {
	var txndb_cache UTXOCache
	decoder := gob.NewDecoder(bytes.NewReader(d))
	err := decoder.Decode(&txndb_cache)
	if err != nil {
		log.Panic(err)
	}
	return db_cache
}

