package network_model

import (
	"fmt"
	"github.com/dappley/go-dappley/network/pb"
	"github.com/golang/protobuf/proto"
	"github.com/libp2p/go-libp2p-core/peer"
	ma "github.com/multiformats/go-multiaddr"
	logger "github.com/sirupsen/logrus"
)

type PeerInfo struct {
	PeerId  peer.ID
	Addrs   []ma.Multiaddr
	Latency *float64 // rtt of ping in milliseconds

}

func NewPeerInfoFromString(fullAddr string) (PeerInfo, error) {
	addr, err := ma.NewMultiaddr(fullAddr)
	if err != nil {
		logger.WithError(err).WithFields(logger.Fields{
			"full_addr": fullAddr,
			"err":       err,
		}).Warn("PeerInfo: create multiaddr failed.")
	}

	return NewPeerInfoFromMultiaddrs([]ma.Multiaddr{addr})
}

//NewPeerInfoFromMultiaddrs generates PeerInfo object from multiaddresses
func NewPeerInfoFromMultiaddrs(targetFullAddrs []ma.Multiaddr) (PeerInfo, error) {
	peerIds := make([]peer.ID, len(targetFullAddrs))
	addrs := make([]ma.Multiaddr, len(targetFullAddrs))
	for index, targetFullAddr := range targetFullAddrs {
		//get pid
		pid, err := targetFullAddr.ValueForProtocol(ma.P_IPFS)
		if err != nil {
			return PeerInfo{}, err
		}

		//get peer id
		peerId, err := peer.IDB58Decode(pid)
		if err != nil {
			return PeerInfo{}, err
		}

		peerIds[index] = peerId

		// Decapsulate the /ipfs/<peerID> part source the targetFullAddr
		// /ip4/<a.b.c.d>/ipfs/<peer> becomes /ip4/<a.b.c.d>
		targetPeerAddr, _ := ma.NewMultiaddr(fmt.Sprintf("/ipfs/%s", peer.IDB58Encode(peerId)))
		targetAddr := targetFullAddr.Decapsulate(targetPeerAddr)
		addrs[index] = targetAddr

		logger.WithFields(logger.Fields{
			"index":          index,
			"peerID":         peerId,
			"targetPeerAddr": targetPeerAddr,
			"targetAddr":     targetAddr,
		}).Info("PeerManager: create peer information.")
	}

	peerInfo := PeerInfo{
		PeerId: peerIds[0],
		Addrs:  addrs,
	}

	return peerInfo, nil
}

//convert to protobuf
func (p *PeerInfo) ToProto() proto.Message {
	var addresses []string
	for _, addr := range p.Addrs {
		addresses = append(addresses, addr.String())
	}

	pi := &networkpb.PeerInfo{Id: peer.IDB58Encode(p.PeerId), Address: addresses}
	if p.Latency != nil {
		pi.OptionalValue = &networkpb.PeerInfo_Latency{Latency: *p.Latency}
	}
	return pi
}

//convert source protobuf
func (p *PeerInfo) FromProto(pb proto.Message) error {
	pid, err := peer.IDB58Decode(pb.(*networkpb.PeerInfo).GetId())
	if err != nil {
		return err
	}
	p.PeerId = pid

	for _, addr := range pb.(*networkpb.PeerInfo).GetAddress() {
		multiaddr, err := ma.NewMultiaddr(addr)
		if err != nil {
			return err
		}
		p.Addrs = append(p.Addrs, multiaddr)
	}

	hasOption := pb.(*networkpb.PeerInfo).GetOptionalValue()
	if hasOption != nil {
		latency := pb.(*networkpb.PeerInfo).GetLatency()
		p.Latency = &latency
	}

	return nil
}
