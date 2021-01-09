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

package downloadmanager

import (
	"bytes"
	"encoding/hex"
	"errors"
	"github.com/dappley/go-dappley/common/log"
	"github.com/dappley/go-dappley/logic/blockproducer"
	"math"
	"sync"
	"time"

	"github.com/dappley/go-dappley/common/hash"
	"github.com/dappley/go-dappley/common/pubsub"
	"github.com/dappley/go-dappley/core/block"
	blockpb "github.com/dappley/go-dappley/core/block/pb"
	"github.com/dappley/go-dappley/logic/lblockchain"
	"github.com/dappley/go-dappley/network"
	"github.com/dappley/go-dappley/network/networkmodel"
	"github.com/golang/protobuf/proto"

	networkpb "github.com/dappley/go-dappley/network/pb"
	"github.com/libp2p/go-libp2p-core/peer"
	logger "github.com/sirupsen/logrus"
)

const (
	PeerStatusInit   int = 0
	PeerStatusReady  int = 1
	PeerStatusFailed int = 2

	DownloadStatusInit             int = 0
	DownloadStatusSyncCommonBlocks int = 1
	DownloadStatusDownloading      int = 2
	DownloadStatusIdle             int = 3

	CheckMaxWaitTime    time.Duration = 5
	DownloadMaxWaitTime time.Duration = 180
	MaxRetryCount       int           = 3
	MinRequestHashesNum int           = 20

	BlockchainInfoRequest   = "BlockchainInfoRequest"
	BlockchainInfoResponse  = "BlockchainInfoResponse"
	GetBlocksRequest        = "GetBlocksRequest"
	GetBlocksResponse       = "GetBlocksResponse"
	GetCommonBlocksRequest  = "GetCommonBlocksRequest"
	GetCommonBlocksResponse = "GetCommonBlocksResponse"

	maxGetBlocksNum = 10
)

var (
	ErrEmptyBlocks      = errors.New("received no block")
	ErrPeerNotFound     = errors.New("peerId not in checklist")
	ErrMismatchResponse = errors.New("response is not for waiting command")
)

var (
	dmSubscribedTopics = []string{
		BlockchainInfoRequest,
		BlockchainInfoResponse,
		GetBlocksRequest,
		GetBlocksResponse,
		GetCommonBlocksRequest,
		GetCommonBlocksResponse,
		network.TopicOnStreamStop,
	}
)

type PeerBlockInfo struct {
	peerid    peer.ID
	height    uint64
	libHeight uint64
	status    int
}

type DownloadCommand struct {
	startHashes []hash.Hash
}

type DownloadingCommandInfo struct {
	startHashes []hash.Hash
	retryCount  int
	finished    bool
}

type SyncCommandBlocksHeader struct {
	hash   hash.Hash
	height uint64
}

type SyncCommonBlocksCommand struct {
	msgId        int32
	blockHeaders []*SyncCommandBlocksHeader
}

type ExecuteCommand struct {
	command    interface{}
	retryCount int
}

type DownloadManager struct {
	peersInfo             map[peer.ID]*PeerBlockInfo
	downloadingPeer       *PeerBlockInfo
	currentCmd            *ExecuteCommand
	bm                    *lblockchain.BlockchainManager
	node                  NetService
	mutex                 sync.RWMutex
	status                int
	commonHeight          uint64
	msgId                 int32
	finishCh              chan bool
	numOfMinRequestHashes int
	bp                    *blockproducer.BlockProducer
}

func NewDownloadManager(node NetService, bm *lblockchain.BlockchainManager, numOfProducers int, bp *blockproducer.BlockProducer) *DownloadManager {

	downloadManager := &DownloadManager{
		peersInfo:             make(map[peer.ID]*PeerBlockInfo),
		downloadingPeer:       nil,
		currentCmd:            nil,
		bm:                    bm,
		node:                  node,
		mutex:                 sync.RWMutex{},
		status:                DownloadStatusIdle,
		msgId:                 0,
		commonHeight:          0,
		finishCh:              nil,
		numOfMinRequestHashes: numOfProducers,
		bp:                    bp,
	}
	if downloadManager.numOfMinRequestHashes < MinRequestHashesNum {
		downloadManager.numOfMinRequestHashes = MinRequestHashesNum
	}
	downloadManager.Subscribe()
	return downloadManager
}

func (downloadManager *DownloadManager) Start() {
	downloadManager.StartDownloadRequestListener()
}

