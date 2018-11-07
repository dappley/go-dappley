package core

import (
	"bytes"
	"encoding/gob"
	logger "github.com/sirupsen/logrus"
	"sync"
)

type ScLocalStorage map[string]string

type ScState struct{
	states map[string]ScLocalStorage
	mutex *sync.RWMutex
}

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
