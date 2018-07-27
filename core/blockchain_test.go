package core

import (
	"testing"
	"github.com/dappley/go-dappley/storage"
	"github.com/stretchr/testify/assert"
)

func TestCreateBlockchain(t *testing.T) {
	//create a new block chain
	s := storage.NewRamStorage()
	addr := NewAddress("16PencPNnF8CiSx2EBGEd1axhf7vuHCouj")
	bc,err := CreateBlockchain(addr, s)
	assert.Nil(t, err)

	//find next block. This block should be the genesis block and its prev hash should be empty
	blk,err := bc.Next()
	assert.Nil(t, err)
	assert.Empty(t, blk.GetPrevHash())
}

func TestBlockchain_HigherThanBlockchainTestHigher(t *testing.T) {
	//create a new block chain
	s := storage.NewRamStorage()
	addr := NewAddress("16PencPNnF8CiSx2EBGEd1axhf7vuHCouj")
	bc,err := CreateBlockchain(addr, s)
	assert.Nil(t, err)

	blk = GenerateMockBlock()
	blk.height = 1
	assert.True(t,bc.HigherThanBlockchain(blk))
}

func TestBlockchain_HigherThanBlockchainTestLower(t *testing.T) {
	//create a new block chain
	s := storage.NewRamStorage()
	addr := NewAddress("16PencPNnF8CiSx2EBGEd1axhf7vuHCouj")
	bc,err := CreateBlockchain(addr, s)
	assert.Nil(t, err)

	blk = GenerateMockBlock()
	blk.height = 1
	bc.UpdateNewBlock(blk)

	assert.False(t,bc.HigherThanBlockchain(blk))
}

func TestBlockchain_IsInBlockchain(t *testing.T) {
	//create a new block chain
	s := storage.NewRamStorage()
	addr := NewAddress("16PencPNnF8CiSx2EBGEd1axhf7vuHCouj")
	bc,err := CreateBlockchain(addr, s)
	assert.Nil(t, err)

	blk = GenerateMockBlock()
	blk.SetHash([]byte("hash1"))
	blk.height = 1
	bc.UpdateNewBlock(blk)

	isFound := bc.IsInBlockchain([]byte("hash1"))
	assert.True(t,isFound)

	isFound = bc.IsInBlockchain([]byte("hash2"))
	assert.False(t,isFound)
}

func TestBlockchain_RollbackToABlock(t *testing.T) {
	//create a mock blockchain with max height of 5
	bc := GenerateMockBlockchain(5)
	defer bc.DB.Close()

	blk,err := bc.GetTailBlock()
	assert.Nil(t,err)

	//find the hash at height 3 (5-2)
	for i:=0; i<2; i++{
		blk,err = bc.GetBlockByHash(blk.GetPrevHash())
		assert.Nil(t,err)
	}

	//rollback to height 3
	bc.RollbackToABlock(blk.GetHash())

	//the height 3 block should be the new tail block
	newTailBlk,err := bc.GetTailBlock()
	assert.Nil(t,err)
	assert.Equal(t,blk.GetHash(),newTailBlk.GetHash())

}