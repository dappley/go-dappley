package scState

import (
	"encoding/hex"
	"testing"

	scstatepb "github.com/dappley/go-dappley/core/scState/pb"
	"github.com/dappley/go-dappley/storage"
	proto "github.com/golang/protobuf/proto"
	"github.com/stretchr/testify/assert"
)

func TestScState_Serialize(t *testing.T) {
	ss := NewScState()
	ls := make(map[string]string)
	ls["key1"] = "value1"
	ss.states["addr1"] = ls
	rawBytes := ss.serialize()
	ssRet := deserializeScState(rawBytes)
	assert.Equal(t, ss.states, ssRet.states)
}

func TestScState_LoadFromDatabase(t *testing.T) {
	db := storage.NewRamStorage()
	ss := NewScState()
	ss.Set("addr1", "key1", "Value")
	err := ss.SaveToDatabase(db)
	assert.Nil(t, err)
	ss1 := LoadScStateFromDatabase(db)
	assert.Equal(t, "Value", ss1.Get("addr1", "key1"))
}

func TestScState_ToProto(t *testing.T) {
	ss := NewScState()
	ss.Set("addr1", "key1", "Value")
	expected := "0a180a056164647231120f0a0d0a046b657931120556616c7565"
	rawBytes, err := proto.Marshal(ss.ToProto())
	assert.Nil(t, err)
	assert.Equal(t, expected, hex.EncodeToString(rawBytes))
}

func TestScState_FromProto(t *testing.T) {
	serializedBytes, err := hex.DecodeString("0a180a056164647231120f0a0d0a046b657931120556616c7565")
	assert.Nil(t, err)
	scStateProto := &scstatepb.ScState{}
	err = proto.Unmarshal(serializedBytes, scStateProto)
	assert.Nil(t, err)
	ss := NewScState()
	ss.FromProto(scStateProto)

	ss1 := NewScState()
	ss1.Set("addr1", "key1", "Value")

	assert.Equal(t, ss1, ss)
}
