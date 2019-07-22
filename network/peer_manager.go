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
	"context"
	"github.com/asaskevich/EventBus"
	"github.com/dappley/go-dappley/network/network_model"
	"github.com/libp2p/go-libp2p-core/network"
	"math/rand"
	"sync"
	"time"

	"github.com/dappley/go-dappley/network/pb"
	"github.com/dappley/go-dappley/storage"
	"github.com/golang/protobuf/proto"
	"github.com/libp2p/go-libp2p-core/peer"
	pstore "github.com/libp2p/go-libp2p-core/peerstore"
	ma "github.com/multiformats/go-multiaddr"
	logger "github.com/sirupsen/logrus"
)

type ConnectionType int

const (
	syncPeersKey = "SyncPeers"

	ConnectionTypeSeed ConnectionType = 0
	ConnectionTypeIn   ConnectionType = 1
	ConnectionTypeOut  ConnectionType = 2

	maxSyncPeersCount            = 32
	defaultMaxConnectionOutCount = 16
	defaultMaxConnectionInCount  = 128
	syncPeersWaitTime            = 10 * time.Second
	syncPeersScheduleTime        = 10 * time.Minute
	checkSeedsConnectionTime     = 15 * time.Minute

	topicStreamStop     = "StreamStop"
	GetPeerListRequest  = "GetPeerListRequest"
	GetPeerListResponse = "GetPeerListResponse"
)

var (
	subscribedTopics = []string{
		GetPeerListRequest,
		GetPeerListResponse,
	}
)

type onStreamStopFunc func(stream *Stream)

type PeerManager struct {
	host      *network_model.Host
	seeds     map[peer.ID]*network_model.PeerInfo
	syncPeers map[peer.ID]*network_model.PeerInfo

	streams            map[peer.ID]*StreamInfo
	connectionConfig   network_model.PeerConnectionConfig
	connectionOutCount int //Connection that current node connect to other nodes, exclude seed nodes
	connectionInCount  int //Connection that other node connection to current node.

	syncPeerContext *SyncPeerContext

	streamStopNotificationCh chan *Stream
	streamMsgReceiveCh       chan *network_model.DappPacketContext
	commandSendCh            chan *network_model.DappSendCmdContext
	eventNotifier            EventBus.Bus
	db                       Storage

	mutex sync.RWMutex
}

//NewPeerManager create a new peer manager object
func NewPeerManager(config network_model.PeerConnectionConfig, streamMessageReceiveCh chan *network_model.DappPacketContext, db Storage) *PeerManager {

	if config.GetMaxConnectionOutCount() == 0 {
		config.SetMaxConnectionOutCount(defaultMaxConnectionOutCount)
	}

	if config.GetMaxConnectionInCount() == 0 {
		config.SetMaxConnectionInCount(defaultMaxConnectionInCount)
	}

	return &PeerManager{
		seeds:                    make(map[peer.ID]*network_model.PeerInfo),
		syncPeers:                make(map[peer.ID]*network_model.PeerInfo),
		streams:                  make(map[peer.ID]*StreamInfo),
		mutex:                    sync.RWMutex{},
		connectionConfig:         config,
		streamMsgReceiveCh:       streamMessageReceiveCh,
		commandSendCh:            nil,
		streamStopNotificationCh: make(chan *Stream, 10),
		eventNotifier:            EventBus.New(),
		db:                       db,
	}
}

//GetSubscribedTopics returns subscribed topics
func (pm *PeerManager) GetSubscribedTopics() []string {
	return subscribedTopics
}

//SetCommandSendCh sets the command send channel
func (pm *PeerManager) SetCommandSendCh(commandSendCh chan *network_model.DappSendCmdContext) {
	pm.commandSendCh = commandSendCh
}

//GetCommandHandler returns the corresponding command handler
func (pm *PeerManager) GetCommandHandler(cmd string) network_model.CommandHandlerFunc {
	switch cmd {
	case GetPeerListRequest:
		return pm.GetPeerListRequestHandler
	case GetPeerListResponse:
		return pm.GetPeerListResponseHandler
	}
	return nil
}

//AddSeeds add seed peers
func (pm *PeerManager) AddSeeds(seeds []string) {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	for _, seed := range seeds {
		pm.addSeedByString(seed)
	}
}

