package network

import (
	"context"
	"errors"
	"github.com/dappley/go-dappley/network/network_model"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/peerstore"
	ma "github.com/multiformats/go-multiaddr"
	logger "github.com/sirupsen/logrus"
	"math/rand"
	"sync"
	"time"
)

var (
	ErrConnectionsFull        = errors.New("Connection is full")
	ErrStreamAlreadyConnected = errors.New("Stream is already connected")
)

type OnStreamCbFunc func(stream *Stream)

type StreamManager struct {
	host                     *network_model.Host
	streams                  map[peer.ID]*StreamInfo
	connectionManager        *ConnectionManager
	streamStopNotificationCh chan *Stream
	streamMsgReceiveCh       chan *network_model.DappPacketContext
	onStreamStopCb           OnStreamCbFunc
	onStreamConnectedCb      OnStreamCbFunc

	mutex sync.RWMutex
}

//NewStreamManager creates a new StreamManager instance
func NewStreamManager(config network_model.PeerConnectionConfig, streamMessageReceiveCh chan *network_model.DappPacketContext, onStreamStopCb OnStreamCbFunc, onStreamConnectedCb OnStreamCbFunc) *StreamManager {

	return &StreamManager{
		streams:                  make(map[peer.ID]*StreamInfo),
		connectionManager:        NewConnectionManager(config),
		streamMsgReceiveCh:       streamMessageReceiveCh,
		streamStopNotificationCh: make(chan *Stream, 10),
		onStreamStopCb:           onStreamStopCb,
		onStreamConnectedCb:      onStreamConnectedCb,
		mutex:                    sync.RWMutex{},
	}
}

//Start starts the host and stream service
func (sm *StreamManager) Start(host *network_model.Host) {
	sm.host = host
	sm.StartStreamStopListener()
}

//GetStreams returns all currently connected streams
func (sm *StreamManager) GetStreams() map[peer.ID]*StreamInfo { return sm.streams }

//StartStreamStopListener starts the stream stop listener loop
func (sm *StreamManager) StartStreamStopListener() {
	go func() {
		for {
			if s, ok := <-sm.streamStopNotificationCh; ok {
				sm.OnStreamStop(s)
				if sm.onStreamStopCb != nil {
					go sm.onStreamStopCb(s)
				}
			}
		}
	}()
}

//Broadcast sends a DappPacket to all peers
func (sm *StreamManager) Broadcast(packet *network_model.DappPacket, priority network_model.DappCmdPriority) {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()
	for _, s := range sm.streams {
		s.stream.Send(packet, priority)
	}
}

//Unicast sends a DappPacket to a peer indicated by "pid"
func (sm *StreamManager) Unicast(packet *network_model.DappPacket, pid peer.ID, priority network_model.DappCmdPriority) {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	streamInfo, ok := sm.streams[pid]
	if !ok {
		logger.WithFields(logger.Fields{
			"pid": pid,
		}).Warn("StreamManager: Unicast pid not found.")
		return
	}

	streamInfo.stream.Send(packet, priority)
}

//StreamHandler starts a Stream object when a stream is established (when a peer initiates the connectionManager to the local node)
func (sm *StreamManager) StreamHandler(s network.Stream) {

	stream := NewStream(s)
	peerId := stream.GetPeerId()

	logger.WithFields(logger.Fields{
		"peer_id": stream.GetPeerId(),
		"addr":    stream.GetRemoteAddr(),
	}).Info("StreamManager: New stream connection has been received!")

	if sm.isStreamConnected(peerId) {
		logger.Warn("StreamManager: Stream is already connected")
		stream.StopStream()
		return
	}

	if sm.connectionManager.IsConnectionFull(ConnectionTypeIn) {
		logger.Warn("StreamManager: Connection has reached its limit")
		stream.StopStream()
		return
	}

	stream.Start(sm.streamStopNotificationCh, sm.streamMsgReceiveCh)

	sm.addStream(stream, ConnectionTypeIn)

	sm.onStreamConnectedCb(stream)
}

//addStream records a new stream information
func (sm *StreamManager) addStream(stream *Stream, connectionType ConnectionType) {
	if sm.isStreamConnected(stream.GetPeerId()) {
		return
	}
	sm.streams[stream.GetPeerId()] = &StreamInfo{stream: stream, connectionType: connectionType}
	sm.connectionManager.AddConnection(connectionType)
}

