package stateLog

import (
	stateLogpb "github.com/dappley/go-dappley/core/stateLog/pb"
	"github.com/golang/protobuf/proto"
	logger "github.com/sirupsen/logrus"
)


type StateLog struct {
	Log map[string]map[string]string
}


func NewStateLog() *StateLog {
	return &StateLog{make(map[string]map[string]string)}
}


func (sl *StateLog) ToProto() proto.Message {
	statelog := make(map[string]*stateLogpb.Log)

	for key, val := range sl.Log {
		statelog[key] = &stateLogpb.Log{Log: val}
	}
	return &stateLogpb.StateLog{Log: statelog}
}

func (sl *StateLog) FromProto(pb proto.Message) {
	for key, val := range pb.(*stateLogpb.StateLog).Log {
		sl.Log[key] = val.Log
	}
}

func DeserializeStateLog(d []byte) *StateLog {
	scStateProto := &stateLogpb.StateLog{}
	err := proto.Unmarshal(d, scStateProto)
	if err != nil {
		logger.WithError(err).Panic("ScState: failed to deserialize chaneglog.")
	}
	sl := NewStateLog()
	sl.FromProto(scStateProto)
	return sl
}

func (sl *StateLog) SerializeStateLog() []byte {
	rawBytes, err := proto.Marshal(sl.ToProto())
	if err != nil {
		logger.WithError(err).Panic("ScState: failed to serialize changelog.")
	}
	return rawBytes
}


