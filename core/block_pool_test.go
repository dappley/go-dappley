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
	"github.com/dappley/go-dappley/storage"
)


func TestBlockPool_GetBlockchain(t *testing.T) {
	db := storage.NewRamStorage()
	defer db.Close()
	addr := Address{"17DgRtQVvaytkiKAfXx9XbV23MESASSwUz"}
	bc:= CreateBlockchain(addr,db,nil)

	hash1 := bc.GetTailBlockHash()
	newbc := bc.GetBlockPool().GetBlockchain()

	hash2 := newbc.GetTailBlockHash()
	assert.ElementsMatch(t,hash1, hash2)
}

func TestLRUCacheWithIntKeyAndValue(t *testing.T){
	bp:= NewBlockPool(5)
	assert.Equal(t, 0, bp.blkCache.Len())
	const addCount = 200
	for i:=0;i < addCount; i++ {
		if bp.blkCache.Len() == BlockPoolLRUCacheLimit{
			bp.blkCache.RemoveOldest()
		}
		bp.blkCache.Add(i, i )
	}
	//test blkCache is full
	assert.Equal(t, BlockPoolLRUCacheLimit, bp.blkCache.Len())
	//test blkCache contains last added key
	assert.Equal(t, true, bp.blkCache.Contains(199))
	//test blkCache oldest key = addcount - BlockPoolLRUCacheLimit
	assert.Equal(t, addCount - BlockPoolLRUCacheLimit, bp.blkCache.Keys()[0])
}


