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
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/libp2p/go-libp2p-core/peer"
	pstore "github.com/libp2p/go-libp2p-core/peerstore"
	"github.com/libp2p/go-libp2p/p2p/protocol/ping"
	ma "github.com/multiformats/go-multiaddr"
	logger "github.com/sirupsen/logrus"

	networkpb "github.com/dappley/go-dappley/network/pb"
	"github.com/dappley/go-dappley/storage"
)

type ConnectionType int

const (
	syncPeersKey = "SyncPeers"

	ConnectionTypeSeed ConnectionType = 0
	ConnectionTypeIn   ConnectionType = 1
	ConnectionTypeOut  ConnectionType = 2

	defaultMaxConnectionOutCount = 16
	defaultMaxConnectionInCount  = 128
	syncPeersWaitTime            = 10 * time.Second
	syncPeersScheduleTime        = 10 * time.Minute
	checkSeedsConnectionTime     = 15 * time.Minute
)

type PeerInfo struct {
	PeerId  peer.ID
	Addrs   []ma.Multiaddr
	Latency *float64 // rtt of ping in milliseconds
}

type StreamInfo struct {
	stream         *Stream
	connectionType ConnectionType
	latency        *float64 // refer to PeerInfo.Latency
}

type SyncPeerContext struct {
	checkingStreams map[peer.ID]*StreamInfo
	newPeers        map[peer.ID]*PeerInfo
}

type PingService struct {
	service *ping.PingService
	stop    chan bool
}

type PeerManager struct {
	seeds     map[peer.ID]*PeerInfo
	syncPeers map[peer.ID]*PeerInfo

	streams               map[peer.ID]*StreamInfo
	maxConnectionOutCount int
	connectionOutCount    int //Connection that current node connect to other nodes, exclude seed nodes
	maxConnectionInCount  int
	connectionInCount     int //Connection that other node connection to current node.

	syncPeerContext *SyncPeerContext

	mutex sync.RWMutex
	node  *Node
	ping  *PingService
}

func (pm *PeerManager) GetMaxConnectionInCount() int {
	return pm.maxConnectionInCount
}

func (pm *PeerManager) GetMaxConnectionOutCount() int {
	return pm.maxConnectionOutCount
}

func (pm *PeerManager) SetMaxConnectionInCount(maxConnectionInCount int) {
	pm.maxConnectionInCount = maxConnectionInCount
}

func (pm *PeerManager) SetMaxConnectionOutCount(maxConnectionOutCount int) {
	pm.maxConnectionOutCount = maxConnectionOutCount
}

func NewPeerManager(node *Node, config *NodeConfig) *PeerManager {

	maxConnectionOutCount := defaultMaxConnectionOutCount
	maxConnectionInCount := defaultMaxConnectionInCount

	if config != nil {
		if config.MaxConnectionOutCount != 0 {
			maxConnectionOutCount = config.MaxConnectionOutCount
		}

		if config.MaxConnectionInCount != 0 {
			maxConnectionInCount = config.MaxConnectionInCount
		}
	}

	return &PeerManager{
		seeds:                 make(map[peer.ID]*PeerInfo),
		syncPeers:             make(map[peer.ID]*PeerInfo),
		streams:               make(map[peer.ID]*StreamInfo),
		mutex:                 sync.RWMutex{},
		maxConnectionOutCount: maxConnectionOutCount,
		maxConnectionInCount:  maxConnectionInCount,
		node:                  node,
	}
}

