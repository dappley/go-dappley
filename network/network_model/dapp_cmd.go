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

package network_model

import (
	"github.com/golang/protobuf/proto"
	"github.com/google/uuid"
	logger "github.com/sirupsen/logrus"

	"github.com/dappley/go-dappley/network/pb"
)

type DappCmd struct {
	name        string
	data        []byte
	isBroadcast bool
	magicNum    uint32
}

//NewDappCmd creates a new instance of DappCmd
func NewDappCmd(cmd string, data []byte, isBroadcast bool) *DappCmd {
	id, _ := uuid.NewUUID()
	return &DappCmd{cmd, data, isBroadcast, id.ID()}
}

//GetName returns the name of the DappCmd
func (dc *DappCmd) GetName() string {
	return dc.name
}

//GetData returns the content of the DappCmd
func (dc *DappCmd) GetData() []byte {
	return dc.data
}

//ParseDappMsgFromDappPacket creates a new DappCmd from a DappPacket
func ParseDappMsgFromDappPacket(packet *DappPacket) *DappCmd {
	return DeserializeDappCmd(packet.GetData())
}

//DeserializeDappCmd deserializes bytes into a DappCmd
func DeserializeDappCmd(bytes []byte) *DappCmd {
	dmpb := &networkpb.DappCmd{}

	//unmarshal byte to proto
	if err := proto.Unmarshal(bytes, dmpb); err != nil {
		logger.WithError(err).Warn("Stream: Unable to")
	}

	dm := &DappCmd{}
	dm.FromProto(dmpb)
	return dm
}

//Serialize serializes a DappCmd into rawa bytes
func (dc *DappCmd) Serialize() []byte {
	data, err := proto.Marshal(dc.ToProto())
	if err != nil {
		logger.WithError(err).Error("DappCmd: Dapp Command can not be converted into raw bytes")
	}
	return data
}

//ToProto converts a DappCmd into proto message
func (dc *DappCmd) ToProto() proto.Message {
	return &networkpb.DappCmd{
		Cmd:         dc.name,
		Data:        dc.data,
		IsBroadcast: dc.isBroadcast,
		MagicNumber: dc.magicNum,
	}
}

//FromProto extracts a DappCmd from a proto message
func (dc *DappCmd) FromProto(pb proto.Message) {
	dc.name = pb.(*networkpb.DappCmd).GetCmd()
	dc.data = pb.(*networkpb.DappCmd).GetData()
	dc.isBroadcast = pb.(*networkpb.DappCmd).GetIsBroadcast()
	dc.magicNum = pb.(*networkpb.DappCmd).GetMagicNumber()
}
