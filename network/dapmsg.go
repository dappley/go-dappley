package network

import (
	"github.com/gogo/protobuf/proto"
	"github.com/dappley/go-dappley/network/pb"
)

type Dapmsg struct{
	cmd 	string
	data 	[]byte
}

func NewDapmsg(cmd string, data []byte) *Dapmsg {
	return &Dapmsg{cmd, data,}
}

func (dm *Dapmsg) GetCmd() string{
	return dm.cmd
}

func (dm *Dapmsg) GetData() []byte{
	return dm.data
}

func (dm *Dapmsg) ToProto() proto.Message{
	return &networkpb.Dapmsg{
		Cmd: dm.cmd,
		Data: dm.data,
	}
}

func (dm *Dapmsg) FromProto(pb proto.Message){
	dm.cmd = pb.(*networkpb.Dapmsg).Cmd
	dm.data = pb.(*networkpb.Dapmsg).Data
}