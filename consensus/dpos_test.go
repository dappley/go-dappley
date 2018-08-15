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
	dynasty := NewDynastyWithMiners(miners)
	dynasty.SetTimeBetweenBlk(2)
	dynasty.SetMaxProducers(2)
	dpos.SetDynasty(*dynasty)
	//3 seconds should be enough to mine a block with difficulty 14
	dpos.SetTargetBit(14)

	dpos.Start()
	//wait for the block gets mined
	time.Sleep(time.Second*6)
	dpos.Stop()

	assert.True(t, bc.GetMaxHeight()>=1)
}

func TestDpos_MultipleMiners(t *testing.T){

	miners := []string{
		"121yKAXeG4cw6uaGCBYjWk9yTWmMkhcoDD",
		"1MeSBgufmzwpiJNLemUe1emxAussBnz7a7",
	}
	dynasty := NewDynastyWithMiners(miners)
	dynasty.SetTimeBetweenBlk(5)
	dynasty.SetMaxProducers(len(miners))
	dposArray := []*Dpos{}
	var firstNode *network.Node
	for i:=0;i<len(miners);i++{
		dpos := NewDpos()
		dpos.SetDynasty(*dynasty)
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

	firstNode.SyncPeers()

	for i:=0;i<len(miners);i++{
		dposArray[i].Start()
	}

	time.Sleep(time.Second*time.Duration(dynasty.dynastyTime*2+1))

	for i:=0;i<len(miners);i++{
		dposArray[i].Stop()
	}


	time.Sleep(time.Second)

	for i:=0;i<len(miners);i++{
		assert.True(t, dposArray[i].bc.GetMaxHeight()>=3)
	}
}