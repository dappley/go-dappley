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

func newTargetRequirement(bit int) Requirement {
	target := big.NewInt(1)
	target = target.Lsh(target, uint(256-bit))
	return func(block *core.Block) bool {
		var hashInt big.Int
		var hashInt2 big.Int

		hash := block.GetHash()
		hashInt.SetBytes(hash)

		hashFromNonce := block.CalculateHashWithNonce(block.GetNonce())
		hashInt2.SetBytes(hashFromNonce)
		return hashInt.Cmp(target) == -1 && hashInt2.Cmp(target) == -1
	}
}

func TestMiner_VerifyNonce(t *testing.T) {

	miner := NewMiner()
	miner.SetRequirement(newTargetRequirement(14))
	cbAddr := core.NewAddress("1FoupuhmPN4q1wiUrM5QaYZjYKKLLXzPPg")
	keystr := "ac0a17dd3025b433ca0307d227241430ff4dda4be5e01a6c6cc6d2ccfaec895b"
	bc := core.CreateBlockchain(
		cbAddr,
		storage.NewRamStorage(),
		nil,
		128,
		nil,
	)
	defer bc.GetDb().Close()

	miner.Setup(bc, cbAddr.String(), nil)
	miner.SetPrivateKey(keystr)

	//prepare a block with correct nonce value
	newBlock := core.NewBlock(nil, nil)
	nonce := int64(0)
mineloop2:
	for {
		hash := newBlock.CalculateHashWithNonce(nonce)
		newBlock.SetNonce(nonce)
		newBlock.SetHash(newBlock.CalculateHashWithNonce(nonce))
		fulfilled := miner.requirement(newBlock)
		if fulfilled {
			newBlock.SignBlock(miner.key, hash)
			break mineloop2
		} else {
			nonce++
		}
	}

	//check if the verifyNonce function returns true
	assert.True(t, miner.requirement(newBlock))

	//input a wrong nonce value, check if it returns false
	newBlock.SetNonce(nonce - 1)
	assert.False(t, miner.requirement(newBlock))
}

func TestMiner_ValidateDifficulty(t *testing.T) {

	miner := NewMiner()
	miner.SetRequirement(newTargetRequirement(defaultTargetBits))

	//create a block that has a hash value larger than the target
	blk := core.GenerateMockBlock()
	target := big.NewInt(1)
	target.Lsh(target, uint(256-defaultTargetBits+1))

	blk.SetHash(target.Bytes())

	assert.False(t, miner.requirement(blk))

	//create a block that has a hash value smaller than the target
	target = big.NewInt(1)
	target.Lsh(target, uint(256-defaultTargetBits-1))
	blk.SetHash(target.Bytes())

	assert.True(t, miner.requirement(blk))
}

func TestMiner_Start(t *testing.T) {
	miner := NewMiner()
	miner.SetRequirement(newTargetRequirement(defaultTargetBits))
	cbAddr := "1FoupuhmPN4q1wiUrM5QaYZjYKKLLXzPPg"
	keystr := "ac0a17dd3025b433ca0307d227241430ff4dda4be5e01a6c6cc6d2ccfaec895b"
	bc := core.CreateBlockchain(
		core.NewAddress(cbAddr),
		storage.NewRamStorage(),
		nil,
		128,
		nil,
	)
	retCh := make(chan *NewBlock, 0)
	miner.Setup(bc, cbAddr, retCh)
	miner.SetPrivateKey(keystr)
	miner.Start()
	blk := <-retCh
	assert.True(t, blk.IsValid)
	assert.True(t, blk.VerifyHash())
	assert.True(t, miner.requirement(blk.Block))
}
