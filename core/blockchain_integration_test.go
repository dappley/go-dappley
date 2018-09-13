// +build integration

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

package core

import (
	"testing"
	"github.com/stretchr/testify/assert"
)

func TestBlockchain_MergeForkCoinbaseTxOnly(t *testing.T) {
	//mock a blockchain and a fork whose parent is the tail of the blockchain
	bc := GenerateMockBlockchainWithCoinbaseTxOnly(5)
	defer bc.db.Close()
	blk,err:= bc.GetTailBlock()
	assert.Nil(t, err)

	//find the hash at height 3 (5-2)
	for i:=0; i<2; i++{
		blk,err = bc.GetBlockByHash(blk.GetPrevHash())
		assert.Nil(t,err)
	}

	//generate a fork that is forked from height 3
	bc.SetBlockPool(GenerateBlockPoolWithFakeFork(5,blk))

	//get the last fork hash
	forkTailBlockHash := bc.GetBlockPool().GetForkPool()[0].GetHash()

	bc.MergeFork()

	//the highest block should have the height of 8 -> 3+5
	assert.Equal(t, uint64(8), bc.GetMaxHeight())
	tailBlkHash := bc.GetTailBlockHash()
	assert.ElementsMatch(t,forkTailBlockHash,tailBlkHash)

}

func TestBlockchain_MergeForkInvalidTransaction(t *testing.T) {
	//mock a blockchain and a fork whose parent is the tail of the blockchain
	bc := GenerateMockBlockchainWithCoinbaseTxOnly(5)
	defer bc.db.Close()
	blk,err:= bc.GetTailBlock()
	assert.Nil(t, err)

	//find the hash at height 3 (5-2)
	for i:=0; i<2; i++{
		blk,err = bc.GetBlockByHash(blk.GetPrevHash())
		assert.Nil(t,err)
	}

	tailBlkHash := bc.GetTailBlockHash()

	//generate a fork that is forked from height 3
	bc.SetBlockPool(GenerateBlockPoolwithFakeForkWithInvalidTx(5,blk))

	//the merge should fail since the transactions are invalid
	bc.MergeFork()

	//the highest block should have the height of 5
	assert.Equal(t, uint64(5), bc.GetMaxHeight())
	newTailBlkHash := bc.GetTailBlockHash()
	assert.ElementsMatch(t,tailBlkHash,newTailBlkHash)
}
