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
	"bytes"
	"math"
	"sync"
	"time"

	"github.com/dappley/go-dappley/core"
	"github.com/dappley/go-dappley/core/pb"
	"github.com/dappley/go-dappley/network/pb"
	"github.com/libp2p/go-libp2p-peer"
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
	DownloadMaxWaitTime time.Duration = 10
	MaxRetryCount       int           = 3
	MinRequestHashesNum int           = 20
)

type PeerBlockInfo struct {
	peerid peer.ID
	height uint64
	status int
}

type DownloadCommand struct {
	startHashes []core.Hash
}

type DownloadingCommandInfo struct {
	startHashes []core.Hash
	retryCount  int
}

type SyncCommandBlocksHeader struct {
	hash   core.Hash
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
	peersInfo       map[peer.ID]*PeerBlockInfo
	downloadingPeer *PeerBlockInfo
	currentCmd      *ExecuteCommand
	node            *Node
	mutex           sync.RWMutex
	status          int
	commonHeight    uint64
	msgId           int32
	isMerging       bool
	finishChan      chan bool
}

func NewDownloadManager(node *Node) *DownloadManager {
	return &DownloadManager{
		peersInfo:       make(map[peer.ID]*PeerBlockInfo),
		downloadingPeer: nil,
		currentCmd:      nil,
		node:            node,
		mutex:           sync.RWMutex{},
		status:          DownloadStatusInit,
		msgId:           0,
		commonHeight:    0,
		isMerging:       false,
		finishChan:      nil,
	}
}

func (downloadManager *DownloadManager) StartDownloadBlockchain(finishChan chan bool) {
	downloadManager.mutex.Lock()

	downloadManager.peersInfo = make(map[peer.ID]*PeerBlockInfo)
	downloadManager.finishChan = finishChan

	for _, peer := range downloadManager.node.GetPeerManager().CloneStreamsToSlice() {
		downloadManager.peersInfo[peer.stream.peerID] = &PeerBlockInfo{peerid: peer.stream.peerID, height: 0, status: PeerStatusInit}
	}
	peersNum := len(downloadManager.peersInfo)

	if peersNum == 0 {
		downloadManager.finishDownload()
		downloadManager.mutex.Unlock()
		return
	}
	downloadManager.mutex.Unlock()

	downloadManager.node.BroadcastGetBlockchainInfo()
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

func (downloadManager *DownloadManager) AddPeerBlockChainInfo(peerId peer.ID, height uint64) {
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
	blockPeerInfo.status = PeerStatusReady

	if downloadManager.canStartDownload() {
		downloadManager.startGetCommonBlocks(0)
	}
}

func (downloadManager *DownloadManager) GetBlocksDataHandler(blocksPb *networkpb.ReturnBlocks, peerId peer.ID) {
	downloadManager.mutex.Lock()

	if downloadManager.downloadingPeer.peerid != peerId {
		logger.WithFields(logger.Fields{
			"cmd": "ReturnBlocks",
		}).Info("DownloadManager: peerId not in checklist")
		downloadManager.mutex.Unlock()
		return
	}

	hashes := make([]core.Hash, len(blocksPb.StartBlockHashes))
	for index, hash := range blocksPb.StartBlockHashes {
		hashes[index] = core.Hash(hash)
	}
	if !downloadManager.isSameDownloadCommand(hashes) {
		logger.WithFields(logger.Fields{
			"cmd": "ReturnBlocks",
		}).Info("DownloadManager: response is not for waiting command.")
		downloadManager.mutex.Unlock()
		return
	}

	downloadManager.isMerging = true
	downloadManager.mutex.Unlock()

	var blocks []*core.Block
	for _, pbBlock := range blocksPb.Blocks {
		block := &core.Block{}
		block.FromProto(pbBlock)

		if !downloadManager.node.bm.VerifyBlock(block) {
			logger.WithFields(logger.Fields{
				"cmd": "ReturnBlocks",
			}).Warn("DownloadManager: Verify block failed.")
			return
		}

		blocks = append(blocks, block)
	}

	if err := downloadManager.node.bm.MergeFork(blocks, blocks[len(blocks)-1].GetPrevHash()); err != nil {
		logger.WithFields(logger.Fields{
			"cmd": "ReturnBlocks",
		}).Info("DownloadManager: Merge fork failed.")
		return
	}

	downloadManager.mutex.Lock()
	downloadManager.isMerging = false
	if downloadManager.node.GetBlockchain().GetMaxHeight() >= downloadManager.downloadingPeer.height {
		downloadManager.finishDownload()
		downloadManager.mutex.Unlock()
		return
	}

	var nextHashes []core.Hash
	for _, block := range blocks {
		nextHashes = append(nextHashes, block.GetHash())
	}

	downloadManager.sendDownloadCommand(nextHashes, peerId, 0)
	downloadManager.mutex.Unlock()
}

func (downloadManager *DownloadManager) GetCommonBlockDataHandler(blocksPb *networkpb.ReturnCommonBlocks, peerId peer.ID) {
	downloadManager.mutex.Lock()

	if downloadManager.downloadingPeer.peerid != peerId {
		logger.WithFields(logger.Fields{
			"cmd": "ReturnCommonBlocks",
		}).Info("DownloadManager: PeerId not in checklist.")
		downloadManager.mutex.Unlock()
		return
	}

	if !downloadManager.isSameGetCommonBlocksCommand(blocksPb.MsgId) {
		logger.WithFields(logger.Fields{
			"cmd": "ReturnCommonBlocks",
		}).Info("DownloadManager: response is not for waiting command.")
		downloadManager.mutex.Unlock()
		return
	}

	downloadManager.checkGetCommonBlocksResult(blocksPb.BlockHeaders)
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
			block, _ := downloadManager.node.GetBlockchain().GetBlockByHeight(currentHeight)
			blockHeaders = append(blockHeaders, &SyncCommandBlocksHeader{hash: block.GetHash(), height: currentHeight})
		}
	}

	return blockHeaders
}

