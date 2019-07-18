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

func TestNewNode(t *testing.T) {
	priv, _, _ := crypto.GenerateEd25519Key(rand.Reader)
	crypto.MarshalPrivateKey(priv)
}

func TestNode_Stop(t *testing.T) {
	logger.SetLevel(logger.DebugLevel)
	cbAddr := core.Address{"dPGZmHd73UpZhrM6uvgnzu49ttbLp4AzU8"}
	mockConsensus := new(mocks.Consensus)
	bc := core.CreateBlockchain(cbAddr, storage.NewRamStorage(), mockConsensus, 128, nil, 100000)
	node := NewNode(bc.GetDb())
	err := node.Start(22100, nil)
	if err != nil {
		t.Fatal(err)
	}

	time.Sleep(time.Second)
	node.Stop()
	_, ok := <-node.network.host.Network().Process().Closed()
	assert.False(t, ok)
}
