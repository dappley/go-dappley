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
	bc, err:= CreateBlockchain(addr,db)
	assert.Nil(t, err)
	hash1, err:= bc.GetTailHash()
	assert.Nil(t, err)
	newbc := bc.blockPool.GetBlockchain()

	hash2, err := newbc.GetTailHash()
	assert.Nil(t, err)
	assert.ElementsMatch(t,hash1, hash2)
}

func TestBlockPool_AddParentToForkPoolWhenEmpty(t *testing.T) {
	db := storage.NewRamStorage()
	defer db.Close()
	addr := Address{"17DgRtQVvaytkiKAfXx9XbV23MESASSwUz"}
	bc, err:= CreateBlockchain(addr,db)
	assert.Nil(t, err)

	bp := NewBlockPool(10, bc)
	blk1 := GenerateMockBlock()
	bp.AddParentToForkPool(blk1)

	assert.Equal(t,blk1,bp.forkPool[0])
}

func TestBlockPool_AddParentToForkPool(t *testing.T) {
	db := storage.NewRamStorage()
	defer db.Close()
	addr := Address{"17DgRtQVvaytkiKAfXx9XbV23MESASSwUz"}
	bc, err:= CreateBlockchain(addr,db)
	assert.Nil(t, err)

	bp := NewBlockPool(10, bc)
	blk1 := GenerateMockBlock()
	blk2 := GenerateMockBlock()
	bp.forkPool = append(bp.forkPool, blk1)
	bp.AddParentToForkPool(blk2)

	assert.Equal(t,blk2,bp.forkPool[1])
}

func TestBlockPool_AddTailToForkPoolWhenEmpty(t *testing.T) {
	db := storage.NewRamStorage()
	defer db.Close()
	addr := Address{"17DgRtQVvaytkiKAfXx9XbV23MESASSwUz"}
	bc, err:= CreateBlockchain(addr,db)
	assert.Nil(t, err)

	bp := NewBlockPool(10, bc)
	blk1 := GenerateMockBlock()
	bp.AddTailToForkPool(blk1)

	assert.Equal(t,blk1,bp.forkPool[0])
}

func TestBlockPool_AddTailToForkPool(t *testing.T) {
	db := storage.NewRamStorage()
	defer db.Close()
	addr := Address{"17DgRtQVvaytkiKAfXx9XbV23MESASSwUz"}
	bc, err:= CreateBlockchain(addr,db)
	assert.Nil(t, err)

	bp := NewBlockPool(10, bc)
	blk1 := GenerateMockBlock()
	blk2 := GenerateMockBlock()
	bp.forkPool = append(bp.forkPool, blk1)
	bp.AddTailToForkPool(blk2)

	assert.Equal(t,blk2,bp.forkPool[0])
}

func TestBlockPool_ForkPoolLen(t *testing.T) {
	db := storage.NewRamStorage()
	defer db.Close()
	addr := Address{"17DgRtQVvaytkiKAfXx9XbV23MESASSwUz"}
	bc, err:= CreateBlockchain(addr,db)
	assert.Nil(t, err)

	bp := NewBlockPool(10, bc)
	blk1 := GenerateMockBlock()
	blk2 := GenerateMockBlock()
	bp.forkPool = append(bp.forkPool, blk1)
	bp.forkPool = append(bp.forkPool, blk2)

	assert.Equal(t,2, bp.ForkPoolLen())
}

func TestBlockPool_GetForkPoolHeadBlk(t *testing.T) {
	db := storage.NewRamStorage()
	defer db.Close()
	addr := Address{"17DgRtQVvaytkiKAfXx9XbV23MESASSwUz"}
	bc, err:= CreateBlockchain(addr,db)
	assert.Nil(t, err)

	bp := NewBlockPool(10, bc)
	blk1 := GenerateMockBlock()
	blk2 := GenerateMockBlock()
	bp.forkPool = append(bp.forkPool, blk1)
	bp.forkPool = append(bp.forkPool, blk2)

	assert.Equal(t,blk2, bp.GetForkPoolHeadBlk())
}

func TestBlockPool_GetForkPoolTailBlk(t *testing.T) {
	db := storage.NewRamStorage()
	defer db.Close()
	addr := Address{"17DgRtQVvaytkiKAfXx9XbV23MESASSwUz"}
	bc, err:= CreateBlockchain(addr,db)
	assert.Nil(t, err)

	bp := NewBlockPool(10, bc)
	blk1 := GenerateMockBlock()
	blk2 := GenerateMockBlock()
	bp.forkPool = append(bp.forkPool, blk1)
	bp.forkPool = append(bp.forkPool, blk2)

	assert.Equal(t,blk1, bp.GetForkPoolTailBlk())
}

func TestBlockPool_IsParentOfFork(t *testing.T) {
	db := storage.NewRamStorage()
	defer db.Close()
	addr := Address{"17DgRtQVvaytkiKAfXx9XbV23MESASSwUz"}
	bc, err:= CreateBlockchain(addr,db)
	assert.Nil(t, err)

	bp := NewBlockPool(10, bc)
	blk1 := GenerateMockBlock()
	blk2 := GenerateMockBlock()
	bp.forkPool = append(bp.forkPool, blk1)
	bp.forkPool = append(bp.forkPool, blk2)

	blk3 := GenerateMockBlock()
	assert.False(t, bp.IsParentOfFork(blk3))

	blk3.SetHash(blk2.GetPrevHash())
	blk2.height = blk3.height + 1

	assert.True(t, bp.IsParentOfFork(blk3))
}

func TestBlockPool_IsTailOfFork(t *testing.T) {
	db := storage.NewRamStorage()
	defer db.Close()
	addr := Address{"17DgRtQVvaytkiKAfXx9XbV23MESASSwUz"}
	bc, err:= CreateBlockchain(addr,db)
	assert.Nil(t, err)

	bp := NewBlockPool(10, bc)
	blk1 := GenerateMockBlock()
	blk2 := GenerateMockBlock()
	bp.forkPool = append(bp.forkPool, blk1)
	bp.forkPool = append(bp.forkPool, blk2)

	blk3 := GenerateMockBlock()
	assert.False(t, bp.IsParentOfFork(blk3))

	blk1.SetHash(blk3.GetPrevHash())
	blk3.height = blk1.height + 1

	assert.True(t, bp.IsTailOfFork(blk3))
}