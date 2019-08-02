package network_model

import (
	"github.com/libp2p/go-libp2p-core/peer"
)

type CommandHandlerFunc func(command *DappRcvdCmdContext)

type DappRcvdCmdContext struct {
	command *DappCmd
	source  peer.ID
}

//NewDappRcvdCmdContext returns a DappRcvdCmdContext object
func NewDappRcvdCmdContext(command *DappCmd, source peer.ID) *DappRcvdCmdContext {
	return &DappRcvdCmdContext{
		command: command,
		source:  source,
	}
}

//GetCommand returns the command
func (dcc *DappRcvdCmdContext) GetCommand() *DappCmd {
	return dcc.command
}

//GetCommandName returns the command name
func (dcc *DappRcvdCmdContext) GetCommandName() string {
	return dcc.command.GetName()
}

//GetData returns the raw data bytes in the command
func (dcc *DappRcvdCmdContext) GetData() []byte {
	return dcc.command.GetData()
}

//GetSource returns the sender of the command
func (dcc *DappRcvdCmdContext) GetSource() peer.ID {
	return dcc.source
}

//IsBroadcast returns if the command is a broadcast
func (dcc *DappRcvdCmdContext) IsBroadcast() bool {
	return dcc.command.isBroadcast
}
