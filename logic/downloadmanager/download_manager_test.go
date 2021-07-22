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
	blockpb "github.com/dappley/go-dappley/core/block/pb"
	"io/ioutil"
	"os"
	"path"
	"strconv"
	"testing"

	"github.com/dappley/go-dappley/logic/lblockchain/mocks"
	networkpb "github.com/dappley/go-dappley/network/pb"
	"github.com/stretchr/testify/mock"

	"github.com/dappley/go-dappley/core/blockchain"
	"github.com/dappley/go-dappley/logic/lblockchain"
	"github.com/dappley/go-dappley/network"
	"github.com/dappley/go-dappley/storage"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/stretchr/testify/assert"
)

const (
	multiPortEqualStart      int = 10301
	multiPortSuccessStart    int = 10310
	multiPortDisconnectStart int = 10320
	multiPortNotEqualStart   int = 10330
	multiPortReturnBlocks    int = 10340
	multiPortDisconnectPeer  int = 10350
	multiPortCommonBlocks    int = 10360
	confDir                      = "../../storage/fakeFileLoaders/"
)

func createTestBlockchains(size int, portStart int) ([]*lblockchain.BlockchainManager, []*network.Node) {
	bms := make([]*lblockchain.BlockchainManager, size)
	nodes := make([]*network.Node, size)
	bc := lblockchain.GenerateMockBlockchainWithCoinbaseTxOnly(size)
	consensus := &mocks.Consensus{}
	consensus.On("Validate", mock.Anything).Return(true)
	consensus.On("ChangeDynasty", mock.Anything).Return(true)
	consensus.On("SetDynasty", mock.Anything).Return(true)
	consensus.On("AddReplacement", mock.Anything).Return(true)
	for i := 0; i < size; i++ {
		rfl := storage.NewRamFileLoader(confDir, "dl"+strconv.Itoa(i)+".conf")
		node := network.NewNode(rfl.File, nil)
		node.Start(portStart+i, "")
		bm := lblockchain.NewBlockchainManager(bc, blockchain.NewBlockPool(nil), node, consensus)
		bms[i] = bm
		nodes[i] = node
	}
	return bms, nodes
}

func TestMultiEqualNode(t *testing.T) {
	bms, nodes := createTestBlockchains(5, multiPortEqualStart)
	defer deleteConfFolderFiles()
	//setup download manager for the first node
	bm := bms[0]
	bm.Getblockchain().SetState(blockchain.BlockchainInit)
	node := nodes[0]
	downloadManager := NewDownloadManager(node, bm, 0, nil)
	downloadManager.Start()

	//Connect all other nodes to the first node
	for i := 1; i < len(nodes); i++ {
		node.GetNetwork().ConnectToSeed(nodes[i].GetHostPeerInfo())
	}

	oldHeight := bm.Getblockchain().GetMaxHeight()

	finishCh := make(chan bool, 1)
	downloadManager.StartDownloadBlockchain(finishCh)
	<-finishCh

	assert.Equal(t, oldHeight, bm.Getblockchain().GetMaxHeight())
}

func TestMultiNotEqualNode(t *testing.T) {
	bms, nodes := createTestBlockchains(5, multiPortNotEqualStart)
	defer deleteConfFolderFiles()
	bm := bms[0]
	bm.Getblockchain().SetState(blockchain.BlockchainInit)
	node := nodes[0]
	downloadManager := NewDownloadManager(node, bm, 0, nil)
	downloadManager.Start()

	highestChain := bms[1]
	lblockchain.AddBlockToGeneratedBlockchain(highestChain.Getblockchain(), 100)

	for i := 1; i < len(nodes); i++ {
		node.GetNetwork().ConnectToSeed(nodes[i].GetHostPeerInfo())
	}

	highestChain.Getblockchain().SetState(blockchain.BlockchainInit)
	highestChainNode := nodes[1]
	highestChainDownloadManager := NewDownloadManager(highestChainNode, highestChain, 0, nil)
	highestChainDownloadManager.Start()

	finishCh := make(chan bool, 1)
	bm.Getblockchain().SetState(blockchain.BlockchainDownloading)
	downloadManager.StartDownloadBlockchain(finishCh)
	<-finishCh
	bm.Getblockchain().SetState(blockchain.BlockchainReady)

	assert.Equal(t, highestChain.Getblockchain().GetMaxHeight(), bm.Getblockchain().GetMaxHeight())
}

