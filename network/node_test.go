package network

import (
	"testing"
	"github.com/stretchr/testify/assert"
	"github.com/dappley/go-dappley/core"
	"time"
	"github.com/dappley/go-dappley/storage"
	"github.com/dappley/go-dappley/logic"
)

const(
	test_port1 = 10000 + iota
	test_port2
	test_port3
	test_port4
)

func TestNetwork_Setup(t *testing.T) {

	bc := mockBlockchain(t)

	//create node1
	node1, err := NewNode(test_port1, bc)
	assert.Nil(t, err)

	//currently it should only have itself as its node
	assert.Len(t, node1.host.Network().Peerstore().Peers(), 1)

	//create node2
	node2, err := NewNode(test_port2, bc)
	assert.Nil(t, err)

	//set node2 as the peer of node1
	err = node1.AddStream(node2.GetMultiaddr())
	assert.Nil(t, err)
	assert.Len(t, node1.host.Network().Peerstore().Peers(), 2)
}

func TestNetwork_SendBlock(t *testing.T){
	bc := mockBlockchain(t)

	node1, err := NewNode(test_port3, bc)
	assert.Nil(t, err)

	node2, err := NewNode(test_port4, bc)
	assert.Nil(t, err)

	err = node2.AddStream(node1.GetMultiaddr())
	assert.Nil(t, err)

	b2 := core.GenerateMockBlock()
	node2.SendBlock(b2)

	time.Sleep(time.Second)

	b1:= node1.GetBlocks()
	assert.NotEmpty(t, b1)
	assert.Equal(t,*b2,*b1[0])
}

func mockBlockchain(t *testing.T) *core.Blockchain{
	db := storage.OpenDatabase(core.BlockchainDbFile)
	defer db.Close()
	addr,err := logic.CreateWallet()
	assert.Nil(t, err)
	bc,err := core.CreateBlockchain(addr,*db)
	assert.Nil(t, err)
	return bc
}
