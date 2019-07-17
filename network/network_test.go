package network

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNetwork_RecordMessage(t *testing.T) {
	net := NewNetwork(nil, nil, nil, nil)
	data1 := ConstructDappPacketFromData([]byte("data1"), Broadcast)
	data2 := ConstructDappPacketFromData([]byte("data2"), Broadcast)
	net.RecordMessage(data1)
	assert.True(t, net.IsNetworkRadiation(data1))
	assert.False(t, net.IsNetworkRadiation(data2))

}
