// +build integration

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

	"github.com/dappley/go-dappley/mocks"
	"github.com/dappley/go-dappley/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"testing"
	"time"
)

const (
	test_port1 = 10000 + iota
	test_port2
	test_port3
	test_port4
	test_port5
	test_port6
	test_port7
	test_port8
	test_port9
)

func TestNetwork_Setup(t *testing.T) {

	db := storage.NewRamStorage()
	defer db.Close()
	addr := core.Address{"17DgRtQVvaytkiKAfXx9XbV23MESASSwUz"}
	bc := core.CreateBlockchain(addr, db, nil, 128)

	//create node1
	node1 := NewNode(bc)
	err := node1.Start(test_port1)
	assert.Nil(t, err)

	//currently it should only have itself as its node
	assert.Len(t, node1.host.Network().Peerstore().Peers(), 1)

	//create node2
	node2 := NewNode(bc)
	err = node2.Start(test_port2)
	assert.Nil(t, err)

	//set node2 as the peer of node1
	err = node1.AddStream(node2.GetPeerID(), node2.GetPeerMultiaddr())
	assert.Nil(t, err)
	assert.Len(t, node1.host.Network().Peerstore().Peers(), 2)
}


func TestNetwork_SendBlock(t *testing.T){
	bp0 := new(mocks.BlockPoolInterface)
	bp1 := new(mocks.BlockPoolInterface)

	//setup node 0
	db0 := storage.NewRamStorage()
	defer db0.Close()


	bp0.On("SetBlockchain", mock.Anything).Return(nil)
	bp0.On("BlockRequestCh").Return(make(chan core.BlockRequestPars))
	bc0 := core.CreateBlockchain(core.Address{"17DgRtQVvaytkiKAfXx9XbV23MESASSwUz"}, db0, nil,128)
	bp1.On("SetBlockchain", mock.Anything)
	bp1.On("BlockRequestCh").Return(make(chan core.BlockRequestPars))
	bp1.On("Push", mock.Anything,mock.Anything).Return(nil)
	bc0.SetBlockPool(bp0)
	assert.Equal(t, bc0.GetBlockPool(),  bp0)
	bp0.AssertCalled(t, "SetBlockchain", mock.Anything)
	n0 := FakeNodeWithPidAndAddr(bc0,"QmWyMUMBeWxwU4R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ","/ip4/192.168.10.110/tcp/10000")

	//setup node 1
	db1 := storage.NewRamStorage()
	defer db1.Close()


	bc1 := core.CreateBlockchain(core.Address{"17DgRtQVvaytkiKAfXx9XbV23MESASSwUz"}, db1, nil,128)
	bc1.SetBlockPool(bp1)

	bp0.AssertCalled(t, "SetBlockchain", mock.Anything)

	n1 := FakeNodeWithPidAndAddr(bc1,"QmWyMUMBeWxwU4R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ","/ip4/192.168.10.110/tcp/10001")

	n0.Start(test_port3)
	time.Sleep(1*time.Second)
	bp0.AssertCalled(t, "BlockRequestCh")
	n1.Start(test_port4)
	time.Sleep(1*time.Second)
	bp1.AssertCalled(t, "BlockRequestCh")
	//add node0 as a stream peer in node1 and node2
	err := n1.AddStream(n0.GetPeerID(), n0.GetPeerMultiaddr())
	assert.Nil(t, err)

	//node 0 broadcast a block
	blk := core.GenerateMockBlock()
	n0.BroadcastBlock(blk)
	time.Sleep(2*time.Second)
	bp1.AssertCalled(t, "Push", mock.Anything, mock.Anything)
}

func TestNode_SyncPeers(t *testing.T) {
	db := storage.NewRamStorage()
	defer db.Close()
	addr := core.Address{"17DgRtQVvaytkiKAfXx9XbV23MESASSwUz"}
	bc := core.CreateBlockchain(addr, db, nil, 128)

	//create node1
	node1 := NewNode(bc)
	err := node1.Start(test_port6)
	assert.Nil(t, err)

	//create node 2 and add node1 as a peer
	node2 := NewNode(bc)
	err = node2.Start(test_port7)
	assert.Nil(t, err)
	err = node2.AddStream(node1.GetPeerID(), node1.GetPeerMultiaddr())
	assert.Nil(t, err)

	//create node 3 and add node1 as a peer
	node3 := NewNode(bc)
	err = node3.Start(test_port8)
	assert.Nil(t, err)
	err = node3.AddStream(node1.GetPeerID(), node1.GetPeerMultiaddr())
	assert.Nil(t, err)

	time.Sleep(time.Second)

	//node 1 broadcast syncpeers
	node1.SyncPeersBroadcast()

	time.Sleep(time.Second * 2)

	//node2 should have node 3 as its peer
	assert.True(t, node2.peerList.IsInPeerlist(node3.GetInfo()))

	//node3 should have node 2 as its peer
	assert.True(t, node3.peerList.IsInPeerlist(node2.GetInfo()))

	time.Sleep(time.Second)

}

func TestNode_RequestBlockUnicast(t *testing.T) {

	bp0 := new(mocks.BlockPoolInterface)
	bp1 := new(mocks.BlockPoolInterface)

	bp0.On("SetBlockchain", mock.Anything).Return(nil)
	bp0.On("BlockRequestCh").Return(make(chan core.BlockRequestPars))
	bp1.On("SetBlockchain", mock.Anything).Return(nil)
	bp1.On("BlockRequestCh").Return(make(chan core.BlockRequestPars))
	bp1.On("Push", mock.Anything,mock.Anything).Return(nil)

	//setup node 0
	db0 := storage.NewRamStorage()
	defer db0.Close()

	bc0 := core.CreateBlockchain(core.Address{"17DgRtQVvaytkiKAfXx9XbV23MESASSwUz"}, db0, nil,128)
	bc0.SetBlockPool(bp0)

	n0 := FakeNodeWithPidAndAddr(bc0, "QmWyMUMBeWxwU4R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ", "/ip4/192.168.10.110/tcp/10000")

	//setup node 1
	db1 := storage.NewRamStorage()
	defer db1.Close()

	bc1 := core.CreateBlockchain(core.Address{"17DgRtQVvaytkiKAfXx9XbV23MESASSwUz"}, db1, nil,128)

	bc1.SetBlockPool(bp1)
	n1 := FakeNodeWithPidAndAddr(bc1,"QmWyMUMBeWxwU4R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ","/ip4/192.168.10.110/tcp/10000")

	n0.Start(test_port9)
	time.Sleep(1*time.Second)
	bp0.AssertCalled(t, "BlockRequestCh")

	n1.Start(test_port5)
	time.Sleep(1*time.Second)
	bp1.AssertCalled(t, "BlockRequestCh")

	//add node0 as a stream peer in node1 and node2
	err := n1.AddStream(n0.GetPeerID(), n0.GetPeerMultiaddr())
	assert.Nil(t, err)

	//generate a block and store it in node0 blockchain
	blk := core.GenerateMockBlock()
	err = n0.bc.GetDb().Put(blk.GetHash(), blk.Serialize())
	assert.Nil(t, err)

	//node1 request the block
	n1.RequestBlockUnicast(blk.GetHash(),n0.GetPeerID())

	//wait for node 1 to receive response
	time.Sleep(2*time.Second)
	bp1.AssertCalled(t, "Push", mock.Anything, mock.Anything)

}
