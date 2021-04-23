package stateLog

import (
	stateLogpb "github.com/dappley/go-dappley/core/stateLog/pb"
	"github.com/golang/protobuf/proto"
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_StateLogToProto(t *testing.T) {
	stLog := NewStateLog()
	stLog.Log = map[string]map[string]string{"dGDrVKjCG3sdXtDUgWZ7Fp3Q97tLhqWivf": {"Account1": "99"}}
	expected := &stateLogpb.StateLog{Log: map[string]*stateLogpb.Log{"dGDrVKjCG3sdXtDUgWZ7Fp3Q97tLhqWivf": {Log: map[string]string{"Account1": "99"}}}}

	assert.Equal(t, expected, stLog.ToProto())
}

func Test_StateLogFromProto(t *testing.T) {
	stLog := NewStateLog()
	logProto := &stateLogpb.StateLog{Log: map[string]*stateLogpb.Log{"dGDrVKjCG3sdXtDUgWZ7Fp3Q97tLhqWivf": {Log: map[string]string{"Account1": "99"}}}}
	stLog.FromProto(logProto)
	expected := &StateLog{map[string]map[string]string{"dGDrVKjCG3sdXtDUgWZ7Fp3Q97tLhqWivf": {"Account1": "99"}}}

	assert.Equal(t, expected, stLog)
}

func Test_SerializeStateLog(t *testing.T) {
	slToProto := &stateLogpb.StateLog{Log: map[string]*stateLogpb.Log{"dGDrVKjCG3sdXtDUgWZ7Fp3Q97tLhqWivf": {Log: map[string]string{"Account1": "99"}}}}
	rawBytes, err := proto.Marshal(slToProto)

	assert.Nil(t, err)
	assert.Equal(t, []uint8([]byte{0xa, 0x36, 0xa, 0x22, 0x64, 0x47, 0x44, 0x72, 0x56, 0x4b, 0x6a, 0x43, 0x47, 0x33, 0x73, 0x64, 0x58, 0x74, 0x44, 0x55, 0x67, 0x57, 0x5a, 0x37, 0x46, 0x70, 0x33, 0x51, 0x39, 0x37, 0x74, 0x4c, 0x68, 0x71, 0x57, 0x69, 0x76, 0x66, 0x12, 0x10, 0xa, 0xe, 0xa, 0x8, 0x41, 0x63, 0x63, 0x6f, 0x75, 0x6e, 0x74, 0x31, 0x12, 0x2, 0x39, 0x39}), rawBytes)
}

func Test_DeserializeStateLog(t *testing.T) {
	scStateProto := &stateLogpb.StateLog{}
	d := []byte{0xa, 0x36, 0xa, 0x22, 0x64, 0x47, 0x44, 0x72, 0x56, 0x4b, 0x6a, 0x43, 0x47, 0x33, 0x73, 0x64, 0x58, 0x74, 0x44, 0x55, 0x67, 0x57, 0x5a, 0x37, 0x46, 0x70, 0x33, 0x51, 0x39, 0x37, 0x74, 0x4c, 0x68, 0x71, 0x57, 0x69, 0x76, 0x66, 0x12, 0x10, 0xa, 0xe, 0xa, 0x8, 0x41, 0x63, 0x63, 0x6f, 0x75, 0x6e, 0x74, 0x31, 0x12, 0x2, 0x39, 0x39}
	err := proto.Unmarshal(d, scStateProto)
	assert.Nil(t, err)

	sl := NewStateLog()
	sl.FromProto(scStateProto)

	stLog := NewStateLog()
	stLog.Log = map[string]map[string]string{"dGDrVKjCG3sdXtDUgWZ7Fp3Q97tLhqWivf": {"Account1": "99"}}

	assert.Equal(t, sl, stLog)
}
