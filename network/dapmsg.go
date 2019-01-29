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
	"github.com/dappley/go-dappley/network/pb"
	"github.com/gogo/protobuf/proto"
	"time"
)

type DapMsg struct {
	cmd            string
	data           []byte
	unixTimeRecvd  int64
	key            string
	uniOrBroadcast int ``
	counter        uint64
}

func NewDapmsg(cmd string, data []byte, msgKey string, uniOrBroadcast int, counter *uint64) *DapMsg {
	if *counter > uint64(MaxMsgCountBeforeReset) {
		*counter = 0
	}
	*counter++
	return &DapMsg{cmd, data, time.Now().Unix(), msgKey, uniOrBroadcast, *counter}
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

func (dm *DapMsg) ToProto() proto.Message {
	return &networkpb.Dapmsg{
		Cmd:           dm.cmd,
		Data:          dm.data,
		UnixTimeReceived: dm.unixTimeRecvd,
		Key:           dm.key,
	}
}

func (dm *DapMsg) FromProto(pb proto.Message) {
	dm.cmd = pb.(*networkpb.Dapmsg).Cmd
	dm.data = pb.(*networkpb.Dapmsg).Data
	dm.unixTimeRecvd = pb.(*networkpb.Dapmsg).UnixTimeReceived
	dm.key = pb.(*networkpb.Dapmsg).Key

}
