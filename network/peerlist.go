package network

import (
	"fmt"

	"github.com/dappley/go-dappley/network/pb"
	"github.com/gogo/protobuf/proto"
	"github.com/libp2p/go-libp2p-peer"
	"github.com/multiformats/go-multiaddr"
	logger "github.com/sirupsen/logrus"
)

type PeerList struct {
	peers [64]*Peer
}

type Peer struct {
	peerid peer.ID
	addr   multiaddr.Multiaddr
}

func CreatePeerFromMultiaddr(targetFullAddr multiaddr.Multiaddr) (*Peer, error) {
	//get pid
	pid, err := targetFullAddr.ValueForProtocol(multiaddr.P_IPFS)
	if err != nil {
		return nil, err
	}

	//get peer id
	peerid, err := peer.IDB58Decode(pid)
	if err != nil {
		return nil, err
	}

	// Decapsulate the /ipfs/<peerID> part from the targetFullAddr
	// /ip4/<a.b.c.d>/ipfs/<peer> becomes /ip4/<a.b.c.d>
	targetPeerAddr, _ := multiaddr.NewMultiaddr(fmt.Sprintf("/ipfs/%s", peer.IDB58Encode(peerid)))
	targetAddr := targetFullAddr.Decapsulate(targetPeerAddr)
	return &Peer{
		peerid,
		targetAddr,
	}, nil
}

func CreatePeerFromString(targetFullAddr string) (*Peer, error) {
	ma, err := multiaddr.NewMultiaddr(targetFullAddr)
	if err != nil {
		return nil, err
	}
	return CreatePeerFromMultiaddr(ma)
}

//create new peerList with multiaddress
func NewPeerList(p []*Peer) *PeerList {

	pl := &PeerList{}
	//filter out duplicated message
	pl.AddMultiple(p)

	return pl
}

//create new peerList with strings
func NewPeerListStr(strs []string) *PeerList {
	var ps []*Peer
	for _, str := range strs {
		peer, err := CreatePeerFromString(str)
		if err != nil {
			logger.Warn("Address Unrecognized:", str)
		}
		ps = append(ps, peer)
	}
	return NewPeerList(ps)
}

//Add a multiadress.
func (pl *PeerList) Add(p *Peer) {
	//add only if it is not already existed in the list
	if !pl.IsInPeerlist(p) {
		pl.peers = append(pl.peers, p)
	}
}

//add multiple addresses
func (pl *PeerList) AddMultiple(ps []*Peer) {
	for _, p := range ps {
		pl.Add(p)
	}
}

//merge two peerlists
func (pl *PeerList) MergePeerlist(newpl *PeerList) {
	pl.AddMultiple(newpl.GetPeerlist())
}

//find the peers in newpl that are not contained in current pl
func (pl *PeerList) FindNewPeers(newpl *PeerList) *PeerList {
	retpl := &PeerList{}
	for _, m := range newpl.GetPeerlist() {
		if !pl.IsInPeerlist(m) {
			retpl.Add(m)
		}
	}
	return retpl
}

//Get peerList
func (pl *PeerList) GetPeerlist() []*Peer {
	return pl.peers
}

//Check if a multiaddress is already existed in the list
func (pl *PeerList) IsInPeerlist(p *Peer) bool {
	for _, ps := range pl.peers {
		if ps.peerid.String() == p.peerid.String() {
			return true
		}
	}
	return false
}

//convert to protobuf
func (p *Peer) ToProto() proto.Message {
	return &networkpb.Peer{
		Peerid: peer.IDB58Encode(p.peerid),
		Addr:   p.addr.String(),
	}
}

//convert from protobuf
func (p *Peer) FromProto(pb proto.Message) error {
	pid, err := peer.IDB58Decode(pb.(*networkpb.Peer).Peerid)
	if err != nil {
		return err
	}
	p.peerid = pid
	p.addr, err = multiaddr.NewMultiaddr(pb.(*networkpb.Peer).Addr)
	if err != nil {
		return err
	}
	return nil
}

//convert to protobuf
func (pl *PeerList) ToProto() proto.Message {

	var peerlist []*networkpb.Peer
	for i := range pl.peers {
		peerlist = append(peerlist, pl.peers[i].ToProto().(*networkpb.Peer))
	}

	return &networkpb.Peerlist{
		Peerlist: peerlist,
	}
}

//convert from protobuf
func (pl *PeerList) FromProto(pb proto.Message) {
	peerlist := pb.(*networkpb.Peerlist).Peerlist
	pl.peers = nil
	for _, peer := range peerlist {
		p := &Peer{}
		p.FromProto(peer)
		pl.peers = append(pl.peers, p)
	}
}
