package network

import (
	"testing"
	"github.com/stretchr/testify/assert"
	"github.com/dappley/go-dappley/core"
	"time"
	"github.com/dappley/go-dappley/storage"
	"os"
	logger "github.com/sirupsen/logrus"
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

func TestMain(m *testing.M){

	logger.SetLevel(logger.WarnLevel)
	retCode := m.Run()
	os.Exit(retCode)
}

func TestNetwork_Setup(t *testing.T) {

	db := storage.NewRamStorage()
	defer db.Close()
	addr := core.Address{"17DgRtQVvaytkiKAfXx9XbV23MESASSwUz"}
	bc,err := core.CreateBlockchain(addr,db)
	assert.Nil(t, err)

	//create node1
	node1 := NewNode(bc)
	err = node1.Start(test_port1)
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

	var bcInputs = []struct{
		addr 		core.Address
		testPort 	int
	}{
		{core.Address{"17DgRtQVvaytkiKAfXx9XbV23MESASSwUz"},test_port3},
		{core.Address{"17DgRtQVvaytkiKAfXx9XbV23MESASSwUx"},test_port4},
		{core.Address{"17DgRtQVvaytkiKAfXx9XbV23MESASSwUy"},test_port5},
	}

	var nodes []*Node

	//setup each node
	for _,input := range bcInputs{
		db := storage.NewRamStorage()
		defer db.Close()
		bc, err := core.CreateBlockchain(input.addr, db)
		assert.Nil(t, err)
		n := NewNode(bc)
		n.Start(input.testPort)
		nodes = append(nodes, n)
	}

	//add node0 as a stream peer in node1 and node2
	err := nodes[1].AddStream(nodes[0].GetPeerID(),nodes[0].GetPeerMultiaddr())
	assert.Nil(t, err)

	err = nodes[2].AddStream(nodes[0].GetPeerID(),nodes[0].GetPeerMultiaddr())
	assert.Nil(t, err)


	//node 0 broadcast a block
	b1 := core.GenerateMockBlock()
	nodes[0].SendBlock(b1)

	time.Sleep(time.Second)

	//node2 receives the block
	b2:= nodes[1].GetBlocks()
	assert.NotEmpty(t, b2)
	assert.Equal(t,*b1,*b2[0])

	//node3 receives the block
	b3:= nodes[2].GetBlocks()
	assert.NotEmpty(t, b3)
	assert.Equal(t,*b1,*b3[0])

/*	for _,s:=range node1.streams{
		s.Send([]byte{4,2,3,1,4})
	}
	time.Sleep(time.Second)*/
}

func TestNode_SyncPeers(t *testing.T){
	db := storage.NewRamStorage()
	defer db.Close()
	addr := core.Address{"17DgRtQVvaytkiKAfXx9XbV23MESASSwUz"}
	bc,err := core.CreateBlockchain(addr,db)
	assert.Nil(t, err)

	//create node1
	node1 := NewNode(bc)
	err = node1.Start(test_port6)
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
	assert.True(t,node2.peerList.IsInPeerlist(node3.GetInfo()))

	//node3 should have node 2 as its peer
	assert.True(t,node3.peerList.IsInPeerlist(node2.GetInfo()))

	time.Sleep(time.Second)

	/*	for _,s:=range node1.streams{
			s.Send([]byte{4,2,3,1,4})
		}
		time.Sleep(time.Second)*/
}

func TestNode_RequestBlockUnicast(t *testing.T) {

	//set up two nodes
	nodes := []*Node{}
	for i:=0; i<2;i++{
		db := storage.NewRamStorage()
		defer db.Close()
		addr := core.Address{"17DgRtQVvaytkiKAfXx9XbV23MESASSwUz"}
		bc,err := core.CreateBlockchain(addr,db)
		assert.Nil(t, err)
		//create node
		node := NewNode(bc)
		err = node.Start(test_port9+i)
		assert.Nil(t, err)

		if i!=0 {
			err = node.AddStream(nodes[0].GetPeerID(),nodes[0].GetPeerMultiaddr())
			assert.Nil(t, err)
		}
		nodes = append(nodes, node)
	}
	time.Sleep(time.Second)

	//generate a block and store it in node0 blockchain
	blk := core.GenerateMockBlock()
	nodes[0].bc.DB.Put(blk.GetHash(),blk.Serialize())

	//node1 request the block
	nodes[1].RequestBlockUnicast(blk.GetHash(),nodes[0].GetPeerID())
	time.Sleep(time.Second*2)
	assert.Equal(t, blk, nodes[1].GetBlocks()[0])

}

/*func TestNetwork_node0(t *testing.T){
	bc := mockBlockchain(t)

	node1 := NewNode(bc)
	node1.Start(test_port1)
	select{}
}

const node0_addr = "/ip4/127.0.0.1/tcp/12345/ipfs/QmRuJ1V6xtj2H2gVnGD8bqt9KTViSu7gcHGhjy5yEM3FZm"

func TestNetwork_node1(t *testing.T){
	bc := mockBlockchain(t)
	logger.SetLevel(logger.DebugLevel)
	node1 := NewNode(bc)
	err := node1.Start(test_port2)
	assert.Nil(t, err)
	err = node1.AddStreamString(node0_addr)
	assert.Nil(t, err)
	//node1.AddStreamString("/ip4/192.168.10.90/tcp/10200/ipfs/QmQMzVX4XqCYPNbdAzsSDXNWijKQnoRNbDXQsgto7ZRyod")
	select{}
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