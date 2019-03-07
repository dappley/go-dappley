package core

import (
	"sync"

	corepb "github.com/dappley/go-dappley/core/pb"
	"github.com/golang/protobuf/proto"

	logger "github.com/sirupsen/logrus"

	"github.com/dappley/go-dappley/storage"
)

type ScState struct {
	states map[string]map[string]string
	events []*Event
	mutex  *sync.RWMutex
}

const (
	scStateMapKey = "scState"
)

func NewScState() *ScState {
	return &ScState{make(map[string]map[string]string), make([]*Event, 0), &sync.RWMutex{}}
}

func (ss *ScState) GetEvents() []*Event { return ss.events }
func (ss *ScState) RecordEvent(event *Event) {
	ss.events = append(ss.events, event)
}

func deserializeScState(d []byte) *ScState {
	scStateProto := &corepb.ScState{}
	err := proto.Unmarshal(d, scStateProto)
	if err != nil {
		logger.WithError(err).Panic("ScState: failed to deserialize UTXO states.")
	}
	ss := NewScState()
	ss.FromProto(scStateProto)
	return ss
}

func (ss *ScState) serialize() []byte {
	rawBytes, err := proto.Marshal(ss.ToProto())
	if err != nil {
		logger.WithError(err).Panic("ScState: failed to serialize UTXO states.")
	}
	return rawBytes
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

func GetScStateKey(blkHash Hash) []byte {
	return []byte(scStateMapKey + blkHash.String())
}

//LoadScStateFromDatabase loads states from database
func LoadScStateFromDatabase(db storage.Storage, blkHash Hash) *ScState {

	rawBytes, err := db.Get([]byte(scStateMapKey + blkHash.String()))

	if err != nil && err.Error() == storage.ErrKeyInvalid.Error() || len(rawBytes) == 0 {
		return NewScState()
	}
	return deserializeScState(rawBytes)
}

//SaveToDatabase saves states to database
func (ss *ScState) SaveToDatabase(db storage.Storage, blkHash Hash) error {
	return db.Put([]byte(scStateMapKey+blkHash.String()), ss.serialize())
}

func (ss *ScState) ToProto() proto.Message {
	scState := make(map[string]*corepb.State)

	for key, val := range ss.states {
		scState[key] = &corepb.State{State: val}
	}
	return &corepb.ScState{States: scState}
}

func (ss *ScState) FromProto(pb proto.Message) {
	for key, val := range pb.(*corepb.ScState).States {
		ss.states[key] = val.State
	}
}
