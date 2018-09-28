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

package consensus

import (
	"testing"
	"time"

	"github.com/dappley/go-dappley/core"
	"github.com/dappley/go-dappley/network"
	"github.com/dappley/go-dappley/storage"
	"github.com/stretchr/testify/assert"
)

func TestDpos_Start(t *testing.T) {

	dpos := NewDpos()
	cbAddr := core.Address{"1ArH9WoB9F7i6qoJiAi7McZMFVQSsBKXZR"}
	keystr := "5a66b0fdb69c99935783059bb200e86e97b506ae443a62febd7d0750cd7fac55"
	bc := core.CreateBlockchain(cbAddr, storage.NewRamStorage(), dpos)
	node := network.NewNode(bc)
	node.Start(21100)
	dpos.Setup(node, cbAddr.Address)
	dpos.SetKey(keystr)

	miners := []string{cbAddr.Address}
	dynasty := NewDynastyWithProducers(miners)
	dynasty.SetTimeBetweenBlk(2)
	dynasty.SetMaxProducers(2)
	dpos.SetDynasty(dynasty)
	//3 seconds should be enough to mine a block with difficulty 14
	dpos.SetTargetBit(14)
	//wait for the block gets mined
	currentTime := time.Now().UTC().Unix()
	dpos.Start()
	//wait for the block gets mined
	for bc.GetMaxHeight() <= 0 && !core.IsTimeOut(currentTime, int64(50)) {
	}
	dpos.Stop()

	assert.True(t, bc.GetMaxHeight() >= 1)
}

func TestDpos_MultipleMiners(t *testing.T) {
	const (
		timeBetweenBlock = 2
		dposRounds       = 3
		bufferTime       = 1
	)

	miners := []string{
		"1ArH9WoB9F7i6qoJiAi7McZMFVQSsBKXZR",
		"1BpXBb3uunLa9PL8MmkMtKNd3jzb5DHFkG",
	}
	keystrs := []string{
		"5a66b0fdb69c99935783059bb200e86e97b506ae443a62febd7d0750cd7fac55",
		"bb23d2ff19f5b16955e8a24dca34dd520980fe3bddca2b3e1b56663f0ec1aa7e",
	}
	dynasty := NewDynastyWithProducers(miners)
	dynasty.SetTimeBetweenBlk(timeBetweenBlock)
	dynasty.SetMaxProducers(len(miners))
	dposArray := []*Dpos{}
	var firstNode *network.Node
	for i := 0; i < len(miners); i++ {
		dpos := NewDpos()
		dpos.SetDynasty(dynasty)
		dpos.SetTargetBit(0)
		bc := core.CreateBlockchain(core.Address{miners[0]}, storage.NewRamStorage(), dpos)
		node := network.NewNode(bc)
		node.Start(21200 + i)
		if i == 0 {
			firstNode = node
		} else {
			node.AddStream(firstNode.GetPeerID(), firstNode.GetPeerMultiaddr())
		}
		dpos.Setup(node, miners[i])
		dpos.SetKey(keystrs[i])
		dposArray = append(dposArray, dpos)
	}

	firstNode.SyncPeersBroadcast()

	for i := 0; i < len(miners); i++ {
		dposArray[i].Start()
	}

	time.Sleep(time.Second * time.Duration(dynasty.dynastyTime*dposRounds+bufferTime))

	for i := 0; i < len(miners); i++ {
		dposArray[i].Stop()
	}

	for i := 0; i < len(miners); i++ {
		v := dposArray[i]
		core.WaitFullyStop(v, 20)
	}

	for i := 0; i < len(miners); i++ {
		assert.Equal(t, uint64(dynasty.dynastyTime*dposRounds/timeBetweenBlock), dposArray[i].bc.GetMaxHeight())
	}
}
