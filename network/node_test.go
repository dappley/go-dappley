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
	"bytes"
	"crypto/rand"
	"os"
	"testing"
	"time"

	"github.com/dappley/go-dappley/core"
	"github.com/dappley/go-dappley/mocks"
	"github.com/dappley/go-dappley/network/pb"
	"github.com/dappley/go-dappley/storage"
	"github.com/golang/protobuf/proto"
	"github.com/libp2p/go-libp2p-core/crypto"
	logger "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {

	logger.SetLevel(logger.WarnLevel)
	retCode := m.Run()
	os.Exit(retCode)
}

func TestNode_prepareData(t *testing.T) {
	tests := []struct {
		name    string
		msgData proto.Message
		cmd     string
		retData []byte
		retErr  error
	}{
		{
			name:    "CorrectProtoMsg",
			msgData: &networkpb.PeerInfo{Id: "pid", Address: []string{"addr"}},
			cmd:     SyncPeerList,
			retData: []byte{10, 12, 83, 121, 110, 99, 80, 101, 101, 114, 76, 105, 115, 116, 18, 11, 10, 3, 112, 105, 100, 18, 4, 97, 100, 100, 114},
			retErr:  nil,
		},
		{
			name:    "NoDataInput",
			msgData: nil,
			cmd:     SyncPeerList,
			retData: []byte{10, 12, 83, 121, 110, 99, 80, 101, 101, 114, 76, 105, 115, 116},
			retErr:  nil,
		},
		{
			name:    "NoCmdInput",
			msgData: &networkpb.PeerInfo{Id: "pid", Address: []string{"addr"}},
			cmd:     "",
			retData: nil,
			retErr:  ErrDapMsgNoCmd,
		},
		{
			name:    "NoInput",
			msgData: nil,
			cmd:     "",
			retData: nil,
			retErr:  ErrDapMsgNoCmd,
		},
	}
	n := FakeNodeWithPidAndAddr(nil, nil, "asd", "test")
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := n.prepareData(tt.msgData, tt.cmd, Unicast, "")
			//dapley msgs returned contains timestamp of when it was created. We only check the non-timestamp contents to make sure it is there.
			assert.Equal(t, true, bytes.Contains(data, tt.retData))
			assert.Equal(t, tt.retErr, err)
		})
	}
}

func TestNewNode(t *testing.T) {
	priv, _, _ := crypto.GenerateEd25519Key(rand.Reader)
	crypto.MarshalPrivateKey(priv)
}

func TestNode_Stop(t *testing.T) {
	logger.SetLevel(logger.DebugLevel)
	cbAddr := core.Address{"dPGZmHd73UpZhrM6uvgnzu49ttbLp4AzU8"}
	mockConsensus := new(mocks.Consensus)
	bc := core.CreateBlockchain(cbAddr, storage.NewRamStorage(), mockConsensus, 128, nil, 100000)
	pool := core.NewBlockPool(0)
	node := NewNode(bc, pool)
	err := node.Start(22100, nil)
	if err != nil {
		t.Fatal(err)
	}

	time.Sleep(time.Second)
	node.Stop()
	_, ok := <-node.network.host.Network().Process().Closed()
	assert.False(t, ok)
}
