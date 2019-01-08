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
	"github.com/dappley/go-dappley/core"
	"github.com/dappley/go-dappley/network/pb"
	"reflect"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p-peer"
	logger "github.com/sirupsen/logrus"
)

const (
	PeerStatusInit   int = 0
	PeerStatusReady  int = 1
	PeerStatusFailed int = 2

	DownloadStatusInit        int = 0
	DownloadStatusDownloading int = 1
	DownloadStatusFinish      int = 2

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

type DownloadingCommandInfo struct {
	startHashes []core.Hash
	retryCount  int
}

type DownloadManager struct {
	peersInfo       map[peer.ID]*PeerBlockInfo
	downloadingPeer *PeerBlockInfo
	downloadingCmd  *DownloadingCommandInfo
	node            *Node
	mutex           sync.RWMutex
	status          int
	finishChan      chan bool
}

func NewDownloadManager(node *Node) *DownloadManager {
	return &DownloadManager{
		peersInfo:       make(map[peer.ID]*PeerBlockInfo),
		downloadingPeer: nil,
		downloadingCmd:  nil,
		node:            node,
		mutex:           sync.RWMutex{},
		status:          DownloadStatusInit,
		finishChan:      nil,
	}
}

func (downloadManager *DownloadManager) StartDownloadBlockchain(finishChan chan bool) {
	downloadManager.mutex.Lock()
	defer downloadManager.mutex.Unlock()

	downloadManager.peersInfo = make(map[peer.ID]*PeerBlockInfo)
	downloadManager.finishChan = finishChan

	for _, peer := range downloadManager.node.GetPeerList().GetPeerlist() {
		downloadManager.peersInfo[peer.peerid] = &PeerBlockInfo{peerid: peer.peerid, height: 0, status: PeerStatusInit}
	}

	if len(downloadManager.peersInfo) == 0 {
		downloadManager.finishDownload()
		return
	}

	downloadManager.node.BroadcastGetBlockchainInfo()
	waitTimer := time.NewTimer(CheckMaxWaitTime * time.Second)
	logger.Warn("Wait peer information")
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
		logger.Warn("Start download")
		downloadManager.startDownload(0)
	}()
}

func (downloadManager *DownloadManager) AddPeerBlockChainInfo(peerId peer.ID, height uint64) {
	logger.Debugf("DownloadManager: Receive blockchain info %v %v \n", peerId, height)
	downloadManager.mutex.Lock()
	defer downloadManager.mutex.Unlock()

	if downloadManager.status != DownloadStatusInit {
		logger.Info("DownloadManager: Download peer started, skip peerId ", peerId)
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
		downloadManager.startDownload(0)
	}
}

func (downloadManager *DownloadManager) GetBlocksDataHandler(blocksPb *networkpb.ReturnBlocks, peerId peer.ID) {
	downloadManager.mutex.Lock()
	if downloadManager.status != DownloadStatusDownloading {
		logger.WithFields(logger.Fields{
			"cmd": "ReturnBlocks",
		}).Info("DownloadManager: Not in downloading state.")
		downloadManager.mutex.Unlock()
		return
	}

	if downloadManager.downloadingPeer.peerid != peerId {
		logger.WithFields(logger.Fields{
			"cmd": "ReturnBlocks",
		}).Info("DownloadManager: PeerId not in checklist")
		downloadManager.mutex.Unlock()
		return
	}
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

	if downloadManager.node.GetBlockchain().GetMaxHeight() >= downloadManager.downloadingPeer.height {
		downloadManager.finishDownload()
		return
	}

	var hashes []core.Hash
	for _, block := range blocks {
		hashes = append(hashes, block.GetHash())
	}

	downloadManager.sendDownloadCommand(hashes, peerId, 0)
}

func (downloadManager *DownloadManager) DisconnectPeer(peerId peer.ID) {
	downloadManager.mutex.Lock()
	defer downloadManager.mutex.Unlock()

	if downloadManager.status == DownloadStatusFinish {
		return
	}

	delete(downloadManager.peersInfo, peerId)
	if downloadManager.status == DownloadStatusDownloading {
		if downloadManager.downloadingPeer.peerid == peerId {
			downloadManager.status = DownloadStatusInit
			downloadManager.downloadingPeer = nil
			downloadManager.downloadingCmd = nil
			downloadManager.startDownload(0)
		}
	}
}

func (downloadManager *DownloadManager) startDownload(retryCount int) {
	if downloadManager.status != DownloadStatusInit {
		logger.Info("DownloadManager: Download peers started")
		return
	}

	downloadManager.status = DownloadStatusDownloading
	highestPeer := downloadManager.selectHighestPeer()

	if highestPeer.peerid == downloadManager.node.GetPeerID() {
		downloadManager.finishDownload()
		logger.Warn("DownloadManager: Current node has the highest block")
		return
	}

	downloadManager.downloadingPeer = highestPeer

	producerNum := len(downloadManager.node.GetBlockchain().GetConsensus().GetProducers())
	if producerNum < MinRequestHashesNum {
		producerNum = MinRequestHashesNum
	}

	startBlockHeight := downloadManager.node.GetBlockchain().GetMaxHeight()

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
	downloadManager.downloadingCmd = &DownloadingCommandInfo{startHashes: hashes}
	downloadManager.node.DownloadBlocksUnicast(hashes, peerId)

	downloadTimer := time.NewTimer(DownloadMaxWaitTime * time.Second)
	go func() {
		<-downloadTimer.C
		downloadTimer.Stop()
		downloadManager.checkDownloadCommand(hashes, peerId, retryCount)
	}()
}

func (downloadManager *DownloadManager) checkDownloadCommand(hashes []core.Hash, peerId peer.ID, retryCount int) {
	downloadManager.mutex.Lock()
	defer downloadManager.mutex.Unlock()

	if downloadManager.downloadingCmd == nil {
		return
	}

	if !reflect.DeepEqual(downloadManager.downloadingCmd.startHashes, hashes){
		return
	}

	if retryCount >= MaxRetryCount {
		peerInfo, ok := downloadManager.peersInfo[peerId]
		if ok {
			peerInfo.status = PeerStatusFailed
		}
		downloadManager.status = DownloadStatusInit
		downloadManager.downloadingPeer = nil
		downloadManager.downloadingCmd = nil
		downloadManager.startDownload(0)
	} else {
		downloadManager.sendDownloadCommand(hashes, peerId, retryCount+1)
	}
}

func (downloadManager *DownloadManager) finishDownload() {
	downloadManager.status = DownloadStatusFinish
	downloadManager.downloadingPeer = nil
	downloadManager.downloadingCmd = nil
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
