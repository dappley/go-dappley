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
	"github.com/dappley/go-dappley/network"
	"testing"
	"time"

	"github.com/dappley/go-dappley/consensus"
	"github.com/dappley/go-dappley/network/pb"
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

func createTestBlockchains(size int, portStart int) ([]*BlockChainManager, []*network.Node) {
	bms := make([]*BlockChainManager, size)
	nodes := make([]*network.Node, size)
	for i := 0; i < size; i++ {
		keyPair := NewKeyPair()
		address := keyPair.GenerateAddress(false)
		pow := consensus.NewProofOfWork()
		pow.SetTargetBit(0)
		bm := NewBlockChainManager(nil, nil)
		bc := CreateBlockchain(NewAddress(genesisAddr), storage.NewRamStorage(), pow, 128, nil, 100000)
		bc.SetState(BlockchainReady)
		bm.SetBlockchain(bc)
		bm.SetBlockPool(NewBlockPool(100))
		node := network.NewNode(bc.GetDb())
		bm.SetDownloadManager(NewDownloadManager(node, bm))
		bms[i] = bm
		nodes[i] = node
		pow.Setup(node, address.Address, bm)
		pow.SetTargetBit(10)
		node.Start(portStart+i, nil)
	}
	return bms, nodes
}

func fillBlockchains(bms []*BlockChainManager) {
	generateChain := bms[0].Getblockchain()

	generateChain.GetConsensus().Start()
	for generateChain.GetMaxHeight() < 100 {
	}
	generateChain.GetConsensus().Stop()
	time.Sleep(2 * time.Second)

	for i := 1; uint64(i) <= generateChain.GetMaxHeight(); i++ {
		block, _ := generateChain.GetBlockByHeight(uint64(i))
		for j := 1; j < len(bms); j++ {
			current := bms[j].Getblockchain()
			current.AddBlockContextToTail(PrepareBlockContext(current, block))
		}
	}
}

func TestSingleNode(t *testing.T) {
	bms, _ := createTestBlockchains(1, singlePortStart)

	bm := bms[0]
	bm.Getblockchain().SetState(BlockchainInit)

	finishChan := make(chan bool, 1)

	bm.Getblockchain().SetState(BlockchainDownloading)
	bm.GetDownloadManager().StartDownloadBlockchain(finishChan)
	<-finishChan
	bm.Getblockchain().SetState(BlockchainReady)

	assert.Equal(t, uint64(0), bm.Getblockchain().GetMaxHeight())
}

func TestMultiEqualNode(t *testing.T) {
	bms, nodes := createTestBlockchains(5, multiPortEqualStart)
	fillBlockchains(bms)

	bm := bms[0]
	bm.Getblockchain().SetState(BlockchainInit)
	node := nodes[0]

	for i := 1; i < len(nodes); i++ {
		node.GetNetwork().AddPeer(nodes[i].GetInfo())
	}

	oldHeight := bm.Getblockchain().GetMaxHeight()

	finishChan := make(chan bool, 1)

	bm.Getblockchain().SetState(BlockchainDownloading)
	bm.GetDownloadManager().StartDownloadBlockchain(finishChan)
	<-finishChan
	bm.Getblockchain().SetState(BlockchainReady)

	assert.Equal(t, oldHeight, bm.Getblockchain().GetMaxHeight())
}

func TestMultiNotEqualNode(t *testing.T) {
	bms, nodes := createTestBlockchains(5, multiPortNotEqualStart)
	fillBlockchains(bms)

	for _, bm := range bms {
		bm.Getblockchain().GetConsensus().Start()
	}
	time.Sleep(3 * time.Second)
	for _, blockchain := range bms {
		blockchain.Getblockchain().GetConsensus().Stop()
	}
	time.Sleep(2 * time.Second)

	bm := bms[0]
	bm.Getblockchain().SetState(BlockchainInit)
	node := nodes[0]

	highestChain := bms[1]
	highestChain.Getblockchain().GetConsensus().Start()
	nextHeight := highestChain.Getblockchain().GetMaxHeight() + 100
	for highestChain.Getblockchain().GetMaxHeight() < nextHeight {
	}
	highestChain.Getblockchain().GetConsensus().Stop()
	time.Sleep(2 * time.Second)

	for i := 1; i < len(nodes); i++ {
		node.GetNetwork().AddPeer(nodes[i].GetInfo())
	}

	bm.DownloadBlocks()

	assert.Equal(t, highestChain.Getblockchain().GetMaxHeight(), bm.Getblockchain().GetMaxHeight())
}

