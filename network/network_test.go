package network

import (
	"github.com/dappley/go-dappley/network/network_model"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNetwork_RecordMessage(t *testing.T) {
	net := NewNetwork(nil, nil, nil)
	data1 := network_model.ConstructDappPacketFromData([]byte("data1"), Broadcast)
	data2 := network_model.ConstructDappPacketFromData([]byte("data2"), Broadcast)
	net.RecordMessage(data1)
	assert.True(t, net.IsNetworkRadiation(data1))
	assert.False(t, net.IsNetworkRadiation(data2))

}
