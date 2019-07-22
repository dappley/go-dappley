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
	"testing"
	"time"

	"github.com/dappley/go-dappley/client"
	"github.com/dappley/go-dappley/consensus"
	"github.com/dappley/go-dappley/core"
	networkpb "github.com/dappley/go-dappley/network/pb"
	"github.com/dappley/go-dappley/storage"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/stretchr/testify/assert"
)

const (
	genesisAddr                  = "121yKAXeG4cw6uaGCBYjWk9yTWmMkhcoDD"
	singlePortStart          int = 10300
	multiPortEqualStart      int = 10301
	multiPortSuccessStart    int = 10310
	multiPortDisconnectStart int = 10320
	multiPortNotEqualStart   int = 10330
	//multiPortRetryStart      int = 10230
	multiPortReturnBlocks int = 10340
)

func createTestBlockchains(size int, portStart int) ([]*core.Blockchain, []*Node) {
	blockchains := make([]*core.Blockchain, size)
	nodes := make([]*Node, size)
	for i := 0; i < size; i++ {
		keyPair := client.NewKeyPair()
		address := keyPair.GenerateAddress()
		pow := consensus.NewProofOfWork()
		pow.SetTargetBit(0)
		bc := core.CreateBlockchain(client.NewAddress(genesisAddr), storage.NewRamStorage(), pow, 128, nil, 100000)
		bc.SetState(core.BlockchainReady)
		blockchains[i] = bc
		pool := core.NewBlockPool(100)
		node := NewNode(bc, pool)
		nodes[i] = node
		pow.Setup(node, address.Address)
		pow.SetTargetBit(10)
		node.Start(portStart + i)
	}
	return blockchains, nodes
}

func fillBlockchains(blockchains []*core.Blockchain) {
	generateChain := blockchains[0]

	generateChain.GetConsensus().Start()
	for generateChain.GetMaxHeight() < 100 {
	}
	generateChain.GetConsensus().Stop()
	time.Sleep(2 * time.Second)

	for i := 1; uint64(i) <= generateChain.GetMaxHeight(); i++ {
		block, _ := generateChain.GetBlockByHeight(uint64(i))
		for j := 1; j < len(blockchains); j++ {
			current := blockchains[j]
			current.AddBlockContextToTail(core.PrepareBlockContext(current, block))
		}
	}
}

func TestSingleNode(t *testing.T) {
	blockchains, nodes := createTestBlockchains(1, singlePortStart)

	blockchain := blockchains[0]
	blockchain.SetState(core.BlockchainInit)

	node := nodes[0]

	downloadManager := node.GetDownloadManager()
	finishChan := make(chan bool, 1)

	blockchain.SetState(core.BlockchainDownloading)
	downloadManager.StartDownloadBlockchain(finishChan)
	<-finishChan
	blockchain.SetState(core.BlockchainReady)

	assert.Equal(t, uint64(0), blockchain.GetMaxHeight())
}

func TestMultiEqualNode(t *testing.T) {
	blockchains, nodes := createTestBlockchains(5, multiPortEqualStart)
	fillBlockchains(blockchains)

	blockchain := blockchains[0]
	blockchain.SetState(core.BlockchainInit)
	node := nodes[0]

	for i := 1; i < len(nodes); i++ {
		node.GetPeerManager().AddAndConnectPeer(nodes[i].GetInfo())
	}

	oldHeight := blockchain.GetMaxHeight()

	downloadManager := node.GetDownloadManager()
	finishChan := make(chan bool, 1)

	blockchain.SetState(core.BlockchainDownloading)
	downloadManager.StartDownloadBlockchain(finishChan)
	<-finishChan
	blockchain.SetState(core.BlockchainReady)

	assert.Equal(t, oldHeight, blockchain.GetMaxHeight())
}

func TestMultiNotEqualNode(t *testing.T) {
	blockchains, nodes := createTestBlockchains(5, multiPortNotEqualStart)
	fillBlockchains(blockchains)

	for _, blockchain := range blockchains {
		blockchain.GetConsensus().Start()
	}
	time.Sleep(3 * time.Second)
	for _, blockchain := range blockchains {
		blockchain.GetConsensus().Stop()
	}
	time.Sleep(2 * time.Second)

	blockchain := blockchains[0]
	blockchain.SetState(core.BlockchainInit)
	node := nodes[0]

	highestChain := blockchains[1]
	highestChain.GetConsensus().Start()
	nextHeight := highestChain.GetMaxHeight() + 100
	for highestChain.GetMaxHeight() < nextHeight {
	}
	highestChain.GetConsensus().Stop()
	time.Sleep(2 * time.Second)

	for i := 1; i < len(nodes); i++ {
		node.GetPeerManager().AddAndConnectPeer(nodes[i].GetInfo())
	}

	downloadManager := node.GetDownloadManager()
	finishChan := make(chan bool, 1)

	blockchain.SetState(core.BlockchainDownloading)
	downloadManager.StartDownloadBlockchain(finishChan)
	<-finishChan
	blockchain.SetState(core.BlockchainReady)

	assert.Equal(t, highestChain.GetMaxHeight(), blockchain.GetMaxHeight())
}

