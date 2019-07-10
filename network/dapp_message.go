package network

import "github.com/libp2p/go-libp2p-core/peer"

type DappMsg struct {
	cmd         *DappCmd
	destination peer.ID
	isBroadcast bool
	priority    DappCmdPriority
}

func NewDappMsg(cmd string, data []byte, isBroadcast bool, priority DappCmdPriority) *DappMsg {
	dm := NewDapCmd(cmd, data, isBroadcast)

	return &DappMsg{
		cmd:         dm,
		isBroadcast: isBroadcast,
		priority:    priority,
	}
}
