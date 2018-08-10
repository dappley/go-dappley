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
	bc := CreateBlockchain(addr,db,nil)

	bp := NewBlockPool(10)
	bp.SetBlockchain(bc)
	blk1 := GenerateMockBlock()
	bp.addParentToForkPool(blk1)

	assert.Equal(t,blk1,bp.forkPool[0])
}

func TestBlockPool_AddParentToForkPool(t *testing.T) {
	db := storage.NewRamStorage()
	defer db.Close()
	addr := Address{"17DgRtQVvaytkiKAfXx9XbV23MESASSwUz"}
	bc := CreateBlockchain(addr,db,nil)


	bp := NewBlockPool(10)
	bp.SetBlockchain(bc)
	blk1 := GenerateMockBlock()
	blk2 := GenerateMockBlock()
	bp.forkPool = append(bp.forkPool, blk1)
	bp.addParentToForkPool(blk2)

	assert.Equal(t,blk2,bp.forkPool[1])
}

func TestBlockPool_AddTailToForkPoolWhenEmpty(t *testing.T) {
	db := storage.NewRamStorage()
	defer db.Close()
	addr := Address{"17DgRtQVvaytkiKAfXx9XbV23MESASSwUz"}
	bc:= CreateBlockchain(addr,db,nil)


	bp := NewBlockPool(10)
	bp.SetBlockchain(bc)
	blk1 := GenerateMockBlock()
	bp.addTailToForkPool(blk1)

	assert.Equal(t,blk1,bp.forkPool[0])
}

func TestBlockPool_AddTailToForkPool(t *testing.T) {
	db := storage.NewRamStorage()
	defer db.Close()
	addr := Address{"17DgRtQVvaytkiKAfXx9XbV23MESASSwUz"}
	bc:= CreateBlockchain(addr,db,nil)

	bp := NewBlockPool(10)
	bp.SetBlockchain(bc)
	blk1 := GenerateMockBlock()
	blk2 := GenerateMockBlock()
	bp.forkPool = append(bp.forkPool, blk1)
	bp.addTailToForkPool(blk2)

	assert.Equal(t,blk2,bp.forkPool[0])
}

