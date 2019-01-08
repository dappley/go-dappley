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

	"github.com/dappley/go-dappley/consensus"
	"github.com/dappley/go-dappley/core"
	"github.com/dappley/go-dappley/storage"
	"github.com/stretchr/testify/assert"
)

const (
	singlePortStart          int = 10300
	multiPortEqualStart      int = 10301
	multiPortSuccessStart    int = 10310
	multiPortDisconnectStart int = 10320
	//multiPortRetryStart      int = 10230
)

func createTestBlockchains(size int, portStart int) ([]*core.Blockchain, []*Node) {
	blockchains := make([]*core.Blockchain, size)
	nodes := make([]*Node, size)
	for i := 0; i < size; i++ {
		keyPair := core.NewKeyPair()
		address := keyPair.GenerateAddress(false)
		pow := consensus.NewProofOfWork()
		pow.SetTargetBit(0)
		bc := core.CreateBlockchain(address, storage.NewRamStorage(), pow, 128, nil)
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

	for i := 1; uint64(i) <= generateChain.GetMaxHeight(); i++ {
		block, _ := generateChain.GetBlockByHeight(uint64(i))
		for j := 1; j < len(blockchains); j++ {
			current := blockchains[j]
			current.AddBlockToTail(block)
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
		currNode := nodes[i]
		node.AddStream(currNode.GetPeerID(), currNode.GetPeerMultiaddr())
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

	for i := 1; i < len(nodes); i++ {
		currNode := nodes[i]
		node.AddStream(currNode.GetPeerID(), currNode.GetPeerMultiaddr())
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

	secondChain := blockchains[2]
	secondChain.GetConsensus().Start()
	for secondChain.GetMaxHeight() < 300 {
	}
	secondChain.GetConsensus().Stop()

	for i := 1; i < len(nodes); i++ {
		currNode := nodes[i]
		node.AddStream(currNode.GetPeerID(), currNode.GetPeerMultiaddr())
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
