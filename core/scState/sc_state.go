package scState

import (
	"github.com/dappley/go-dappley/core/utxo"
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
	states map[string]map[string]string //address key, value
	events []*Event
	cache *utxo.UTXOCache
	mutex  *sync.RWMutex
}

const (
	scStateValueIsNotExist = "scStateValueIsNotExist"
)

func NewChangeLog() *ChangeLog {
	return &ChangeLog{make(map[string]map[string]string)}
}

func NewScState(cache *utxo.UTXOCache) *ScState {
	return &ScState{
		make(map[string]map[string]string),
		make([]*Event, 0),
		cache,
		&sync.RWMutex{},
	}
}

func (ss *ScState) GetEvents() []*Event { return ss.events }

func (ss *ScState) RecordEvent(event *Event) {
	ss.events = append(ss.events, event)
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
func (ss *ScState) Save(blkHash hash.Hash) error {
	changeLog := NewChangeLog()
	for address, state := range ss.states {
		if _, ok := changeLog.log[address]; !ok {
			changeLog.log[address] = make(map[string]string)
		}
		for key, value := range state {
			//before saving, read out the original value and save it in the changelog
			val, err := ss.cache.GetScStates(address, key)
			if err != nil {
				changeLog.log[address][key] = scStateValueIsNotExist
			} else {
				changeLog.log[address][key] = val
			}
			//update new states in db
			if value == scStateValueIsNotExist {
				err := ss.cache.DelScStates(address, key)
				if err != nil {
					return err
				}
			} else {
				err := ss.cache.AddScStates(address, key, value)
				if err != nil {
					return err
				}
			}
		}
	}

	err :=ss.cache.AddStateLog(util.Bytes2str(blkHash),changeLog.SerializeChangeLog())
	if err != nil {
		return err
	}

	return nil
}

func (ss *ScState) RevertState(blkHash hash.Hash) {
	changelog := ss.getChangeLog(blkHash)

	for address, state := range changelog.log {
		for key, value := range state {
			ss.states[address] = map[string]string{key: value}
		}
	}
}

func (ss *ScState) getChangeLog(blkHash hash.Hash) *ChangeLog {
	changeLog := NewChangeLog()
	rawBytes, err :=ss.cache.GetStateLog(util.Bytes2str(blkHash))
	if err != nil {
		return changeLog
	}
	return DeserializeChangeLog(rawBytes)
}

func (ss *ScState)deleteLog(prevHash hash.Hash) error {
	err := ss.cache.DelStateLog(util.Bytes2str(prevHash))
	return err
}

func DeserializeChangeLog(d []byte) *ChangeLog {
	scStateProto := &scstatepb.ChangeLog{}
	err := proto.Unmarshal(d, scStateProto)
	if err != nil {
		logger.WithError(err).Panic("ScState: failed to deserialize chaneglog.")
	}
	cl := NewChangeLog()
	cl.FromProto(scStateProto)
	return cl
}

func (cl *ChangeLog) SerializeChangeLog() []byte {
	rawBytes, err := proto.Marshal(cl.ToProto())
	if err != nil {
		logger.WithError(err).Panic("ScState: failed to serialize changelog.")
	}
	return rawBytes
}

func (ss *ScState) GetStateValue(address, key string) string {
	if _, ok := ss.states[address]; ok {
		if value, ok := ss.states[address][key]; ok {
			if value == scStateValueIsNotExist {
				return ""
			} else {
				return value
			}
		}
	} else {
		ss.states[address] = make(map[string]string)
	}
	value, err := ss.cache.GetScStates(address, key)
	if err != nil {
		logger.Warn("get state value failed: ", err)
	}
	ss.states[address][key] = value
	return value
}

func (ss *ScState) SetStateValue(address, key, value string) {
	if _, ok := ss.states[address]; !ok {
		ss.states[address] = make(map[string]string)
	}
	ss.states[address][key] = value
}

func (ss *ScState) DelStateValue(db storage.Storage, address, key string) {
	if _, ok := ss.states[address]; ok {
		if _, ok := ss.states[address][key]; ok {
			ss.states[address][key] = scStateValueIsNotExist
			return
		}
	}
}
