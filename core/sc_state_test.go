package core

import (
	"testing"

	"github.com/dappley/go-dappley/storage"
	"github.com/stretchr/testify/assert"
)

func TestScState_Serialize(t *testing.T) {
	ss := NewScState()
	ls := make(map[string]string)
	ls["key1"] = "value1"
	ss.states["addr1"] = ls
	rawBytes := serialize(ss.states)
	ssRet := NewScState()
	ssRet.states = deserializeScState(rawBytes)
	assert.Equal(t, ss.states, ssRet.states)
}

func TestScState_Get(t *testing.T) {
	ss := NewScState()
	ls := make(map[string]string)
	ls["key1"] = "value1"
	ss.states["addr1"] = ls
	assert.Equal(t, "value1", ss.Get("addr1", "key1"))
}

func TestScState_Set(t *testing.T) {
	ss := NewScState()
	ss.Set("addr1", "key1", "Value")
	assert.Equal(t, "Value", ss.Get("addr1", "key1"))
}

func TestScState_Del(t *testing.T) {
	ss := NewScState()
	ls := make(map[string]string)
	ls["key1"] = "value1"
	ss.states["addr1"] = ls
	ss.Del("addr1", "key1")
	assert.Equal(t, "", ss.Get("addr1", "key1"))
}

func TestScState_LoadFromDatabase(t *testing.T) {
	db := storage.NewRamStorage()
	ss := NewScState()
	ssOld := NewScState()
	ss.Set("addr1", "key1", "Value")
	hash := []byte("testhash")
	err := ssOld.SaveToDatabase(db, hash, ss)
	assert.Nil(t, err)
	ss.LoadFromDatabase(db)
	assert.Equal(t, "Value", ss.Get("addr1", "key1"))
}

func TestScState_ReverState(t *testing.T) {
	ss := NewScState()
	ls := make(map[string]string)
	ls["key1"] = "value1"
	ss.states["addr1"] = ls

	changeLog1 := make(map[string]map[string]string)
	changeLog2 := make(map[string]map[string]string)
	changeLog3 := make(map[string]map[string]string)

	changePair1 := make(map[string]string)
	changePair2 := make(map[string]string)
	changePair3 := make(map[string]string)
	changePair4 := make(map[string]string)

	expect1 := make(map[string]map[string]string)
	expect2 := make(map[string]map[string]string)
	expect3 := make(map[string]map[string]string)

	changePair1["key1"] = "2"
	changePair2["key3"] = "3"
	changePair3["key4"] = "4"
	changePair4["key1"] = "2"
	changePair4["key4"] = "4"

	expect1["addr1"] = changePair1

	expect2["addr1"] = changePair4
	expect2["addr2"] = changePair2

	changeLog1["addr1"] = changePair1
	ss.ReverState(changeLog1)
	assert.Equal(t, expect1, ss.states)

	changeLog2["addr2"] = changePair2
	changeLog2["addr1"] = changePair3
	ss.ReverState(changeLog2)
	assert.Equal(t, expect2, ss.states)

	changeLog3["addr2"] = nil
	changeLog3["addr1"] = nil
	ss.ReverState(changeLog3)
	assert.Equal(t, expect3, ss.states)
	assert.Equal(t, 0, len(ss.states))

}

func TestScState_FindChangedValue(t *testing.T) {
	newSS := NewScState()
	oldSS := NewScState()

	ls1 := make(map[string]string)
	ls2 := make(map[string]string)
	ls3 := make(map[string]string)

	ls1["key1"] = "value1"
	ls1["key2"] = "value2"
	ls1["key3"] = "value3"

	ls2["key1"] = "value1"
	ls2["key2"] = "value2"
	ls2["key3"] = "4"

	ls3["key1"] = "value1"
	ls3["key3"] = "4"

	expect1 := make(map[string]map[string]string)
	expect2 := make(map[string]map[string]string)
	expect3 := make(map[string]map[string]string)
	expect4 := make(map[string]map[string]string)
	expect5 := make(map[string]map[string]string)
	expect6 := make(map[string]map[string]string)

	expect2["address1"] = nil
	expect4["address1"] = map[string]string{
		"key2": "value2",
		"key3": "value3",
	}

	expect5["address1"] = map[string]string{
		"key2": "value2",
		"key3": "value3",
	}

	expect5["address2"] = nil

	expect6["address2"] = ls2

	change1 := oldSS.findChangedValue(newSS)
	assert.Equal(t, expect1, change1)

	newSS.states["address1"] = ls1
	change2 := oldSS.findChangedValue(newSS)
	assert.Equal(t, 1, len(change2))
	assert.Equal(t, expect2, change2)

	oldSS.states["address1"] = ls1
	change3 := oldSS.findChangedValue(newSS)
	assert.Equal(t, 0, len(change3))
	assert.Equal(t, expect3, change3)

	newSS.states["address1"] = ls3
	change4 := oldSS.findChangedValue(newSS)
	assert.Equal(t, 1, len(change4))
	assert.Equal(t, expect4, change4)

	newSS.states["address2"] = ls2
	change5 := oldSS.findChangedValue(newSS)
	assert.Equal(t, 2, len(change5))
	assert.Equal(t, expect5, change5)

	oldSS.states["address2"] = ls2
	oldSS.states["address1"] = ls3
	delete(newSS.states, "address2")
	change6 := oldSS.findChangedValue(newSS)
	assert.Equal(t, 1, len(change6))
	assert.Equal(t, expect6, change6)

}
