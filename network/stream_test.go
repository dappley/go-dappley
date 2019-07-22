// Copyright (C) 2018 go-dappley authors
//
// This file is part of the go-dappley library.
//
// the go-dappley library is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// the go-dappley library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with the go-dappley library.  If not, see <http://www.gnu.org/licenses/>.
//

package network

import (
	"github.com/dappley/go-dappley/network/network_model"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestStream_Send(t *testing.T) {
	s := &Stream{
		network_model.PeerInfo{},
		nil,
		[]byte{},
		make(chan *network_model.DappPacket, highPriorityChLength),
		make(chan *network_model.DappPacket, normalPriorityChLength),
		make(chan bool, WriteChTotalLength),
		make(chan bool, 1), //two channels to stop
		make(chan bool, 1),
	}

	data1 := network_model.ConstructDappPacketFromData([]byte("data1"), Unicast)
	data2 := network_model.ConstructDappPacketFromData([]byte("data2"), Unicast)
	s.Send(data1, network_model.NormalPriorityCommand)
	s.Send(data2, network_model.HighPriorityCommand)
	assert.Equal(t, 2, len(s.msgNotifyCh))
	assert.Equal(t, 1, len(s.highPriorityWriteCh))
	assert.Equal(t, 1, len(s.normalPriorityWriteCh))

	select {
	case receivedData := <-s.highPriorityWriteCh:
		assert.Equal(t, data2, receivedData)
	default:
		assert.Error(t, nil, "No data in high priority channel")
	}

	select {
	case receivedData := <-s.normalPriorityWriteCh:
		assert.Equal(t, data1, receivedData)
	default:
		assert.Error(t, nil, "No data in normal priority channel")
	}

}