func (downloadManager *DownloadManager) StartDownloadRequestListener() {
	go func() {
		defer log.CrashHandler()

		for {
			select {
			case returnCh := <-downloadManager.bm.GetDownloadRequestCh():
				logger.Info("StartDownloadRequestListener: Received download request.")
				if downloadManager.status != DownloadStatusIdle {
					logger.Warn("DownloadMananger: Blockchain is being downloaded. Received download request is dropped.")
					continue
				}
				logger.Info("StartDownloadRequestListener: Prepare to download.")
				go downloadManager.StartDownloadBlockchain(returnCh)
			}
		}
	}()
}

func (downloadManager *DownloadManager) Subscribe() {
	if downloadManager.node == nil {
		return
	}

	downloadManager.node.Listen(downloadManager)
}

func (downloadManager *DownloadManager) GetSubscribedTopics() []string {
	return dmSubscribedTopics
}

func (downloadManager *DownloadManager) GetTopicHandler(topic string) pubsub.TopicHandler {

	switch topic {
	case network.TopicOnStreamStop:
		return downloadManager.OnStreamStopHandler
	case BlockchainInfoRequest:
		return downloadManager.GetBlockchainInfoRequestHandler
	case BlockchainInfoResponse:
		return downloadManager.GetBlockchainInfoResponseHandler
	case GetBlocksRequest:
		return downloadManager.GetBlocksRequestHandler
	case GetBlocksResponse:
		return downloadManager.GetBlocksResponseHandler
	case GetCommonBlocksRequest:
		return downloadManager.GetCommonBlockRequestHandler
	case GetCommonBlocksResponse:
		return downloadManager.GetCommonBlockResponseHandler
	}
	return nil
}

func (downloadManager *DownloadManager) StartDownloadBlockchain(finishCh chan bool) {
	downloadManager.mutex.Lock()

	downloadManager.peersInfo = make(map[peer.ID]*PeerBlockInfo)
	downloadManager.finishCh = finishCh
	downloadManager.status = DownloadStatusInit

	for _, peer := range downloadManager.node.GetPeers() {
		downloadManager.peersInfo[peer.PeerId] = &PeerBlockInfo{peerid: peer.PeerId, height: 0, libHeight: 0, status: PeerStatusInit}
	}

	if len(downloadManager.peersInfo) == 0 {
		downloadManager.finishDownload()
		downloadManager.mutex.Unlock()
		return
	}
	downloadManager.mutex.Unlock()

	downloadManager.SendGetBlockchainInfoRequest()
	waitTimer := time.NewTimer(CheckMaxWaitTime * time.Second)
	logger.Info("DownloadManager: wait peer information")
	go func() {
		defer log.CrashHandler()

		<-waitTimer.C
		waitTimer.Stop()

		downloadManager.mutex.Lock()
		defer downloadManager.mutex.Unlock()

		if downloadManager.status != DownloadStatusInit {
			return
		}

		for _, peerInfo := range downloadManager.peersInfo {
			if peerInfo.status == PeerStatusInit {
				peerInfo.status = PeerStatusFailed
			}
		}
		logger.Info("DownloadManager: start get common blocks")
		downloadManager.startGetCommonBlocks(0)
	}()
}

func (downloadManager *DownloadManager) AddPeerBlockChainInfo(peerId peer.ID, height uint64, libHeight uint64) {
	logger.Infof("DownloadManager: Receive blockchain info %v %v \n", peerId, height)
	downloadManager.mutex.Lock()
	defer downloadManager.mutex.Unlock()

	if downloadManager.status != DownloadStatusInit {
		logger.Info("DownloadManager: Download peer started, skip PeerId ", peerId)
		return
	}

	blockPeerInfo, isFound := downloadManager.peersInfo[peerId]
	if isFound != true {
		logger.Info("DownloadManager: Peer not in check list ", peerId)
		return
	}

	blockPeerInfo.height = height
	blockPeerInfo.libHeight = libHeight
	blockPeerInfo.status = PeerStatusReady

	if downloadManager.canStartDownload() {
		downloadManager.startGetCommonBlocks(0)
	}
}

