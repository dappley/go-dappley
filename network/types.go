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