//removeStream deletes information of a stream
func (sm *StreamManager) removeStream(stream *Stream) {
	if !sm.isStreamConnected(stream.GetPeerId()) {
		return
	}
	sm.connectionManager.RemoveConnection(sm.streams[stream.GetPeerId()].connectionType)
	delete(sm.streams, stream.GetPeerId())
}

func (sm *StreamManager) isStreamConnected(peerId peer.ID) bool {
	_, isConnected := sm.streams[peerId]
	return isConnected
}

//ConnectPeers connect to multiple peers
func (sm *StreamManager) ConnectPeers(peers []network_model.PeerInfo) {

	numOfPeersToBeConnected := len(peers)

	logger.WithFields(logger.Fields{
		"numOfPeers": numOfPeersToBeConnected,
	}).Debug("StreamManager: ConnectPeers")

	numOfPeersAllowed := sm.connectionManager.GetNumOfConnectionsAllowed(ConnectionTypeOut)
	if numOfPeersAllowed < numOfPeersToBeConnected {
		numOfPeersToBeConnected = numOfPeersAllowed
	}

	peers = ShufflePeers(peers)

	for _, peer := range peers {

		if numOfPeersToBeConnected <= 0 {
			return
		}

		if err := sm.connectPeer(peer, ConnectionTypeOut); err == nil {
			numOfPeersToBeConnected--
		}
	}

}

//connectPeer connects to a peer
func (sm *StreamManager) connectPeer(peerInfo network_model.PeerInfo, connectionType ConnectionType) error {

	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	if peerInfo.PeerId.String() == "" {
		return nil
	}

	if sm.isStreamConnected(peerInfo.PeerId) {
		return ErrStreamAlreadyConnected
	}

	if sm.connectionManager.IsConnectionFull(connectionType) {
		return ErrConnectionsFull
	}

	sm.host.Peerstore().AddAddrs(peerInfo.PeerId, peerInfo.Addrs, peerstore.PermanentAddrTTL)
	s, err := sm.host.NewStream(context.Background(), peerInfo.PeerId, network_model.ProtocalName)
	if err != nil {
		logger.WithError(err).WithFields(logger.Fields{
			"PeerId": peerInfo.PeerId,
		}).Warn("PeerManager: Connect to peer failed")
		return err
	}

	stream := NewStream(s)

	logger.WithFields(logger.Fields{
		"PeerId": peerInfo.PeerId,
		"Addr":   peerInfo.Addrs[0].String(),
	}).Info("StreamManager: Connect to a peer")

	stream.Start(sm.streamStopNotificationCh, sm.streamMsgReceiveCh)
	sm.addStream(stream, connectionType)

	return nil
}

//onStreamStop removes the stream from its peer list
func (sm *StreamManager) OnStreamStop(stream *Stream) {

	logger.WithFields(logger.Fields{
		"peer_id": stream.GetPeerId(),
		"addr":    stream.GetRemoteAddr().String(),
	}).Debug("StreamManager: Stream is stopped")

	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	streamInfo, ok := sm.streams[stream.GetPeerId()]
	if !ok || streamInfo.stream != stream {
		return
	}

	sm.removeStream(stream)
	sm.host.Peerstore().ClearAddrs(stream.GetPeerId())
}

//Stop stops all streams
func (sm *StreamManager) Stop() {

	for _, streamInfo := range sm.streams {
		streamInfo.stream.StopStream()
	}

	sm.host.RemoveStreamHandler(network_model.ProtocalName)
	err := sm.host.Close()
	if err != nil {
		logger.WithError(err).Warn("StreamManager: host was not closed properly.")
	}
}

//GetConnectedPeers return a list of connected peers
func (sm *StreamManager) GetConnectedPeers() map[peer.ID]network_model.PeerInfo {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()
	peers := make(map[peer.ID]network_model.PeerInfo)

	for _, streamInfo := range sm.streams {
		peer := network_model.PeerInfo{PeerId: streamInfo.stream.GetPeerId(), Addrs: []ma.Multiaddr{streamInfo.stream.GetRemoteAddr()}}
		peers[peer.PeerId] = peer
	}

	return peers
}

//ShufflePeers shuffles the order in the PeerInfo slice
func ShufflePeers(peers []network_model.PeerInfo) []network_model.PeerInfo {

	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	r.Shuffle(len(peers), func(i, j int) {
		peers[i], peers[j] = peers[j], peers[i]
	})
	return peers
}
