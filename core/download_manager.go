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

package core

import (
	"bytes"
	"encoding/hex"
	"errors"
	"github.com/dappley/go-dappley/network"
	"github.com/golang/protobuf/proto"
	"math"
	"sync"
	"time"

	"github.com/dappley/go-dappley/core/pb"
	"github.com/dappley/go-dappley/network/pb"
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
	DownloadStatusFinish           int = 3

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
	subscribedTopics = []string{
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
	startHashes []Hash
}

type DownloadingCommandInfo struct {
	startHashes []Hash
	retryCount  int
	finished    bool
}

type SyncCommandBlocksHeader struct {
	hash   Hash
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
	peersInfo        map[peer.ID]*PeerBlockInfo
	downloadingPeer  *PeerBlockInfo
	currentCmd       *ExecuteCommand
	node             *network.Node
	mutex            sync.RWMutex
	status           int
	commonHeight     uint64
	msgId            int32
	finishChan       chan bool
	commandSendCh    chan *network.DappSendCmdContext
	commandReceiveCh chan *network.DappRcvdCmdContext
}

func NewDownloadManager(node *network.Node) *DownloadManager {

	downloadManager := &DownloadManager{
		peersInfo:        make(map[peer.ID]*PeerBlockInfo),
		downloadingPeer:  nil,
		currentCmd:       nil,
		node:             node,
		mutex:            sync.RWMutex{},
		status:           DownloadStatusInit,
		msgId:            0,
		commonHeight:     0,
		finishChan:       nil,
		commandSendCh:    node.GetCommandSendCh(),
		commandReceiveCh: make(chan *network.DappRcvdCmdContext, 100),
	}
	downloadManager.StartCommandListener()
	downloadManager.SubscribeCommandBroker(node.GetCommandBroker())
	return downloadManager
}

func (downloadManager *DownloadManager) StartCommandListener() {
	go func() {
		for {
			select {
			case command := <-downloadManager.commandReceiveCh:
				switch command.GetCommandName() {
				case network.TopicOnStreamStop:
					downloadManager.OnStreamStopHandler(command)
				case BlockchainInfoRequest:
					downloadManager.GetBlockchainInfoRequestHandler(command)
				case BlockchainInfoResponse:
					downloadManager.GetBlockchainInfoResponseHandler(command)
				case GetBlocksRequest:
					downloadManager.GetBlocksRequestHandler(command)
				case GetBlocksResponse:
					downloadManager.GetBlocksResponseHandler(command)
				case GetCommonBlocksRequest:
					downloadManager.GetCommonBlockRequestHandler(command)
				case GetCommonBlocksResponse:
					downloadManager.GetCommonBlockResponseHandler(command)
				}
			}
		}
	}()
}

func (downloadManager *DownloadManager) StartDownloadBlockchain(finishChan chan bool) {
	downloadManager.mutex.Lock()

	downloadManager.peersInfo = make(map[peer.ID]*PeerBlockInfo)
	downloadManager.finishChan = finishChan
	downloadManager.status = DownloadStatusInit

	for _, peer := range downloadManager.node.GetNetwork().GetPeers() {
		downloadManager.peersInfo[peer.PeerId] = &PeerBlockInfo{peerid: peer.PeerId, height: 0, libHeight: 0, status: PeerStatusInit}
	}
	peersNum := len(downloadManager.peersInfo)

	if peersNum == 0 {
		downloadManager.finishDownload()
		downloadManager.mutex.Unlock()
		return
	}
	downloadManager.mutex.Unlock()

	downloadManager.SendGetBlockchainInfoRequest()
	waitTimer := time.NewTimer(CheckMaxWaitTime * time.Second)
	logger.Info("DownloadManager: wait peer information")
	go func() {
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
	logger.Debugf("DownloadManager: Receive blockchain info %v %v \n", peerId, height)
	downloadManager.mutex.Lock()
	defer downloadManager.mutex.Unlock()

	if downloadManager.status != DownloadStatusInit {
		logger.Info("DownloadManager: Download peer started, skip PeerId ", peerId)
		return
	}

	blockPeerInfo, err := downloadManager.peersInfo[peerId]
	if err != true {
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
	returnBlocksLogger := logger.WithFields(logger.Fields{
		"name": "GetBlocksResponse",
	})

	if downloadManager.downloadingPeer == nil || downloadManager.downloadingPeer.peerid != peerId {
		returnBlocksLogger.Info("DownloadManager: peerId not in checklist")
		return nil, ErrPeerNotFound
	}

	if blocksPb.GetBlocks() == nil || len(blocksPb.GetBlocks()) == 0 {
		returnBlocksLogger.Error("DownloadManager: received no block.")
		return nil, ErrEmptyBlocks
	}

	hashes := make([]Hash, len(blocksPb.GetStartBlockHashes()))
	for index, hash := range blocksPb.GetStartBlockHashes() {
		hashes[index] = Hash(hash)
	}

	if downloadManager.isDownloadCommandFinished(hashes) {
		returnBlocksLogger.Info("DownloadManager: response is not for waiting command.")
		return nil, ErrMismatchResponse
	}

	return downloadManager.downloadingPeer, nil
}

func (downloadManager *DownloadManager) GetBlocksDataHandler(blocksPb *networkpb.ReturnBlocks, peerId peer.ID) {
	returnBlocksLogger := logger.WithFields(logger.Fields{
		"name": "GetBlocksResponse",
	})

	downloadManager.mutex.Lock()
	checkingPeer, err := downloadManager.validateReturnBlocks(blocksPb, peerId)
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

	var blocks []*Block
	for _, pbBlock := range blocksPb.GetBlocks() {
		block := &Block{}
		block.FromProto(pbBlock)

		if !downloadManager.node.GetBlockchainManager().VerifyBlock(block) {
			returnBlocksLogger.Warn("DownloadManager: verify block failed.")
			return
		}

		blocks = append(blocks, block)
	}

	logger.Warnf("DownloadManager: receive blocks source %v to %v.", blocks[0].GetHeight(), blocks[len(blocks)-1].GetHeight())

	if err := downloadManager.node.GetBlockchainManager().MergeFork(blocks, blocks[len(blocks)-1].GetPrevHash()); err != nil {
		returnBlocksLogger.Info("DownloadManager: merge fork failed.")
		return
	}

	downloadManager.mutex.Lock()
	defer downloadManager.mutex.Unlock()
	if downloadManager.node.GetBlockchain().GetMaxHeight() >= checkingPeer.height {
		downloadManager.finishDownload()
		lib, _ := downloadManager.node.GetBlockchain().GetBlockByHeight(checkingPeer.libHeight)
		downloadManager.node.GetBlockchain().SetLIBHash(lib.GetHash())
		logger.WithFields(logger.Fields{
			"lib_hash": lib.GetHash(),
		}).Info("DownloadManager: finishing download blocks.")
		return
	}

	var nextHashes []Hash
	for _, block := range blocks {
		nextHashes = append(nextHashes, block.GetHash())
	}

	downloadManager.sendDownloadCommand(nextHashes, peerId, 0)
}

func (downloadManager *DownloadManager) GetCommonBlockDataHandler(blocksPb *networkpb.ReturnCommonBlocks, peerId peer.ID) {
	downloadManager.mutex.Lock()

	if downloadManager.downloadingPeer == nil || downloadManager.downloadingPeer.peerid != peerId {
		logger.WithFields(logger.Fields{
			"name": "GetCommonBlocksResponse",
		}).Info("DownloadManager: PeerId not in checklist.")
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
	downloadManager.mutex.Unlock()
}

func (downloadManager *DownloadManager) DisconnectPeer(peerId peer.ID) {
	downloadManager.mutex.Lock()
	defer downloadManager.mutex.Unlock()

	if downloadManager.status == DownloadStatusFinish {
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
			block, err := downloadManager.node.GetBlockchain().GetBlockByHeight(currentHeight)
			if err != nil {
				continue
			}
			blockHeaders = append(blockHeaders, &SyncCommandBlocksHeader{hash: block.GetHash(), height: currentHeight})
		}
	}

	return blockHeaders
}

func (downloadManager *DownloadManager) FindCommonBlock(blockHeaders []*corepb.BlockHeader) (int, *Block) {
	findIndex := -1
	var commonBlock *Block

	for index, blockHeader := range blockHeaders {
		block, err := downloadManager.node.GetBlockchain().GetBlockByHeight(blockHeader.GetHeight())
		if err != nil {
			continue
		}

		if bytes.Compare([]byte(block.GetHash()), []byte(blockHeader.GetHash())) == 0 {
			findIndex = index
			commonBlock = block
			break
		}
	}
	if findIndex == -1 {
		logger.Panic("DownloadManager: invalid get common blocks result.")
	}
	return findIndex, commonBlock
}

func (downloadManager *DownloadManager) CheckGetCommonBlockCommand(msgId int32, peerId peer.ID, retryCount int) {
	downloadManager.mutex.Lock()
	defer downloadManager.mutex.Unlock()

	if !downloadManager.isSameGetCommonBlocksCommand(msgId) {
		return
	}

	if retryCount >= MaxRetryCount {
		peerInfo, ok := downloadManager.peersInfo[peerId]
		if ok {
			peerInfo.status = PeerStatusFailed
		}
		downloadManager.status = DownloadStatusSyncCommonBlocks
		downloadManager.downloadingPeer = nil
		downloadManager.currentCmd = nil
		downloadManager.startGetCommonBlocks(0)
	} else {
		syncCommand, _ := downloadManager.currentCmd.command.(*SyncCommonBlocksCommand)
		downloadManager.sendGetCommonBlockCommand(syncCommand.blockHeaders, peerId, retryCount+1)
	}
}

func (downloadManager *DownloadManager) CheckDownloadCommand(hashes []Hash, peerId peer.ID, retryCount int) {
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

	downloadManager.status = DownloadStatusSyncCommonBlocks
	highestPeer := downloadManager.selectHighestPeer()

	if highestPeer.peerid == downloadManager.node.GetPeerID() {
		downloadManager.finishDownload()
		logger.Info("DownloadManager: Current node has the highest block")
		return
	}

	downloadManager.downloadingPeer = highestPeer
	maxHeight := downloadManager.node.GetBlockchain().GetMaxHeight()
	blockHeaders := downloadManager.GetCommonBlockCheckPoint(0, maxHeight)
	downloadManager.sendGetCommonBlockCommand(blockHeaders, highestPeer.peerid, 0)
}

func (downloadManager *DownloadManager) sendGetCommonBlockCommand(blockHeaders []*SyncCommandBlocksHeader, peerId peer.ID, retryCount int) {
	downloadManager.msgId++
	msgId := downloadManager.msgId
	syncCommand := &SyncCommonBlocksCommand{msgId: msgId, blockHeaders: blockHeaders}

	downloadManager.currentCmd = &ExecuteCommand{command: syncCommand, retryCount: retryCount}
	downloadManager.SendGetCommonBlockRequest(blockHeaders, peerId, msgId)

	downloadTimer := time.NewTimer(DownloadMaxWaitTime * time.Second)
	go func() {
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

func (downloadManager *DownloadManager) checkGetCommonBlocksResult(blockHeaders []*corepb.BlockHeader) {
	findIndex, commonBlock := downloadManager.FindCommonBlock(blockHeaders)

	if findIndex == 0 || blockHeaders[findIndex-1].GetHeight()-blockHeaders[findIndex].GetHeight() == 1 {
		logger.Warnf("BlockManager: common height %v", commonBlock.GetHeight())
		downloadManager.commonHeight = commonBlock.GetHeight()
		downloadManager.currentCmd = nil
		downloadManager.startDownload(0)
	} else {
		blockHeaders := downloadManager.GetCommonBlockCheckPoint(
			blockHeaders[findIndex].GetHeight(),
			blockHeaders[findIndex-1].GetHeight(),
		)
		downloadManager.sendGetCommonBlockCommand(blockHeaders, downloadManager.downloadingPeer.peerid, 0)
	}
}

func (downloadManager *DownloadManager) startDownload(retryCount int) {
	if downloadManager.status != DownloadStatusSyncCommonBlocks {
		return
	}

	downloadManager.status = DownloadStatusDownloading

	producerNum := len(downloadManager.node.GetBlockchain().GetConsensus().GetProducers())
	if producerNum < MinRequestHashesNum {
		producerNum = MinRequestHashesNum
	}

	startBlockHeight := downloadManager.commonHeight
	var hashes []Hash
	for i := 0; i < producerNum && startBlockHeight-uint64(i) > 0; i++ {
		block, err := downloadManager.node.GetBlockchain().GetBlockByHeight(startBlockHeight - uint64(i))
		if err != nil {
			break
		}
		hashes = append(hashes, block.GetHash())
	}

	downloadManager.sendDownloadCommand(hashes, downloadManager.downloadingPeer.peerid, retryCount)
}

func (downloadManager *DownloadManager) sendDownloadCommand(hashes []Hash, peerId peer.ID, retryCount int) {
	downloadingCmd := &DownloadingCommandInfo{startHashes: hashes, finished: false}
	downloadManager.currentCmd = &ExecuteCommand{command: downloadingCmd, retryCount: retryCount}
	downloadManager.SendGetBlocksRequest(hashes, peerId)

	downloadTimer := time.NewTimer(DownloadMaxWaitTime * time.Second)
	go func() {
		<-downloadTimer.C
		downloadTimer.Stop()
		downloadManager.CheckDownloadCommand(hashes, peerId, retryCount)
	}()
}

func (downloadManager *DownloadManager) isDownloadCommandFinished(hashes []Hash) bool {
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
	downloadManager.status = DownloadStatusFinish
	downloadManager.downloadingPeer = nil
	downloadManager.currentCmd = nil
	downloadManager.finishChan <- true
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
	currentBlockInfo := &PeerBlockInfo{
		peerid:    downloadManager.node.GetPeerID(),
		height:    downloadManager.node.GetBlockchain().GetMaxHeight(),
		libHeight: downloadManager.node.GetBlockchain().GetLIBHeight(),
		status:    PeerStatusReady,
	}

	for _, peerInfo := range downloadManager.peersInfo {
		if peerInfo.status == PeerStatusReady && peerInfo.libHeight > currentBlockInfo.libHeight {
			currentBlockInfo = peerInfo
		} else if peerInfo.status == PeerStatusReady && peerInfo.libHeight == currentBlockInfo.libHeight && peerInfo.height > currentBlockInfo.height {
			currentBlockInfo = peerInfo
		}
	}
	return currentBlockInfo
}

func (downloadManager *DownloadManager) SubscribeCommandBroker(broker *network.CommandBroker) {
	for _, topic := range subscribedTopics {
		err := broker.Subscribe(topic, downloadManager.commandReceiveCh)
		if err != nil {
			logger.WithError(err).WithFields(logger.Fields{
				"command": topic,
			}).Warn("DownloadManager: Unable to subscribe to a command")
		}
	}
}

func (downloadManager *DownloadManager) SendGetCommonBlockRequest(blockHeaders []*SyncCommandBlocksHeader, pid peer.ID, msgId int32) {
	var blockHeaderPbs []*corepb.BlockHeader
	for _, blockHeader := range blockHeaders {
		blockHeaderPbs = append(blockHeaderPbs,
			&corepb.BlockHeader{Hash: blockHeader.hash, Height: blockHeader.height})
	}

	getCommonBlocksPb := &networkpb.GetCommonBlocks{MsgId: msgId, BlockHeaders: blockHeaderPbs}

	command := network.NewDappSendCmdContext(GetCommonBlocksRequest, getCommonBlocksPb, pid, network.Unicast, network.HighPriorityCommand)

	command.Send(downloadManager.commandSendCh)
}

func (downloadManager *DownloadManager) GetCommonBlockRequestHandler(command *network.DappRcvdCmdContext) {

	param := &networkpb.GetCommonBlocks{}
	if err := proto.Unmarshal(command.GetData(), param); err != nil {
		logger.WithFields(logger.Fields{
			"name": "GetCommonBlocksRequest",
		}).Warn("DownloadManager: parse data failed.")
		return
	}

	downloadManager.SendGetCommonBlockResponse(param.GetBlockHeaders(), param.GetMsgId(), command.GetSource())

}

func (downloadManager *DownloadManager) SendGetCommonBlockResponse(blockHeaders []*corepb.BlockHeader, msgId int32, destination peer.ID) {
	index, _ := downloadManager.FindCommonBlock(blockHeaders)
	var blockHeaderPbs []*corepb.BlockHeader
	if index == 0 {
		blockHeaderPbs = blockHeaders[:1]
	} else {
		blockHeaders := downloadManager.GetCommonBlockCheckPoint(
			blockHeaders[index].GetHeight(),
			blockHeaders[index-1].GetHeight(),
		)
		for _, blockHeader := range blockHeaders {
			blockHeaderPbs = append(blockHeaderPbs,
				&corepb.BlockHeader{Hash: blockHeader.hash, Height: blockHeader.height})
		}
	}

	result := &networkpb.ReturnCommonBlocks{MsgId: msgId, BlockHeaders: blockHeaderPbs}

	command := network.NewDappSendCmdContext(GetCommonBlocksResponse, result, destination, network.Unicast, network.HighPriorityCommand)

	command.Send(downloadManager.commandSendCh)

}

func (downloadManager *DownloadManager) GetCommonBlockResponseHandler(command *network.DappRcvdCmdContext) {
	param := &networkpb.ReturnCommonBlocks{}

	if err := proto.Unmarshal(command.GetData(), param); err != nil {
		logger.WithFields(logger.Fields{
			"name": "GetCommonBlocksResponse",
		}).Info("DownloadManager: parse data failed.")
	}

	downloadManager.GetCommonBlockDataHandler(param, command.GetSource())
}

func (downloadManager *DownloadManager) SendGetBlocksRequest(hashes []Hash, pid peer.ID) {
	blkHashes := make([][]byte, len(hashes))
	for index, hash := range hashes {
		blkHashes[index] = hash
	}

	getBlockPb := &networkpb.GetBlocks{StartBlockHashes: blkHashes}

	command := network.NewDappSendCmdContext(GetBlocksRequest, getBlockPb, pid, network.Unicast, network.HighPriorityCommand)

	command.Send(downloadManager.commandSendCh)
}

func (downloadManager *DownloadManager) GetBlocksRequestHandler(command *network.DappRcvdCmdContext) {

	param := &networkpb.GetBlocks{}
	if err := proto.Unmarshal(command.GetData(), param); err != nil {
		logger.WithFields(logger.Fields{
			"name": "GetBlocksRequest",
		}).Info("DownloadManager: parse data failed.")
		return
	}

	downloadManager.SendGetBlocksResponse(param.GetStartBlockHashes(), command.GetSource())

}

func (downloadManager *DownloadManager) SendGetBlocksResponse(startBlockHashes [][]byte, destination peer.ID) {

	block := downloadManager.findBlockInRequestHash(startBlockHashes)

	// Reach the blockchain's tail
	if block.GetHeight() >= downloadManager.node.GetBlockchain().GetMaxHeight() {
		logger.WithFields(logger.Fields{
			"name": "GetBlocksRequest",
		}).Info("DownloadManager: reach blockchain tail.")
		return
	}

	var blocks []*Block

	block, err := downloadManager.node.GetBlockchain().GetBlockByHeight(block.GetHeight() + 1)
	for i := int32(0); i < maxGetBlocksNum && err == nil; i++ {
		if block.GetHeight() == 0 {
			logger.Panicf("Error %v", hex.EncodeToString(block.GetHash()))
		}
		blocks = append(blocks, block)
		block, err = downloadManager.node.GetBlockchain().GetBlockByHeight(block.GetHeight() + 1)
	}

	var blockPbs []*corepb.Block
	for i := len(blocks) - 1; i >= 0; i-- {
		blockPbs = append(blockPbs, blocks[i].ToProto().(*corepb.Block))
	}

	result := &networkpb.ReturnBlocks{Blocks: blockPbs, StartBlockHashes: startBlockHashes}

	command := network.NewDappSendCmdContext(GetBlocksResponse, result, destination, network.Unicast, network.HighPriorityCommand)

	command.Send(downloadManager.commandSendCh)

}

func (downloadManager *DownloadManager) GetBlocksResponseHandler(command *network.DappRcvdCmdContext) {
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
	request := &networkpb.GetBlockchainInfo{Version: network.ProtocalName}

	var destination peer.ID
	command := network.NewDappSendCmdContext(BlockchainInfoRequest, request, destination, network.Broadcast, network.NormalPriorityCommand)

	command.Send(downloadManager.commandSendCh)
}

func (downloadManager *DownloadManager) GetBlockchainInfoRequestHandler(command *network.DappRcvdCmdContext) {

	downloadManager.SendGetBlockchainInfoResponse(command.GetSource())

}

func (downloadManager *DownloadManager) SendGetBlockchainInfoResponse(destination peer.ID) {

	tailBlock, err := downloadManager.node.GetBlockchain().GetTailBlock()
	if err != nil {
		logger.WithFields(logger.Fields{
			"name": "GetBlockchainInfoRequest",
		}).Warn("DownloadManager: get tail block failed.")
		return
	}

	result := &networkpb.ReturnBlockchainInfo{
		TailBlockHash: downloadManager.node.GetBlockchain().GetTailBlockHash(),
		BlockHeight:   downloadManager.node.GetBlockchain().GetMaxHeight(),
		Timestamp:     tailBlock.GetTimestamp(),
		LibHash:       downloadManager.node.GetBlockchain().GetLIBHash(),
		LibHeight:     downloadManager.node.GetBlockchain().GetLIBHeight(),
	}

	command := network.NewDappSendCmdContext(BlockchainInfoResponse, result, destination, network.Unicast, network.NormalPriorityCommand)

	command.Send(downloadManager.commandSendCh)

}

func (downloadManager *DownloadManager) GetBlockchainInfoResponseHandler(command *network.DappRcvdCmdContext) {
	blockchainInfo := &networkpb.ReturnBlockchainInfo{}
	if err := proto.Unmarshal(command.GetData(), blockchainInfo); err != nil {
		logger.WithFields(logger.Fields{
			"name": "BlockchainInfoResponse",
		}).Info("DownloadManager: parse data failed.")
		return
	}

	downloadManager.AddPeerBlockChainInfo(command.GetSource(), blockchainInfo.GetBlockHeight(), blockchainInfo.GetLibHeight())
}
func (downloadManager *DownloadManager) OnStreamStopHandler(command *network.DappRcvdCmdContext) {

	peerInfopb := &networkpb.PeerInfo{}
	if err := proto.Unmarshal(command.GetData(), peerInfopb); err != nil {
		logger.WithFields(logger.Fields{
			"name": "OnStreamStop",
		}).Info("DownloadManager: parse data failed.")
		return
	}

	var peerInfo network.PeerInfo
	if err := peerInfo.FromProto(peerInfopb); err != nil {
		logger.WithFields(logger.Fields{
			"name": "OnStreamStop",
		}).Info("DownloadManager: parse data from proto message failed.")
		return
	}

	downloadManager.DisconnectPeer(peerInfo.PeerId)

}

func (downloadManager *DownloadManager) findBlockInRequestHash(startBlockHashes [][]byte) *Block {
	for _, hash := range startBlockHashes {
		// hash in blockchain, return
		if block, err := downloadManager.node.GetBlockchain().GetBlockByHash(hash); err == nil {
			return block
		}
	}

	// Return Genesis Block
	block, _ := downloadManager.node.GetBlockchain().GetBlockByHeight(0)
	return block
}
