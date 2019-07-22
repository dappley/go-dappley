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

func NewDappSendCmdContextFromDappCmd(cmd *DappCmd, destination peer.ID, priority DappCmdPriority) *DappSendCmdContext {
	return &DappSendCmdContext{
		command:     cmd,
		destination: destination,
		priority:    priority,
	}
}

func (dcc *DappSendCmdContext) GetCommandName() string {
	return dcc.command.name
}

func (dcc *DappSendCmdContext) GetCommand() *DappCmd {
	return dcc.command
}

func (dcc *DappSendCmdContext) GetPriority() DappCmdPriority {
	return dcc.priority
}

func (dcc *DappSendCmdContext) GetDestination() peer.ID {
	return dcc.destination
}

func (dcc *DappSendCmdContext) IsBroadcast() bool {
	return dcc.command.isBroadcast
}

func (dcc *DappSendCmdContext) Send(commandSendCh chan *DappSendCmdContext) {
	select {
	case commandSendCh <- dcc:
	default:
		logger.WithFields(logger.Fields{
			"lenOfDispatchChan": len(commandSendCh),
		}).Warn("DappSendCmdContext: request channel full")
	}
}
