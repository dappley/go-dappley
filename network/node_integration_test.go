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
	"encoding/hex"
	"testing"

	"github.com/dappley/go-dappley/client"
	"github.com/dappley/go-dappley/core"
	"github.com/dappley/go-dappley/storage"
	logger "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

const (
	test_port1 = 20600 + iota
	test_port2
	test_port3
	test_port4
	test_port5
	test_port6
	test_port7
	test_port8
	test_port9
	test_port10
	test_port11
	test_port12
	test_port13
	test_port14
)

func initNode(address string, port int, seedPeer *PeerInfo, db storage.Storage) (*Node, error) {
	addr := client.Address{address}
	bc := core.CreateBlockchain(addr, db, nil, 128, nil, 100000)
	pool := core.NewBlockPool(0)
	n := NewNode(bc, pool)

	if seedPeer != nil {
		n.GetPeerManager().AddSeedByPeerInfo(seedPeer)
	}
	err := n.Start(port)
	return n, err
}

func initNodeWithConfig(address string, port, connectionInCount, connectionOutCount int, seedPeer *PeerInfo, db storage.Storage) (*Node, error) {
	addr := client.Address{address}
	bc := core.CreateBlockchain(addr, db, nil, 128, nil, 100000)
	pool := core.NewBlockPool(0)
	config := &NodeConfig{MaxConnectionInCount: connectionInCount, MaxConnectionOutCount: connectionOutCount}
	n := NewNodeWithConfig(bc, pool, config)

	if seedPeer != nil {
		n.GetPeerManager().AddSeedByPeerInfo(seedPeer)
	}
	err := n.Start(port)
	return n, err
}

func TestNetwork_AddStream(t *testing.T) {

	db := storage.NewRamStorage()
	defer db.Close()

	//create node1
	n1, err := initNode("17DgRtQVvaytkiKAfXx9XbV23MESASSwUz", test_port1, nil, db)
	defer n1.Stop()
	assert.Nil(t, err)

	//currently it should only have itself as its node
	assert.Len(t, n1.host.Network().Peerstore().Peers(), 1)

	//create node2
	n2, err := initNode("17DgRtQVvaytkiKAfXx9XbV23MESASSwUz", test_port2, nil, db)
	defer n2.Stop()
	assert.Nil(t, err)

	//set node2 as the peer of node1
	err = n1.GetPeerManager().AddAndConnectPeer(n2.GetInfo())
	assert.Nil(t, err)
	assert.Len(t, n1.host.Network().Peerstore().Peers(), 2)
}

func TestNetwork_BroadcastBlock(t *testing.T) {
	//setup node 0
	db := storage.NewRamStorage()
	defer db.Close()

	n1, err := initNode("QmWyMUMBeWxwU4R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ", test_port3, nil, db)
	defer n1.Stop()
	assert.Nil(t, err)

	//setup node 1
	n2, err := initNode("QmWyMUMBeWxwU4R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ", test_port4, nil, db)
	defer n2.Stop()
	assert.Nil(t, err)

	err = n2.GetPeerManager().AddAndConnectPeer(n1.GetInfo())
	assert.Nil(t, err)

	blk := core.GenerateMockBlock()
	n1.BroadcastBlock(blk)

	//wait for node 1 to receive response
	core.WaitDoneOrTimeout(func() bool {
		blk, _ := n2.recentlyRcvedDapMsgs.Get(hex.EncodeToString(blk.GetHash()))
		return blk != nil
	}, 5)

	assert.True(t, true)
}

func TestNode_RequestBlockUnicast(t *testing.T) {

	//setup node 1
	db := storage.NewRamStorage()
	defer db.Close()
	n1, err := initNode("17DgRtQVvaytkiKAfXx9XbV23MESASSwUz", test_port5, nil, db)
	defer n1.Stop()
	assert.Nil(t, err)
	//setup node 2
	n2, err := initNode("17DgRtQVvaytkiKAfXx9XbV23MESASSwUz", test_port6, nil, db)
	defer n2.Stop()
	assert.Nil(t, err)

	err = n2.GetPeerManager().AddAndConnectPeer(n1.GetInfo())
	assert.Nil(t, err)

	blk := core.GenerateMockBlock()

	err = n1.GetBlockchain().GetDb().Put(blk.GetHash(), blk.Serialize())
	assert.Nil(t, err)

	n2.RequestBlockUnicast(blk.GetHash(), n1.GetPeerID())
	//wait for node 1 to receive response
	core.WaitDoneOrTimeout(func() bool {
		blk, _ := n2.recentlyRcvedDapMsgs.Get(hex.EncodeToString(blk.GetHash()))
		return blk != nil
	}, 5)

	assert.True(t, true)
}

