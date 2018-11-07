package core

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestScState_Serialize(t *testing.T) {
	ss := NewScState()
	ls := NewScLocalStorage()
	ls["key1"] = "value1"
	ss.states["addr1"] = ls
	rawBytes := ss.serialize()
	ssRet := deserializeScState(rawBytes)
	assert.Equal(t,ss.states,ssRet.states)
}

func TestScState_Get(t *testing.T) {
	ss := NewScState()
	ls := NewScLocalStorage()
	ls["key1"] = "value1"
	ss.states["addr1"] = ls
	assert.Equal(t, "value1", ss.Get("addr1","key1"))
}

func TestScState_Set(t *testing.T) {
	ss := NewScState()
	ss.Set("addr1","key1","value")
	assert.Equal(t, "value", ss.Get("addr1","key1"))
}

func TestScState_Del(t *testing.T) {
	ss := NewScState()
	ls := NewScLocalStorage()
	ls["key1"] = "value1"
	ss.states["addr1"] = ls
	ss.Del("addr1","key1")
	assert.Equal(t, "", ss.Get("addr1","key1"))
}