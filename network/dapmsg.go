package network

import (
	"github.com/gogo/protobuf/proto"
	"github.com/dappley/go-dappley/network/pb"
	"time"
)

type Dapmsg struct{
	cmd 	string
	data 	[]byte
	unixTimeRecvd int64
}

func NewDapmsg(cmd string, data []byte) *Dapmsg {
	return &Dapmsg{cmd, data, time.Now().Unix()}
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

func (dm *Dapmsg) ToProto() proto.Message{
	return &networkpb.Dapmsg{
		Cmd: dm.cmd,
		Data: dm.data,
		UnixTimeRecvd: dm.unixTimeRecvd,
	}
}

func (dm *Dapmsg) FromProto(pb proto.Message){
	dm.cmd = pb.(*networkpb.Dapmsg).Cmd
	dm.data = pb.(*networkpb.Dapmsg).Data
	dm.unixTimeRecvd =pb.(*networkpb.Dapmsg).UnixTimeRecvd
}