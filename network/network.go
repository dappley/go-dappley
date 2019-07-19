package network

import (
	"github.com/dappley/go-dappley/network/network_model"
	"github.com/hashicorp/golang-lru"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/peer"
	logger "github.com/sirupsen/logrus"
)

type Network struct {
	host                  *Host
	peerManager           *PeerManager
	streamMsgRcvCh        chan *network_model.DappPacketContext
	streamMsgDispatcherCh chan *network_model.DappPacketContext
	recentlyRcvdDapMsgs   *lru.Cache
}

func NewNetwork(config *NodeConfig, streamMsgDispatcherCh chan *network_model.DappPacketContext, commandSendCh chan *network_model.DappSendCmdContext, db Storage) *Network {

	var err error
	streamMsgRcvCh := make(chan *network_model.DappPacketContext, dispatchChLen)

	net := &Network{
		peerManager:           NewPeerManager(config, streamMsgRcvCh, commandSendCh, db),
		streamMsgRcvCh:        streamMsgRcvCh,
		streamMsgDispatcherCh: streamMsgDispatcherCh,
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
	net.StartReceivedMsgHandler()
	return nil
}

func (net *Network) StartReceivedMsgHandler() {
	go func() {
		for {
			select {
			case msg := <-net.streamMsgRcvCh:

				if net.IsNetworkRadiation(msg.Packet) {
					continue
				}
				net.RecordMessage(msg.Packet)
				select {
				case net.streamMsgDispatcherCh <- msg:
				default:
					logger.WithFields(logger.Fields{
						"dispatcherCh_len": len(net.streamMsgDispatcherCh),
					}).Warn("Stream: message streamMsgDispatcherCh channel full! Message disgarded")
					return
				}
			}
		}
	}()
}

func (net *Network) IsNetworkRadiation(msg *network_model.DappPacket) bool {
	return msg.IsBroadcast() && net.recentlyRcvdDapMsgs.Contains(string(msg.GetRawBytes()))
}

func (net *Network) RecordMessage(msg *network_model.DappPacket) {
	net.recentlyRcvdDapMsgs.Add(string(msg.GetRawBytes()), true)
}

func (net *Network) Stop() {
	net.peerManager.StopAllStreams(nil)
	net.host.RemoveStreamHandler(ProtocalName)
	err := net.host.Close()
	if err != nil {
		logger.WithError(err).Warn("Node: host was not closed properly.")
	}
}

func (net *Network) OnStreamStop(cb onStreamStopFunc) {
	net.peerManager.SubscribeOnStreamStop(cb)
}

func (net *Network) Unicast(data []byte, pid peer.ID, priority network_model.DappCmdPriority) {
	packet := network_model.ConstructDappPacketFromData(data, false)

	net.RecordMessage(packet)
	net.peerManager.Unicast(packet, pid, priority)
}

func (net *Network) Broadcast(data []byte, priority network_model.DappCmdPriority) {
	packet := network_model.ConstructDappPacketFromData(data, true)

	net.RecordMessage(packet)
	net.peerManager.Broadcast(packet, priority)
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

func (net *Network) Subscirbe(broker *CommandBroker) {
	net.peerManager.Subscirbe(broker)
}
