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
	logger "github.com/sirupsen/logrus"
	"time"

	"github.com/golang/protobuf/proto"

	"github.com/dappley/go-dappley/network/pb"
)

type DappCmd struct {
	cmd            string
	data           []byte
	unixTimeRecvd  int64
	key            string
	uniOrBroadcast int ``
}

func NewDapCmd(cmd string, data []byte, msgKey string, uniOrBroadcast int) *DappCmd {
	return &DappCmd{cmd, data, time.Now().Unix(), msgKey, uniOrBroadcast}
}

func (dc *DappCmd) GetCmd() string {
	return dc.cmd
}

func (dc *DappCmd) GetData() []byte {
	return dc.data
}

func (dc *DappCmd) GetTimestamp() int64 {
	return dc.unixTimeRecvd
}

func (dc *DappCmd) GetFrom() string {
	return dc.key
}

//used to lookup dapmsg cache (key:unix time of command + command in string, value: 1 if received recently, 0 if not).
func (dc *DappCmd) GetKey() string {
	return dc.key
}

func ParseDappMsgFromDappPacket(packet *DappPacket) *DappCmd {
	return ParseDappMsgFromRawBytes(packet.GetData())
}

func ParseDappMsgFromRawBytes(bytes []byte) *DappCmd {
	dmpb := &networkpb.Dapmsg{}

	//unmarshal byte to proto
	if err := proto.Unmarshal(bytes, dmpb); err != nil {
		logger.WithError(err).Warn("Stream: Unable to")
	}

	dm := &DappCmd{}
	dm.FromProto(dmpb)
	return dm
}

func (dc *DappCmd) GetRawBytes() []byte {
	data, err := proto.Marshal(dc.ToProto())
	if err != nil {
		logger.WithError(err).Error("DappCmd: Dapp Command can not be converted into raw bytes")
	}
	return data
}

func (dc *DappCmd) ToProto() proto.Message {
	return &networkpb.Dapmsg{
		Cmd:              dc.cmd,
		Data:             dc.data,
		UnixTimeReceived: dc.unixTimeRecvd,
		Key:              dc.key,
	}
}

func (dc *DappCmd) FromProto(pb proto.Message) {
	dc.cmd = pb.(*networkpb.Dapmsg).GetCmd()
	dc.data = pb.(*networkpb.Dapmsg).GetData()
	dc.unixTimeRecvd = pb.(*networkpb.Dapmsg).GetUnixTimeReceived()
	dc.key = pb.(*networkpb.Dapmsg).GetKey()

}