func (downloadManager *DownloadManager) validateReturnBlocks(blocksPb *networkpb.ReturnBlocks, peerId peer.ID) (*PeerBlockInfo, error) {
	downloadingPeer := ""
	if downloadManager.downloadingPeer != nil {
		downloadingPeer = downloadManager.downloadingPeer.peerid.String()
	}
	returnBlocksLogger := logger.WithFields(logger.Fields{
		"name":               "GetBlocksResponse",
		"downloadingPeer.id": downloadingPeer,
	})

	if downloadManager.downloadingPeer == nil || downloadManager.downloadingPeer.peerid != peerId {
		returnBlocksLogger.Info("validateReturnBlocks: downloadingPeer is empty or peerId is not match.")
		return nil, ErrPeerNotFound
	}

	if blocksPb.GetBlocks() == nil || len(blocksPb.GetBlocks()) == 0 {
		returnBlocksLogger.Error("DownloadManager: received no block.")
		return nil, ErrEmptyBlocks
	}

	hashes := make([]hash.Hash, len(blocksPb.GetStartBlockHashes()))
	for index, h := range blocksPb.GetStartBlockHashes() {
		hashes[index] = hash.Hash(h)
	}

	if downloadManager.isDownloadCommandFinished(hashes) {
		returnBlocksLogger.Info("DownloadManager: response is not for waiting command.")
		return nil, ErrMismatchResponse
	}

	return downloadManager.downloadingPeer, nil
}

func (downloadManager *DownloadManager) GetBlocksDataHandler(blocksPb *networkpb.ReturnBlocks, peerInfo networkmodel.PeerInfo) {
	returnBlocksLogger := logger.WithFields(logger.Fields{
		"name": "GetBlocksResponse",
	})

	downloadManager.mutex.Lock()
	checkingPeer, err := downloadManager.validateReturnBlocks(blocksPb, peerInfo.PeerId)
	if err != nil {
		returnBlocksLogger.WithFields(logger.Fields{"error": err}).Error("DownloadManager:")
		downloadManager.mutex.Unlock()
		return
	}

	downloadingCmd, ok := downloadManager.currentCmd.command.(*DownloadingCommandInfo)
	if ok {
		downloadingCmd.finished = true
	}

	downloadManager.mutex.Unlock()

	var blocks []*block.Block
	for _, pbBlock := range blocksPb.GetBlocks() {
		block := &block.Block{}
		block.FromProto(pbBlock)

		if !downloadManager.bm.VerifyBlock(block) {
			returnBlocksLogger.WithFields(logger.Fields{
				"height": block.GetHeight(),
				"hash":   block.GetHash(),
			}).Warn("DownloadManager: verify block failed.")
			return
		}

		blocks = append(blocks, block)
	}
	logger.Infof("DownloadManager: receive blocks source %v to %v.", blocks[0].GetHeight(), blocks[len(blocks)-1].GetHeight())
	logger.Info("DownloadManager: set blockchain status to downloading.")

	if err := downloadManager.bm.MergeFork(blocks, blocks[len(blocks)-1].GetPrevHash()); err != nil {
		downloadManager.finishDownload()
		returnBlocksLogger.WithError(err).Warn("DownloadManager: merge fork failed:", err)
		return
	}

	downloadManager.mutex.Lock()
	defer downloadManager.mutex.Unlock()
	if downloadManager.bm.Getblockchain().GetMaxHeight() >= checkingPeer.height {
		downloadManager.finishDownload()
		lib, _ := downloadManager.bm.Getblockchain().GetBlockByHeight(checkingPeer.libHeight)
		downloadManager.bm.Getblockchain().SetLIBHash(lib.GetHash())
		logger.WithFields(logger.Fields{
			"lib_hash": lib.GetHash(),
		}).Info("DownloadManager: finishing download blocks.")
		return
	}

	var nextHashes []hash.Hash
	for _, block := range blocks {
		nextHashes = append(nextHashes, block.GetHash())
	}

	logger.WithFields(logger.Fields{
		"peerInfo.PeerId":  peerInfo.PeerId.String(),
		"CurrentMaxHeight": downloadManager.bm.Getblockchain().GetMaxHeight(),
	}).Info("GetBlocksDataHandler: start the next download.")
	downloadManager.sendDownloadCommand(nextHashes, peerInfo.PeerId, 0)
}

