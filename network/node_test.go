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
	err = node1.AddStreamMultiAddr(node2.GetMultiaddr())
	assert.Nil(t, err)
	assert.Len(t, node1.host.Network().Peerstore().Peers(), 2)
}

func TestNetwork_SendBlock(t *testing.T){
	bc := mockBlockchain(t)

	node1 := NewNode(bc)
	err := node1.Start(test_port3)
	assert.Nil(t, err)

	node2 := NewNode(bc)
	err = node2.Start(test_port4)
	assert.Nil(t, err)

	err = node2.AddStreamMultiAddr(node1.GetMultiaddr())
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
	bc,err := core.CreateBlockchain(addr,db)
	assert.Nil(t, err)
	return bc
}

/*func TestNetwork_node0(t *testing.T){
	bc := mockBlockchain(t)

	node1 := NewNode(bc)
	node1.Start(test_port1)
	//node1.AddStreamString("/ip4/127.0.0.1/tcp/10009/ipfs/QmcrXvSkD7JcVSi2UQ4RRED8McfsoGG2p7x8Ev9tUyZ584")
	//node1.AddStreamString("/ip4/192.168.10.90/tcp/10200/ipfs/QmQMzVX4XqCYPNbdAzsSDXNWijKQnoRNbDXQsgto7ZRyod")
	select{}

}

const node0_addr = "/ip4/127.0.0.1/tcp/12345/ipfs/Qma6Jq6JSH7MCTRKtFRY2SBYXW2xB4EFV3TeGwKUG9isDm"

func TestNetwork_node1(t *testing.T){
	bc := mockBlockchain(t)

	node1 := NewNode(bc)
	node1.Start(test_port2)
	node1.AddStreamString(node0_addr)
	//node1.AddStreamString("/ip4/192.168.10.90/tcp/10200/ipfs/QmQMzVX4XqCYPNbdAzsSDXNWijKQnoRNbDXQsgto7ZRyod")
	//select{}
	b := core.GenerateMockBlock()
	for{
		node1.SendBlock(b)
		time.Sleep(time.Second*15)
	}
}

func TestNetwork_node2(t *testing.T){
	bc := mockBlockchain(t)

	node1 := NewNode(bc)
	node1.Start(test_port3)
	node1.AddStreamString(node0_addr)
	//node1.AddStreamString("/ip4/192.168.10.90/tcp/10200/ipfs/QmQMzVX4XqCYPNbdAzsSDXNWijKQnoRNbDXQsgto7ZRyod")
	//select{}
	b := core.GenerateMockBlock()
	for{
		node1.SendBlock(b)
		time.Sleep(time.Second*15)
	}
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