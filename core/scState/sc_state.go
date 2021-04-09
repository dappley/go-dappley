package scState

import (
	"github.com/dappley/go-dappley/util"
	"sync"

	"github.com/dappley/go-dappley/common/hash"

	scstatepb "github.com/dappley/go-dappley/core/scState/pb"
	"github.com/dappley/go-dappley/storage"
	"github.com/golang/protobuf/proto"
	logger "github.com/sirupsen/logrus"
)

type ChangeLog struct {
	log map[string]map[string]string
}

type ScState struct {
	states map[string]map[string]string//address key, value
	events []*Event
	mutex  *sync.RWMutex
}

type ContractState struct {
	address string
	state map[string]string//key, value
	events []*Event
	mutex  *sync.RWMutex
}

const (
	scStateLogKey = "scLog"
	scStateMapKey = "scState"
	scStateValueIsNotExist="scStateValueIsNotExist"
)

func NewChangeLog() *ChangeLog {
	return &ChangeLog{make(map[string]map[string]string)}
}

func NewScState() *ScState {
	return &ScState{make(map[string]map[string]string), make([]*Event, 0), &sync.RWMutex{}}
}

func (ss *ScState) GetEvents() []*Event { return ss.events }

func (ss *ScState) RecordEvent(event *Event) {
	ss.events = append(ss.events, event)
}

