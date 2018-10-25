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
	"testing"
	"github.com/stretchr/testify/assert"
	"github.com/dappley/go-dappley/core"
	"time"
	"github.com/dappley/go-dappley/storage"
	"github.com/golang/mock/gomock"
	core_mock "github.com/dappley/go-dappley/core/mock"
)

const(
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
	bc := core.CreateBlockchain(addr,db,nil)

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
	err = node1.AddStream(node2.GetPeerID(),node2.GetPeerMultiaddr())
	assert.Nil(t, err)
	assert.Len(t, node1.host.Network().Peerstore().Peers(), 2)
}

func TestNetwork_SendBlock(t *testing.T){

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	//setup node 0
	db0 := storage.NewRamStorage()
	defer db0.Close()
	bc0 := core.CreateBlockchain(core.Address{"17DgRtQVvaytkiKAfXx9XbV23MESASSwUz"}, db0, nil)
	mockBp0 := core_mock.NewMockBlockPoolInterface(mockCtrl)
	mockBp0.EXPECT().SetBlockchain(bc0)
	bc0.SetBlockPool(mockBp0)

	n0 := FakeNodeWithPidAndAddr(bc0,"QmWyMUMBeWxwU4R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ","/ip4/192.168.10.110/tcp/10000")

	//setup node 1
	db1 := storage.NewRamStorage()
	defer db1.Close()
	bc1 := core.CreateBlockchain(core.Address{"17DgRtQVvaytkiKAfXx9XbV23MESASSwUz"}, db1, nil)
	mockBp1 := core_mock.NewMockBlockPoolInterface(mockCtrl)
	mockBp1.EXPECT().SetBlockchain(bc1)
	bc1.SetBlockPool(mockBp1)
	n1 := FakeNodeWithPidAndAddr(bc1,"QmWyMUMBeWxwU4R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ","/ip4/192.168.10.110/tcp/10001")

	n0.Start(test_port3)
	mockBp0.EXPECT().BlockRequestCh()
	n1.Start(test_port4)
	mockBp1.EXPECT().BlockRequestCh()
	//add node0 as a stream peer in node1 and node2
	err := n1.AddStream(n0.GetPeerID(),n0.GetPeerMultiaddr())
	assert.Nil(t, err)

	//node 0 broadcast a block
	blk := core.GenerateMockBlock()
	n0.BroadcastBlock(blk)

	mockBp1.EXPECT().Push(blk, n0.GetPeerID())
	time.Sleep(time.Second)
}

func TestNode_SyncPeers(t *testing.T){
	db := storage.NewRamStorage()
	defer db.Close()
	addr := core.Address{"17DgRtQVvaytkiKAfXx9XbV23MESASSwUz"}
	bc := core.CreateBlockchain(addr,db,nil)


	//create node1
	node1 := NewNode(bc)
	err := node1.Start(test_port6)
	assert.Nil(t, err)

	//create node 2 and add node1 as a peer
	node2 := NewNode(bc)
	err = node2.Start(test_port7)
	assert.Nil(t, err)
	err = node2.AddStream(node1.GetPeerID(),node1.GetPeerMultiaddr())
	assert.Nil(t, err)

	//create node 3 and add node1 as a peer
	node3 := NewNode(bc)
	err = node3.Start(test_port8)
	assert.Nil(t, err)
	err = node3.AddStream(node1.GetPeerID(),node1.GetPeerMultiaddr())
	assert.Nil(t, err)

	time.Sleep(time.Second)

	//node 1 broadcast syncpeers
	node1.SyncPeersBroadcast()

	time.Sleep(time.Second*2)

	//node2 should have node 3 as its peer
	assert.True(t,node2.peerList.IsInPeerlist(node3.GetInfo()))

	//node3 should have node 2 as its peer
	assert.True(t,node3.peerList.IsInPeerlist(node2.GetInfo()))

	time.Sleep(time.Second)

}

func TestNode_RequestBlockUnicast(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	//setup node 0
	db0 := storage.NewRamStorage()
	defer db0.Close()
	bc0 := core.CreateBlockchain(core.Address{"17DgRtQVvaytkiKAfXx9XbV23MESASSwUz"}, db0, nil)
	mockBp0 := core_mock.NewMockBlockPoolInterface(mockCtrl)
	mockBp0.EXPECT().SetBlockchain(bc0)
	bc0.SetBlockPool(mockBp0)

	n0 := FakeNodeWithPidAndAddr(bc0,"QmWyMUMBeWxwU4R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ","/ip4/192.168.10.110/tcp/10000")

	//setup node 1
	db1 := storage.NewRamStorage()
	defer db1.Close()
	bc1 := core.CreateBlockchain(core.Address{"17DgRtQVvaytkiKAfXx9XbV23MESASSwUz"}, db1, nil)
	mockBp1 := core_mock.NewMockBlockPoolInterface(mockCtrl)
	mockBp1.EXPECT().SetBlockchain(bc1)
	bc1.SetBlockPool(mockBp1)
	n1 := FakeNodeWithPidAndAddr(bc1,"QmWyMUMBeWxwU4R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ","/ip4/192.168.10.110/tcp/10000")

	n0.Start(test_port9)
	mockBp0.EXPECT().BlockRequestCh()
	n1.Start(test_port5)
	mockBp1.EXPECT().BlockRequestCh()

	//add node0 as a stream peer in node1 and node2
	err := n1.AddStream(n0.GetPeerID(),n0.GetPeerMultiaddr())
	assert.Nil(t, err)

	//generate a block and store it in node0 blockchain
	blk := core.GenerateMockBlock()
	err = n0.bc.GetDb().Put(blk.GetHash(),blk.Serialize())
	assert.Nil(t, err)

	//node1 request the block
	n1.RequestBlockUnicast(blk.GetHash(),n0.GetPeerID())
	mockBp1.EXPECT().Push(blk, n0.GetPeerID())

	time.Sleep(time.Second)
}
