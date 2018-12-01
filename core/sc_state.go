package core

import (
	"bytes"
	"encoding/gob"
	"github.com/dappley/go-dappley/storage"
	logger "github.com/sirupsen/logrus"
	"sync"
)

type ScState struct{
	states map[string]map[string]string
	mutex *sync.RWMutex
}

const (
	scStateMapKey = "scState"
	scRewardKey = "scStateRewardKey"
)

func NewScState() *ScState{
	return &ScState{make(map[string]map[string]string), &sync.RWMutex{}}
}

func deserializeScState(d []byte) map[string]map[string]string {
	scState := make(map[string]map[string]string)
	decoder := gob.NewDecoder(bytes.NewReader(d))
	err := decoder.Decode(&scState)
	if err != nil {
		logger.Panic("Failed to deserialize utxo states:", err)
	}
	return scState
}

func (ss *ScState) serialize() []byte {

	var encoded bytes.Buffer
	enc := gob.NewEncoder(&encoded)
	err := enc.Encode(ss.states)
	if err != nil {
		logger.Panic("Serialize utxo states failed. Err:", err)
	}
	return encoded.Bytes()
}

//Get gets an item in scStorage
func (ss *ScState) Get(pubKeyHash, key string) string{
	ss.mutex.Lock()
	defer ss.mutex.Unlock()
	if len(ss.states[pubKeyHash]) == 0 {
		return ""
	}
	return ss.states[pubKeyHash][key]
}

//Set sets an item in scStorage
func (ss *ScState) Set(pubKeyHash, key, value string) int{
	ss.mutex.Lock()
	defer ss.mutex.Unlock()

	if len(ss.states[pubKeyHash]) == 0 {
		ls := make(map[string]string)
		ss.states[pubKeyHash] = ls
	}
	ss.states[pubKeyHash][key] = value
	return 0
}

//Del deletes an item in scStorage
func (ss *ScState) Del(pubKeyHash, key string) int{
	ss.mutex.Lock()
	defer ss.mutex.Unlock()
	if len(ss.states[pubKeyHash]) == 0 {
		return 1;
	}
	if ss.states[pubKeyHash][key] == ""{
		return 1
	}

	delete(ss.states[pubKeyHash],key)
	return 0
}

//GetStorageByAddress gets a storage map by address
func (ss *ScState) GetStorageByAddress(address string) map[string]string{
	if len(ss.states[address]) == 0 {
		//initializes the map with dummy data
		ss.states[address] = map[string]string{"init":"i"}
	}
	return ss.states[address]
}

//LoadFromDatabase loads states from database
func (ss *ScState) LoadFromDatabase(db storage.Storage, blkHash Hash){
	ss.mutex.Lock()
	defer ss.mutex.Unlock()
	rawBytes, err := db.Get([]byte(scStateMapKey + blkHash.String()))

	if err != nil && err.Error() == storage.ErrKeyInvalid.Error() || len(rawBytes) == 0 {
		return
	}
	ss.states = deserializeScState(rawBytes)
}

//SaveToDatabase saves states to database
func (ss *ScState) SaveToDatabase(db storage.Storage, blkHash Hash) error{
	return db.Put([]byte(scStateMapKey + blkHash.String()), ss.serialize())
}

//Update updates smart contract states by executing all input transactions
func (ss *ScState) Update(txs []*Transaction, index UTXOIndex, manager ScEngineManager, currBlkHeight uint64, parentBlk *Block){
	ss.mutex.Lock()
	defer ss.mutex.Unlock()
	for _,tx := range txs{
		tx.Execute(index,ss,nil,manager.CreateEngine(), currBlkHeight, parentBlk)
	}
}