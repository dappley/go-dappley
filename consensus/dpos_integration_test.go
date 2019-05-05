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
	"github.com/dappley/go-dappley/util"
	"github.com/stretchr/testify/assert"
)

func TestDpos_Start(t *testing.T) {
	dpos := NewDPOS()
	cbAddr := core.Address{"dPGZmHd73UpZhrM6uvgnzu49ttbLp4AzU8"}
	keystr := "5a66b0fdb69c99935783059bb200e86e97b506ae443a62febd7d0750cd7fac55"
	bc := core.CreateBlockchain(cbAddr, storage.NewRamStorage(), dpos, 128, nil, 100000)
	pool := core.NewBlockPool(0)
	node := network.NewNode(bc, pool)
	node.Start(22100)
	defer node.Stop()
	dpos.Setup(node, cbAddr.String())
	dpos.SetKey(keystr)

	miners := []string{cbAddr.String()}
	dynasty := NewDynasty(miners, 2, 2)
	dpos.SetDynasty(dynasty)
	//wait for the block gets mined
	currentTime := time.Now().UTC().Unix()
	dpos.Start()
	//wait for the block gets mined
	for bc.GetMaxHeight() <= 0 && !util.IsTimeOut(currentTime, int64(50)) {
	}
	dpos.Stop()

	assert.True(t, bc.GetMaxHeight() >= 1)
}

func TestDpos_MultipleMiners(t *testing.T) {
	const (
		timeBetweenBlock = 2
		dposRounds       = 3
	)

	miners := []string{
		"dPGZmHd73UpZhrM6uvgnzu49ttbLp4AzU8",
		"dQEooMsqp23RkPsvZXj3XbsRh9BUyGz2S9",
	}
	keystrs := []string{
		"5a66b0fdb69c99935783059bb200e86e97b506ae443a62febd7d0750cd7fac55",
		"bb23d2ff19f5b16955e8a24dca34dd520980fe3bddca2b3e1b56663f0ec1aa7e",
	}
	dynasty := NewDynasty(miners, len(miners), timeBetweenBlock)
	var dposArray []*DPOS
	var nodeArray []*network.Node
	var firstNode *network.Node
	for i, miner := range miners {
		dpos := NewDPOS()
		dpos.SetDynasty(dynasty)
		bc := core.CreateBlockchain(core.Address{miners[0]}, storage.NewRamStorage(), dpos, 128, nil, 100000)
		pool := core.NewBlockPool(0)
		node := network.NewNode(bc, pool)
		node.Start(21200 + i)
		nodeArray = append(nodeArray, node)
		if i == 0 {
			firstNode = node
		} else {
			node.GetPeerManager().AddAndConnectPeer(firstNode.GetInfo())
		}
		dpos.Setup(node, miner)
		dpos.SetKey(keystrs[i])
		dposArray = append(dposArray, dpos)
	}

	firstNode.SyncPeersBroadcast()

	for i := range miners {
		dposArray[i].Start()
	}

	time.Sleep(time.Second*time.Duration(dynasty.dynastyTime*dposRounds) + time.Second/2)

	for i := range miners {
		dposArray[i].Stop()
		nodeArray[i].Stop()
	}
	//Waiting block sync to other nodes
	time.Sleep(time.Second * 2)
	for i := range miners {
		v := dposArray[i]
		core.WaitDoneOrTimeout(func() bool {
			return !v.IsProducingBlock()
		}, 20)
	}

	for i := range miners {
		assert.Equal(t, uint64(dynasty.dynastyTime*dposRounds/timeBetweenBlock), dposArray[i].node.GetBlockchain().GetMaxHeight())
	}
}

func TestDPOS_UpdateLIB(t *testing.T) {
	const (
		timeBetweenBlock = 5
		dposRounds       = 3
	)

	miners := []string{
		"dPGZmHd73UpZhrM6uvgnzu49ttbLp4AzU8",
		"dQEooMsqp23RkPsvZXj3XbsRh9BUyGz2S9",
		"dastXXWLe5pxbRYFhcyUq8T3wb5srWkHKa",
		"dUuPPYshbBgkzUrgScEHWvdGbSxC8z4R12",
		"dPGD4t6ibpmyKZnXH1TNbbPw98EDaaZq8C",
	}
	keystrs := []string{
		"5a66b0fdb69c99935783059bb200e86e97b506ae443a62febd7d0750cd7fac55",
		"bb23d2ff19f5b16955e8a24dca34dd520980fe3bddca2b3e1b56663f0ec1aa7e",
		"300c0338c4b0d49edc66113e3584e04c6b907f9ded711d396d522aae6a79be1a",
		"da9282440fae188c371165e01615a2e1b14af68b3eaae51e6608c0bd86d4e6a6",
		"7c918ed7660d55759b7fc42b25f26bdab3caf8fc07586b2659a26470fb8dfc69",
	}
	dynasty := NewDynasty(miners, len(miners), timeBetweenBlock)

	var dposArray []*DPOS
	var nodeArray []*network.Node

	for i, miner := range miners {
		dpos := NewDPOS()
		dpos.SetDynasty(dynasty)
		bc := core.CreateBlockchain(core.Address{miners[0]}, storage.NewRamStorage(), dpos, 128, nil, 100000)
		pool := core.NewBlockPool(0)
		node := network.NewNode(bc, pool)
		node.Start(21200 + i)
		nodeArray = append(nodeArray, node)

		dpos.Setup(node, miner)
		dpos.SetKey(keystrs[i])
		dposArray = append(dposArray, dpos)
	}

	for i := range miners {
		for j := range miners {
			if i != j {
				nodeArray[i].GetPeerManager().AddAndConnectPeer(nodeArray[j].GetInfo())
			}
		}

		dposArray[i].Start()
	}

	time.Sleep(time.Second*time.Duration(dynasty.dynastyTime*dposRounds) + time.Second/2)

	for i := range miners {
		dposArray[i].Stop()
		nodeArray[i].Stop()
	}

	block0, _ := dposArray[0].node.GetBlockchain().GetLIB()
	assert.NotEqual(t, 0, block0.GetHeight())
	for i := range miners {
		block, _ := dposArray[i].node.GetBlockchain().GetLIB()
		assert.Equal(t, block0.GetHash(), block.GetHash())
	}
}
