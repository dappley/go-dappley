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
	"github.com/stretchr/testify/assert"
	"github.com/dappley/go-dappley/core"
	"github.com/dappley/go-dappley/storage"
	"github.com/dappley/go-dappley/network"
	"time"
)

func TestNewDpos(t *testing.T) {
	dpos := NewDpos()
	assert.Equal(t,1, cap(dpos.mintBlkCh))
	assert.Equal(t,1, cap(dpos.quitCh))
	assert.Nil(t,dpos.node)
}

func TestDpos_Setup(t *testing.T) {
	dpos := NewDpos()
	cbAddr := "abcdefg"
	bc := core.CreateBlockchain(core.Address{cbAddr},storage.NewRamStorage(),dpos)
	node := network.NewNode(bc)

	dpos.Setup(node, cbAddr)

	assert.Equal(t, bc, dpos.bc)
	assert.Equal(t, node, dpos.node)
}

func TestDpos_Stop(t *testing.T) {
	dpos := NewDpos()
	dpos.Stop()
	select{
	case <-dpos.quitCh:
	default:
		t.Error("Failed!")
	}
}

func TestDpos_Start(t *testing.T) {

	dpos := NewDpos()
	cbAddr := core.Address{"121yKAXeG4cw6uaGCBYjWk9yTWmMkhcoDD"}
	bc := core.CreateBlockchain(cbAddr,storage.NewRamStorage(),dpos)
	node := network.NewNode(bc)
	node.Start(21100)
	dpos.Setup(node, cbAddr.Address)

	miners := []string{cbAddr.Address}
	dynasty := NewDynastyWithProducers(miners)
	dynasty.SetTimeBetweenBlk(2)
	dynasty.SetMaxProducers(2)
	dpos.SetDynasty(dynasty)
	//3 seconds should be enough to mine a block with difficulty 14
	dpos.SetTargetBit(14)

	dpos.Start()
	//wait for the block gets mined
	time.Sleep(time.Second*6)
	dpos.Stop()

	assert.True(t, bc.GetMaxHeight()>=1)
}

func TestDpos_MultipleMiners(t *testing.T){
	const ( timeBetweenBlock = 2
			dposRounds = 3
			bufferTime = 1
			)

	miners := []string{
		"121yKAXeG4cw6uaGCBYjWk9yTWmMkhcoDD",
		"1MeSBgufmzwpiJNLemUe1emxAussBnz7a7",
	}
	dynasty := NewDynastyWithProducers(miners)
	dynasty.SetTimeBetweenBlk(timeBetweenBlock)
	dynasty.SetMaxProducers(len(miners))
	dposArray := []*Dpos{}
	var firstNode *network.Node
	for i:=0;i<len(miners);i++{
		dpos := NewDpos()
		dpos.SetDynasty(dynasty)
		dpos.SetTargetBit(14)
		bc := core.CreateBlockchain(core.Address{miners[0]},storage.NewRamStorage(),dpos)
		node := network.NewNode(bc)
		node.Start(21200+i)
		if i==0{
			firstNode = node
		}else{
			node.AddStream(firstNode.GetPeerID(),firstNode.GetPeerMultiaddr())
		}
		dpos.Setup(node, miners[i])
		dposArray = append(dposArray, dpos)
	}

	firstNode.SyncPeersBroadcast()

	for i:=0;i<len(miners);i++{
		dposArray[i].Start()
	}

	time.Sleep(time.Second*time.Duration(dynasty.dynastyTime*dposRounds+bufferTime))

	for i:=0;i<len(miners);i++{
		dposArray[i].Stop()
	}

	time.Sleep(time.Second)

	for i:=0;i<len(miners);i++{
		assert.Equal(t, uint64(dynasty.dynastyTime*dposRounds/timeBetweenBlock) , dposArray[i].bc.GetMaxHeight())
	}
}