func (downloadManager *DownloadManager) GetCommonBlockDataHandler(blocksPb *networkpb.ReturnCommonBlocks, peerInfo networkmodel.PeerInfo) {
	downloadManager.mutex.Lock()
	defer downloadManager.mutex.Unlock()

	downloadingPeer := ""
	if downloadManager.downloadingPeer != nil {
		downloadingPeer = downloadManager.downloadingPeer.peerid.String()
	}
	if downloadManager.downloadingPeer == nil || downloadManager.downloadingPeer.peerid != peerInfo.PeerId {
		logger.WithFields(logger.Fields{
			"name":               "GetCommonBlocksResponse",
			"downloadingPeer.id": downloadingPeer,
		}).Info("GetCommonBlockDataHandler: downloadingPeer is empty or peerId is not match.")
		downloadManager.mutex.Unlock()
		return
	}

	if !downloadManager.isSameGetCommonBlocksCommand(blocksPb.GetMsgId()) {
		logger.WithFields(logger.Fields{
			"name": "GetCommonBlocksResponse",
		}).Info("DownloadManager: response is not for waiting command.")
		downloadManager.mutex.Unlock()
		return
	}

	downloadManager.checkGetCommonBlocksResult(blocksPb.GetBlockHeaders())
}

func (downloadManager *DownloadManager) DisconnectPeer(peerId peer.ID) {
	downloadManager.mutex.Lock()
	defer downloadManager.mutex.Unlock()

	if downloadManager.status == DownloadStatusIdle {
		return
	}

	delete(downloadManager.peersInfo, peerId)
	if downloadManager.status != DownloadStatusInit {
		if downloadManager.downloadingPeer.peerid == peerId {
			downloadManager.status = DownloadStatusInit
			downloadManager.downloadingPeer = nil
			downloadManager.currentCmd = nil
			downloadManager.startGetCommonBlocks(0)
		}
	}
}

func (downloadManager *DownloadManager) GetCommonBlockCheckPoint(startHeight uint64, endHeight uint64) []*SyncCommandBlocksHeader {
	var blockHeaders []*SyncCommandBlocksHeader
	lastHeight := uint64(math.MaxUint64)
	interval := endHeight - startHeight
	for i := 32; i >= 0; i-- {
		currentHeight := startHeight + interval*uint64(i)/uint64(32)
		if lastHeight != currentHeight {
			lastHeight = currentHeight
			block, err := downloadManager.bm.Getblockchain().GetBlockByHeight(currentHeight)
			if err != nil {
				continue
			}
			blockHeaders = append(blockHeaders, &SyncCommandBlocksHeader{hash: block.GetHash(), height: currentHeight})
		}
	}

	return blockHeaders
}

func (downloadManager *DownloadManager) FindCommonBlock(blockHeaders []*blockpb.BlockHeader) (int, *block.Block) {
	findIndex := -1
	var commonBlock *block.Block

	for index, blockHeader := range blockHeaders {
		block, err := downloadManager.bm.Getblockchain().GetBlockByHeight(blockHeader.GetHeight())
		if err != nil {
			continue
		}

		if bytes.Compare([]byte(block.GetHash()), []byte(blockHeader.GetHash())) == 0 {
			findIndex = index
			commonBlock = block
			break
		}
		if blockHeader.GetHeight() == 0 {
			logger.Warn("DownloadManager: invalid get common blocks result. Genesis block hash is different with the request source node.")
			return findIndex, nil
		}
	}
	return findIndex, commonBlock
}

func (downloadManager *DownloadManager) CheckGetCommonBlockCommand(msgId int32, peerInfo networkmodel.PeerInfo, retryCount int) {
	downloadManager.mutex.Lock()
	defer downloadManager.mutex.Unlock()

	if !downloadManager.isSameGetCommonBlocksCommand(msgId) {
		return
	}

	if retryCount >= MaxRetryCount {
		peerInfo, ok := downloadManager.peersInfo[peerInfo.PeerId]
		if ok {
			peerInfo.status = PeerStatusFailed
		}
		downloadManager.status = DownloadStatusSyncCommonBlocks
		downloadManager.downloadingPeer = nil
		downloadManager.currentCmd = nil
		downloadManager.startGetCommonBlocks(0)
	} else {
		syncCommand, _ := downloadManager.currentCmd.command.(*SyncCommonBlocksCommand)
		downloadManager.sendGetCommonBlockCommand(syncCommand.blockHeaders, peerInfo, retryCount+1)
	}
}

