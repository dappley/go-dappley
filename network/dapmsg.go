// Copyright (C) 2018 go-dappley authors
//
// This file is part of the go-dappley library.
//
// the go-dappley library is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// the go-dappley library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with the go-dappley library.  If not, see <http://www.gnu.org/licenses/>.
//

package network

import (
	"time"

	"github.com/golang/protobuf/proto"

	"github.com/dappley/go-dappley/network/pb"

	"github.com/google/uuid"
)

type DapMsg struct {
	cmd            string
	data           []byte
	unixTimeRecvd  int64
	key            string
	uniOrBroadcast int ``
	counter        uint64
	uuid		   string
}

func NewDapmsg(cmd string, data []byte, msgKey string, uniOrBroadcast int, counter *uint64) *DapMsg {
	if *counter > uint64(MaxMsgCountBeforeReset) {
		*counter = 0
	}
	*counter++
	uuidByte,_ := uuid.NewUUID()
	return &DapMsg{cmd, data, time.Now().Unix(), msgKey, uniOrBroadcast, *counter,uuidByte.String()}
}

func (dm *DapMsg) GetCmd() string {
	return dm.cmd
}

func (dm *DapMsg) GetData() []byte {
	return dm.data
}

func (dm *DapMsg) GetTimestamp() int64 {
	return dm.unixTimeRecvd
}

func (dm *DapMsg) GetFrom() string {
	return dm.key
}

//used to lookup dapmsg cache (key:unix time of command + command in string, value: 1 if received recently, 0 if not).
func (dm *DapMsg) GetKey() string {
	return dm.key
}

func (dm *DapMsg) GetUuid() string {
	return dm.uuid
}

func (dm *DapMsg) ToProto() proto.Message {
	return &networkpb.Dapmsg{
		Cmd:              dm.cmd,
		Data:             dm.data,
		UnixTimeReceived: dm.unixTimeRecvd,
		Key:              dm.key,
		Counter:    	  dm.counter,
		UniOrBroadcast:	  int64(dm.uniOrBroadcast),
		Uuid: dm.uuid,
	}
}

func (dm *DapMsg) FromProto(pb proto.Message) {
	dm.cmd = pb.(*networkpb.Dapmsg).GetCmd()
	dm.data = pb.(*networkpb.Dapmsg).GetData()
	dm.unixTimeRecvd = pb.(*networkpb.Dapmsg).GetUnixTimeReceived()
	dm.key = pb.(*networkpb.Dapmsg).GetKey()
	dm.uniOrBroadcast = int(pb.(*networkpb.Dapmsg).GetUniOrBroadcast())
	dm.counter = pb.(*networkpb.Dapmsg).GetCounter()
	dm.uuid = pb.(*networkpb.Dapmsg).GetUuid()
}
