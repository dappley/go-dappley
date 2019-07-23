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
	"testing"

	"github.com/dappley/go-dappley/core"
	"github.com/dappley/go-dappley/storage"
	"github.com/stretchr/testify/assert"
)

func TestBlockProducer_ProduceBlock(t *testing.T) {
	bp := NewBlockProducer()
	cbAddr := "1FoupuhmPN4q1wiUrM5QaYZjYKKLLXzPPg"
	bc := core.CreateBlockchain(
		core.NewAddress(cbAddr),
		storage.NewRamStorage(),
		nil,
		core.NewTransactionPool(nil, 128),
		nil,
		100000,
	)
	bp.Setup(bc, cbAddr)
	processRuns := false
	bp.SetProcess(func(ctx *core.BlockContext) {
		processRuns = true
	})
	block := bp.ProduceBlock(0)
	assert.True(t, processRuns)
	assert.NotNil(t, block)

}