func (downloadManager *DownloadManager) CheckDownloadCommand(hashes []hash.Hash, peerId peer.ID, retryCount int) {
	downloadManager.mutex.Lock()
	defer downloadManager.mutex.Unlock()

	if downloadManager.isDownloadCommandFinished(hashes) {
		return
	}

	if retryCount >= MaxRetryCount {
		peerInfo, ok := downloadManager.peersInfo[peerId]
		if ok {
			peerInfo.status = PeerStatusFailed
		}
		downloadManager.status = DownloadStatusInit
		downloadManager.downloadingPeer = nil
		downloadManager.currentCmd = nil
		downloadManager.startGetCommonBlocks(0)
	} else {
		downloadManager.sendDownloadCommand(hashes, peerId, retryCount+1)
	}
}

func (downloadManager *DownloadManager) startGetCommonBlocks(retryCount int) {
	if downloadManager.status != DownloadStatusInit {
		logger.Info("DownloadManager: start get common blocks failed, download status incorrect")
		return
	}

	if downloadManager.bp != nil {
		logger.Info("startGetCommonBlocks: prepared to stop the block producer")
		downloadManager.bp.Stop()
	}

	downloadManager.status = DownloadStatusSyncCommonBlocks
	highestPeer := downloadManager.selectHighestPeer()

	if highestPeer.peerid == downloadManager.node.GetHostPeerInfo().PeerId {
		downloadManager.finishDownload()
		logger.Info("DownloadManager: Current node has the highest block")
		return
	}

	downloadManager.downloadingPeer = highestPeer
	maxHeight := downloadManager.bm.Getblockchain().GetMaxHeight()
	blockHeaders := downloadManager.GetCommonBlockCheckPoint(0, maxHeight)
	downloadManager.sendGetCommonBlockCommand(blockHeaders, networkmodel.PeerInfo{PeerId: highestPeer.peerid}, 0)
}

func (downloadManager *DownloadManager) sendGetCommonBlockCommand(blockHeaders []*SyncCommandBlocksHeader, peerId networkmodel.PeerInfo, retryCount int) {
	downloadManager.msgId++
	msgId := downloadManager.msgId
	syncCommand := &SyncCommonBlocksCommand{msgId: msgId, blockHeaders: blockHeaders}

	downloadManager.currentCmd = &ExecuteCommand{command: syncCommand, retryCount: retryCount}
	downloadManager.SendGetCommonBlockRequest(blockHeaders, peerId, msgId)

	downloadTimer := time.NewTimer(DownloadMaxWaitTime * time.Second)
	go func() {
		defer log.CrashHandler()

		<-downloadTimer.C
		downloadTimer.Stop()
		downloadManager.CheckGetCommonBlockCommand(msgId, peerId, retryCount)
	}()
}

func (downloadManager *DownloadManager) isSameGetCommonBlocksCommand(msgId int32) bool {
	if downloadManager.status != DownloadStatusSyncCommonBlocks {
		return false
	}

	if downloadManager.currentCmd == nil {
		return false
	}

	syncCommand, ok := downloadManager.currentCmd.command.(*SyncCommonBlocksCommand)
	if !ok || syncCommand.msgId != msgId {
		return false
	}

	return true
}

func (downloadManager *DownloadManager) checkGetCommonBlocksResult(blockHeaders []*blockpb.BlockHeader) {
	findIndex, commonBlock := downloadManager.FindCommonBlock(blockHeaders)

	if findIndex == -1 {
		// no common blocks, code version is different
		logger.Panic("checkGetCommonBlocksResult: genesis block hash is different from other nodes. Check code version or synchronize db files from other nodes.")
	}
	if findIndex == 0 || blockHeaders[findIndex-1].GetHeight()-blockHeaders[findIndex].GetHeight() == 1 {
		logger.Warnf("checkGetCommonBlocksResult: common height %v", commonBlock.GetHeight())
		downloadManager.commonHeight = commonBlock.GetHeight()
		downloadManager.currentCmd = nil
		downloadManager.startDownload(0)
	} else {
		blockHeaders := downloadManager.GetCommonBlockCheckPoint(
			blockHeaders[findIndex].GetHeight(),
			blockHeaders[findIndex-1].GetHeight(),
		)
		downloadManager.sendGetCommonBlockCommand(blockHeaders, networkmodel.PeerInfo{PeerId: downloadManager.downloadingPeer.peerid}, 0)
	}
}

