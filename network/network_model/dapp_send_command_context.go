package network_model

import (
	"github.com/golang/protobuf/proto"
	"github.com/libp2p/go-libp2p-core/peer"
	logger "github.com/sirupsen/logrus"
)

type DappSendCmdContext struct {
	command     *DappCmd
	destination peer.ID
	priority    DappCmdPriority
}

type DappCmdPriority int

const (
	Unicast   = false
	Broadcast = true
)

const (
	HighPriorityCommand = iota
	NormalPriorityCommand
)

//NewDappSendCmdContext constructs a DappSendCmdContext object from raw information
func NewDappSendCmdContext(cmd string, protoMessage proto.Message, destination peer.ID, isBroadcast bool, priority DappCmdPriority) *DappSendCmdContext {
	bytes, err := proto.Marshal(protoMessage)

	if err != nil {
		logger.WithError(err).Error("DappSendCmdContext: Marshal proto message failed")
	}

	dm := NewDappCmd(cmd, bytes, isBroadcast)

	return &DappSendCmdContext{
		command:     dm,
		destination: destination,
		priority:    priority,
	}
}

//NewDappSendCmdContext constructs a DappSendCmdContext object from an existing DappCmd
func NewDappSendCmdContextFromDappCmd(cmd *DappCmd, destination peer.ID, priority DappCmdPriority) *DappSendCmdContext {
	return &DappSendCmdContext{
		command:     cmd,
		destination: destination,
		priority:    priority,
	}
}

//GetCommandName returns the command name
func (dcc *DappSendCmdContext) GetCommandName() string {
	return dcc.command.name
}

//GetCommand returns the DappCmd
func (dcc *DappSendCmdContext) GetCommand() *DappCmd {
	return dcc.command
}

//GetPriority returns the priority of the command
func (dcc *DappSendCmdContext) GetPriority() DappCmdPriority {
	return dcc.priority
}

//GetDestination returns the receiver of the command
func (dcc *DappSendCmdContext) GetDestination() peer.ID {
	return dcc.destination
}

//IsBroadcast returns if the DappSendCmdContext is a broadcast
func (dcc *DappSendCmdContext) IsBroadcast() bool {
	return dcc.command.isBroadcast
}
