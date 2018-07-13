package network

import "github.com/multiformats/go-multiaddr"

type Peerlist struct{
	peers	[]multiaddr.Multiaddr
}

func NewPeerlist() *Peerlist{
	return &Peerlist{}
}

func (pl *Peerlist) Add(m multiaddr.Multiaddr){
	pl.peers = append(pl.peers, m)
}

func (pl *Peerlist) GetPeerlist() []multiaddr.Multiaddr{
	return pl.peers
}