func (downloadManager *DownloadManager) startDownload(retryCount int) {
	if downloadManager.status != DownloadStatusSyncCommonBlocks {
		return
	}

	downloadManager.status = DownloadStatusDownloading

	startBlockHeight := downloadManager.commonHeight
	var hashes []hash.Hash
	for i := 0; i < downloadManager.numOfMinRequestHashes && startBlockHeight-uint64(i) > 0; i++ {
		block, err := downloadManager.bm.Getblockchain().GetBlockByHeight(startBlockHeight - uint64(i))
		if err != nil {
			break
		}
		hashes = append(hashes, block.GetHash())
	}

	downloadManager.sendDownloadCommand(hashes, downloadManager.downloadingPeer.peerid, retryCount)
}

func (downloadManager *DownloadManager) sendDownloadCommand(hashes []hash.Hash, peerId peer.ID, retryCount int) {
	downloadingCmd := &DownloadingCommandInfo{startHashes: hashes, finished: false}
	downloadManager.currentCmd = &ExecuteCommand{command: downloadingCmd, retryCount: retryCount}
	downloadManager.SendGetBlocksRequest(hashes, networkmodel.PeerInfo{PeerId: peerId})

	downloadTimer := time.NewTimer(DownloadMaxWaitTime * time.Second)
	go func() {
		defer log.CrashHandler()

		<-downloadTimer.C
		downloadTimer.Stop()
		downloadManager.CheckDownloadCommand(hashes, peerId, retryCount)
	}()
}

func (downloadManager *DownloadManager) isDownloadCommandFinished(hashes []hash.Hash) bool {
	if downloadManager.currentCmd == nil {
		return true
	}

	if downloadManager.status != DownloadStatusDownloading {
		return true
	}

	downloadingCmd, ok := downloadManager.currentCmd.command.(*DownloadingCommandInfo)

	if !ok {
		return true
	}

	if downloadingCmd.finished {
		return true
	}

	if len(hashes) != len(downloadingCmd.startHashes) {
		return true
	}

	for index, hash := range hashes {
		if bytes.Compare(hash, downloadingCmd.startHashes[index]) != 0 {
			return true
		}
	}

	return false
}

func (downloadManager *DownloadManager) finishDownload() {
	downloadManager.status = DownloadStatusIdle
	downloadManager.downloadingPeer = nil
	downloadManager.currentCmd = nil
	downloadManager.finishCh <- true

	if downloadManager.bp != nil {
		logger.Info("finishDownload: prepared to start the block producer")
		downloadManager.bp.Start()
	}
}

func (downloadManager *DownloadManager) canStartDownload() bool {
	for _, peerInfo := range downloadManager.peersInfo {
		if peerInfo.status == PeerStatusInit {
			return false
		}
	}

	return true
}

func (downloadManager *DownloadManager) selectHighestPeer() *PeerBlockInfo {
	peerWithHighestBlockHeight := &PeerBlockInfo{
		peerid:    downloadManager.node.GetHostPeerInfo().PeerId,
		height:    downloadManager.bm.Getblockchain().GetMaxHeight(),
		libHeight: downloadManager.bm.Getblockchain().GetLIBHeight(),
		status:    PeerStatusReady,
	}

	for _, peerInfo := range downloadManager.peersInfo {
		if peerInfo.status == PeerStatusReady && peerInfo.libHeight > peerWithHighestBlockHeight.libHeight {
			peerWithHighestBlockHeight = peerInfo
		} else if peerInfo.status == PeerStatusReady && peerInfo.libHeight == peerWithHighestBlockHeight.libHeight && peerInfo.height > peerWithHighestBlockHeight.height {
			peerWithHighestBlockHeight = peerInfo
		}
	}
	return peerWithHighestBlockHeight
}

func (downloadManager *DownloadManager) SendGetCommonBlockRequest(blockHeaders []*SyncCommandBlocksHeader, pid networkmodel.PeerInfo, msgId int32) {
	var blockHeaderPbs []*blockpb.BlockHeader

	for _, blockHeader := range blockHeaders {
		blockHeaderPbs = append(blockHeaderPbs,
			&blockpb.BlockHeader{Hash: blockHeader.hash, Height: blockHeader.height})
	}

	getCommonBlocksPb := &networkpb.GetCommonBlocks{MsgId: msgId, BlockHeaders: blockHeaderPbs}

	downloadManager.node.UnicastHighProrityCommand(GetCommonBlocksRequest, getCommonBlocksPb, pid)

}

