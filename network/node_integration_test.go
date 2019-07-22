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
	"github.com/dappley/go-dappley/network/network_model"
	"github.com/dappley/go-dappley/util"
	"testing"

	"github.com/dappley/go-dappley/storage"
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

func initNode(address string, port int, seedPeer *network_model.PeerInfo, db storage.Storage) (*Node, error) {

	n := NewNode(db)

	if seedPeer != nil {
		n.GetNetwork().AddSeed(seedPeer)
	}

	err := n.Start(port, nil, "")
	return n, err
}

func initNodeWithConfig(address string, port, connectionInCount, connectionOutCount int, seedPeer *network_model.PeerInfo, db storage.Storage) (*Node, error) {

	config := &NodeConfig{MaxConnectionInCount: connectionInCount, MaxConnectionOutCount: connectionOutCount}
	n := NewNodeWithConfig(db, config)

	if seedPeer != nil {
		n.GetNetwork().AddSeed(seedPeer)
	}

	err := n.Start(port, nil, "")
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
	assert.Len(t, n1.network.host.Network().Peerstore().Peers(), 1)

	//create node2
	n2, err := initNode("17DgRtQVvaytkiKAfXx9XbV23MESASSwUz", test_port2, nil, db)
	defer n2.Stop()
	assert.Nil(t, err)

	//set node2 as the peer of node1
	err = n1.GetNetwork().AddPeer(n2.GetHostPeerInfo())
	assert.Nil(t, err)
	assert.Len(t, n1.network.host.Network().Peerstore().Peers(), 2)
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
	n2, err := initNode("17DgRtQVvaytkiKAfXx9XbV23MESASSwUz", test_port8, n1.GetHostPeerInfo(), db2)
	defer n2.Stop()
	assert.Nil(t, err)

	util.WaitDoneOrTimeout(func() bool {
		//no condition to be checked
		return false
	}, 1)

	//create node 3 and add node1 as a peer
	db3 := storage.NewRamStorage()
	defer db3.Close()
	n3, err := initNode("17DgRtQVvaytkiKAfXx9XbV23MESASSwUz", test_port9, n1.GetHostPeerInfo(), db3)
	defer n3.Stop()
	assert.Nil(t, err)

	util.WaitDoneOrTimeout(func() bool {
		//no condition to be checked
		return false
	}, 2)

	//node2 should have node 3 as its peer
	_, ok := n2.GetNetwork().peerManager.streams[n3.GetHostPeerInfo().PeerId]
	assert.True(t, ok)
	assert.Equal(t, 1, n2.GetNetwork().peerManager.connectionInCount)

	//node3 should have node 2 as its peer
	_, ok = n3.GetNetwork().peerManager.streams[n2.GetHostPeerInfo().PeerId]
	assert.True(t, ok)
	assert.Equal(t, 1, n3.GetNetwork().peerManager.connectionOutCount)
}

func TestNode_ConnectionFull(t *testing.T) {
	db1 := storage.NewRamStorage()
	defer db1.Close()

	n1, err := initNode("17DgRtQVvaytkiKAfXx9XbV23MESASSwUz", test_port10, nil, db1)
	defer n1.Stop()
	assert.Nil(t, err)

	//create node 2 and add node1 as a peer
	db2 := storage.NewRamStorage()
	defer db2.Close()
	n2, err := initNodeWithConfig("17DgRtQVvaytkiKAfXx9XbV23MESASSwUz", test_port11, 2, 2, n1.GetHostPeerInfo(), db2)
	defer n2.Stop()
	assert.Nil(t, err)

	util.WaitDoneOrTimeout(func() bool {
		return false
	}, 1)

	//create node 3 and add node1 as a peer
	db3 := storage.NewRamStorage()
	defer db3.Close()
	n3, err := initNodeWithConfig("17DgRtQVvaytkiKAfXx9XbV23MESASSwUz", test_port12, 2, 2, n1.GetHostPeerInfo(), db3)
	defer n3.Stop()
	assert.Nil(t, err)

	util.WaitDoneOrTimeout(func() bool {
		return false
	}, 1)

	//create node 3 and add node1 as a peer
	db4 := storage.NewRamStorage()
	defer db4.Close()
	n4, err := initNodeWithConfig("17DgRtQVvaytkiKAfXx9XbV23MESASSwUz", test_port13, 2, 2, n1.GetHostPeerInfo(), db4)
	defer n4.Stop()
	assert.Nil(t, err)

	util.WaitDoneOrTimeout(func() bool {
		return false
	}, 1)

	//create node 3 and add node1 as a peer
	db5 := storage.NewRamStorage()
	defer db5.Close()
	n5, err := initNode("17DgRtQVvaytkiKAfXx9XbV23MESASSwUz", test_port14, nil, db5)
	defer n5.Stop()
	assert.Nil(t, err)

	n5.GetNetwork().AddPeer(n2.GetHostPeerInfo())
	n4.GetNetwork().AddPeer(n5.GetHostPeerInfo())

	util.WaitDoneOrTimeout(func() bool {
		return false
	}, 2)

	assert.Equal(t, 2, n2.GetNetwork().peerManager.connectionInCount)
	assert.Equal(t, 0, n2.GetNetwork().peerManager.connectionOutCount)
	assert.Equal(t, 1, n3.GetNetwork().peerManager.connectionInCount)
	assert.Equal(t, 1, n3.GetNetwork().peerManager.connectionOutCount)
	assert.Equal(t, 0, n4.GetNetwork().peerManager.connectionInCount)
	assert.Equal(t, 2, n4.GetNetwork().peerManager.connectionOutCount)
	assert.Equal(t, 0, n5.GetNetwork().peerManager.connectionInCount)
	assert.Equal(t, 0, n5.GetNetwork().peerManager.connectionOutCount)
}
