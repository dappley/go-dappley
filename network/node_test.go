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
	test_port5
	test_port6
	test_port7
	test_port8
	test_port9
	test_port10

)

const blockchainDbFile = "../bin/networktest.db"

func TestNetwork_Setup(t *testing.T) {

	bc := mockBlockchain(t)

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
	bc := mockBlockchain(t)

	//create node1
	node1 := NewNode(bc)
	err := node1.Start(test_port3)
	assert.Nil(t, err)

	//create node 2 and add node1 as a peer
	node2 := NewNode(bc)
	err = node2.Start(test_port4)
	assert.Nil(t, err)
	err = node2.AddStream(node1.GetPeerID(),node1.GetPeerMultiaddr())
	assert.Nil(t, err)

	//create node 3 and add node1 as a peer
	node3 := NewNode(bc)
	err = node3.Start(test_port5)
	assert.Nil(t, err)
	err = node3.AddStream(node1.GetPeerID(),node1.GetPeerMultiaddr())
	assert.Nil(t, err)

	//node 1 broadcast a block
	b1 := core.GenerateMockBlock()
	node1.SendBlock(b1)

	time.Sleep(time.Second)

	//node2 receives the block
	b2:= node2.GetBlocks()
	assert.NotEmpty(t, b2)
	assert.Equal(t,*b1,*b2[0])

	//node3 receives the block
	b3:= node3.GetBlocks()
	assert.NotEmpty(t, b3)
	assert.Equal(t,*b1,*b3[0])

/*	for _,s:=range node1.streams{
		s.Send([]byte{4,2,3,1,4})
	}
	time.Sleep(time.Second)*/
}

func TestNode_SyncPeers(t *testing.T){
	bc := mockBlockchain(t)

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
	node1.SyncPeers()

	time.Sleep(time.Second*2)

	//node2 should have node 3 as its peer
	assert.True(t,node2.peerlist.IsInPeerlist(node3.GetInfo()))

	//node3 should have node 2 as its peer
	assert.True(t,node3.peerlist.IsInPeerlist(node2.GetInfo()))

	time.Sleep(time.Second)

	/*	for _,s:=range node1.streams{
			s.Send([]byte{4,2,3,1,4})
		}
		time.Sleep(time.Second)*/
}

func mockBlockchain(t *testing.T) *core.Blockchain{
	db := storage.OpenDatabase(blockchainDbFile)
	defer db.Close()

	addr,err := logic.CreateWallet()
	assert.Nil(t, err)
	bc,err := core.CreateBlockchain(addr,db)
	assert.Nil(t, err)
	return bc
}

/*func TestNetwork_node0(t *testing.T){
	bc := mockBlockchain(t)

	node1 := NewNode(bc)
	node1.Start(test_port1)
	select{}
}

const node0_addr = "/ip4/127.0.0.1/tcp/10000/ipfs/QmRtBnqSBJ6a4yixnT765KGga7dxMH5wqU8FpzEj1ryyfx"

func TestNetwork_node1(t *testing.T){
	bc := mockBlockchain(t)

	node1 := NewNode(bc)
	err := node1.Start(test_port2)
	assert.Nil(t, err)
	err = node1.AddStreamString(node0_addr)
	assert.Nil(t, err)
	//node1.AddStreamString("/ip4/192.168.10.90/tcp/10200/ipfs/QmQMzVX4XqCYPNbdAzsSDXNWijKQnoRNbDXQsgto7ZRyod")
	b := core.GenerateMockBlock()
	for{
		node1.SendBlock(b)
		time.Sleep(time.Second*10)
	}
}

func TestNetwork_node2(t *testing.T){
	bc := mockBlockchain(t)

	node1 := NewNode(bc)
	node1.Start(test_port3)
	node1.AddStreamString(node0_addr)
	//node1.AddStreamString("/ip4/192.168.10.90/tcp/10200/ipfs/QmQMzVX4XqCYPNbdAzsSDXNWijKQnoRNbDXQsgto7ZRyod")
	select{}
}

func TestNetwork_node3(t *testing.T){
	bc := mockBlockchain(t)

	node1 := NewNode(bc)
	node1.Start(test_port4)
	node1.AddStreamString(node0_addr)
	//node1.AddStreamString("/ip4/192.168.10.90/tcp/10200/ipfs/QmQMzVX4XqCYPNbdAzsSDXNWijKQnoRNbDXQsgto7ZRyod")
	//select{}
	b := core.GenerateMockBlock()
	for{
		node1.SendBlock(b)
		time.Sleep(time.Second*15)
	}
}*/