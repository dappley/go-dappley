// Copyright (C) 2018 go-dappley authors
//
// This file is part of the go-dappley library.
//
// the go-dappley library is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either pubKeyHash 3 of the License, or
// (at your option) any later pubKeyHash.
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

	"github.com/dappley/go-dappley/common"
	"github.com/dappley/go-dappley/storage"
)

func TestLRUCacheWithIntKeyAndValue(t *testing.T) {
	bp := NewBlockPool(5)
	assert.Equal(t, 0, bp.blkCache.Len())
	const addCount = 200
	for i := 0; i < addCount; i++ {
		if bp.blkCache.Len() == ForkCacheLRUCacheLimit {
			bp.blkCache.RemoveOldest()
		}
		bp.blkCache.Add(i, i)
	}
	//test blkCache is full
	assert.Equal(t, ForkCacheLRUCacheLimit, bp.blkCache.Len())
	//test blkCache contains last added key
	assert.Equal(t, true, bp.blkCache.Contains(199))
	//test blkCache oldest key = addcount - BlockPoolLRUCacheLimit
	assert.Equal(t, addCount-ForkCacheLRUCacheLimit, bp.blkCache.Keys()[0])
}

func TestBlockPool_NumForks(t *testing.T) {
	bc := CreateBlockchain(NewAddress(""), storage.NewRamStorage(), nil, 100, nil, 100)
	blk, err := bc.GetTailBlock()
	assert.Nil(t, err)
	b1 := &Block{header: &BlockHeader{hash: []byte{1}, height: 1, prevHash: blk.GetHash()}}
	b3 := &Block{header: &BlockHeader{hash: []byte{3}, height: 2, prevHash: b1.GetHash()}}
	b6 := &Block{header: &BlockHeader{hash: []byte{6}, height: 3, prevHash: b3.GetHash()}}

	err = bc.AddBlockContextToTail(&BlockContext{Block: b1, UtxoIndex: NewUTXOIndex(nil), State: NewScState()})
	assert.Nil(t, err)
	err = bc.AddBlockContextToTail(&BlockContext{Block: b3, UtxoIndex: NewUTXOIndex(nil), State: NewScState()})
	assert.Nil(t, err)
	err = bc.AddBlockContextToTail(&BlockContext{Block: b6, UtxoIndex: NewUTXOIndex(nil), State: NewScState()})
	assert.Nil(t, err)

	b2 := &Block{header: &BlockHeader{hash: []byte{2}, height: 2, prevHash: b1.GetHash()}}
	b4 := &Block{header: &BlockHeader{hash: []byte{4}, height: 3, prevHash: b2.GetHash()}}
	b5 := &Block{header: &BlockHeader{hash: []byte{5}, height: 3, prevHash: b2.GetHash()}}
	b7 := &Block{header: &BlockHeader{hash: []byte{7}, height: 4, prevHash: b4.GetHash()}}

	/*
		              b1
		            b2  b3
		          b4 b5  b6
		        b7
			BlockChain:  Genesis - b1 - b3 - b6
	*/

	t2, _ := common.NewTree(b2.GetHash().String(), b2)
	t4, _ := common.NewTree(b4.GetHash().String(), b4)
	t5, _ := common.NewTree(b5.GetHash().String(), b5)
	t7, _ := common.NewTree(b7.GetHash().String(), b7)

	bp := NewBlockPool(10)
	bp.CacheBlock(t2, 0) // maxHeight 0 to ensure it caches
	bp.CacheBlock(t4, 0)
	bp.CacheBlock(t5, 0)
	bp.CacheBlock(t7, 0)

	// adding block that is not connected to BlockChain should be ignored
	b8 := &Block{header: &BlockHeader{hash: []byte{8}, height: 4, prevHash: []byte{9}}}
	t8, _ := common.NewTree(b8.GetHash().String(), b8)
	bp.CacheBlock(t8, 0)

	numForks, longestFork := bp.NumForks(bc)
	assert.EqualValues(t, 2, numForks)
	assert.EqualValues(t, 3, longestFork)

	// create a new fork off b6
	b9 := &Block{header: &BlockHeader{hash: []byte{9}, height: 4, prevHash: b6.GetHash()}}
	t9, _ := common.NewTree(b9.GetHash().String(), b9)
	bp.CacheBlock(t9, 0)

	numForks, longestFork = bp.NumForks(bc)
	assert.EqualValues(t, 3, numForks)
	assert.EqualValues(t, 3, longestFork)
}
