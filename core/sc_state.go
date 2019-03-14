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

	for address, oldMap := range ss.states {
		if _, ok := newState.states[address]; !ok {
			change[address] = oldMap
		}
	}

	return change
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
func (ss *ScState) LoadFromDatabase(db storage.Storage) {
	ss.mutex.Lock()
	defer ss.mutex.Unlock()
	rawBytes, err := db.Get([]byte(scStateMapKey))

	if err != nil && err.Error() == storage.ErrKeyInvalid.Error() || len(rawBytes) == 0 {
		return
	}
	ss.states = deserializeScState(rawBytes)
}

//Save saves states to database
func (ss *ScState) Save(db storage.Storage, blkHash Hash) error {
	scStateOld := NewScState()
	scStateOld.LoadFromDatabase(db)
	change := scStateOld.findChangedValue(ss)

	err := db.Put([]byte(scStateLogKey+blkHash.String()), serialize(change))
	if err != nil {
		return err
	}

	err = db.Put([]byte(scStateMapKey), serialize(ss.states))
	if err != nil {
		return err
	}

	return err
}

func (ss *ScState) saveToDatabase(db storage.Storage) error {
	err := db.Put([]byte(scStateMapKey), serialize(ss.states))
	return err
}

func (ss *ScState) RevertStateAndSave(db storage.Storage, prevHash Hash) error {
	changelog := getChangeLog(db, prevHash)
	ss.revertState(changelog)
	err := deleteLog(db, prevHash)
	if err != nil {
		return err
	}
	return ss.saveToDatabase(db)

}

func (ss *ScState) revertState(changelog map[string]map[string]string) {
	for address, pair := range changelog {
		if pair == nil {
			delete(ss.states, address)
		} else {
			if _, ok := ss.states[address]; !ok {
				ss.states[address] = pair
			} else {
				for key, value := range pair {
					if value != "" {
						ss.states[address][key] = value
					} else {
						delete(ss.states[address], key)
					}
				}
			}
		}
	}
}

func getChangeLog(db storage.Storage, prevHash Hash) map[string]map[string]string {
	change := make(map[string]map[string]string)

	rawBytes, err := db.Get([]byte(scStateLogKey + prevHash.String()))

	if err != nil && err.Error() == storage.ErrKeyInvalid.Error() || len(rawBytes) == 0 {
		return change
	}
	change = deserializeScState(rawBytes)

	return change
}

func deleteLog(db storage.Storage, prevHash Hash) error {
	err := db.Del([]byte(scStateLogKey + prevHash.String()))

	return err
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

func serialize(ss map[string]map[string]string) []byte {

	var encoded bytes.Buffer
	enc := gob.NewEncoder(&encoded)
	err := enc.Encode(ss)
	if err != nil {
		logger.WithError(err).Panic("ScState: failed to serialize UTXO states.")
	}
	return encoded.Bytes()
}
