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
	"github.com/dappley/go-dappley/core"
	"github.com/dappley/go-dappley/storage"
	"github.com/stretchr/testify/assert"
	"testing"
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

func initNode(address string, port int, db storage.Storage) (*Node, error){
	addr := core.Address{address}
	bc := core.CreateBlockchain(addr ,db,nil, 128)
	n := NewNode(bc)
	err := n.Start(port)
	return n, err
}

func TestNetwork_AddStream(t *testing.T) {

	db := storage.NewRamStorage()
	defer db.Close()

	//create node1
	n1,err := initNode("17DgRtQVvaytkiKAfXx9XbV23MESASSwUz",test_port1, db)
	assert.Nil(t, err)

	//currently it should only have itself as its node
	assert.Len(t, n1.host.Network().Peerstore().Peers(), 1)

	//create node2
	n2, err := initNode("17DgRtQVvaytkiKAfXx9XbV23MESASSwUz",test_port2,db)
	assert.Nil(t, err)

	//set node2 as the peer of node1
	err = n1.AddStream(n2.GetPeerID(),n2.GetPeerMultiaddr())
	assert.Nil(t, err)
	assert.Len(t, n1.host.Network().Peerstore().Peers(), 2)
}


func TestNetwork_BroadcastBlock(t *testing.T){
	//setup node 0
	db := storage.NewRamStorage()
	defer db.Close()

	n1, err := initNode("QmWyMUMBeWxwU4R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ",test_port3, db)
	assert.Nil(t, err)

	//setup node 1
	n2 , err := initNode("QmWyMUMBeWxwU4R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ",test_port4, db)
	assert.Nil(t, err)

	err = n2.AddStream(n1.GetPeerID(),n1.GetPeerMultiaddr())
	assert.Nil(t, err)

	blk := core.GenerateMockBlock()
	n1.BroadcastBlock(blk)

	//wait for node 1 to receive response
	core.WaitDoneOrTimeout(func() bool {
		blk, _ := n2.recentlyRcvedDapMsgs.Load(hex.EncodeToString(blk.GetHash()))
		return blk != nil
	}, 5)

	assert.True(t, true)
}

func TestNode_SyncPeers(t *testing.T) {
	db := storage.NewRamStorage()
	defer db.Close()

	n1, err := initNode("17DgRtQVvaytkiKAfXx9XbV23MESASSwUz",test_port7, db)
	assert.Nil(t, err)

	//create node 2 and add node1 as a peer
	n2, err := initNode("17DgRtQVvaytkiKAfXx9XbV23MESASSwUz",test_port8, db)
	assert.Nil(t, err)

	err = n2.AddStream(n1.GetPeerID(),n1.GetPeerMultiaddr())
	assert.Nil(t, err)

	//create node 3 and add node1 as a peer
	n3, err := initNode("17DgRtQVvaytkiKAfXx9XbV23MESASSwUz",test_port9, db)
	assert.Nil(t, err)

	err = n3.AddStream(n1.GetPeerID(),n1.GetPeerMultiaddr())
	assert.Nil(t, err)

	//node 1 broadcast syncpeers
	n1.SyncPeersBroadcast()

	core.WaitDoneOrTimeout(func() bool {
		//no condition to be checked
		return false
	}, 5)

	//node2 should have node 3 as its peer
	assert.True(t,n2.peerList.IsInPeerlist(n3.GetInfo()))

	//node3 should have node 2 as its peer
	assert.True(t,n3.peerList.IsInPeerlist(n2.GetInfo()))

}

func TestNode_RequestBlockUnicast(t *testing.T) {

	//setup node 1
	db := storage.NewRamStorage()
	defer db.Close()
	n1, err := initNode("17DgRtQVvaytkiKAfXx9XbV23MESASSwUz",test_port5, db)
	assert.Nil(t, err)
	//setup node 2
	n2, err := initNode("17DgRtQVvaytkiKAfXx9XbV23MESASSwUz",test_port6, db)
	assert.Nil(t, err)

	err = n2.AddStream(n1.GetPeerID(),n1.GetPeerMultiaddr())
	assert.Nil(t, err)

	blk := core.GenerateMockBlock()

	err = n1.bc.GetDb().Put(blk.GetHash(),blk.Serialize())
	assert.Nil(t, err)

	n2.RequestBlockUnicast(blk.GetHash(),n1.GetPeerID())
	//wait for node 1 to receive response
	core.WaitDoneOrTimeout(func() bool {
		blk, _ := n2.recentlyRcvedDapMsgs.Load(hex.EncodeToString(blk.GetHash()))
		return blk != nil
	}, 5)

	assert.True(t, true)
}
