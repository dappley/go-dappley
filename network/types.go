package network

import (
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	ma "github.com/multiformats/go-multiaddr"
)

type StreamInfo struct {
	stream         *Stream
	connectionType ConnectionType
}

type SyncPeerContext struct {
	checkingStreams map[peer.ID]*StreamInfo
	newPeers        map[peer.ID]*PeerInfo
}

type Host struct {
	host    host.Host
	info    *PeerInfo
	privKey crypto.PrivKey
}
