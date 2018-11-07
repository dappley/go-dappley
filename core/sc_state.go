package core

import (
	"bytes"
	"encoding/gob"
	"github.com/dappley/go-dappley/storage"
	logger "github.com/sirupsen/logrus"
	"sync"
)

type ScLocalStorage map[string]string

type ScState struct{
	states map[string]ScLocalStorage
	mutex *sync.RWMutex
}

const scStateMapKey = "scState"

func NewScLocalStorage() ScLocalStorage{
	return make(map[string]string)
}

func NewScState() *ScState{
	return &ScState{make(map[string]ScLocalStorage), &sync.RWMutex{}}
}

func deserializeScState(d []byte) *ScState {
	scState := NewScState()
	decoder := gob.NewDecoder(bytes.NewReader(d))
	err := decoder.Decode(&scState.states)
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

//Get deletes an item in scStorage
func (ss *ScState) Get(pubKeyHash, key string) string{
	if len(ss.states[pubKeyHash]) == 0 {
		return ""
	}
	return ss.states[pubKeyHash][key]
}

//Set deletes an item in scStorage
func (ss *ScState) Set(pubKeyHash, key, value string) int{
	if len(ss.states[pubKeyHash]) == 0 {
		ls := NewScLocalStorage()
		ss.states[pubKeyHash] = ls
	}
	ss.states[pubKeyHash][key] = value
	return 0
}

//Del deletes an item in scStorage
func (ss *ScState) Del(pubKeyHash, key string) int{
	if len(ss.states[pubKeyHash]) == 0 {
		return 1;
	}
	if ss.states[pubKeyHash][key] == ""{
		return 1
	}

	delete(ss.states[pubKeyHash],key)
	return 0
}

//LoadFromDatabase loads states from database
func (ss *ScState) LoadFromDatabase(db storage.Storage){
	rawBytes, err := db.Get([]byte(scStateMapKey))

	if err != nil && err.Error() == storage.ErrKeyInvalid.Error() || len(rawBytes) == 0 {
		return
	}
	ss = deserializeScState(rawBytes)
}

//SaveToDatabase saves states to database
func (ss *ScState) SaveToDatabase(db storage.Storage) error{
	return db.Put([]byte(scStateMapKey), ss.serialize())
}