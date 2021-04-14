package scState

import (
	"encoding/hex"
	"github.com/dappley/go-dappley/core/utxo"
	"github.com/dappley/go-dappley/storage"
	"testing"
	scstatepb "github.com/dappley/go-dappley/core/scState/pb"
	"github.com/golang/protobuf/proto"
	"github.com/stretchr/testify/assert"
)

func TestScState_ToProto(t *testing.T) {
	ss := NewScState(utxo.NewUTXOCache(storage.NewRamStorage()))
	ss.states["addr1"] = map[string]string{"key1": "Value"}
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
	ss := NewScState(&utxo.UTXOCache{ScStateCache:utxo.NewScStateCache()})
	ss.FromProto(scStateProto)

	ss1 := NewScState(&utxo.UTXOCache{ScStateCache:utxo.NewScStateCache()})
	ss1.states["addr1"] = map[string]string{"key1": "Value"}

	assert.Equal(t, ss1, ss)
}
