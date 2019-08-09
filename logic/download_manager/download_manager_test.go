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

package download_manager

import (
	"github.com/dappley/go-dappley/core/blockchain"
	"github.com/dappley/go-dappley/logic/blockchain_logic"
	"testing"
	"time"

	"github.com/dappley/go-dappley/core"
	"github.com/dappley/go-dappley/core/account"
	"github.com/dappley/go-dappley/network"

	"github.com/dappley/go-dappley/consensus"
	networkpb "github.com/dappley/go-dappley/network/pb"
	"github.com/dappley/go-dappley/storage"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/stretchr/testify/assert"
)

const (
	genesisAddr                  = "121yKAXeG4cw6uaGCBYjWk9yTWmMkhcoDD"
	multiPortEqualStart      int = 10301
	multiPortSuccessStart    int = 10310
	multiPortDisconnectStart int = 10320
	multiPortNotEqualStart   int = 10330
	multiPortReturnBlocks    int = 10340
)

func createTestBlockchains(size int, portStart int) ([]*blockchain_logic.BlockchainManager, []*network.Node) {
	bms := make([]*blockchain_logic.BlockchainManager, size)
	nodes := make([]*network.Node, size)
	for i := 0; i < size; i++ {
		keyPair := account.NewKeyPair()
		address := keyPair.GenerateAddress()
		pow := consensus.NewProofOfWork()
		pow.SetTargetBit(0)
		db := storage.NewRamStorage()
		node := network.NewNode(db, nil)

		bc := blockchain_logic.CreateBlockchain(account.NewAddress(genesisAddr), db, pow, core.NewTransactionPool(node, 128), nil, 100000)
		bc.SetState(blockchain.BlockchainReady)

		bm := blockchain_logic.NewBlockchainManager(bc, core.NewBlockPool(), node)

		bms[i] = bm
		nodes[i] = node
		pow.Setup(address.String(), bm)
		pow.SetTargetBit(10)
		node.Start(portStart+i, "")
	}
	return bms, nodes
}

func fillBlockchains(bms []*blockchain_logic.BlockchainManager) {
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
			current.AddBlockContextToTail(blockchain_logic.PrepareBlockContext(current, block))
		}
	}
}

func TestMultiEqualNode(t *testing.T) {
	bms, nodes := createTestBlockchains(5, multiPortEqualStart)
	fillBlockchains(bms)

	//setup download manager for the first node
	bm := bms[0]
	bm.Getblockchain().SetState(blockchain.BlockchainInit)
	node := nodes[0]
	downloadManager := NewDownloadManager(node, bm)
	downloadManager.Start()
	bm.SetDownloadRequestCh(downloadManager.GetDownloadRequestCh())

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
	bm.Getblockchain().SetState(blockchain.BlockchainInit)
	node := nodes[0]
	downloadManager := NewDownloadManager(node, bm)
	downloadManager.Start()

	highestChain := bms[1]
	highestChain.Getblockchain().GetConsensus().Start()
	nextHeight := highestChain.Getblockchain().GetMaxHeight() + 100

	for highestChain.Getblockchain().GetMaxHeight() < nextHeight {
	}
	highestChain.Getblockchain().GetConsensus().Stop()

	time.Sleep(2 * time.Second)

	for i := 1; i < len(nodes); i++ {
		node.GetNetwork().ConnectToSeed(nodes[i].GetHostPeerInfo())
	}

	highestChain.Getblockchain().SetState(blockchain.BlockchainInit)
	highestChainNode := nodes[1]
	highestChainDownloadManager := NewDownloadManager(highestChainNode, highestChain)
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
	fillBlockchains(bms)

	bm := bms[0]
	bm.Getblockchain().SetState(blockchain.BlockchainInit)
	node := nodes[0]
	downloadManager := NewDownloadManager(node, bm)
	downloadManager.Start()

	highestChain := bms[1]
	highestChain.Getblockchain().GetConsensus().Start()
	for highestChain.Getblockchain().GetMaxHeight() < 200 {
	}
	highestChain.Getblockchain().GetConsensus().Stop()
	time.Sleep(2 * time.Second)

	for i := 1; i < len(nodes); i++ {
		node.GetNetwork().ConnectToSeed(nodes[i].GetHostPeerInfo())
	}

	highestChain.Getblockchain().SetState(blockchain.BlockchainInit)
	highestChainNode := nodes[1]
	highestChainDownloadManager := NewDownloadManager(highestChainNode, highestChain)
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
	fillBlockchains(bms)

	bm := bms[0]
	bm.Getblockchain().SetState(blockchain.BlockchainInit)
	node := nodes[0]
	downloadManager := NewDownloadManager(node, bm)
	downloadManager.Start()

	highestChain := bms[1]
	highestChain.Getblockchain().GetConsensus().Start()
	for highestChain.Getblockchain().GetMaxHeight() < 400 {
	}
	highestChain.Getblockchain().GetConsensus().Stop()
	time.Sleep(2 * time.Second)
	highestChainNode := nodes[1]
	highestChainDownloadManager := NewDownloadManager(highestChainNode, highestChain)
	highestChainDownloadManager.Start()

	secondChain := bms[2]
	secondChain.Getblockchain().GetConsensus().Start()
	for secondChain.Getblockchain().GetMaxHeight() < 300 {
	}
	secondChain.Getblockchain().GetConsensus().Stop()
	time.Sleep(2 * time.Second)
	secondChainNode := nodes[2]
	secondChainDownloadManager := NewDownloadManager(secondChainNode, secondChain)
	secondChainDownloadManager.Start()

	for i := 1; i < len(nodes); i++ {
		node.GetNetwork().ConnectToSeed(nodes[i].GetHostPeerInfo())
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
	fillBlockchains(bms)

	bm := bms[0]
	bm.Getblockchain().SetState(blockchain.BlockchainInit)
	node := nodes[0]
	downloadManager := NewDownloadManager(node, bm)
	bm.SetDownloadRequestCh(downloadManager.GetDownloadRequestCh())

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
