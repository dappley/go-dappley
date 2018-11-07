package core

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestScState_Serialize(t *testing.T) {
	ss := NewScState()
	ss.states["key1"] = "value1"
	rawBytes := ss.serialize()
	ssRet := deserializeScState(rawBytes)
	assert.Equal(t,ss.states,ssRet.states)
}
