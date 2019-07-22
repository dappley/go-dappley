package network

import (
	"github.com/dappley/go-dappley/network/network_model"
	"github.com/libp2p/go-libp2p-core/peer"
)

type StreamInfo struct {
	stream         *Stream
	connectionType ConnectionType
}

type SyncPeerContext struct {
	checkingStreams map[peer.ID]*StreamInfo
	newPeers        map[peer.ID]*network_model.PeerInfo
}

type NodeConfig struct {
	MaxConnectionOutCount int
	MaxConnectionInCount  int
}
