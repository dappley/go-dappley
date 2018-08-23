package network

import (
	"testing"
	"github.com/dappley/go-dappley/network/pb"
	"github.com/stretchr/testify/assert"
)

func TestDapmsg_ToProto(t *testing.T) {
	msg :=Dapmsg{"cmd", []byte{1,2,3,4}, 11111111}
	retMsg := &networkpb.Dapmsg{Cmd: "cmd",Data: []byte{1,2,3,4}, UnixTimeRecvd: 11111111}

	assert.Equal(t,msg.ToProto(),retMsg)
}

func TestDapmsg_FromProto(t *testing.T) {
	msg :=Dapmsg{"cmd", []byte{1,2,3,4}, 11111111}
	retMsg := &networkpb.Dapmsg{Cmd: "cmd",Data: []byte{1,2,3,4}, UnixTimeRecvd:11111111}
	msg2 := Dapmsg{}
	msg2.FromProto(retMsg)

	assert.Equal(t,msg,msg2)
}