func TestMultiSuccessNode(t *testing.T) {
	bms, nodes := createTestBlockchains(5, multiPortSuccessStart)
	fillBlockchains(bms)

	bm := bms[0]
	bm.Getblockchain().SetState(BlockchainInit)
	node := nodes[0]

	highestChain := bms[1]
	highestChain.Getblockchain().GetConsensus().Start()
	for highestChain.Getblockchain().GetMaxHeight() < 200 {
	}
	highestChain.Getblockchain().GetConsensus().Stop()
	time.Sleep(2 * time.Second)

	for i := 1; i < len(nodes); i++ {
		node.GetNetwork().AddPeer(nodes[i].GetInfo())
	}

	bm.DownloadBlocks()

	assert.Equal(t, highestChain.Getblockchain().GetMaxHeight(), bm.Getblockchain().GetMaxHeight())
}

func TestDisconnectNode(t *testing.T) {
	bms, nodes := createTestBlockchains(5, multiPortDisconnectStart)
	fillBlockchains(bms)

	bm := bms[0]
	bm.Getblockchain().SetState(BlockchainInit)
	node := nodes[0]

	highestChain := bms[1]
	highestChain.Getblockchain().GetConsensus().Start()
	for highestChain.Getblockchain().GetMaxHeight() < 400 {
	}
	highestChain.Getblockchain().GetConsensus().Stop()
	time.Sleep(2 * time.Second)

	secondChain := bms[2]
	secondChain.Getblockchain().GetConsensus().Start()
	for secondChain.Getblockchain().GetMaxHeight() < 300 {
	}
	secondChain.Getblockchain().GetConsensus().Stop()
	time.Sleep(2 * time.Second)

	for i := 1; i < len(nodes); i++ {
		node.GetNetwork().AddPeer(nodes[i].GetInfo())
	}

	bm.DownloadBlocks()
	nodes[1].Stop()

	assert.Equal(t, secondChain.Getblockchain().GetMaxHeight(), bm.Getblockchain().GetMaxHeight())
}

func TestValidateReturnBlocks(t *testing.T) {
	// Test empty blocks in GetBlocksResponse message
	bms, nodes := createTestBlockchains(2, multiPortReturnBlocks)
	fillBlockchains(bms)

	bm := bms[0]
	bm.Getblockchain().SetState(BlockchainInit)
	node := nodes[0]
	peerNode := nodes[1]

	node.GetNetwork().AddPeer(peerNode.GetInfo())
	downloadManager := bm.GetDownloadManager()
	downloadManager.peersInfo = make(map[peer.ID]*PeerBlockInfo)

	for _, p := range downloadManager.node.GetNetwork().GetPeers() {
		downloadManager.peersInfo[p.PeerId] = &PeerBlockInfo{peerid: p.PeerId, height: 0, status: PeerStatusInit}
		downloadManager.downloadingPeer = downloadManager.peersInfo[p.PeerId]
	}
	bm.Getblockchain().SetState(BlockchainDownloading)

	// test invalid peer id
	_, err := downloadManager.validateReturnBlocks(nil, "foo")
	assert.Equal(t, ErrPeerNotFound, err)

	// test empty blocks
	fakeReturnMsg := &networkpb.ReturnBlocks{Blocks: nil, StartBlockHashes: nil}
	_, err = downloadManager.validateReturnBlocks(fakeReturnMsg, peerNode.GetInfo().PeerId)
	assert.Equal(t, ErrEmptyBlocks, err)
}
