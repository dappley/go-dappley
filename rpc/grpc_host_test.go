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

package rpc

import (
	"testing"

	"github.com/dappley/go-dappley/consensus"
	"github.com/dappley/go-dappley/storage"

	"github.com/dappley/go-dappley/network"
	"github.com/stretchr/testify/assert"
)

const grpcConfDir = "../storage/fakeFileLoaders/"

func TestNewGrpcServer(t *testing.T) {
	rfl := storage.NewRamFileLoader(grpcConfDir, "test.conf")
	defer rfl.DeleteFolder()
	node := network.NewNode(rfl.File, nil)
	grpcServer := NewGrpcServer(node, nil, consensus.NewDPOS(nil), "password")
	assert.Equal(t, node, grpcServer.node)
	assert.Equal(t, "password", grpcServer.password)
}
