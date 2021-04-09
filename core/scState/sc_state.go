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
	changeLog := NewChangeLog()
	for address, state := range ss.states {
		if _, ok := changeLog.log[address]; !ok {
			changeLog.log[address] = make(map[string]string)
		}
		for key, value := range state {
			//before saving, read out the original value and save it in the changelog
			valBytes, err := db.Get(util.Str2bytes(scStateMapKey + address + key))
			if err != nil {
				changeLog.log[address][key] = scStateValueIsNotExist
			}else{
				changeLog.log[address][key] = util.Bytes2str(valBytes)
			}
			//update new states in db
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

	err := db.Put(util.Str2bytes(scStateLogKey+blkHash.String()), changeLog.serializeChangeLog())
	if err != nil {
		return err
	}


	return nil
}

func (ss *ScState) RevertState(db storage.Storage, blkHash hash.Hash)  {
	changelog := getChangeLog(db, blkHash)

	for address,state:=range changelog.log{
		for key,value:=range state{
			ss.states[address]=map[string]string{key: value}
		}
	}
}

func getChangeLog(db storage.Storage, blkHash hash.Hash) *ChangeLog {
	changeLog :=NewChangeLog()

	rawBytes, err := db.Get(util.Str2bytes(scStateLogKey + blkHash.String()))
	if err != nil {
		return changeLog
	}

	return deserializeChangeLog(rawBytes)
}

func deleteLog(db storage.Storage, prevHash hash.Hash) error {
	err := db.Del(util.Str2bytes(scStateLogKey + prevHash.String()))
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

func (ss *ScState) GetStateValue(db storage.Storage, address, key string) string {
	if _, ok := ss.states[address]; ok {
		if value, ok := ss.states[address][key]; ok {
			if value == scStateValueIsNotExist {
				return ""
			} else {
				return value
			}
		}
	}else{
		ss.states[address]=make(map[string]string)
	}

	valBytes, err := db.Get(util.Str2bytes(scStateMapKey + address + key))
	if err != nil {
		logger.Warn("get state value failed: ", err)
	}
	value := util.Bytes2str(valBytes)
	ss.states[address][key] = value
	return value
}

func (ss *ScState) SetStateValue(db storage.Storage, address, key, value string)  {
	if _,ok:=ss.states[address];!ok{
		ss.states[address]=make(map[string]string)
	}
	ss.states[address][key]=value
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