//addSeedByString adds seed peer by multiaddr string
func (pm *PeerManager) addSeedByString(fullAddr string) {

	peerInfo, err := network_model.NewPeerInfoFromString(fullAddr)
	if err != nil {
		logger.WithError(err).WithFields(logger.Fields{
			"full_addr": fullAddr,
		}).Warn("PeerManager: create PeerInfo failed.")
	}

	pm.seeds[peerInfo.PeerId] = peerInfo
}

//AddSeedByPeerInfo adds seed by peerInfo
func (pm *PeerManager) AddSeedByPeerInfo(peerInfo *network_model.PeerInfo) error {
	pm.seeds[peerInfo.PeerId] = peerInfo
	return nil
}

//AddAndConnectPeer adds a peer and starts a connection right away
func (pm *PeerManager) AddAndConnectPeer(peerInfo *network_model.PeerInfo) error {

	logger.Info("PeerManager: AddAndConnectPeer")

	_, err := pm.connectPeer(peerInfo, ConnectionTypeOut)
	if err != nil {
		logger.WithError(err).WithFields(logger.Fields{
			"peerId": peerInfo.PeerId,
		}).Warn("PeerManager: connect PeerInfo failed.")
	}
	return err
}

//Start starts peer manager thread
func (pm *PeerManager) Start(host *network_model.Host, seeds []string) {
	pm.host = host
	pm.AddSeeds(seeds)
	pm.loadSyncPeers()
	pm.startConnectSeeds()
	pm.startConnectSyncPeers()
	pm.startSyncPeersSchedule()
	pm.checkSeedsConnectionSchedule()
	pm.StartExitListener()
}

//SubscribeOnStreamStop allows another object to subscribe to stream stop event
func (pm *PeerManager) SubscribeOnStreamStop(cb onStreamStopFunc) {
	pm.eventNotifier.SubscribeAsync(topicStreamStop, cb, false)
}

//StartExitListener starts the exit listener loop
func (pm *PeerManager) StartExitListener() {
	go func() {
		for {
			if s, ok := <-pm.streamStopNotificationCh; ok {
				pm.OnStreamStop(s)
				pm.eventNotifier.Publish(topicStreamStop, s)
			}
		}
	}()
}

//startConnectSeeds connects to all seed peers
func (pm *PeerManager) startConnectSeeds() {
	unConnectedSeeds := pm.getUnConnectedSeeds()

	wg := sync.WaitGroup{}
	wg.Add(len(unConnectedSeeds))
	pm.doConnectSeeds(&wg, unConnectedSeeds)
	wg.Wait()
}

//getUnConnectedSeeds returns all unconnected seed peers
func (pm *PeerManager) getUnConnectedSeeds() []*network_model.PeerInfo {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()

	var unConnectedSeeds []*network_model.PeerInfo

	for _, seed := range pm.seeds {
		if _, ok := pm.streams[seed.PeerId]; !ok {
			unConnectedSeeds = append(unConnectedSeeds, seed)
		}
	}

	return unConnectedSeeds
}

//Broadcast sends a DappPacket to all peers
func (pm *PeerManager) Broadcast(packet *network_model.DappPacket, priority network_model.DappCmdPriority) {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()
	for _, s := range pm.streams {
		s.stream.Send(packet, priority)
	}
}

//Unicast sends a DappPacket to a peer indicated by "pid"
func (pm *PeerManager) Unicast(packet *network_model.DappPacket, pid peer.ID, priority network_model.DappCmdPriority) {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()

	streamInfo, ok := pm.streams[pid]
	if !ok {
		logger.WithFields(logger.Fields{
			"pid": pid,
		}).Warn("PeerManager: Unicast pid not found.")
		return
	}

	streamInfo.stream.Send(packet, priority)
}

//ReceivePeers adds one of its peers' peerlist to its own peerlist
func (pm *PeerManager) ReceivePeers(peerId peer.ID, peers []*network_model.PeerInfo) {
	pm.addSyncPeersResult(peerId, peers)

	if pm.isSyncPeerFinish() {
		pm.collectSyncPeersResult()
		pm.saveSyncPeers()
		go func() {
			pm.startConnectSyncPeers()
		}()
	}
}

//StreamHandler starts a Stream object when a stream is established (when a peer initiates the connection to the local node)
func (pm *PeerManager) StreamHandler(s network.Stream) {

	stream := NewStream(s)
	stream.Start(pm.streamStopNotificationCh, pm.streamMsgReceiveCh)

	logger.WithFields(logger.Fields{
		"peer_id": stream.GetPeerId(),
		"addr":    stream.GetRemoteAddr().String(),
	}).Info("PeerManager: Add Stream")

	connectionType := pm.getStreamConnectionType(stream)
	if !pm.checkAndAddStream(stream.GetPeerId(), connectionType, stream) {
		stream.StopStream(nil)
	}
}

