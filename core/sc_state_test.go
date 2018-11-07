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
