package network

import (
	"github.com/dappley/go-dappley/network/network_model"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNetwork_RecordMessage(t *testing.T) {
	net := NewNetwork(nil, network_model.PeerConnectionConfig{}, nil, nil)
	data1 := network_model.ConstructDappPacketFromData([]byte("data1"), network_model.Broadcast)
	data2 := network_model.ConstructDappPacketFromData([]byte("data2"), network_model.Broadcast)
	net.RecordMessage(data1)
	assert.True(t, net.IsNetworkRadiation(data1))
	assert.False(t, net.IsNetworkRadiation(data2))

}
