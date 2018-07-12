package network

import (
	"github.com/gogo/protobuf/proto"
	"github.com/dappley/go-dappley/network/pb"
)

type Depmsg struct{
	cmd 	string
	data 	[]byte
}

func NewDepmsg(cmd string, data []byte) *Depmsg{
	return &Depmsg{cmd, data,}
}

func (dm *Depmsg) GetCmd() string{
	return dm.cmd
}

func (dm *Depmsg) GetData() []byte{
	return dm.data
}

func (dm *Depmsg) ToProto() proto.Message{
	return &networkpb.Depmsg{
		Cmd: dm.cmd,
		Data: dm.data,
	}
}

func (dm *Depmsg) FromProto(pb proto.Message){
	dm.cmd = pb.(*networkpb.Depmsg).Cmd
	dm.data = pb.(*networkpb.Depmsg).Data
}