//OnStreamStop removes the stream from its peer list
func (pm *PeerManager) OnStreamStop(stream *Stream) {

	logger.WithFields(logger.Fields{
		"peer_id": stream.GetPeerId(),
		"addr":    stream.GetRemoteAddr().String(),
	}).Info("PeerManager: Stop Stream")

	pm.mutex.Lock()
	streamInfo, ok := pm.streams[stream.GetPeerId()]
	if !ok || streamInfo.stream != stream {
		pm.mutex.Unlock()
		return
	}

	switch streamInfo.connectionType {
	case ConnectionTypeIn:
		pm.connectionInCount--

	case ConnectionTypeOut:
		pm.connectionOutCount--

	default:
		//pass
	}
	delete(pm.streams, stream.peerInfo.PeerId)
	pm.host.Peerstore().ClearAddrs(stream.GetPeerId())
	streamLen := len(pm.streams)
	pm.mutex.Unlock()
	if streamLen == 0 {
		go func() {
			pm.startConnectSeeds()
			pm.startConnectSyncPeers()
		}()
	}
}

//StopAllStreams stops all streams
func (pm *PeerManager) StopAllStreams(err error) {
	for _, streamInfo := range pm.streams {
		streamInfo.stream.StopStream(err)
	}
}

//GetRandomConnectedPeers get a number of random connected peers
func (pm *PeerManager) GetRandomConnectedPeers(number int) []*network_model.PeerInfo {
	streams := pm.CloneStreamsToSlice()
	chooseStreams := randomChooseStreams(number, streams)
	peers := make([]*network_model.PeerInfo, len(chooseStreams))

	for i, streamInfo := range chooseStreams {
		peers[i] = &network_model.PeerInfo{PeerId: streamInfo.stream.GetPeerId(), Addrs: []ma.Multiaddr{streamInfo.stream.GetRemoteAddr()}}
	}
	return peers
}

//doConnectSeeds connects to all its seed peers
func (pm *PeerManager) doConnectSeeds(wg *sync.WaitGroup, peers []*network_model.PeerInfo) {

	logger.WithFields(logger.Fields{
		"num_of_peers": len(peers),
	}).Info("PeerManager: Connect seed peers")

	for _, peerInfo := range peers {
		currentPeer := peerInfo
		go func() {
			pm.connectPeer(currentPeer, ConnectionTypeSeed)
			wg.Done()
		}()
	}
}

//startConnectSyncPeers connects to random peers
func (pm *PeerManager) startConnectSyncPeers() {

	logger.WithFields(logger.Fields{
		"num_of_peers": len(pm.syncPeers),
	}).Info("PeerManager: Connect sync peers")

	if len(pm.syncPeers) == 0 {
		return
	}

	leftConnectionOut := pm.connectionConfig.GetMaxConnectionOutCount() - pm.connectionOutCount
	if leftConnectionOut < 0 {
		return
	}

	toCheckPeers := pm.cloneUnconnectedSyncPeersToSlice()
	randomChoosePeers := randomChoosePeers(leftConnectionOut, toCheckPeers)
	wg := &sync.WaitGroup{}
	wg.Add(len(randomChoosePeers))

	logger.WithFields(logger.Fields{
		"maxConnectionOutCount":     pm.connectionConfig.GetMaxConnectionOutCount(),
		"connectionOutCount":        pm.connectionOutCount,
		"num_of_reconnecting_peers": len(randomChoosePeers),
	}).Info("PeerManager: Connect sync peers")

	for _, peerInfo := range randomChoosePeers {
		currentPeer := peerInfo
		go func() {
			stream, err := pm.connectPeer(currentPeer, ConnectionTypeOut)
			if stream == nil && err == nil {
				pm.removeStaleSyncPeer(peerInfo.PeerId)
			}
			wg.Done()
		}()
	}
	wg.Wait()
}

//startSyncPeersSchedule starts synchronize peer list with its peers
func (pm *PeerManager) startSyncPeersSchedule() {
	// Start first sync peers task
	go func() {
		pm.startSyncPeers()
	}()

	go func() {
		ticker := time.NewTicker(syncPeersScheduleTime)
		for {
			select {
			case <-ticker.C:
				pm.startSyncPeers()
			}
		}
	}()
}

