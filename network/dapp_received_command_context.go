package network

import (
	"github.com/libp2p/go-libp2p-core/peer"
)

type DappRcvdCmdContext struct {
	command *DappCmd
	source  peer.ID
}

func NewDappRcvdCmdContext(command *DappCmd, source peer.ID) *DappRcvdCmdContext {
	return &DappRcvdCmdContext{
		command: command,
		source:  source,
	}
}

func (dcc *DappRcvdCmdContext) GetCommand() *DappCmd {
	return dcc.command
}

func (dcc *DappRcvdCmdContext) GetCommandName() string {
	return dcc.command.GetName()
}

func (dcc *DappRcvdCmdContext) GetData() []byte {
	return dcc.command.GetData()
}

func (dcc *DappRcvdCmdContext) GetSource() peer.ID {
	return dcc.source
}

func (dcc *DappRcvdCmdContext) IsBroadcast() bool {
	return dcc.command.isBroadcast
}
