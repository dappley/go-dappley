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
	"github.com/dappley/go-dappley/core/block"
	"github.com/dappley/go-dappley/logic/blockchain_logic"
	"github.com/dappley/go-dappley/logic/blockchain_manager"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/dappley/go-dappley/common"
	"github.com/dappley/go-dappley/core"
	"github.com/dappley/go-dappley/core/account"
	"github.com/dappley/go-dappley/network"
	"github.com/dappley/go-dappley/storage"
)

func TestNewDpos(t *testing.T) {
	dpos := NewDPOS()
	assert.Equal(t, 1, cap(dpos.newBlockCh))
	assert.Equal(t, 1, cap(dpos.stopCh))
	assert.Nil(t, dpos.node)
}

func TestDpos_Setup(t *testing.T) {
	dpos := NewDPOS()
	cbAddr := "abcdefg"
	bc := blockchain_logic.CreateBlockchain(account.NewAddress(cbAddr), storage.NewRamStorage(), dpos, core.NewTransactionPool(nil, 128), nil, 100000)
	pool := core.NewBlockPool()

	node := network.NewNode(bc.GetDb(), nil)

	bm := blockchain_manager.NewBlockchainManager(bc, pool, node)

	dpos.Setup(node, cbAddr, bm)

	assert.Equal(t, bc, dpos.bm.Getblockchain())
	assert.Equal(t, node, dpos.node)
}

func TestDpos_beneficiaryIsProducer(t *testing.T) {
	producers := []string{
		"121yKAXeG4cw6uaGCBYjWk9yTWmMkhcoDD",
		"1MeSBgufmzwpiJNLemUe1emxAussBnz7a7",
		"1LCn8D5W7DLV1CbKE3buuJgNJjSeoBw2ct"}

	cbtx := core.NewCoinbaseTX(account.NewAddress(producers[0]), "", 0, common.NewAmount(0))
	cbtxInvalidProducer := core.NewCoinbaseTX(account.NewAddress(producers[0]), "", 0, common.NewAmount(0))

	tests := []struct {
		name     string
		block    *block.Block
		expected bool
	}{
		{
			name: "BeneficiaryIsProducer",
			block: core.FakeNewBlockWithTimestamp(
				46,
				[]*core.Transaction{
					core.MockTransaction(),
					&cbtx,
				},
				nil,
			),
			expected: true,
		},
		{
			name: "ProducerNotAtItsTurn",
			block: core.FakeNewBlockWithTimestamp(
				44,
				[]*core.Transaction{
					core.MockTransaction(),
					&cbtx,
				},
				nil,
			),
			expected: false,
		},
		{
			name: "NotAProducer",
			block: core.FakeNewBlockWithTimestamp(
				44,
				[]*core.Transaction{
					core.MockTransaction(),
					&cbtxInvalidProducer,
				},
				nil,
			),
			expected: false,
		},
		{
			name:     "EmptyBlock",
			block:    nil,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dpos := NewDPOS()
			dpos.SetDynasty(NewDynasty(producers, len(producers), defaultTimeBetweenBlk))
			assert.Equal(t, tt.expected, dpos.beneficiaryIsProducer(tt.block))
		})
	}
}

func TestDPOS_isDoubleMint(t *testing.T) {
	dpos := NewDPOS()
	dpos.SetDynasty(NewDynasty(nil, defaultMaxProducers, defaultTimeBetweenBlk))
	blk1Time := int64(1548979365)
	blk2Time := int64(1548979366)

	// Both timestamps fall in the same DPoS time slot
	assert.Equal(t, int(blk1Time/defaultTimeBetweenBlk), int(blk2Time/defaultTimeBetweenBlk))

	blk1 := core.FakeNewBlockWithTimestamp(blk1Time, []*core.Transaction{}, nil)
	dpos.AddBlockToSlot(blk1)
	blk2 := core.FakeNewBlockWithTimestamp(blk2Time, []*core.Transaction{}, nil)

	assert.True(t, dpos.isDoubleMint(blk2))
}
