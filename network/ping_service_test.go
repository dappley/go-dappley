package network

import (
	"testing"
	"time"

	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/stretchr/testify/require"

	"github.com/dappley/go-dappley/network/network_model"
)

func TestNewPingService(t *testing.T) {
	// invalid host
	_, err := NewPingService(nil, 1)
	require.Error(t, err)

	// invalid duration
	_, err = NewPingService(network_model.NewHost(0, nil, nil), 0)
	require.Error(t, err)
}

func TestPingService_Start(t *testing.T) {
	startPingService(t)
}

func TestPingService_Stop(t *testing.T) {
	ps := startPingService(t)
	require.Nil(t, ps.Stop())
	require.Error(t, ps.Stop())
}

func startPingService(t *testing.T) *PingService {
	h0 := network_model.NewHost(0, nil, nil)
	ps, err := NewPingService(h0, time.Second)
	require.Nil(t, err)
	err = ps.Start(func() map[peer.ID]network_model.PeerInfo {
		return make(map[peer.ID]network_model.PeerInfo, 0)
	}, func(results []*PingResult) {})
	require.Nil(t, err)
	return ps
}
