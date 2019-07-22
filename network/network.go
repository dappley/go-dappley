package network

import (
	"github.com/dappley/go-dappley/network/network_model"
	"github.com/hashicorp/golang-lru"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/peer"
	logger "github.com/sirupsen/logrus"
)

type Network struct {
	host                  *network_model.Host
	peerManager           *PeerManager
	streamMsgRcvCh        chan *network_model.DappPacketContext
	streamMsgDispatcherCh chan *network_model.DappPacketContext
	recentlyRcvdDapMsgs   *lru.Cache
}

//NewNetwork creates a network instance
func NewNetwork(config network_model.PeerConnectionConfig, streamMsgDispatcherCh chan *network_model.DappPacketContext, db Storage) *Network {

	var err error
	streamMsgRcvCh := make(chan *network_model.DappPacketContext, dispatchChLen)

	net := &Network{
		peerManager:           NewPeerManager(config, streamMsgRcvCh, db),
		streamMsgRcvCh:        streamMsgRcvCh,
		streamMsgDispatcherCh: streamMsgDispatcherCh,
	}

	net.recentlyRcvdDapMsgs, err = lru.New(1024000)
	if err != nil {
		logger.WithError(err).Panic("Network: Can not initialize lru cache for recentlyRcvdDapMsgs!")
	}

	return net
}

//GetPeers returns a list of peers in the network
func (net *Network) GetPeers() []*network_model.PeerInfo {
	return net.peerManager.CloneStreamsToPeerInfoSlice()
}

//Start starts the network
func (net *Network) Start(listenPort int, privKey crypto.PrivKey, seeds []string) error {
	net.host = network_model.NewHost(listenPort, privKey, net.peerManager.StreamHandler)
	net.peerManager.Start(net.host, seeds)
	net.StartReceivedMsgHandler()
	return nil
}

//StartReceivedMsgHandler starts a listening loop that listens to new message from all streams
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

//IsNetworkRadiation decides if a message is a network radiation (a message that it has received already)
func (net *Network) IsNetworkRadiation(msg *network_model.DappPacket) bool {
	return msg.IsBroadcast() && net.recentlyRcvdDapMsgs.Contains(string(msg.GetRawBytes()))
}

//RecordMessage records a message that is already received or sent
func (net *Network) RecordMessage(msg *network_model.DappPacket) {
	net.recentlyRcvdDapMsgs.Add(string(msg.GetRawBytes()), true)
}

//Stop stops the network
func (net *Network) Stop() {
	net.peerManager.StopAllStreams(nil)
	net.host.RemoveStreamHandler(network_model.ProtocalName)
	err := net.host.Close()
	if err != nil {
		logger.WithError(err).Warn("Node: host was not closed properly.")
	}
}

//OnStreamStop runs cb function upon any stream stops
func (net *Network) OnStreamStop(cb onStreamStopFunc) {
	net.peerManager.SubscribeOnStreamStop(cb)
}

//Unicast sends a message to a peer
func (net *Network) Unicast(data []byte, pid peer.ID, priority network_model.DappCmdPriority) {
	packet := network_model.ConstructDappPacketFromData(data, false)

	net.RecordMessage(packet)
	net.peerManager.Unicast(packet, pid, priority)
}

//Broadcast sends a message to all peers
func (net *Network) Broadcast(data []byte, priority network_model.DappCmdPriority) {
	packet := network_model.ConstructDappPacketFromData(data, true)

	net.RecordMessage(packet)
	net.peerManager.Broadcast(packet, priority)
}

//AddPeer adds a peer to its network and starts the connection
func (net *Network) AddPeer(peerInfo *network_model.PeerInfo) error {
	return net.peerManager.AddAndConnectPeer(peerInfo)
}

//AddSeed Add a seed peer to its network
func (net *Network) AddSeed(peerInfo *network_model.PeerInfo) error {
	return net.peerManager.AddSeedByPeerInfo(peerInfo)
}

//AddPeerByString adds a peer by its full address string and starts the connection
func (net *Network) AddPeerByString(fullAddr string) error {

	peerInfo, err := network_model.NewPeerInfoFromString(fullAddr)
	if err != nil {
		logger.WithError(err).WithFields(logger.Fields{
			"full_addr": fullAddr,
		}).Warn("Network: create PeerInfo failed.")
	}

	return net.peerManager.AddAndConnectPeer(peerInfo)
}