func (downloadManager *DownloadManager) GetCommonBlockRequestHandler(input interface{}) {

	var command *networkmodel.DappRcvdCmdContext
	command = input.(*networkmodel.DappRcvdCmdContext)

	param := &networkpb.GetCommonBlocks{}
	if err := proto.Unmarshal(command.GetData(), param); err != nil {
		logger.WithFields(logger.Fields{
			"name": "GetCommonBlocksRequest",
		}).Warn("DownloadManager: parse data failed.")
		return
	}

	downloadManager.SendGetCommonBlockResponse(param.GetBlockHeaders(), param.GetMsgId(), command.GetSource())

}

func (downloadManager *DownloadManager) SendGetCommonBlockResponse(blockHeaders []*blockpb.BlockHeader, msgId int32, destination networkmodel.PeerInfo) {

	index, _ := downloadManager.FindCommonBlock(blockHeaders)
	var blockHeaderPbs []*blockpb.BlockHeader
	if index == 0 {
		blockHeaderPbs = blockHeaders[:1]
	} else if index > 0 {
		blockHeaders := downloadManager.GetCommonBlockCheckPoint(
			blockHeaders[index].GetHeight(),
			blockHeaders[index-1].GetHeight(),
		)
		for _, blockHeader := range blockHeaders {
			blockHeaderPbs = append(blockHeaderPbs,
				&blockpb.BlockHeader{Hash: blockHeader.hash, Height: blockHeader.height})
		}
	}

	result := &networkpb.ReturnCommonBlocks{MsgId: msgId, BlockHeaders: blockHeaderPbs}

	downloadManager.node.UnicastHighProrityCommand(GetCommonBlocksResponse, result, destination)
}

func (downloadManager *DownloadManager) GetCommonBlockResponseHandler(input interface{}) {

	var command *networkmodel.DappRcvdCmdContext
	command = input.(*networkmodel.DappRcvdCmdContext)

	param := &networkpb.ReturnCommonBlocks{}

	if err := proto.Unmarshal(command.GetData(), param); err != nil {
		logger.WithFields(logger.Fields{
			"name": "GetCommonBlockResponseHandler",
		}).Info("DownloadManager: parse data failed.")
	}

	downloadManager.GetCommonBlockDataHandler(param, command.GetSource())
}

func (downloadManager *DownloadManager) SendGetBlocksRequest(hashes []hash.Hash, peerInfo networkmodel.PeerInfo) {

	blkHashes := make([][]byte, len(hashes))
	for index, hash := range hashes {
		blkHashes[index] = hash
	}

	getBlockPb := &networkpb.GetBlocks{StartBlockHashes: blkHashes}

	downloadManager.node.UnicastHighProrityCommand(GetBlocksRequest, getBlockPb, peerInfo)
}

func (downloadManager *DownloadManager) GetBlocksRequestHandler(input interface{}) {

	var command *networkmodel.DappRcvdCmdContext
	command = input.(*networkmodel.DappRcvdCmdContext)

	param := &networkpb.GetBlocks{}
	if err := proto.Unmarshal(command.GetData(), param); err != nil {
		logger.WithFields(logger.Fields{
			"name": "GetBlocksRequest",
		}).Info("DownloadManager: parse data failed.")
		return
	}

	downloadManager.SendGetBlocksResponse(param.GetStartBlockHashes(), command.GetSource())

}

func (downloadManager *DownloadManager) SendGetBlocksResponse(startBlockHashes [][]byte, destination networkmodel.PeerInfo) {

	blk := downloadManager.findBlockInRequestHash(startBlockHashes)

	// Reach the blockchain's tail
	if blk.GetHeight() >= downloadManager.bm.Getblockchain().GetMaxHeight() {
		logger.WithFields(logger.Fields{
			"name": "GetBlocksRequest",
		}).Info("DownloadManager: reach blockchain tail.")
		return
	}

	var blks []*block.Block

	blk, err := downloadManager.bm.Getblockchain().GetBlockByHeight(blk.GetHeight() + 1)
	for i := int32(0); i < maxGetBlocksNum && err == nil; i++ {
		if blk.GetHeight() == 0 {
			logger.Panicf("Error %v", hex.EncodeToString(blk.GetHash()))
		}
		blks = append(blks, blk)
		blk, err = downloadManager.bm.Getblockchain().GetBlockByHeight(blk.GetHeight() + 1)
	}

	var blockPbs []*blockpb.Block
	for i := len(blks) - 1; i >= 0; i-- {
		blockPbs = append(blockPbs, blks[i].ToProto().(*blockpb.Block))
	}

	result := &networkpb.ReturnBlocks{Blocks: blockPbs, StartBlockHashes: startBlockHashes}

	downloadManager.node.UnicastHighProrityCommand(GetBlocksResponse, result, destination)

}

