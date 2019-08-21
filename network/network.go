package network

import (
	"time"

	lru "github.com/hashicorp/golang-lru"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/peer"
	logger "github.com/sirupsen/logrus"

	"github.com/dappley/go-dappley/network/network_model"
)

type Network struct {
	streamManager         *StreamManager
	peerManager           *PeerManager
	streamMsgRcvCh        chan *network_model.DappPacketContext
	streamMsgDispatcherCh chan *network_model.DappPacketContext
	recentlyRcvdDapMsgs   *lru.Cache
	onStreamStopCb        OnStreamCbFunc
}

type NetworkContext struct {
	netService            NetService
	config                network_model.PeerConnectionConfig
	streamMsgDispatcherCh chan *network_model.DappPacketContext
	db                    Storage
	onStreamStopCb        OnStreamCbFunc
	seeds                 []string
}

//NewNetwork creates a network instance
func NewNetwork(netContext *NetworkContext) *Network {

	var err error

	net := &Network{
		streamMsgRcvCh:        make(chan *network_model.DappPacketContext, dispatchChLen),
		streamMsgDispatcherCh: netContext.streamMsgDispatcherCh,
		onStreamStopCb:        netContext.onStreamStopCb,
	}

	net.recentlyRcvdDapMsgs, err = lru.New(1024000)
	net.streamManager = NewStreamManager(netContext.config, net.streamMsgRcvCh, net.onStreamStop, net.onStreamConnected)
	net.peerManager = NewPeerManager(netContext.netService, netContext.db, net.onPeerListReceived, netContext.seeds)

	if err != nil {
		logger.WithError(err).Panic("Network: Can not initialize lru cache for recentlyRcvdDapMsgs!")
	}

	return net
}

//GetConnectedPeers returns a list of peers in the network
func (net *Network) GetConnectedPeers() []network_model.PeerInfo {
	peers := net.streamManager.GetConnectedPeers()
	peersInSlice := []network_model.PeerInfo{}
	for _, peer := range peers {
		peersInSlice = append(peersInSlice, peer)
	}
	return peersInSlice
}

//GetHost returns a list of peers in the network
func (net *Network) GetHost() *network_model.Host {
	if net.streamManager == nil {
		return nil
	}

	return net.streamManager.host
}

func (net *Network) GetStreamManager() *StreamManager {
	return net.streamManager
}

func (net *Network) StartNewPingService(interval time.Duration) error {
	return net.streamManager.StartNewPingService(interval)
}

//Start starts the network
func (net *Network) Start(listenPort int, privKey crypto.PrivKey) error {
	host := network_model.NewHost(listenPort, privKey, net.streamManager.StreamHandler)
	net.streamManager.Start(host)
	net.connectToAllPeers()
	net.peerManager.Start()
	net.peerManager.SetHostPeerId(host.GetPeerInfo().PeerId)
	net.startStreamMsgHandler()
	net.startPeerConnectionSchedule()
	return nil
}

//Stop stops the network
func (net *Network) Stop() {
	net.streamManager.Stop()
}

//Unicast sends a message to a peer
func (net *Network) Unicast(data []byte, pid peer.ID, priority network_model.DappCmdPriority) {
	packet := network_model.ConstructDappPacketFromData(data, false)

	net.recordMessage(packet)
	net.streamManager.Unicast(packet, pid, priority)
}

//Broadcast sends a message to all peers
func (net *Network) Broadcast(data []byte, priority network_model.DappCmdPriority) {
	packet := network_model.ConstructDappPacketFromData(data, true)

	net.recordMessage(packet)
	net.streamManager.Broadcast(packet, priority)
}

//ConnectToSeed adds a peer to its network and starts the connectionManager
func (net *Network) ConnectToSeed(peerInfo network_model.PeerInfo) error {
	err := net.streamManager.connectPeer(peerInfo, ConnectionTypeOut)
	if err != nil {
		return err
	}
	net.peerManager.AddSeedByPeerInfo(peerInfo)
	return nil
}

//ConnectToSeedByString adds a peer by its full address string and starts the connectionManager
func (net *Network) ConnectToSeedByString(fullAddr string) error {

	peerInfo, err := network_model.NewPeerInfoFromString(fullAddr)
	if err != nil {
		logger.WithError(err).WithFields(logger.Fields{
			"full_addr": fullAddr,
		}).Warn("Network: create PeerInfo failed.")
		return err
	}

	return net.ConnectToSeed(peerInfo)
}

//AddSeed Add a seed peer to its network
func (net *Network) AddSeed(peerInfo network_model.PeerInfo) {
	net.peerManager.AddSeedByPeerInfo(peerInfo)
}

//updatePeers removes stale sync peers from peer manager
func (net *Network) updatePeers() {
	peers := net.streamManager.GetConnectedPeers()
	net.peerManager.UpdateSyncPeers(peers)
}

//startStreamMsgHandler starts a listening loop that listens to new message from all streams
func (net *Network) startStreamMsgHandler() {
	go func() {
		for {
			select {
			case msg := <-net.streamMsgRcvCh:

				if net.isNetworkRadiation(msg.Packet) {
					continue
				}
				net.recordMessage(msg.Packet)
				select {
				case net.streamMsgDispatcherCh <- msg:
				default:
					logger.WithFields(logger.Fields{
						"dispatcherCh_len": len(net.streamMsgDispatcherCh),
					}).Warn("Network: message streamMsgDispatcherCh channel full! Message disgarded")
					return
				}
			}
		}
	}()
}

//startPeerConnectionSchedule trys to connect to seed peers periodically
func (net *Network) startPeerConnectionSchedule() {
	go func() {
		ticker := time.NewTicker(PeerConnectionInterval)
		for {
			select {
			case <-ticker.C:
				net.connectToAllPeers()
				net.updatePeers()
			}
		}
	}()
}

//connectToAllPeers first connect to all seeds and then connect to sync peers
func (net *Network) connectToAllPeers() {
	net.connectToSeeds()
	net.connectToSyncPeers()
}

//connectToSeeds connects to all seeds
func (net *Network) connectToSeeds() {
	net.streamManager.ConnectPeers(net.peerManager.GetSeeds())
}

//ConnectToSyncPeers connects to sync peers
func (net *Network) connectToSyncPeers() {
	net.streamManager.ConnectPeers(net.peerManager.GetSyncPeers())
}

//isNetworkRadiation decides if a message is a network radiation (a message that it has received already)
func (net *Network) isNetworkRadiation(msg *network_model.DappPacket) bool {
	return msg.IsBroadcast() && net.recentlyRcvdDapMsgs.Contains(string(msg.GetRawBytes()))
}

//recordMessage records a message that is already received or sent
func (net *Network) recordMessage(msg *network_model.DappPacket) {
	net.recentlyRcvdDapMsgs.Add(string(msg.GetRawBytes()), true)
}

//onStreamStop runs cb function upon any stream stops
func (net *Network) onStreamStop(stream *Stream) {
	net.onStreamStopCb(stream)

	if len(net.streamManager.GetStreams()) == 0 {
		logger.Info("Network: The network has no streams. Attempt to connect to all peers again...")
		net.connectToAllPeers()
	}
}

//onPeerListReceived connects to new peers when a peer list is received
func (net *Network) onPeerListReceived(newPeers []network_model.PeerInfo) {
	net.streamManager.ConnectPeers(newPeers)
}

//onStreamConnected adds a peer in peer manager when a stream is connected
func (net *Network) onStreamConnected(stream *Stream) {
	net.peerManager.AddSyncPeer(stream.peerInfo)
}