func CreatePeerInfoFromMultiaddrs(targetFullAddrs []ma.Multiaddr) (*PeerInfo, error) {
	peerIds := make([]peer.ID, len(targetFullAddrs))
	addrs := make([]ma.Multiaddr, len(targetFullAddrs))
	for index, targetFullAddr := range targetFullAddrs {
		//get pid
		pid, err := targetFullAddr.ValueForProtocol(ma.P_IPFS)
		if err != nil {
			return nil, err
		}

		//get peer id
		peerId, err := peer.IDB58Decode(pid)
		if err != nil {
			return nil, err
		}

		peerIds[index] = peerId

		// Decapsulate the /ipfs/<peerID> part from the targetFullAddr
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

	peerInfo := &PeerInfo{
		PeerId: peerIds[0],
		Addrs:  addrs,
	}

	return peerInfo, nil
}

func (pm *PeerManager) AddSeedByString(fullAddr string) error {
	peerInfo, err := createPeerInfoFromString(fullAddr)
	if err != nil {
		logger.WithError(err).WithFields(logger.Fields{
			"full_addr": fullAddr,
		}).Warn("PeerManager: create PeerInfo failed.")
	}

	pm.seeds[peerInfo.PeerId] = peerInfo
	return nil
}

func (pm *PeerManager) AddSeedByPeerInfo(peerInfo *PeerInfo) error {
	pm.seeds[peerInfo.PeerId] = peerInfo
	return nil
}

func (pm *PeerManager) AddAndConnectPeerByString(fullAddr string) error {

	logger.Info("PeerManager: AddAndConnectPeerByString")

	peerInfo, err := createPeerInfoFromString(fullAddr)
	if err != nil {
		logger.WithError(err).WithFields(logger.Fields{
			"full_addr": fullAddr,
		}).Warn("PeerManager: create PeerInfo failed.")
	}

	_, err = pm.connectPeer(peerInfo, ConnectionTypeOut)
	if err != nil {
		logger.WithError(err).WithFields(logger.Fields{
			"full_addr": fullAddr,
		}).Warn("PeerManager: connect PeerInfo failed.")
	}
	return err
}

func (pm *PeerManager) AddAndConnectPeer(peerInfo *PeerInfo) error {

	logger.Info("PeerManager: AddAndConnectPeer")

	_, err := pm.connectPeer(peerInfo, ConnectionTypeOut)
	if err != nil {
		logger.WithError(err).WithFields(logger.Fields{
			"peerId": peerInfo.PeerId,
		}).Warn("PeerManager: connect PeerInfo failed.")
	}
	return err
}

func (pm *PeerManager) Start() {
	pm.loadSyncPeers()
	pm.startConnectSeeds()
	pm.startConnectSyncPeers()
	pm.startSyncPeersSchedule()
	pm.checkSeedsConnectionSchedule()
}

func (pm *PeerManager) startConnectSeeds() {
	unConnectedSeeds := pm.getUnConnectedSeeds()

	wg := sync.WaitGroup{}
	wg.Add(len(unConnectedSeeds))
	pm.doConnectSeeds(&wg, unConnectedSeeds)
	wg.Wait()
}

func (pm *PeerManager) getUnConnectedSeeds() []*PeerInfo {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()

	var unConnectedSeeds []*PeerInfo

	for _, seed := range pm.seeds {
		if _, ok := pm.streams[seed.PeerId]; !ok {
			unConnectedSeeds = append(unConnectedSeeds, seed)
		}
	}

	return unConnectedSeeds
}

func (pm *PeerManager) Broadcast(data []byte, priority int) {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()
	for _, s := range pm.streams {
		pbDm := networkpb.Dapmsg{}
		if err := proto.Unmarshal(data,&pbDm); err != nil{
			logger.Debugf("Unmarshal failed: %v", err.Error())
		}
		logger.Debugf("Broadcast to stream of peerID %v, type: %v, sendTime: %v,counter: %v, uuid: %v",s.stream.peerID,pbDm.Cmd,pbDm.UnixTimeReceived,pbDm.Counter, pbDm.Uuid)
		s.stream.Send(data, priority)
	}
}

func (pm *PeerManager) Unicast(data []byte, pid peer.ID, priority int) {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()

	streamInfo, ok := pm.streams[pid]
	if !ok {
		logger.WithFields(logger.Fields{
			"pid": pid,
		}).Warn("PeerManager: Unicast pid not found.")
		return
	}

	streamInfo.stream.Send(data, priority)
}

func (pm *PeerManager) ReceivePeers(peerId peer.ID, peers []*PeerInfo) {
	pm.addSyncPeersResult(peerId, peers)

	if pm.isSyncPeerFinish() {
		pm.collectSyncPeersResult()
		pm.saveSyncPeers()
		go func() {
			pm.startConnectSyncPeers()
		}()
	}
}

func (pm *PeerManager) AddStream(stream *Stream) {

	logger.WithFields(logger.Fields{
		"peer_id": stream.peerID,
		"addr":    stream.remoteAddr.String(),
	}).Info("PeerManager: Add Stream")

	connectionType := pm.getStreamConnectionType(stream)
	if !pm.checkAndAddStream(stream.peerID, connectionType, stream) {
		stream.StopStream(nil)
	}
}

func (pm *PeerManager) StopStream(stream *Stream) {

	logger.WithFields(logger.Fields{
		"peer_id": stream.peerID,
		"addr":    stream.remoteAddr.String(),
	}).Info("PeerManager: Stop Stream")

	pm.mutex.Lock()
	streamInfo, ok := pm.streams[stream.peerID]
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
	delete(pm.streams, stream.peerID)
	pm.node.host.Peerstore().ClearAddrs(stream.peerID)
	streamLen := len(pm.streams)
	pm.mutex.Unlock()
	if streamLen == 0 {
		go func() {
			pm.startConnectSeeds()
			pm.startConnectSyncPeers()
		}()
	}
}

func (pm *PeerManager) StopAllStreams(err error) {
	for _, streamInfo := range pm.streams {
		streamInfo.stream.StopStream(err)
	}
}

func (pm *PeerManager) RandomGetConnectedPeers(number int) []*PeerInfo {
	streams := pm.CloneStreamsToSlice()
	chooseStreams := randomChooseStreams(number, streams)
	peers := make([]*PeerInfo, len(chooseStreams))

	for i, streamInfo := range chooseStreams {
		peers[i] = &PeerInfo{PeerId: streamInfo.stream.peerID, Addrs: []ma.Multiaddr{streamInfo.stream.remoteAddr}}
	}
	return peers
}

func (pm *PeerManager) StartNewPingService(interval time.Duration) bool {
	if pm.ping != nil {
		logger.Warn("PeerManager: Ping service already running.")
		return false
	}

	if pm.node.host == nil {
		logger.Warn("PeerManager: Unable to start ping service.")
		return false
	}

	pm.ping = &PingService{ping.NewPingService(pm.node.host), make(chan bool)}
	go func() {
		logger.Debug("PeerManager: Starting ping service...")
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				pm.pingPeers()
			case <-pm.ping.stop:
				logger.Debug("PeerManager: Stopping ping service...")
				pm.ping.stop <- true
				return
			}
		}
	}()
	return true
}

