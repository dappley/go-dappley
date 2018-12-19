// Copyright (C) 2018 go-dappley authors
//
// This file is part of the go-dappley library.
//
// the go-dappley library is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// the go-dappley library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with the go-dappley library.  If not, see <http://www.gnu.org/licenses/>.
//

package network

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/libp2p/go-libp2p-peer"
	"github.com/multiformats/go-multiaddr"
	logger "github.com/sirupsen/logrus"

	"github.com/dappley/go-dappley/network/pb"
)

var PEERLISTMAXSIZE = 20

type PeerList struct {
	peers []*Peer
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
			logger.WithFields(logger.Fields{
				"address": str,
			}).Warn("PeerList: cannot recognize address.")
		} else {
			ps = append(ps, peer)
		}
	}

	return NewPeerList(ps)
}

func (pl *PeerList) ListIsFull() bool {
	if len(pl.peers) < PEERLISTMAXSIZE {
		return false
	}
	return true
}

//remove a random ip to add a new ip
func (pl *PeerList) RemoveRandomIP() {
	rand.Seed(time.Now().UnixNano())
	randPeer := rand.Intn(PEERLISTMAXSIZE)
	pl.peers = append(pl.peers[:randPeer], pl.peers[randPeer + 1:]...)
}

func (pl *PeerList) DeletePeer(p *Peer) {
	for i, peer := range pl.GetPeerlist() {
		if peer.peerid.String() == p.peerid.String() || peer.addr.String() == p.addr.String() {
			pl.peers = append(pl.peers[:i], pl.peers[i+1:]...)
			return
		}
	}
}

//Add a multiadress.
func (pl *PeerList) Add(p *Peer) {
	//add only if it is not already existed in the list
	if !pl.IsInPeerlist(p) && (len(pl.peers) < PEERLISTMAXSIZE) {
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

	if p == nil {
		return false
	}

	if p.addr == nil || p.peerid == "" {
		return false
	}

	for _, ps := range pl.peers {
		if ps.peerid.String() == p.peerid.String() || ps.addr.String() == p.addr.String() {
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