func TestMultiSuccessNode(t *testing.T) {
	bms, nodes := createTestBlockchains(5, multiPortSuccessStart)
	defer deleteConfFolderFiles()
	bm := bms[0]
	bm.Getblockchain().SetState(blockchain.BlockchainInit)
	node := nodes[0]
	downloadManager := NewDownloadManager(node, bm, 0, nil)
	downloadManager.Start()

	highestChain := bms[1]
	lblockchain.AddBlockToGeneratedBlockchain(highestChain.Getblockchain(), 200)

	for i := 1; i < len(nodes); i++ {
		node.GetNetwork().ConnectToSeed(nodes[i].GetHostPeerInfo())
	}

	highestChain.Getblockchain().SetState(blockchain.BlockchainInit)
	highestChainNode := nodes[1]
	highestChainDownloadManager := NewDownloadManager(highestChainNode, highestChain, 0, nil)
	highestChainDownloadManager.Start()

	finishCh := make(chan bool, 1)
	bm.Getblockchain().SetState(blockchain.BlockchainDownloading)
	downloadManager.StartDownloadBlockchain(finishCh)
	<-finishCh
	bm.Getblockchain().SetState(blockchain.BlockchainReady)

	assert.Equal(t, highestChain.Getblockchain().GetMaxHeight(), bm.Getblockchain().GetMaxHeight())
}

func TestDisconnectNode(t *testing.T) {
	bms, nodes := createTestBlockchains(5, multiPortDisconnectStart)
	defer deleteConfFolderFiles()
	bm := bms[0]
	bm.Getblockchain().SetState(blockchain.BlockchainInit)
	node := nodes[0]
	downloadManager := NewDownloadManager(node, bm, 0, nil)
	downloadManager.Start()

	highestChain := bms[1]
	lblockchain.AddBlockToGeneratedBlockchain(highestChain.Getblockchain(), 400)
	highestChainNode := nodes[1]
	highestChainDownloadManager := NewDownloadManager(highestChainNode, highestChain, 0, nil)
	highestChainDownloadManager.Start()

	secondChain := bms[2]
	lblockchain.AddBlockToGeneratedBlockchain(highestChain.Getblockchain(), 300)
	secondChainNode := nodes[2]
	secondChainDownloadManager := NewDownloadManager(secondChainNode, secondChain, 0, nil)
	secondChainDownloadManager.Start()

	for i := 1; i < len(nodes); i++ {
		node.GetNetwork().ConnectToSeed(nodes[i].GetHostPeerInfo())
	}
	info := node.GetPeers()
	for _, i := range info {
		println(i.PeerId)
	}
	finishCh := make(chan bool, 1)
	bm.Getblockchain().SetState(blockchain.BlockchainDownloading)
	downloadManager.StartDownloadBlockchain(finishCh)
	highestChainNode.Stop()
	<-finishCh
	bm.Getblockchain().SetState(blockchain.BlockchainReady)

	assert.Equal(t, secondChain.Getblockchain().GetMaxHeight(), bm.Getblockchain().GetMaxHeight())
}

func TestValidateReturnBlocks(t *testing.T) {
	// Test empty blocks in GetBlocksResponse message
	bms, nodes := createTestBlockchains(2, multiPortReturnBlocks)
	defer deleteConfFolderFiles()
	bm := bms[0]
	bm.Getblockchain().SetState(blockchain.BlockchainInit)
	node := nodes[0]
	downloadManager := NewDownloadManager(node, bm, 0, nil)

	peerNode := nodes[1]

	node.GetNetwork().ConnectToSeed(peerNode.GetHostPeerInfo())
	downloadManager.peersInfo = make(map[peer.ID]*PeerBlockInfo)

	for _, p := range downloadManager.node.GetPeers() {
		downloadManager.peersInfo[p.PeerId] = &PeerBlockInfo{peerid: p.PeerId, height: 0, status: PeerStatusInit}
		downloadManager.downloadingPeer = downloadManager.peersInfo[p.PeerId]
	}
	bm.Getblockchain().SetState(blockchain.BlockchainDownloading)

	// test invalid peer id
	_, err := downloadManager.validateReturnBlocks(nil, "foo")
	assert.Equal(t, ErrPeerNotFound, err)

	// test empty blocks
	fakeReturnMsg := &networkpb.ReturnBlocks{Blocks: nil, StartBlockHashes: nil}
	_, err = downloadManager.validateReturnBlocks(fakeReturnMsg, peerNode.GetHostPeerInfo().PeerId)
	assert.Equal(t, ErrEmptyBlocks, err)
}

