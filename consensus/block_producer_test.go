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

	"github.com/stretchr/testify/assert"

	"github.com/dappley/go-dappley/core"
	"github.com/dappley/go-dappley/storage"
)

func TestBlockProducer_Validate(t *testing.T) {

	bp := NewBlockProducer()

	blk := core.GenerateMockBlock()
	target := big.NewInt(1)
	target.Lsh(target, uint(256-defaulttargetBits-1))
	blk.SetHash(target.Bytes())

	assert.True(t, bp.Validate(blk))
}

func TestBlockProducer_Start(t *testing.T) {
	bp := NewBlockProducer()
	cbAddr := "1FoupuhmPN4q1wiUrM5QaYZjYKKLLXzPPg"
	keystr := "ac0a17dd3025b433ca0307d227241430ff4dda4be5e01a6c6cc6d2ccfaec895b"
	bc := core.CreateBlockchain(
		core.Address{cbAddr},
		storage.NewRamStorage(),
		nil,
		128,
	)
	retCh := make(chan *MinedBlock, 0)
	bp.Setup(bc, cbAddr, retCh)
	bp.SetPrivKey(keystr)
	bp.Start()
	blk := <-retCh
	assert.True(t, blk.isValid)
	assert.True(t, blk.block.VerifyHash())
	assert.True(t, bp.Validate(blk.block))
}