func TestNode_SyncPeers(t *testing.T) {
	db1 := storage.NewRamStorage()
	defer db1.Close()

	n1, err := initNode("17DgRtQVvaytkiKAfXx9XbV23MESASSwUz", test_port7, nil, db1)
	defer n1.Stop()
	assert.Nil(t, err)

	//create node 2 and add node1 as a peer
	db2 := storage.NewRamStorage()
	defer db2.Close()
	n2, err := initNode("17DgRtQVvaytkiKAfXx9XbV23MESASSwUz", test_port8, n1.GetInfo(), db2)
	defer n2.Stop()
	assert.Nil(t, err)

	core.WaitDoneOrTimeout(func() bool {
		//no condition to be checked
		return false
	}, 1)

	//create node 3 and add node1 as a peer
	db3 := storage.NewRamStorage()
	defer db3.Close()
	n3, err := initNode("17DgRtQVvaytkiKAfXx9XbV23MESASSwUz", test_port9, n1.GetInfo(), db3)
	defer n3.Stop()
	assert.Nil(t, err)

	core.WaitDoneOrTimeout(func() bool {
		//no condition to be checked
		return false
	}, 2)

	//node2 should have node 3 as its peer
	_, ok := n2.GetPeerManager().streams[n3.GetPeerID()]
	assert.True(t, ok)
	assert.Equal(t, 1, n2.GetPeerManager().connectionInCount)

	//node3 should have node 2 as its peer
	_, ok = n3.GetPeerManager().streams[n2.GetPeerID()]
	assert.True(t, ok)
	assert.Equal(t, 1, n3.GetPeerManager().connectionOutCount)
}

func TestNode_ConnectionFull(t *testing.T) {
	logger.SetLevel(logger.InfoLevel)
	db1 := storage.NewRamStorage()
	defer db1.Close()

	n1, err := initNode("17DgRtQVvaytkiKAfXx9XbV23MESASSwUz", test_port10, nil, db1)
	defer n1.Stop()
	assert.Nil(t, err)

	//create node 2 and add node1 as a peer
	db2 := storage.NewRamStorage()
	defer db2.Close()
	n2, err := initNodeWithConfig("17DgRtQVvaytkiKAfXx9XbV23MESASSwUz", test_port11, 2, 2, n1.GetInfo(), db2)
	defer n2.Stop()
	assert.Nil(t, err)

	core.WaitDoneOrTimeout(func() bool {
		return false
	}, 1)

	//create node 3 and add node1 as a peer
	db3 := storage.NewRamStorage()
	defer db3.Close()
	n3, err := initNodeWithConfig("17DgRtQVvaytkiKAfXx9XbV23MESASSwUz", test_port12, 2, 2, n1.GetInfo(), db3)
	defer n3.Stop()
	assert.Nil(t, err)

	core.WaitDoneOrTimeout(func() bool {
		return false
	}, 1)

	//create node 3 and add node1 as a peer
	db4 := storage.NewRamStorage()
	defer db4.Close()
	n4, err := initNodeWithConfig("17DgRtQVvaytkiKAfXx9XbV23MESASSwUz", test_port13, 2, 2, n1.GetInfo(), db4)
	defer n4.Stop()
	assert.Nil(t, err)

	core.WaitDoneOrTimeout(func() bool {
		return false
	}, 1)

	//create node 3 and add node1 as a peer
	db5 := storage.NewRamStorage()
	defer db5.Close()
	n5, err := initNode("17DgRtQVvaytkiKAfXx9XbV23MESASSwUz", test_port14, nil, db5)
	defer n5.Stop()
	assert.Nil(t, err)

	n5.GetPeerManager().AddAndConnectPeer(n2.GetInfo())
	n4.GetPeerManager().AddAndConnectPeer(n5.GetInfo())

	core.WaitDoneOrTimeout(func() bool {
		return false
	}, 2)

	assert.Equal(t, 2, n2.GetPeerManager().connectionInCount)
	assert.Equal(t, 0, n2.GetPeerManager().connectionOutCount)
	assert.Equal(t, 1, n3.GetPeerManager().connectionInCount)
	assert.Equal(t, 1, n3.GetPeerManager().connectionOutCount)
	assert.Equal(t, 0, n4.GetPeerManager().connectionInCount)
	assert.Equal(t, 2, n4.GetPeerManager().connectionOutCount)
	assert.Equal(t, 0, n5.GetPeerManager().connectionInCount)
	assert.Equal(t, 0, n5.GetPeerManager().connectionOutCount)
}