func TestMultiSuccessNode(t *testing.T) {
	blockchains, nodes := createTestBlockchains(5, multiPortSuccessStart)
	fillBlockchains(blockchains)

	blockchain := blockchains[0]
	blockchain.SetState(core.BlockchainInit)
	node := nodes[0]

	highestChain := blockchains[1]
	highestChain.GetConsensus().Start()
	for highestChain.GetMaxHeight() < 200 {
	}
	highestChain.GetConsensus().Stop()
	time.Sleep(2 * time.Second)

	for i := 1; i < len(nodes); i++ {
		node.GetPeerManager().AddAndConnectPeer(nodes[i].GetInfo())
	}

	downloadManager := node.GetDownloadManager()
	finishChan := make(chan bool, 1)

	blockchain.SetState(core.BlockchainDownloading)
	downloadManager.StartDownloadBlockchain(finishChan)
	<-finishChan
	blockchain.SetState(core.BlockchainReady)

	assert.Equal(t, highestChain.GetMaxHeight(), blockchain.GetMaxHeight())
}

func TestDisconnectNode(t *testing.T) {
	blockchains, nodes := createTestBlockchains(5, multiPortDisconnectStart)
	fillBlockchains(blockchains)

	blockchain := blockchains[0]
	blockchain.SetState(core.BlockchainInit)
	node := nodes[0]

	highestChain := blockchains[1]
	highestChain.GetConsensus().Start()
	for highestChain.GetMaxHeight() < 400 {
	}
	highestChain.GetConsensus().Stop()
	time.Sleep(2 * time.Second)

	secondChain := blockchains[2]
	secondChain.GetConsensus().Start()
	for secondChain.GetMaxHeight() < 300 {
	}
	secondChain.GetConsensus().Stop()
	time.Sleep(2 * time.Second)

	for i := 1; i < len(nodes); i++ {
		node.GetPeerManager().AddAndConnectPeer(nodes[i].GetInfo())
	}

	downloadManager := node.GetDownloadManager()
	finishChan := make(chan bool, 1)

	blockchain.SetState(core.BlockchainDownloading)
	downloadManager.StartDownloadBlockchain(finishChan)
	nodes[1].Stop()

	<-finishChan
	blockchain.SetState(core.BlockchainReady)

	assert.Equal(t, secondChain.GetMaxHeight(), blockchain.GetMaxHeight())
}

func TestValidateReturnBlocks(t *testing.T) {
	// Test empty blocks in ReturnBlocks message
	blockchains, nodes := createTestBlockchains(2, multiPortReturnBlocks)
	fillBlockchains(blockchains)

	blockchain := blockchains[0]
	blockchain.SetState(core.BlockchainInit)
	node := nodes[0]
	peerNode := nodes[1]

	node.GetPeerManager().AddAndConnectPeer(peerNode.GetInfo())
	downloadManager := node.GetDownloadManager()
	downloadManager.peersInfo = make(map[peer.ID]*PeerBlockInfo)

	for _, p := range downloadManager.node.GetPeerManager().CloneStreamsToSlice() {
		downloadManager.peersInfo[p.stream.peerID] = &PeerBlockInfo{peerid: p.stream.peerID, height: 0, status: PeerStatusInit}
		downloadManager.downloadingPeer = downloadManager.peersInfo[p.stream.peerID]
	}
	blockchain.SetState(core.BlockchainDownloading)

	// test invalid peer id
	_, err := downloadManager.validateReturnBlocks(nil, "foo")
	assert.Equal(t, ErrPeerNotFound, err)

	// test empty blocks
	fakeReturnMsg := &networkpb.ReturnBlocks{Blocks: nil, StartBlockHashes: nil}
	_, err = downloadManager.validateReturnBlocks(fakeReturnMsg, peerNode.GetInfo().PeerId)
	assert.Equal(t, ErrEmptyBlocks, err)
}