func (downloadManager *DownloadManager) GetBlocksResponseHandler(input interface{}) {

	var command *networkmodel.DappRcvdCmdContext
	command = input.(*networkmodel.DappRcvdCmdContext)

	param := &networkpb.ReturnBlocks{}
	if err := proto.Unmarshal(command.GetData(), param); err != nil {
		logger.WithFields(logger.Fields{
			"name": "GetBlocksResponse",
		}).Info("DownloadManager: parse data failed.")
		return
	}

	downloadManager.GetBlocksDataHandler(param, command.GetSource())
}

func (downloadManager *DownloadManager) SendGetBlockchainInfoRequest() {
	request := &networkpb.GetBlockchainInfo{Version: networkmodel.ProtocalName}

	downloadManager.node.BroadcastNormalPriorityCommand(BlockchainInfoRequest, request)

}

func (downloadManager *DownloadManager) GetBlockchainInfoRequestHandler(input interface{}) {

	var command *networkmodel.DappRcvdCmdContext
	command = input.(*networkmodel.DappRcvdCmdContext)

	downloadManager.SendGetBlockchainInfoResponse(command.GetSource())

}

func (downloadManager *DownloadManager) SendGetBlockchainInfoResponse(destination networkmodel.PeerInfo) {

	tailBlock, err := downloadManager.bm.Getblockchain().GetTailBlock()
	if err != nil {
		logger.WithFields(logger.Fields{
			"name": "GetBlockchainInfoRequest",
		}).Warn("DownloadManager: get tail block failed.")
		return
	}

	result := &networkpb.ReturnBlockchainInfo{
		TailBlockHash: tailBlock.GetHash(),
		BlockHeight:   tailBlock.GetHeight(),
		Timestamp:     tailBlock.GetTimestamp(),
		LibHash:       downloadManager.bm.Getblockchain().GetLIBHash(),
		LibHeight:     downloadManager.bm.Getblockchain().GetLIBHeight(),
	}

	downloadManager.node.UnicastNormalPriorityCommand(BlockchainInfoResponse, result, destination)

}

func (downloadManager *DownloadManager) GetBlockchainInfoResponseHandler(input interface{}) {

	var command *networkmodel.DappRcvdCmdContext
	command = input.(*networkmodel.DappRcvdCmdContext)

	blockchainInfo := &networkpb.ReturnBlockchainInfo{}
	if err := proto.Unmarshal(command.GetData(), blockchainInfo); err != nil {
		logger.WithFields(logger.Fields{
			"name": "BlockchainInfoResponse",
		}).Info("DownloadManager: parse data failed.")
		return
	}

	downloadManager.AddPeerBlockChainInfo(command.GetSource().PeerId, blockchainInfo.GetBlockHeight(), blockchainInfo.GetLibHeight())
}
func (downloadManager *DownloadManager) OnStreamStopHandler(input interface{}) {

	var command *networkmodel.DappRcvdCmdContext
	command = input.(*networkmodel.DappRcvdCmdContext)

	peerInfopb := &networkpb.PeerInfo{}
	if err := proto.Unmarshal(command.GetData(), peerInfopb); err != nil {
		logger.WithFields(logger.Fields{
			"name": "onStreamStop",
		}).Info("DownloadManager: parse data failed.")
		return
	}

	var peerInfo networkmodel.PeerInfo
	if err := peerInfo.FromProto(peerInfopb); err != nil {
		logger.WithFields(logger.Fields{
			"name": "onStreamStop",
		}).Info("DownloadManager: parse data from proto message failed.")
		return
	}

	downloadManager.DisconnectPeer(peerInfo.PeerId)

}

func (downloadManager *DownloadManager) findBlockInRequestHash(startBlockHashes [][]byte) *block.Block {
	for _, hash := range startBlockHashes {
		// hash in blockchain, return
		if block, err := downloadManager.bm.Getblockchain().GetBlockByHash(hash); err == nil {
			return block
		}
	}

	// Return Genesis Block
	block, _ := downloadManager.bm.Getblockchain().GetBlockByHeight(0)
	return block
}
