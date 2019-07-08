package network

import (
	"github.com/dappley/go-dappley/storage"
	"github.com/libp2p/go-libp2p-core/crypto"
	logger "github.com/sirupsen/logrus"
)

type Network struct {
	host        *Host
	peerManager *PeerManager
}

func NewNetwork(config *NodeConfig, dispatcher chan *streamMsg, db storage.Storage) *Network {
	return &Network{
		peerManager: NewPeerManager(config, dispatcher, db),
	}
}

func (net *Network) Start(listenPort int, privKey crypto.PrivKey) error {
	net.host = NewHost(listenPort, privKey, net.peerManager.StreamHandler)
	net.peerManager.Start(net.host)
	return nil
}

func (net *Network) Stop() {
	net.peerManager.StopAllStreams(nil)
	net.host.RemoveStreamHandler(protocalName)
	err := net.host.Close()
	if err != nil {
		logger.WithError(err).Warn("Node: host was not closed properly.")
	}
}

func (net *Network) SubScribeOnStreamStop(cb onStreamStopFunc) {
	net.peerManager.SubscribeOnStreamStop(cb)
}
