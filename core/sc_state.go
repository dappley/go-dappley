package core

import (
	"bytes"
	"encoding/gob"
	"sync"

	"github.com/dappley/go-dappley/storage"
	logger "github.com/sirupsen/logrus"
)

type ScState struct {
	states map[string]map[string]string
	events []*Event
	mutex  *sync.RWMutex
}

const (
	scStateLogKey = "scLog"
	scStateMapKey = "scState"
	scRewardKey   = "scStateRewardKey"
)

func NewScState() *ScState {
	return &ScState{make(map[string]map[string]string), make([]*Event, 0), &sync.RWMutex{}}
}

func (ss *ScState) GetEvents() []*Event { return ss.events }
func (ss *ScState) RecordEvent(event *Event) {
	ss.events = append(ss.events, event)
}

func deserializeScState(d []byte) map[string]map[string]string {
	scState := make(map[string]map[string]string)
	decoder := gob.NewDecoder(bytes.NewReader(d))
	err := decoder.Decode(&scState)
	if err != nil {
		logger.WithError(err).Panic("ScState: failed to deserialize UTXO states.")
	}
	return scState
}

func (ss *ScState) findChangedValue(newState *ScState) map[string]map[string]string {
	change := make(map[string]map[string]string)

	for address, newMap := range newState.states {
		if oldMap, ok := ss.states[address]; !ok {
			change[address] = nil
		} else {
			ls := make(map[string]string)
			for key, value := range oldMap {
				if newValue, ok := newMap[key]; ok {
					if newValue != value {
						ls[key] = value
					}
				} else {
					ls[key] = value
				}
			}

			for key, value := range newMap {
				if oldMap[key] != value {
					ls[key] = oldMap[key]
				}
			}

			if len(ls) > 0 {
				change[address] = ls
			}

		}
	}

	return change
}

func serialize(ss map[string]map[string]string) []byte {

	var encoded bytes.Buffer
	enc := gob.NewEncoder(&encoded)
	err := enc.Encode(ss)
	if err != nil {
		logger.WithError(err).Panic("ScState: failed to serialize UTXO states.")
	}
	return encoded.Bytes()
}

//Get gets an item in scStorage
func (ss *ScState) Get(pubKeyHash, key string) string {
	ss.mutex.Lock()
	defer ss.mutex.Unlock()
	if len(ss.states[pubKeyHash]) == 0 {
		return ""
	}
	return ss.states[pubKeyHash][key]
}

//Set sets an item in scStorage
func (ss *ScState) Set(pubKeyHash, key, value string) int {
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
func (ss *ScState) Del(pubKeyHash, key string) int {
	ss.mutex.Lock()
	defer ss.mutex.Unlock()
	if len(ss.states[pubKeyHash]) == 0 {
		return 1
	}
	if ss.states[pubKeyHash][key] == "" {
		return 1
	}

	delete(ss.states[pubKeyHash], key)
	return 0
}

//GetStorageByAddress gets a storage map by address
func (ss *ScState) GetStorageByAddress(address string) map[string]string {
	if len(ss.states[address]) == 0 {
		//initializes the map with dummy data
		ss.states[address] = map[string]string{"init": "i"}
	}
	return ss.states[address]
}

//LoadFromDatabase loads states from database
func (ss *ScState) LoadFromDatabase(db storage.Storage, blkHash Hash) {
	ss.mutex.Lock()
	defer ss.mutex.Unlock()
	rawBytes, err := db.Get([]byte(scStateMapKey + blkHash.String()))

	if err != nil && err.Error() == storage.ErrKeyInvalid.Error() || len(rawBytes) == 0 {
		return
	}
	ss.states = deserializeScState(rawBytes)
}

//SaveToDatabase saves states to database
func (ss *ScState) SaveToDatabase(db storage.Storage, blkHash Hash, newSS *ScState) error {
	change := ss.findChangedValue(newSS)

	err := db.Put([]byte(scStateLogKey+blkHash.String()), serialize(change))
	if err != nil {
		return err
	}

	err = db.Put([]byte(scStateMapKey), serialize(newSS.states))
	if err != nil {
		return err
	}

	return err
}
