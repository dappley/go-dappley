package network

import (
	"github.com/libp2p/go-libp2p-core/peer"
)

type StreamInfo struct {
	stream         *Stream
	connectionType ConnectionType
}

type SyncPeerContext struct {
	checkingStreams map[peer.ID]*StreamInfo
	newPeers        map[peer.ID]*PeerInfo
}

type NodeConfig struct {
	MaxConnectionOutCount int
	MaxConnectionInCount  int
}

type StreamMsg struct {
	msg    *DappPacket
	source peer.ID
}

type DappMsg struct {
	cmd         *DappCmd
	destination peer.ID
	isBroadcast bool
	priority    DappCmdPriority
}

type DappCmdPriority int

const (
	HighPriorityCommand = iota
	NormalPriorityCommand
)
