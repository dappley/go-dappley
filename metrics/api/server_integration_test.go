// +build integration

package metrics

import (
	"testing"

	peerstore "github.com/libp2p/go-libp2p-peerstore"
	logger "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"

	"github.com/dappley/go-dappley/core"
	"github.com/dappley/go-dappley/network"
	"github.com/dappley/go-dappley/storage"
)

func init() {
	InitAPI()
	logger.SetLevel(logger.PanicLevel)
}

func TestStartAPI(t *testing.T) {
	size, err := GetTransactionPoolSize()
	assert.Nil(t, err)
	assert.Equal(t, 0, size)
}

func TestTransactionPoolSize(t *testing.T) {
	// add transaction
	txPool := core.NewTransactionPool(1)
	tx := core.MockTransaction()
	txPool.Push(*tx)
	size, err := GetTransactionPoolSize()
	assert.Nil(t, err)
	assert.Equal(t, 1, size)

	// exceed tx pool limit
	txPool.Push(*core.MockTransaction())
	size, err = GetTransactionPoolSize()
	assert.Nil(t, err)
	assert.Equal(t, 1, size)

	// verify deserialization restores metric
	ramStorage := storage.NewRamStorage()
	err = txPool.SaveToDatabase(ramStorage)
	assert.Nil(t, err)
	newTXPool := core.LoadTxPoolFromDatabase(ramStorage, 1)
	size, err = GetTransactionPoolSize()
	assert.Nil(t, err)
	assert.Equal(t, 1, newTXPool.GetNumOfTxInPool())
	assert.Equal(t, 1, size)

	// remove transaction from pool
	txPool.CleanUpMinedTxs([]*core.Transaction{tx})
	size, err = GetTransactionPoolSize()
	assert.Nil(t, err)
	assert.Equal(t, 0, size)
}

func TestGetConnectedPeersFunc(t *testing.T) {
	bc := core.CreateBlockchain(core.NewAddress(""), storage.NewRamStorage(), nil, 128, nil, 100000)
	node0 := network.NewNode(bc, core.NewBlockPool(10))
	node1 := network.NewNode(bc, core.NewBlockPool(10))

	err := node0.Start(0)
	assert.Nil(t, err)
	err = node1.Start(0)
	assert.Nil(t, err)
	fun := getConnectedPeersFunc(node0)

	err = node0.GetPeerManager().AddAndConnectPeer(node1.GetInfo())
	assert.Nil(t, err)

	peers := fun().([]peerstore.PeerInfo)
	assert.Equal(t, 1, len(peers))
	assert.Equal(t, node1.GetPeerID(), peers[0].ID)
}
