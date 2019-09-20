package network

import (
	"github.com/dappley/go-dappley/network/networkmodel"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNetwork_RecordMessage(t *testing.T) {
	net := NewNetwork(&NetworkContext{nil, networkmodel.PeerConnectionConfig{}, nil, nil, nil, nil})
	data1 := networkmodel.ConstructDappPacketFromData([]byte("data1"), networkmodel.Broadcast)
	data2 := networkmodel.ConstructDappPacketFromData([]byte("data2"), networkmodel.Broadcast)
	net.recordMessage(data1)
	assert.True(t, net.isNetworkRadiation(data1))
	assert.False(t, net.isNetworkRadiation(data2))

}
