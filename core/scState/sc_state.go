package scState

import (
	"github.com/dappley/go-dappley/core/stateLog"
	"github.com/dappley/go-dappley/core/utxo"
	"sync"

	"github.com/dappley/go-dappley/common/hash"

	logger "github.com/sirupsen/logrus"
)

type ScState struct {
	states map[string]map[string]string //address key, value
	events []*Event
	cache  *utxo.UTXOCache
	mutex  *sync.RWMutex
}

const (
	ScStateValueIsNotExist = ""
)

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

// Save data with change logs
func (ss *ScState) Save(blkHash hash.Hash) error {
	stLog := stateLog.NewStateLog()
	for address, state := range ss.states {
		if _, ok := stLog.Log[address]; !ok {
			stLog.Log[address] = make(map[string]string)
		}
		for key, value := range state {
			//before saving, read out the original value and save it in the state log
			val, err := ss.cache.GetScStates(utxo.GetscStateKey(address, key))
			if err != nil {
				stLog.Log[address][key] = ScStateValueIsNotExist
			} else {
				stLog.Log[address][key] = val
			}
			//update new states in db
			if value == ScStateValueIsNotExist {
				err := ss.cache.DelScStates(utxo.GetscStateKey(address, key))
				if err != nil {
					return err
				}
			} else {
				err := ss.cache.AddScStates(utxo.GetscStateKey(address, key), value)
				if err != nil {
					return err
				}
			}
		}
	}

	err := ss.cache.AddStateLog(utxo.GetscStateLogKey(blkHash), stLog)
	if err != nil {
		return err
	}

	return nil
}

func (ss *ScState) RevertState(blkHash hash.Hash) {
	stlog, err := ss.cache.GetStateLog(utxo.GetscStateLogKey(blkHash))
	if err != nil {
		logger.Warn("get state log failed: ", err)
	}

	for address, state := range stlog.Log {
		for key, value := range state {
			ss.states[address] = map[string]string{key: value}
		}
	}
}

func (ss *ScState) GetStateValue(address, key string) (string, bool) {
	if _, ok := ss.states[address]; ok {
		if value, ok := ss.states[address][key]; ok {
			if value == ScStateValueIsNotExist {
				return "", false
			} else {
				return value, true
			}
		}
	} else {
		ss.states[address] = make(map[string]string)
	}
	value, err := ss.cache.GetScStates(utxo.GetscStateKey(address, key))
	if err != nil {
		logger.Debug("get state value failed: ", err)
		ss.states[address][key] = ScStateValueIsNotExist
		return "", false
	}
	ss.states[address][key] = value
	return value, true
}

func (ss *ScState) SetStateValue(address, key, value string) {
	if _, ok := ss.states[address]; !ok {
		ss.states[address] = make(map[string]string)
	}
	ss.states[address][key] = value
}

func (ss *ScState) DelStateValue(address, key string) {
	if _, ok := ss.states[address]; ok {
		if _, ok := ss.states[address][key]; ok {
			ss.states[address][key] = ScStateValueIsNotExist
			return
		}
	}

	_, err := ss.cache.GetScStates(utxo.GetscStateKey(address, key))
	if err != nil {
		logger.Warn("The key to be deleted does not exist.")
		return
	}
	ss.states[address] = map[string]string{key: ScStateValueIsNotExist}
}
