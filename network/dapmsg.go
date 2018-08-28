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
	"github.com/gogo/protobuf/proto"
	"github.com/dappley/go-dappley/network/pb"
	"time"
	"strconv"
	"github.com/libp2p/go-libp2p-peer"
)

type Dapmsg struct{
	cmd 	string
	data 	[]byte
	unixTimeRecvd int64
	from string
	uniOrBroadcast int
}

func NewDapmsg(cmd string, data []byte, from peer.ID, uniOrBroadcast int) *Dapmsg {
	return &Dapmsg{cmd, data, time.Now().Unix(), from.String(), uniOrBroadcast}
}

func (dm *Dapmsg) GetCmd() string{
	return dm.cmd
}

func (dm *Dapmsg) GetData() []byte{
	return dm.data
}

func (dm *Dapmsg) GetTimestamp() int64{
	return dm.unixTimeRecvd
}

func (dm *Dapmsg) GetFrom() string{
	return dm.from
}
//used to lookup dapmsg cache (key:unix time of command + command in string, value: 1 if received recently, 0 if not).
func (dm *Dapmsg) GetKey() string{
	return strconv.Itoa(int(dm.unixTimeRecvd))+dm.cmd+dm.from
}


func (dm *Dapmsg) ToProto() proto.Message{
	return &networkpb.Dapmsg{
		Cmd: dm.cmd,
		Data: dm.data,
		UnixTimeRecvd: dm.unixTimeRecvd,
		From: dm.from,
	}
}

func (dm *Dapmsg) FromProto(pb proto.Message){
	dm.cmd = pb.(*networkpb.Dapmsg).Cmd
	dm.data = pb.(*networkpb.Dapmsg).Data
	dm.unixTimeRecvd =pb.(*networkpb.Dapmsg).UnixTimeRecvd
	dm.from = pb.(*networkpb.Dapmsg).From

}