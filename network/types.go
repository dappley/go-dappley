package network

import (
	"github.com/libp2p/go-libp2p-core/peer"

	"github.com/dappley/go-dappley/network/networkmodel"
)

type StreamInfo struct {
	stream         *Stream
	connectionType ConnectionType
	latency        *float64 // refer to PeerInfo.Latency
}

type SyncPeerContext struct {
	checkingStreams map[peer.ID]*StreamInfo
	newPeers        map[peer.ID]*networkmodel.PeerInfo
}