func TestDownloadManager_DisconnectPeer(t *testing.T) {
	bms, nodes := createTestBlockchains(5, multiPortDisconnectPeer)
	defer deleteConfFolderFiles()
	//setup download manager for the first node
	bm := bms[0]
	bm.Getblockchain().SetState(blockchain.BlockchainInit)
	node := nodes[0]

	downloadManager := NewDownloadManager(node, bm, 0, nil)
	downloadManager.Start()

	//Connect all other nodes to the first node
	for i := 1; i < len(nodes); i++ {
		node.GetNetwork().ConnectToSeed(nodes[i].GetHostPeerInfo())
	}

	finishCh := make(chan bool, 1)
	bm.Getblockchain().SetState(blockchain.BlockchainDownloading)
	downloadManager.StartDownloadBlockchain(finishCh)
	<-finishCh

	downloadManager.status = DownloadStatusInit
	assert.Equal(t, 4, len(downloadManager.peersInfo))
	downloadManager.DisconnectPeer(nodes[3].GetHostPeerInfo().PeerId)
	assert.Equal(t, 3, len(downloadManager.peersInfo))
	downloadManager.DisconnectPeer(nodes[2].GetHostPeerInfo().PeerId)
	assert.Equal(t, 2, len(downloadManager.peersInfo))
}

func TestDownloadManager_FindCommonBlock(t *testing.T) {
	bms, nodes := createTestBlockchains(5, multiPortCommonBlocks)
	defer deleteConfFolderFiles()
	//setup download manager for the first node
	bm := bms[0]
	bm.Getblockchain().SetState(blockchain.BlockchainInit)
	node := nodes[0]

	downloadManager := NewDownloadManager(node, bm, 0, nil)
	downloadManager.Start()

	maxHeight := downloadManager.bm.Getblockchain().GetMaxHeight()
	blockHeaders := downloadManager.GetCommonBlockCheckPoint(0, maxHeight)

	var blockHeaderPbs []*blockpb.BlockHeader
	for _, blockHeader := range blockHeaders {
		blockHeaderPbs = append(blockHeaderPbs,
			&blockpb.BlockHeader{Hash: blockHeader.hash, Height: blockHeader.height})
	}

	// first block is common block
	index, block := downloadManager.FindCommonBlock(blockHeaderPbs)
	expectedIndex := 0
	expectedBlock, _ := downloadManager.bm.Getblockchain().GetBlockByHeight(blockHeaders[0].height)
	assert.Equal(t, expectedIndex, index)
	assert.Equal(t, expectedBlock, block)

	// first block has incorrect hash, second block is common block
	originalHash := blockHeaderPbs[0].GetHash()
	blockHeaderPbs[0].Hash = []byte{}
	index, block = downloadManager.FindCommonBlock(blockHeaderPbs)
	expectedIndex = 1
	expectedBlock, _ = downloadManager.bm.Getblockchain().GetBlockByHeight(blockHeaders[1].height)
	assert.Equal(t, expectedIndex, index)
	assert.Equal(t, expectedBlock, block)
	blockHeaderPbs[0].Hash = originalHash

	// first block at height 0
	blockHeaderPbs[0].Height = 0
	index, block = downloadManager.FindCommonBlock(blockHeaderPbs)
	expectedIndex = -1
	expectedBlock = nil
	assert.Equal(t, expectedIndex, index)
	assert.Equal(t, expectedBlock, block)

	// all blocks at non-existent height
	for _, bh := range blockHeaderPbs {
		bh.Height = 9999
	}
	index, block = downloadManager.FindCommonBlock(blockHeaderPbs)
	assert.Equal(t, expectedIndex, index)
	assert.Equal(t, expectedBlock, block)
}

func deleteConfFolderFiles() error {
	dir, err := ioutil.ReadDir(confDir)
	if err != nil {
		return err
	}
	for _, d := range dir {
		os.RemoveAll(path.Join([]string{confDir, d.Name()}...))
	}
	return nil
}
