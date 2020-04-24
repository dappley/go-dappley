package network

import (
	"github.com/dappley/go-dappley/common/log"
	"time"

	lru "github.com/hashicorp/golang-lru"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/peer"
	logger "github.com/sirupsen/logrus"

	"github.com/dappley/go-dappley/network/networkmodel"
)

type Network struct {
	streamManager         *StreamManager
	peerManager           *PeerManager
	streamMsgRcvCh        chan *networkmodel.DappPacketContext
	streamMsgDispatcherCh chan *networkmodel.DappPacketContext
	recentlyRcvdDapMsgs   *lru.Cache
	onStreamStopCb        OnStreamCbFunc
}

type NetworkContext struct {
	netService            NetService
	config                networkmodel.PeerConnectionConfig
	streamMsgDispatcherCh chan *networkmodel.DappPacketContext
	db                    Storage
	onStreamStopCb        OnStreamCbFunc
	seeds                 []string
}

//NewNetwork creates a network instance
func NewNetwork(netContext *NetworkContext) *Network {

	var err error

	net := &Network{
		streamMsgRcvCh:        make(chan *networkmodel.DappPacketContext, dispatchChLen),
		streamMsgDispatcherCh: netContext.streamMsgDispatcherCh,
		onStreamStopCb:        netContext.onStreamStopCb,
	}

	net.recentlyRcvdDapMsgs, err = lru.New(10240)
	net.streamManager = NewStreamManager(netContext.config, net.streamMsgRcvCh, net.onStreamStop, net.onStreamConnected)
	net.peerManager = NewPeerManager(netContext.netService, netContext.db, net.onPeerListReceived, netContext.seeds)

	if err != nil {
		logger.WithError(err).Panic("Network: Can not initialize lru cache for recentlyRcvdDapMsgs!")
	}

	return net
}

//GetConnectedPeers returns a list of peers in the network
func (net *Network) GetConnectedPeers() []networkmodel.PeerInfo {
	peers := net.streamManager.GetConnectedPeers()
	peersInSlice := []networkmodel.PeerInfo{}
	for _, peer := range peers {
		peersInSlice = append(peersInSlice, peer)
	}
	return peersInSlice
}

//GetHost returns a list of peers in the network
func (net *Network) GetHost() *networkmodel.Host {
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
	host := networkmodel.NewHost(listenPort, privKey, net.streamManager.StreamHandler)
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
func (net *Network) Unicast(data []byte, pid peer.ID, priority networkmodel.DappCmdPriority) {
	packet := networkmodel.ConstructDappPacketFromData(data, false)

	net.recordMessage(packet)
	net.streamManager.Unicast(packet, pid, priority)
}

//Broadcast sends a message to all peers
func (net *Network) Broadcast(data []byte, priority networkmodel.DappCmdPriority) {
	packet := networkmodel.ConstructDappPacketFromData(data, true)

	net.recordMessage(packet)
	net.streamManager.Broadcast(packet, priority)
}

//ConnectToSeed adds a peer to its network and starts the connectionManager
func (net *Network) ConnectToSeed(peerInfo networkmodel.PeerInfo) error {
	err := net.streamManager.connectPeer(peerInfo, ConnectionTypeOut)
	if err != nil {
		return err
	}
	net.peerManager.AddSeedByPeerInfo(peerInfo)
	return nil
}

//ConnectToSeedByString adds a peer by its full address string and starts the connectionManager
func (net *Network) ConnectToSeedByString(fullAddr string) error {

	peerInfo, err := networkmodel.NewPeerInfoFromString(fullAddr)
	if err != nil {
		logger.WithError(err).WithFields(logger.Fields{
			"full_addr": fullAddr,
		}).Warn("Network: create PeerInfo failed.")
		return err
	}

	return net.ConnectToSeed(peerInfo)
}

//AddSeed Add a seed peer to its network
func (net *Network) AddSeed(peerInfo networkmodel.PeerInfo) {
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
		defer log.CrashHandler()

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
		defer log.CrashHandler()

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
func (net *Network) isNetworkRadiation(msg *networkmodel.DappPacket) bool {
	return msg.IsBroadcast() && net.recentlyRcvdDapMsgs.Contains(string(msg.GetRawBytes()))
}

//recordMessage records a message that is already received or sent
func (net *Network) recordMessage(msg *networkmodel.DappPacket) {
	net.recentlyRcvdDapMsgs.Add(string(msg.GetRawBytes()), true)
}

//onStreamStop runs cb function upon any stream stops
func (net *Network) onStreamStop(stream *Stream) {
	net.onStreamStopCb(stream)
}

//onPeerListReceived connects to new peers when a peer list is received
func (net *Network) onPeerListReceived(newPeers []networkmodel.PeerInfo) {
	net.streamManager.ConnectPeers(newPeers)
}

//onStreamConnected adds a peer in peer manager when a stream is connected
func (net *Network) onStreamConnected(stream *Stream) {
	net.peerManager.AddSyncPeer(stream.peerInfo)
}
