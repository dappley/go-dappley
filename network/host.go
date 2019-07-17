package network

import (
	"context"
	"fmt"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
	ma "github.com/multiformats/go-multiaddr"
	logger "github.com/sirupsen/logrus"
)

const (
	ProtocalName = "dappley/1.0.0"
)

type Host struct {
	host.Host
	info *PeerInfo
}

func NewHost(listenPort int, privKey crypto.PrivKey, handler network.StreamHandler) *Host {
	h, addrs, err := createBasicHost(listenPort, privKey)
	if err != nil {
		logger.WithError(err).Error("Network: Failed to create host.")
		return nil
	}

	info, err := NewPeerInfoFromMultiaddrs(addrs)

	if err != nil {
		logger.WithError(err).Error("Network: Failed to get multiaddr source host.")
		return nil
	}

	h.SetStreamHandler(ProtocalName, handler)

	return &Host{
		h,
		info,
	}
}

//create basic host. Returns host object, host address and error
func createBasicHost(listenPort int, priv crypto.PrivKey) (host.Host, []ma.Multiaddr, error) {

	opts := []libp2p.Option{
		libp2p.ListenAddrStrings(fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", listenPort)),
	}

	if priv != nil {
		opts = append(opts, libp2p.Identity(priv))
	}

	basicHost, err := libp2p.New(context.Background(), opts...)

	if err != nil {
		logger.WithError(err).Error("Node: failed to create a new libp2p node.")
		return nil, nil, err
	}

	// Build host multiaddress
	hostAddr, _ := ma.NewMultiaddr(fmt.Sprintf("/ipfs/%s", basicHost.ID().Pretty()))

	// Now we can build a full multiaddress to reach this host
	// by encapsulating both addresses:

	fullAddrs := make([]ma.Multiaddr, len(basicHost.Addrs()))

	for index, addr := range basicHost.Addrs() {
		fullAddr := addr.Encapsulate(hostAddr)
		logger.WithFields(logger.Fields{
			"index":   index,
			"address": fullAddr,
		}).Info("Node: host is up.")

		fullAddrs[index] = fullAddr
	}

	return basicHost, fullAddrs, nil
}