//checkSeedsConnectionSchedule trys to connect to seed peers periodically
func (pm *PeerManager) checkSeedsConnectionSchedule() {
	go func() {
		ticker := time.NewTicker(checkSeedsConnectionTime)
		for {
			select {
			case <-ticker.C:
				pm.startConnectSeeds()
			}
		}
	}()
}

//startSyncPeers starts synchronize its peers' peerlists to its own
func (pm *PeerManager) startSyncPeers() {
	if pm.syncPeerContext != nil {
		logger.Info("PeerManager: another sync is running.")
		return
	}

	pm.createSyncContext()

	pm.SendSyncPeersRequest()

	syncTimer := time.NewTimer(syncPeersWaitTime)
	go func() {
		<-syncTimer.C
		syncTimer.Stop()
		if pm.collectSyncPeersResult() {
			pm.saveSyncPeers()
			pm.startConnectSyncPeers()
		}
	}()
}

//createSyncContext creates a context for synchronizing peer info
func (pm *PeerManager) createSyncContext() {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	pm.syncPeerContext = &SyncPeerContext{
		checkingStreams: make(map[peer.ID]*StreamInfo),
		newPeers:        make(map[peer.ID]*network_model.PeerInfo),
	}

	for key, streamInfo := range pm.streams {
		pm.syncPeerContext.checkingStreams[key] = streamInfo
	}
}

//addSyncPeersResult add the received peer list to its own peer list
func (pm *PeerManager) addSyncPeersResult(peerId peer.ID, peers []*network_model.PeerInfo) bool {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	if pm.syncPeerContext == nil {
		logger.Info("PeerManager: no sync peers task is running.")
		return false
	}

	if _, ok := pm.syncPeerContext.checkingStreams[peerId]; !ok {
		logger.WithFields(logger.Fields{
			"pid": peerId,
		}).Info("PeerManager: PeerId not in check list.")
		return false
	}

	delete(pm.syncPeerContext.checkingStreams, peerId)

	for _, peerInfo := range peers {
		if peerInfo.PeerId == pm.host.GetPeerInfo().PeerId {
			continue
		}

		if _, ok := pm.seeds[peerInfo.PeerId]; ok {
			continue
		}

		if _, ok := pm.syncPeerContext.newPeers[peerInfo.PeerId]; ok {
			continue
		}

		pm.syncPeerContext.newPeers[peerInfo.PeerId] = peerInfo
	}
	return true
}

//isSyncPeerFinish checks if the peer synchronization process is finished
func (pm *PeerManager) isSyncPeerFinish() bool {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	if pm.syncPeerContext == nil {
		return false
	}

	return len(pm.syncPeerContext.checkingStreams) == 0
}

func (pm *PeerManager) collectSyncPeersResult() bool {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	if pm.syncPeerContext == nil {
		logger.Info("PeerManager: no sync peers task is running.")
		return false
	}

	pm.syncPeers = pm.syncPeerContext.newPeers
	// Copy connected stream to syncPeers
	for key, streamInfo := range pm.streams {
		if _, ok := pm.syncPeers[key]; ok {
			continue
		}

		if streamInfo.connectionType == ConnectionTypeSeed {
			continue
		}

		pm.syncPeers[key] = &network_model.PeerInfo{PeerId: key, Addrs: []ma.Multiaddr{streamInfo.stream.GetRemoteAddr()}}

		logger.WithFields(logger.Fields{
			"peer_id": pm.syncPeers[key].PeerId,
			"addr":    pm.syncPeers[key].Addrs[0].String(),
		}).Infof("PeerManager: Collect sync peers")
	}

	logger.WithFields(logger.Fields{
		"num_of_peers": len(pm.syncPeers),
	}).Infof("PeerManager: Collect sync peers")

	pm.syncPeerContext = nil
	return true
}

func (pm *PeerManager) saveSyncPeers() {
	syncPeers := pm.cloneSyncPeers()

	var peerPbs []*networkpb.PeerInfo
	for _, peerInfo := range syncPeers {
		peerPbs = append(peerPbs, peerInfo.ToProto().(*networkpb.PeerInfo))
	}

	bytes, err := proto.Marshal(&networkpb.ReturnPeerList{PeerList: peerPbs})
	if err != nil {
		logger.WithError(err).Info("PeerManager: serialize sync peers failed.")
	}

	err = pm.db.Put([]byte(syncPeersKey), bytes)
	if err != nil {
		logger.WithError(err).Info("PeerManager: save sync peers failed.")
	}
}