func TestBlockPool_ForkPoolLen(t *testing.T) {
	db := storage.NewRamStorage()
	defer db.Close()
	addr := Address{"17DgRtQVvaytkiKAfXx9XbV23MESASSwUz"}
	bc:= CreateBlockchain(addr,db,nil)

	bp := NewBlockPool(10)
	bp.SetBlockchain(bc)
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
	bc:= CreateBlockchain(addr,db,nil)

	bp := NewBlockPool(10)
	bp.SetBlockchain(bc)
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
	bc:= CreateBlockchain(addr,db,nil)

	bp := NewBlockPool(10)
	bp.SetBlockchain(bc)
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
	bc:= CreateBlockchain(addr,db,nil)

	bp := NewBlockPool(10)
	bp.SetBlockchain(bc)
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
	bc:= CreateBlockchain(addr,db,nil)

	bp := NewBlockPool(10)
	bp.SetBlockchain(bc)
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

func TestBlockPool_UpdateForkFromHeadLowerHeight(t *testing.T) {
	bc := GenerateMockBlockchain(5)
	pool := NewBlockPool(5)
	pool.SetBlockchain(bc)

	blk := NewBlock(nil, nil)
	blk.SetHash(blk.CalculateHash())
	blk2 := NewBlock(nil, blk)
	pool.forkPool = append(pool.forkPool, blk2)
	//this will be successful since blk is blk2's parent
	assert.True(t, pool.AddParentToFork(blk))
	//however the fork should be empty since blk2's height is lower than the current blockchain
	assert.Empty(t, pool.forkPool)
	//this will be failed since blk is not blk's parent
	assert.False(t, pool.AddParentToFork(blk))

}

func TestBlockPool_UpdateForkFromHeadHigherHeight(t *testing.T) {
	bc := GenerateMockBlockchain(5)
	pool := NewBlockPool(5)
	pool.SetBlockchain(bc)

	blk := NewBlock(nil,nil)
	blk.SetHash(blk.CalculateHash())
	blk2 := NewBlock(nil,blk)
	blk2.height = 8
	blk.height = 7
	pool.forkPool = append(pool.forkPool, blk2)
	//this will be successful since blk is blk2's parent
	assert.True(t, pool.AddParentToFork(blk))
	//however the fork should not be empty since blk2's height is higher than the current blockchain
	assert.NotEmpty(t, pool.forkPool)
}

func TestBlockPool_UpdateForkFromTailLowerHeight(t *testing.T) {
	bc := GenerateMockBlockchain(5)
	pool := NewBlockPool(5)
	pool.SetBlockchain(bc)

	blk := NewBlock(nil, nil)
	blk.SetHash(blk.CalculateHash())
	blk2 := NewBlock(nil, blk)
	pool.forkPool = append(pool.forkPool, blk)
	//this will be successful since blk is blk2's parent
	assert.True(t, pool.UpdateForkFromTail(blk2))
	//however the fork should be empty since blk2's height is lower than the current blockchain
	assert.Empty(t, pool.forkPool)
	//this will be failed since blk2 is not blk2's parent
	assert.False(t, pool.UpdateForkFromTail(blk2))

}

func TestBlockPool_UpdateForkFromTailHigherHeight(t *testing.T) {
	bc := GenerateMockBlockchain(5)
	pool := NewBlockPool(5)
	pool.SetBlockchain(bc)

	blk := NewBlock(nil,nil)
	blk.SetHash(blk.CalculateHash())
	blk2 := NewBlock(nil,blk)
	blk2.height = 8
	blk.height = 7
	pool.forkPool = append(pool.forkPool, blk)
	//this will be successful since blk is blk2's parent
	assert.True(t, pool.UpdateForkFromTail(blk2))
	//however the fork should not be empty since blk2's height is higher than the current blockchain
	assert.NotEmpty(t, pool.forkPool)
}

func TestBlockPool_IsHigherThanForkSameHeight(t *testing.T) {
	pool := NewBlockPool(5)
	blk := NewBlock(nil,nil)
	blk.height = 5
	pool.forkPool = append(pool.forkPool, blk)

	blk2 := NewBlock(nil,nil)
	blk2.height = 5

	assert.False(t, pool.IsHigherThanFork(blk2))
}

func TestBlockPool_IsHigherThanForkHigherHeight(t *testing.T) {
	pool := NewBlockPool(5)
	blk := NewBlock(nil,nil)
	blk.height = 5
	pool.forkPool = append(pool.forkPool, blk)

	blk2 := NewBlock(nil,nil)
	blk2.height = 6

	assert.True(t, pool.IsHigherThanFork(blk2))
}

func TestBlockPool_IsHigherThanForkLowerHeight(t *testing.T) {
	pool := NewBlockPool(5)
	blk := NewBlock(nil,nil)
	blk.height = 5
	pool.forkPool = append(pool.forkPool, blk)

	blk2 := NewBlock(nil,nil)
	blk2.height = 4

	assert.False(t, pool.IsHigherThanFork(blk2))
}

func TestBlockPool_IsHigherThanForkNilInput(t *testing.T) {
	pool := NewBlockPool(5)
	assert.False(t, pool.IsHigherThanFork(nil))
}

func TestBlockPool_IsHigherThanForkEmptyPool(t *testing.T) {
	pool := NewBlockPool(5)
	blk := NewBlock(nil,nil)
	assert.True(t, pool.IsHigherThanFork(blk))
}

func TestBlockPool_ReInitializeForkPool(t *testing.T) {
	pool := NewBlockPool(5)
	blk := NewBlock(nil,nil)
	blk.height = 5
	pool.forkPool = append(pool.forkPool, blk)

	pool.ResetForkPool()

	assert.Empty(t,pool.forkPool)
}