func (pm *PeerManager) StopPingService() bool {
	if pm.ping != nil {
		/* send stop signal and wait for reply */
		pm.ping.stop <- true
		<-pm.ping.stop
		pm.ping = nil
		return true
	} else {
		logger.Warn("PeerManager: Can not stop a ping service that was never started.")
		return false
	}
}

func (pm *PeerManager) pingPeers() {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()
	logger.Debug("PeerManager: pinging peers...")
	var wg sync.WaitGroup
	wg.Add(len(pm.streams))
	for peerId, streamInfo := range pm.streams {
		go func() {
			defer wg.Done()
			pm.updatePeerLatency(peerId, streamInfo)
		}()
	}
	wg.Wait()
	logger.Debug("PeerManager: done pinging peers...")
}

func (pm *PeerManager) updatePeerLatency(peerId peer.ID, streamInfo *StreamInfo) {
	result := <-pm.ping.service.Ping(context.Background(), peerId)
	if result.Error != nil {
		logger.WithError(result.Error).Errorf("PeerManager: error pinging peer %v", peerId.Pretty())
		streamInfo.latency = nil
	} else {
		rtt := float64(result.RTT) / 1e6
		streamInfo.latency = &rtt
	}
}

func (pm *PeerManager) doConnectSeeds(wg *sync.WaitGroup, peers []*PeerInfo) {

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

func (pm *PeerManager) startConnectSyncPeers() {

	logger.WithFields(logger.Fields{
		"num_of_peers": len(pm.syncPeers),
	}).Info("PeerManager: Connect sync peers")

	if len(pm.syncPeers) == 0 {
		return
	}

	leftConnectionOut := pm.maxConnectionOutCount - pm.connectionOutCount
	if leftConnectionOut < 0 {
		return
	}

	toCheckPeers := pm.cloneUnconnectedSyncPeersToSlice()
	randomChoosePeers := randomChoosePeers(leftConnectionOut, toCheckPeers)
	wg := &sync.WaitGroup{}
	wg.Add(len(randomChoosePeers))

	logger.WithFields(logger.Fields{
		"maxConnectionOutCount":     pm.maxConnectionOutCount,
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

func (pm *PeerManager) startSyncPeers() {
	if pm.syncPeerContext != nil {
		logger.Info("PeerManager: another sync is running.")
		return
	}

	pm.createSyncContext()
	pm.node.SyncPeersBroadcast()

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

func (pm *PeerManager) createSyncContext() {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	pm.syncPeerContext = &SyncPeerContext{
		checkingStreams: make(map[peer.ID]*StreamInfo),
		newPeers:        make(map[peer.ID]*PeerInfo),
	}

	for key, streamInfo := range pm.streams {
		pm.syncPeerContext.checkingStreams[key] = streamInfo
	}
}

func (pm *PeerManager) addSyncPeersResult(peerId peer.ID, peers []*PeerInfo) bool {
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
		if peerInfo.PeerId == pm.node.GetPeerID() {
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

		pm.syncPeers[key] = &PeerInfo{PeerId: key, Addrs: []ma.Multiaddr{streamInfo.stream.remoteAddr}}

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
	db := pm.node.GetBlockchain().GetDb()

	var peerPbs []*networkpb.PeerInfo
	for _, peerInfo := range syncPeers {
		peerPbs = append(peerPbs, peerInfo.ToProto().(*networkpb.PeerInfo))
	}

	bytes, err := proto.Marshal(&networkpb.ReturnPeerList{PeerList: peerPbs})
	if err != nil {
		logger.WithError(err).Info("PeerManager: serialize sync peers failed.")
	}

	err = db.Put([]byte(syncPeersKey), bytes)
	if err != nil {
		logger.WithError(err).Info("PeerManager: save sync peers failed.")
	}
}

func (pm *PeerManager) removeStaleSyncPeer(peerId peer.ID) {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()
	delete(pm.syncPeers, peerId)
}

func (pm *PeerManager) cloneSyncPeers() map[peer.ID]*PeerInfo {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()
	peers := make(map[peer.ID]*PeerInfo)

	for key, peerInfo := range pm.syncPeers {
		peers[key] = peerInfo
	}

	return peers
}

func (pm *PeerManager) cloneUnconnectedSyncPeersToSlice() []*PeerInfo {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()
	var peers []*PeerInfo

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

func (pm *PeerManager) CloneStreamsToPeerInfoSlice() []*PeerInfo {
	streams := pm.CloneStreamsToSlice()
	peers := make([]*PeerInfo, len(streams))

	for i, streamInfo := range streams {
		peers[i] = &PeerInfo{PeerId: streamInfo.stream.peerID, Addrs: []ma.Multiaddr{streamInfo.stream.remoteAddr}, Latency: streamInfo.latency}
	}

	return peers
}

func randomChoosePeers(number int, peers []*PeerInfo) []*PeerInfo {
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

func (pm *PeerManager) connectPeer(peerInfo *PeerInfo, connectionType ConnectionType) (*Stream, error) {
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

	pm.node.host.Peerstore().AddAddrs(peerInfo.PeerId, peerInfo.Addrs, pstore.PermanentAddrTTL)
	// make a new stream
	stream, err := pm.node.host.NewStream(context.Background(), peerInfo.PeerId, protocalName)
	if err != nil {
		logger.WithError(err).WithFields(logger.Fields{
			"PeerId": peerInfo.PeerId,
		}).Warn("PeerManager: Connect to peer failed")
		return nil, err
	}

	peerStream := NewStream(stream)
	logger.WithFields(logger.Fields{
		"PeerId": peerStream.peerID,
		"Addr":   peerStream.remoteAddr.String(),
	}).Info("PeerManager: Connect peer actual stream")
	if !pm.checkAndAddStream(peerInfo.PeerId, connectionType, peerStream) {
		peerStream.StopStream(nil)
		return nil, nil
	}
	pm.node.StartStream(peerStream)
	return peerStream, nil
}

func (pm *PeerManager) getStreamConnectionType(stream *Stream) ConnectionType {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()

	if _, ok := pm.seeds[stream.peerID]; ok {
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
		if pm.connectionInCount >= pm.maxConnectionInCount {
			logger.Info("PeerManager: connection in is full.")
			return false
		}
		pm.connectionInCount++
	case ConnectionTypeOut:
		if pm.connectionOutCount >= pm.maxConnectionOutCount {
			logger.Info("PeerManager: connection out is full.")
			return false
		}
		pm.connectionOutCount++

	default:
		//Pass
	}
	pm.streams[peerId] = &StreamInfo{stream: stream, connectionType: connectionType}
	for k,v:= range pm.streams{
		logger.Printf("pm streams pid: %v, type: %v",k,v.connectionType)
	}
	return true
}

func (pm *PeerManager) loadSyncPeers() error {
	db := pm.node.GetBlockchain().GetDb()
	peersBytes, err := db.Get([]byte(syncPeersKey))

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
		peerInfo := &PeerInfo{}
		if err := peerInfo.FromProto(peerPb); err != nil {
			logger.WithError(err).Warn("PeerManager: parse PeerInfo failed.")
		}

		pm.syncPeers[peerInfo.PeerId] = peerInfo
		logger.WithFields(logger.Fields{
			"peer_id": peerInfo.PeerId,
			"addr":    peerInfo.Addrs[0].String(),
		}).Info("loadSyncPeers")
	}

	logger.Infof("PeerManager: load sync peers count %v.", len(pm.syncPeers))

	return nil
}

func (pm *PeerManager) isStreamExist(peerId peer.ID) bool {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()
	_, ok := pm.streams[peerId]
	return ok
}

func createPeerInfoFromString(fullAddr string) (*PeerInfo, error) {
	addr, err := ma.NewMultiaddr(fullAddr)
	if err != nil {
		logger.WithError(err).WithFields(logger.Fields{
			"full_addr": fullAddr,
			"err":       err,
		}).Warn("PeerManager: create multiaddr failed.")
	}

	return CreatePeerInfoFromMultiaddrs([]ma.Multiaddr{addr})
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

//convert from protobuf
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