func (downloadManager *DownloadManager) FindCommonBlock(blockHeaders []*corepb.BlockHeader) (int, *core.Block) {
	findIndex := -1
	var commonBlock *core.Block

	for index, blockHeader := range blockHeaders {
		block, err := downloadManager.node.GetBlockchain().GetBlockByHeight(blockHeader.Height)
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

func (downloadManager *DownloadManager) CheckDownloadCommand(hashes []core.Hash, peerId peer.ID, retryCount int) {
	downloadManager.mutex.Lock()
	defer downloadManager.mutex.Unlock()

	if !downloadManager.isSameDownloadCommand(hashes) {
		return
	}

	if downloadManager.isMerging {
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
	downloadManager.node.GetCommonBlocksUnicast(blockHeaders, peerId, msgId)

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

	if findIndex == 0 || blockHeaders[findIndex-1].Height-blockHeaders[findIndex].Height == 1 {
		logger.Warnf("BlockManager: common height %v", commonBlock.GetHeight())
		downloadManager.commonHeight = commonBlock.GetHeight()
		downloadManager.currentCmd = nil
		downloadManager.startDownload(0)
	} else {
		blockHeaders := downloadManager.GetCommonBlockCheckPoint(
			blockHeaders[findIndex].Height,
			blockHeaders[findIndex-1].Height,
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
	var hashes []core.Hash
	for i := 0; i < producerNum && startBlockHeight-uint64(i) > 0; i++ {
		block, err := downloadManager.node.GetBlockchain().GetBlockByHeight(startBlockHeight - uint64(i))
		if err != nil {
			break
		}
		hashes = append(hashes, block.GetHash())
	}

	downloadManager.sendDownloadCommand(hashes, downloadManager.downloadingPeer.peerid, retryCount)
}

func (downloadManager *DownloadManager) sendDownloadCommand(hashes []core.Hash, peerId peer.ID, retryCount int) {
	downloadingCmd := &DownloadingCommandInfo{startHashes: hashes}
	downloadManager.currentCmd = &ExecuteCommand{command: downloadingCmd, retryCount: retryCount}
	downloadManager.node.DownloadBlocksUnicast(hashes, peerId)

	downloadTimer := time.NewTimer(DownloadMaxWaitTime * time.Second)
	go func() {
		<-downloadTimer.C
		downloadTimer.Stop()
		downloadManager.CheckDownloadCommand(hashes, peerId, retryCount)
	}()
}

func (downloadManager *DownloadManager) isSameDownloadCommand(hashes []core.Hash) bool {
	if downloadManager.currentCmd == nil {
		return false
	}

	if downloadManager.status != DownloadStatusDownloading {
		return false
	}

	downloadingCmd, ok := downloadManager.currentCmd.command.(*DownloadingCommandInfo)

	if !ok {
		return false
	}

	if len(hashes) != len(downloadingCmd.startHashes) {
		return false
	}

	for index, hash := range hashes {
		if bytes.Compare(hash, downloadingCmd.startHashes[index]) != 0 {
			return false
		}
	}

	return true
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
		peerid: downloadManager.node.GetPeerID(),
		height: downloadManager.node.GetBlockchain().GetMaxHeight(),
		status: PeerStatusReady,
	}

	for _, peerInfo := range downloadManager.peersInfo {
		if peerInfo.status == PeerStatusReady && peerInfo.height > currentBlockInfo.height {
			currentBlockInfo = peerInfo
		}
	}
	return currentBlockInfo
}