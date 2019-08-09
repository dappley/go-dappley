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
	"github.com/dappley/go-dappley/common/hash"
	"github.com/dappley/go-dappley/core/block"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dappley/go-dappley/common"
)

func TestLRUCacheWithIntKeyAndValue(t *testing.T) {
	bp := NewBlockPool()
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

func TestBlockPool_ForkHeadRange(t *testing.T) {
	bp := NewBlockPool()

	parent := block.NewBlockWithRawInfo(hash.Hash("parent"), []byte{0}, 0, 0, 1, nil)
	blk := block.NewBlockWithRawInfo(hash.Hash("blk"), parent.GetHash(), 0, 0, 2, nil)
	child := block.NewBlockWithRawInfo(hash.Hash("child"), blk.GetHash(), 0, 0, 3, nil)

	// cache a blk
	bp.Add(blk)
	readBlk, isFound := bp.blkCache.Get(blk.GetHash().String())
	require.Equal(t, blk, readBlk.(*common.Tree).GetValue().(*block.Block))
	require.True(t, isFound)
	require.ElementsMatch(t, []string{blk.GetHash().String()}, testGetForkHeadHashes(bp))

	// attach child to blk
	bp.Add(child)
	require.ElementsMatch(t, []string{blk.GetHash().String()}, testGetForkHeadHashes(bp))

	// attach parent to blk
	bp.Add(parent)
	require.ElementsMatch(t, []string{parent.GetHash().String()}, testGetForkHeadHashes(bp))

	// cache extraneous block
	unrelatedBlk := block.NewBlockWithRawInfo(hash.Hash("unrelated"), []byte{0}, 0, 0, 1, nil)
	bp.Add(unrelatedBlk)
	require.ElementsMatch(t, []string{parent.GetHash().String(), unrelatedBlk.GetHash().String()}, testGetForkHeadHashes(bp))

	// remove parent
	bp.RemoveFork([]*block.Block{parent})
	require.ElementsMatch(t, []string{unrelatedBlk.GetHash().String()}, testGetForkHeadHashes(bp))

	// remove unrelated
	bp.RemoveFork([]*block.Block{unrelatedBlk})
	require.Nil(t, testGetForkHeadHashes(bp))
}

func testGetForkHeadHashes(bp *BlockPool) []string {
	var hashes []string
	bp.ForkHeadRange(func(blkHash string, tree *common.Tree) {
		hashes = append(hashes, blkHash)
	})
	return hashes
}
