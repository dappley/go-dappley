package network

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/dappley/go-dappley/consensus"
	"github.com/dappley/go-dappley/core"
	"github.com/dappley/go-dappley/storage"
)

var (
	bc   = core.CreateBlockchain(core.NewAddress(""), storage.NewRamStorage(), consensus.NewDPOS(), 100, nil, 100)
	bp   = core.NewBlockPool(100)
	node = NewNode(bc, bp)
)

func TestPeerManager_StartNewPingService(t *testing.T) {
	pm := NewPeerManager(node, nil)
	// no host
	require.False(t, pm.StartNewPingService(time.Second))

	// success
	require.Nil(t, node.Start(0))
	require.True(t, pm.StartNewPingService(time.Second))

	// already running
	require.False(t, pm.StartNewPingService(time.Second))
}

func TestPeerManager_StopPingService(t *testing.T) {
	pm := NewPeerManager(node, nil)
	// not running
	require.False(t, pm.StopPingService())

	// start
	require.Nil(t, node.Start(0))
	require.True(t, pm.StartNewPingService(time.Second))

	// stop
	require.True(t, pm.StopPingService())
	require.False(t, pm.StopPingService())
}
