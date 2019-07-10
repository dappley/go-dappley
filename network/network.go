package network

import (
	"github.com/dappley/go-dappley/storage"
	"github.com/hashicorp/golang-lru"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/peer"
	logger "github.com/sirupsen/logrus"
)

type Network struct {
	host                *Host
	peerManager         *PeerManager
	msgRcvCh            chan *StreamMsg
	recentlyRcvdDapMsgs *lru.Cache
	dispatcher          chan *StreamMsg
}

func NewNetwork(config *NodeConfig, dispatcher chan *StreamMsg, db storage.Storage) *Network {

	var err error
	msgRcvCh := make(chan *StreamMsg, dispatchChLen)

	net := &Network{
		msgRcvCh:    msgRcvCh,
		peerManager: NewPeerManager(config, msgRcvCh, db),
		dispatcher:  dispatcher,
	}

	net.recentlyRcvdDapMsgs, err = lru.New(1024000)
	if err != nil {
		logger.WithError(err).Panic("Network: Can not initialize lru cache for recentlyRcvdDapMsgs!")
	}

	return net
}

func (net *Network) GetPeers() []*PeerInfo {
	return net.peerManager.CloneStreamsToPeerInfoSlice()
}

func (net *Network) Start(listenPort int, privKey crypto.PrivKey, seeds []string) error {
	net.host = NewHost(listenPort, privKey, net.peerManager.StreamHandler)
	net.peerManager.Start(net.host, seeds)
	net.StartReceivedMessageHandler()
	return nil
}

func (net *Network) StartReceivedMessageHandler() {
	go func() {
		for {
			select {
			case msg := <-net.msgRcvCh:

				if net.IsNetworkRadiation(msg.msg) {
					continue
				}

				select {
				case net.dispatcher <- msg:
				default:
					logger.WithFields(logger.Fields{
						"dispatcherCh_len": len(net.dispatcher),
					}).Warn("Stream: message dispatcher channel full! Message disgarded")
					return
				}
			}
		}
	}()
}

func (net *Network) IsNetworkRadiation(msg *DappPacket) bool {
	return net.recentlyRcvdDapMsgs.Contains(msg)
}

func (net *Network) RecordMessage(msg *DappPacket) {
	net.recentlyRcvdDapMsgs.Add(msg, true)
}

func (net *Network) Stop() {
	net.peerManager.StopAllStreams(nil)
	net.host.RemoveStreamHandler(protocalName)
	err := net.host.Close()
	if err != nil {
		logger.WithError(err).Warn("Node: host was not closed properly.")
	}
}

func (net *Network) OnStreamStop(cb onStreamStopFunc) {
	net.peerManager.SubscribeOnStreamStop(cb)
}

func (net *Network) Unicast(data []byte, pid peer.ID, priority DappCmdPriority) {
	net.peerManager.Unicast(data, pid, priority)
}

func (net *Network) Broadcast(data []byte, priority DappCmdPriority) {
	net.peerManager.Broadcast(data, priority)
}

func (net *Network) AddPeer(peerInfo *PeerInfo) error {
	return net.peerManager.AddAndConnectPeer(peerInfo)
}

func (net *Network) AddSeed(peerInfo *PeerInfo) error {
	return net.peerManager.AddSeedByPeerInfo(peerInfo)
}

func (net *Network) AddPeerByString(fullAddr string) error {

	peerInfo, err := NewPeerInfoFromString(fullAddr)
	if err != nil {
		logger.WithError(err).WithFields(logger.Fields{
			"full_addr": fullAddr,
		}).Warn("Network: create PeerInfo failed.")
	}

	return net.peerManager.AddAndConnectPeer(peerInfo)
}
