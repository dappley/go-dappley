package network

import (
	"github.com/multiformats/go-multiaddr"
	"github.com/gogo/protobuf/proto"
	"github.com/dappley/go-dappley/network/pb"
	"log"
)

type Peerlist struct{
	peers	[]multiaddr.Multiaddr
}

func NewPeerlist(m []multiaddr.Multiaddr) *Peerlist{
	return &Peerlist{m}
}

func NewPeerlistStr(strs []string) *Peerlist{
	pl := &Peerlist{}
	for _, str := range strs{
		addr, err := multiaddr.NewMultiaddr(str)
		if err!= nil {
			log.Println("Address Unrecognized:", str)
		}
		pl.peers = append(pl.peers, addr)
	}
	return pl
}

func (pl *Peerlist) Add(m multiaddr.Multiaddr){
	pl.peers = append(pl.peers, m)
}

func (pl *Peerlist) GetPeerlist() []multiaddr.Multiaddr{
	return pl.peers
}

func (pl *Peerlist) ToProto() proto.Message{
	peerlistStr := []string{}
	for i := range pl.peers{
		peerlistStr = append(peerlistStr, pl.peers[i].String())
	}

	return &networkpb.Peerlist{
		Peerlist: peerlistStr,
	}
}

func (pl *Peerlist) FromProto(pb proto.Message) {
	peerlistStr := pb.(*networkpb.Peerlist).Peerlist
	pl.peers = nil
	for _, str := range peerlistStr{
		addr, err := multiaddr.NewMultiaddr(str)
		if err!= nil {
			log.Println("Address Unrecognized:", str)
		}
		pl.peers = append(pl.peers, addr)
	}
}