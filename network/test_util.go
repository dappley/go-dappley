package network

import (
	"github.com/libp2p/go-libp2p-peer"
	"github.com/multiformats/go-multiaddr"
)

func FakeNodeWithPeer(pid, addr string) *Node{

	node := NewNode(nil)
	peerid, _ := peer.IDB58Decode(pid)
	maddr, _ := multiaddr.NewMultiaddr(addr)
	p := &Peer{peerid,maddr}
	node.GetPeerList().Add(p)

	return node
}
