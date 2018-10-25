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

package consensus

import (
	"github.com/dappley/go-dappley/core"
	"github.com/dappley/go-dappley/network"
	"github.com/dappley/go-dappley/storage"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNewDpos(t *testing.T) {
	dpos := NewDpos()
	assert.Equal(t, 1, cap(dpos.mintBlkCh))
	assert.Equal(t, 1, cap(dpos.quitCh))
	assert.Nil(t, dpos.node)
}

func TestDpos_Setup(t *testing.T) {
	dpos := NewDpos()
	cbAddr := "abcdefg"
	bc := core.CreateBlockchain(core.Address{cbAddr}, storage.NewRamStorage(), dpos, 128)
	node := network.NewNode(bc)

	dpos.Setup(node, cbAddr)

	assert.Equal(t, bc, dpos.bc)
	assert.Equal(t, node, dpos.node)
}

func TestDpos_Stop(t *testing.T) {
	dpos := NewDpos()
	dpos.Stop()
	select {
	case <-dpos.quitCh:
	default:
		t.Error("Failed!")
	}
}
