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
	"math/big"
	"testing"

	"github.com/dappley/go-dappley/core"
	"github.com/dappley/go-dappley/network"
	"github.com/stretchr/testify/assert"
)

func TestProofOfWork_NewPoW(t *testing.T) {
	pow := NewProofOfWork()
	assert.Nil(t, pow.node)
	assert.Equal(t, big.NewInt(1).Lsh(big.NewInt(1), uint(256)), pow.target)
}

func TestProofOfWork_Setup(t *testing.T) {
	pow := NewProofOfWork()
	bc := core.GenerateMockBlockchain(5)
	cbAddr := "121yKAXeG4cw6uaGCBYjWk9yTWmMkhcoDD"
	pool := core.NewBlockPool(0)
	bm := core.NewBlockChainManager(bc, pool)
	pow.Setup(network.NewNode(bc.GetDb()), cbAddr, bm)
	assert.Equal(t, bc, pow.bm.Getblockchain())
}

func TestProofOfWork_SetTargetBit(t *testing.T) {
	tests := []struct {
		name     string
		bit      int
		expected int
	}{{"regular", 16, 16},
		{"zero", 0, 0},
		{"negative", -5, 0},
		{"above256", 257, 0},
		{"regular2", 18, 18},
		{"equalTo256", 256, 256},
	}

	pow := NewProofOfWork()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pow.SetTargetBit(tt.bit)
			target := big.NewInt(1)
			target.Lsh(target, uint(256-tt.expected))
			assert.Equal(t, target, pow.target)
		})
	}
}

func TestProofOfWork_isHashBelowTarget(t *testing.T) {

	pow := NewProofOfWork()
	pow.SetTargetBit(defaultTargetBits)

	//create a block that has a hash value larger than the target
	blk := core.GenerateMockBlock()
	hash := big.NewInt(1)
	hash.Lsh(hash, uint(256-defaultTargetBits+1))

	blk.SetHash(hash.Bytes())

	assert.False(t, pow.isHashBelowTarget(blk))

	//create a block that has a hash value smaller than the target
	hash = big.NewInt(1)
	hash.Lsh(hash, uint(256-defaultTargetBits-1))
	blk.SetHash(hash.Bytes())

	assert.True(t, pow.isHashBelowTarget(blk))
}
