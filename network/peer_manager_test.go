package network

import (
	"github.com/dappley/go-dappley/network/network_model"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestPeerManager_StartNewPingService(t *testing.T) {
	pm := NewPeerManager(nil, nil, nil, nil)

	// no host
	require.False(t, pm.StartNewPingService(time.Second))

	pm.SetHost(network_model.NewHost(0, nil, nil))
	// success
	require.True(t, pm.StartNewPingService(time.Second))

	// already running
	require.False(t, pm.StartNewPingService(time.Second))
}

func TestPeerManager_StopPingService(t *testing.T) {
	pm := NewPeerManager(nil, nil, nil, nil)

	// not running
	require.False(t, pm.StopPingService())

	pm.SetHost(network_model.NewHost(0, nil, nil))
	// start
	require.True(t, pm.StartNewPingService(time.Second))

	// stop
	require.True(t, pm.StopPingService())
	require.False(t, pm.StopPingService())
}
