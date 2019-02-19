package core

import (
	"github.com/dappley/go-dappley/storage"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestScState_Serialize(t *testing.T) {
	ss := NewScState()
	ls := make(map[string]string)
	ls["key1"] = "value1"
	ss.states["addr1"] = ls
	rawBytes := ss.serialize()
	ssRet := NewScState()
	ssRet.states = deserializeScState(rawBytes)
	assert.Equal(t,ss.states,ssRet.states)
}

func TestScState_Get(t *testing.T) {
	ss := NewScState()
	ls := make(map[string]string)
	ls["key1"] = "value1"
	ss.states["addr1"] = ls
	assert.Equal(t, "value1", ss.Get("addr1","key1"))
}

func TestScState_Set(t *testing.T) {
	ss := NewScState()
	ss.Set("addr1","key1","Value")
	assert.Equal(t, "Value", ss.Get("addr1","key1"))
}

func TestScState_Del(t *testing.T) {
	ss := NewScState()
	ls := make(map[string]string)
	ls["key1"] = "value1"
	ss.states["addr1"] = ls
	ss.Del("addr1","key1")
	assert.Equal(t, "", ss.Get("addr1","key1"))
}

func TestScState_LoadFromDatabase(t *testing.T) {
	db := storage.NewRamStorage()
	ss := NewScState()
	ss.Set("addr1","key1","Value")
	hash := []byte("testhash")
	err := ss.SaveToDatabase(db, hash)
	assert.Nil(t, err)
	ss.LoadFromDatabase(db, hash)
	assert.Equal(t, "Value", ss.Get("addr1","key1"))
}
