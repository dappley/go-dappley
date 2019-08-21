package network

import (
	"github.com/libp2p/go-libp2p-core/peer"

	"github.com/dappley/go-dappley/network/network_model"
)

type StreamInfo struct {
	stream         *Stream
	connectionType ConnectionType
	latency        *float64 // refer to PeerInfo.Latency
}

type SyncPeerContext struct {
	checkingStreams map[peer.ID]*StreamInfo
	newPeers        map[peer.ID]*network_model.PeerInfo
}
