package network

import (
	"github.com/dappley/go-dappley/network/network_model"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNetwork_RecordMessage(t *testing.T) {
	net := NewNetwork(&NetworkContext{nil, network_model.PeerConnectionConfig{}, nil, nil, nil, nil})
	data1 := network_model.ConstructDappPacketFromData([]byte("data1"), network_model.Broadcast)
	data2 := network_model.ConstructDappPacketFromData([]byte("data2"), network_model.Broadcast)
	net.recordMessage(data1)
	assert.True(t, net.isNetworkRadiation(data1))
	assert.False(t, net.isNetworkRadiation(data2))

}