func deserializeScState(d []byte) *ScState {
	scStateProto := &scstatepb.ScState{}
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
func (ss *ScState) Get(address, key string) string {
	ss.mutex.Lock()
	defer ss.mutex.Unlock()
	if len(ss.states[address]) == 0 {
		return ""
	}
	return ss.states[address][key]
}

//GetByValue gets an item in scStorage by the value
func (ss *ScState) GetByValue(address, value string) string {
	ss.mutex.Lock()
	defer ss.mutex.Unlock()
	if len(ss.states[address]) == 0 {
		return ""
	}
	for key, val := range ss.states[address] {
		if val == value {
			return key
		}
	}
	return ""
}

//Set sets an item in scStorage
func (ss *ScState) Set(address, key, value string) int {
	ss.mutex.Lock()
	defer ss.mutex.Unlock()

	if len(ss.states[address]) == 0 {
		ls := make(map[string]string)
		ss.states[address] = ls
	}
	ss.states[address][key] = value
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

func GetScStateKey(blkHash hash.Hash) []byte {
	return []byte(scStateMapKey)
}

//LoadScStateFromDatabase loads states from database
func LoadScStateFromDatabase(db storage.Storage) *ScState {

	rawBytes, err := db.Get([]byte(scStateMapKey))

	if err != nil && err.Error() == storage.ErrKeyInvalid.Error() || len(rawBytes) == 0 {
		return NewScState()
	}
	return deserializeScState(rawBytes)
}

//SaveToDatabase saves states to database directly
func (ss *ScState) SaveToDatabase(db storage.Storage) error {
	return db.Put([]byte(scStateMapKey), ss.serialize())
}

func (ss *ScState) ToProto() proto.Message {
	scState := make(map[string]*scstatepb.State)

	for key, val := range ss.states {
		scState[key] = &scstatepb.State{State: val}
	}
	return &scstatepb.ScState{States: scState}
}

func (ss *ScState) FromProto(pb proto.Message) {
	for key, val := range pb.(*scstatepb.ScState).States {
		ss.states[key] = val.State
	}
}

func (cl *ChangeLog) ToProto() proto.Message {
	changelog := make(map[string]*scstatepb.Log)

	for key, val := range cl.log {
		changelog[key] = &scstatepb.Log{Log: val}
	}
	return &scstatepb.ChangeLog{Log: changelog}
}

func (cl *ChangeLog) FromProto(pb proto.Message) {
	for key, val := range pb.(*scstatepb.ChangeLog).Log {
		cl.log[key] = val.Log
	}
}

// Save data with change logs
func (ss *ScState) Save(db storage.Storage, blkHash hash.Hash) error {
	for address, state := range ss.states {
		for key, value := range state {
			if value == scStateValueIsNotExist {
				err := db.Del(util.Str2bytes(scStateMapKey + address + key))
				if err != nil {
					return err
				}
			} else {
				err := db.Put(util.Str2bytes(scStateMapKey+address+key), util.Str2bytes(value))
				if err != nil {
					return err
				}
			}
		}
	}

	changeLog := NewChangeLog()
	changeLog.log = ss.states
	err := db.Put(util.Str2bytes(scStateLogKey+blkHash.String()), changeLog.serializeChangeLog())
	if err != nil {
		return err
	}

	return nil
}

func (ss *ScState) RevertState(db storage.Storage, prevHash hash.Hash) error {



	//changelog := getChangeLog(db, prevHash)
	//if len(changelog) < 1 {
	//	return nil
	//}
	//ss.revertState(changelog)
	//err := deleteLog(db, prevHash)
	//if err != nil {
	//	return err
	//}

	return nil
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

func getChangeLog(db storage.Storage, prevHash hash.Hash) map[string]map[string]string {
	change := make(map[string]map[string]string)

	rawBytes, err := db.Get([]byte(scStateLogKey + prevHash.String()))

	if err != nil && err.Error() == storage.ErrKeyInvalid.Error() || len(rawBytes) == 0 {
		return change
	}
	change = deserializeChangeLog(rawBytes).log

	return change
}

func deleteLog(db storage.Storage, prevHash hash.Hash) error {
	err := db.Del([]byte(scStateLogKey + prevHash.String()))
	return err
}

func deserializeChangeLog(d []byte) *ChangeLog {
	scStateProto := &scstatepb.ChangeLog{}
	err := proto.Unmarshal(d, scStateProto)
	if err != nil {
		logger.WithError(err).Panic("ScState: failed to deserialize chaneglog.")
	}
	cl := NewChangeLog()
	cl.FromProto(scStateProto)
	return cl
}

func (cl *ChangeLog) serializeChangeLog() []byte {
	rawBytes, err := proto.Marshal(cl.ToProto())
	if err != nil {
		logger.WithError(err).Panic("ScState: failed to serialize changelog.")
	}
	return rawBytes
}

func (scState *ScState) DeepCopy() *ScState {
	newScState := &ScState{make(map[string]map[string]string), make([]*Event, 0), &sync.RWMutex{}}

	for address, addressState := range scState.states {
		newAddressState := make(map[string]string)
		for key, value := range addressState {
			newAddressState[key] = value
		}

		newScState.states[address] = addressState
	}

	for _, event := range scState.events {
		newScState.events = append(newScState.events, event)
	}

	return newScState
}

func (ss *ScState) GetStateByAddress(scAddress string) *ContractState {
	cs:=NewContractState()
	for address, state := range ss.states {
		if address == scAddress {
			cs.address=address
			cs.state=state
			return cs
		}
	}
	return nil
}

func (ss *ScState) AddState(cs *ContractState){
		if _, ok := ss.states[cs.address]; !ok {
			ss.states[cs.address]=make(map[string]string)
		}
		ss.states[cs.address]=cs.state
}

func NewContractState() *ContractState {
	return &ContractState{"",make(map[string]string), make([]*Event, 0), &sync.RWMutex{}}
}

func (ss *ScState) GetStateValue(db storage.Storage, address, key string) string {
	if _, ok := ss.states[address]; ok {
		if value, ok := ss.states[address][key]; ok {
			if value == scStateValueIsNotExist {
				return ""
			} else {
				return value
			}
		}
	}

	valBytes, err := db.Get(util.Str2bytes(scStateMapKey + address + key))
	if err != nil {
		logger.Warn("get state value failed: ", err)
	}
	value := util.Bytes2str(valBytes)
	ss.states[address] = map[string]string{key: value}
	return value
}

func (ss *ScState) SetStateValue(db storage.Storage, address, key, value string) error {
	if _, ok := ss.states[address]; ok {
		if value, ok := ss.states[address][key]; ok {
			ss.states[address][key] = value
			return nil
		}
	}

	valBytes, err := db.Get(util.Str2bytes(scStateMapKey + address + key))
	if err != nil {
		logger.Warn("get state value failed.")
		return err
	}
	ss.states[address] = map[string]string{key: util.Bytes2str(valBytes)}
	return nil
}

func (ss *ScState) DelStateValue(db storage.Storage, address, key string) {
	if _, ok := ss.states[address]; ok {
		if _, ok := ss.states[address][key]; ok {
			ss.states[address][key] = scStateValueIsNotExist
			return
		}
	}

	_, err := db.Get(util.Str2bytes(scStateMapKey + address + key))
	if err != nil {
		logger.Warn("The key to be deleted does not exist.")
		return
	}
	ss.states[address] = map[string]string{key: scStateValueIsNotExist}
}

func (cs *ContractState) RecordEvent(event *Event) {
	cs.events = append(cs.events, event)
}

// Save data with change logs
func (cs *ContractState) Save(db storage.Storage) error {
	//scStateOld := LoadScStateFromDatabase(db)
	//change := NewChangeLog()
	//change.log = scStateOld.findChangedValue(ss)
	//logger.Warn("changelog:")
	//for k,v:=range change.log {
	//	logger.Info("change.log address: ",k)
	//	for k,v:=range v {
	//		logger.Info("change.log  key: ",k,",value: ",v)
	//	}
	//}

	//err := db.Put([]byte(scStateLogKey+blkHash.String()), change.serializeChangeLog())
	//if err != nil {
	//	return err
	//}

	for key,val:=range cs.state{
		err := db.Put(util.Str2bytes(scStateMapKey+cs.address+key), util.Str2bytes(val))
		if err != nil {
			return err
		}
	}
	return nil
}

func (cs *ContractState) RevertState(db storage.Storage, prevHash hash.Hash) error {
	return nil
}

func (cs *ContractState) SetAddress(address string){
	cs.address=address
}
