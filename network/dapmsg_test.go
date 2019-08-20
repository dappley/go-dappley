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
	"testing"

	"github.com/dappley/go-dappley/network/pb"
	"github.com/stretchr/testify/assert"
)

func TestDapmsg_ToProto(t *testing.T) {
	msg := DapMsg{"cmd", []byte{1, 2, 3, 4}, 11111111, "", Unicast}
	retMsg := &networkpb.Dapmsg{Cmd: "cmd", Data: []byte{1, 2, 3, 4}, UnixTimeReceived: 11111111, Key: "", UniOrBroadcast: Unicast, Counter: uint64(0)}

	assert.Equal(t, msg.ToProto(), retMsg)
}

func TestDapMsg_FromProto(t *testing.T) {
	msg := DapMsg{"cmd", []byte{1, 2, 3, 4}, 11111111, "", Unicast}
	retMsg := &networkpb.Dapmsg{Cmd: "cmd", Data: []byte{1, 2, 3, 4}, UnixTimeReceived: 11111111, Key: "", UniOrBroadcast: Unicast, Counter: uint64(0)}
	msg2 := DapMsg{}
	msg2.FromProto(retMsg)

	assert.Equal(t, msg, msg2)
}