func (pm *PeerManager) removeStaleSyncPeer(peerId peer.ID) {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()
	delete(pm.syncPeers, peerId)
}

func (pm *PeerManager) cloneSyncPeers() map[peer.ID]*network_model.PeerInfo {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()
	peers := make(map[peer.ID]*network_model.PeerInfo)

	for key, peerInfo := range pm.syncPeers {
		peers[key] = peerInfo
	}

	return peers
}

func (pm *PeerManager) cloneUnconnectedSyncPeersToSlice() []*network_model.PeerInfo {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()
	var peers []*network_model.PeerInfo

	for key, peerInfo := range pm.syncPeers {
		// Skip connected peers
		if _, ok := pm.streams[key]; ok {
			continue
		}

		peers = append(peers, peerInfo)
	}

	return peers
}

func (pm *PeerManager) CloneStreamsToSlice() []*StreamInfo {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()
	var streams []*StreamInfo

	for _, streamInfo := range pm.streams {
		streams = append(streams, streamInfo)
	}

	return streams
}

func (pm *PeerManager) CloneStreamsToPeerInfoSlice() []*network_model.PeerInfo {
	streams := pm.CloneStreamsToSlice()
	peers := make([]*network_model.PeerInfo, len(streams))

	for i, streamInfo := range streams {
		peers[i] = &network_model.PeerInfo{PeerId: streamInfo.stream.GetPeerId(), Addrs: []ma.Multiaddr{streamInfo.stream.GetRemoteAddr()}}
	}

	return peers
}

func randomChoosePeers(number int, peers []*network_model.PeerInfo) []*network_model.PeerInfo {
	if number >= len(peers) {
		return peers
	}

	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	r.Shuffle(len(peers), func(i, j int) {
		peers[i], peers[j] = peers[j], peers[i]
	})
	return peers[0:number]
}

func randomChooseStreams(number int, streams []*StreamInfo) []*StreamInfo {
	if number >= len(streams) {
		return streams
	}

	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	r.Shuffle(len(streams), func(i, j int) {
		streams[i], streams[j] = streams[j], streams[i]
	})
	return streams[0:number]
}

func (pm *PeerManager) connectPeer(peerInfo *network_model.PeerInfo, connectionType ConnectionType) (*Stream, error) {
	if pm.isStreamExist(peerInfo.PeerId) {
		logger.WithFields(logger.Fields{
			"PeerId": peerInfo.PeerId,
		}).Info("PeerManager: Stream exist.")
		return nil, nil
	}

	logger.WithFields(logger.Fields{
		"PeerId": peerInfo.PeerId,
		"Addr":   peerInfo.Addrs[0].String(),
	}).Info("PeerManager: Connect peer information.")

	pm.host.Peerstore().AddAddrs(peerInfo.PeerId, peerInfo.Addrs, pstore.PermanentAddrTTL)
	// make a new stream
	stream, err := pm.host.NewStream(context.Background(), peerInfo.PeerId, network_model.ProtocalName)
	if err != nil {
		logger.WithError(err).WithFields(logger.Fields{
			"PeerId": peerInfo.PeerId,
		}).Info("PeerManager: Connect to peer failed")
		return nil, err
	}

	peerStream := NewStream(stream)
	logger.WithFields(logger.Fields{
		"PeerId": peerStream.GetPeerId(),
		"Addr":   peerStream.GetRemoteAddr().String(),
	}).Info("PeerManager: Connect peer actual stream")
	if !pm.checkAndAddStream(peerInfo.PeerId, connectionType, peerStream) {
		peerStream.StopStream(nil)
		return nil, nil
	}

	peerStream.Start(pm.streamStopNotificationCh, pm.streamMsgReceiveCh)
	return peerStream, nil
}

func (pm *PeerManager) getStreamConnectionType(stream *Stream) ConnectionType {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()

	if _, ok := pm.seeds[stream.GetPeerId()]; ok {
		return ConnectionTypeSeed
	}
	return ConnectionTypeIn
}

func (pm *PeerManager) checkAndAddStream(peerId peer.ID, connectionType ConnectionType, stream *Stream) bool {

	logger.Info("PeerManager: checkAndAddStream")

	pm.mutex.Lock()
	defer pm.mutex.Unlock()
	_, ok := pm.streams[peerId]
	if ok {
		return false
	}

	switch connectionType {
	case ConnectionTypeIn:
		if pm.connectionInCount >= pm.connectionConfig.GetMaxConnectionInCount() {
			logger.Info("PeerManager: connection in is full.")
			return false
		}
		pm.connectionInCount++
	case ConnectionTypeOut:
		if pm.connectionOutCount >= pm.connectionConfig.GetMaxConnectionOutCount() {
			logger.Info("PeerManager: connection out is full.")
			return false
		}
		pm.connectionOutCount++

	default:
		//Pass
	}
	pm.streams[peerId] = &StreamInfo{stream: stream, connectionType: connectionType}

	return true
}

func (pm *PeerManager) loadSyncPeers() error {

	peersBytes, err := pm.db.Get([]byte(syncPeersKey))

	if err != nil {
		logger.WithError(err).Warn("PeerManager: load sync peers database failed.")
		if err == storage.ErrKeyInvalid {
			return nil
		}
		return err
	}

	peerListPb := &networkpb.ReturnPeerList{}

	if err := proto.Unmarshal(peersBytes, peerListPb); err != nil {
		logger.WithError(err).Warn("PeerManager: parse Peerlist failed.")
	}

	for _, peerPb := range peerListPb.GetPeerList() {
		peerInfo := &network_model.PeerInfo{}
		if err := peerInfo.FromProto(peerPb); err != nil {
			logger.WithError(err).Warn("PeerManager: parse PeerInfo failed.")
		}

		pm.syncPeers[peerInfo.PeerId] = peerInfo
		logger.WithFields(logger.Fields{
			"peer_id": peerInfo.PeerId,
			"addr":    peerInfo.Addrs[0].String(),
		}).Info("loadSyncPeers")
	}

	logger.WithError(err).Warnf("PeerManager: load sync peers count %v.", len(pm.syncPeers))

	return nil
}

func (pm *PeerManager) isStreamExist(peerId peer.ID) bool {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()
	_, ok := pm.streams[peerId]
	return ok
}

func (pm *PeerManager) SendSyncPeersRequest() {
	getPeerListPb := &networkpb.GetPeerList{
		MaxNumber: int32(maxSyncPeersCount),
	}

	var destination peer.ID
	command := network_model.NewDappSendCmdContext(GetPeerListRequest, getPeerListPb, destination, network_model.Broadcast, network_model.HighPriorityCommand)

	command.Send(pm.commandSendCh)
}

func (pm *PeerManager) SendPeerListMessage(maxNumOfPeers int, destination peer.ID) {

	peers := pm.GetRandomConnectedPeers(maxNumOfPeers)
	var peerPbs []*networkpb.PeerInfo
	for _, peerInfo := range peers {
		peerPbs = append(peerPbs, peerInfo.ToProto().(*networkpb.PeerInfo))
	}

	peerList := &networkpb.ReturnPeerList{PeerList: peerPbs}

	command := network_model.NewDappSendCmdContext(GetPeerListResponse, peerList, destination, network_model.Unicast, network_model.HighPriorityCommand)

	command.Send(pm.commandSendCh)
}

func (pm *PeerManager) GetPeerListRequestHandler(command *network_model.DappRcvdCmdContext) {

	getPeerlistRequest := &networkpb.GetPeerList{}

	//unmarshal byte to proto
	if err := proto.Unmarshal(command.GetData(), getPeerlistRequest); err != nil {
		logger.WithError(err).Warn("Node: parse GetPeerListRequest failed.")
	}

	pm.SendPeerListMessage(int(getPeerlistRequest.GetMaxNumber()), command.GetSource())
}

func (pm *PeerManager) GetPeerListResponseHandler(command *network_model.DappRcvdCmdContext) {
	peerlistPb := &networkpb.ReturnPeerList{}

	if err := proto.Unmarshal(command.GetData(), peerlistPb); err != nil {
		logger.WithError(err).Warn("PeerManager: parse Peerlist failed.")
	}

	var peers []*network_model.PeerInfo
	for _, peerPb := range peerlistPb.GetPeerList() {
		peerInfo := &network_model.PeerInfo{}
		if err := peerInfo.FromProto(peerPb); err != nil {
			logger.WithError(err).Warn("PeerManager: parse PeerInfo failed.")
		}
		peers = append(peers, peerInfo)
	}

	pm.ReceivePeers(command.GetSource(), peers)
}
