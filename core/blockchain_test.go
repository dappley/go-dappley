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

func TestBlockchain_FindHeightInBlockchain(t *testing.T) {
	//create a new block chain
	s := storage.NewRamStorage()
	addr := NewAddress("16PencPNnF8CiSx2EBGEd1axhf7vuHCouj")
	bc,err := CreateBlockchain(addr, s)
	assert.Nil(t, err)

	blk = GenerateMockBlock()
	blk.SetHash([]byte("hash1"))
	blk.height = 1
	bc.UpdateNewBlock(blk)

	height, isFound := bc.FindHeightInBlockchain([]byte("hash1"))
	assert.Equal(t,blk.height,height)
	assert.True(t,isFound)

	_, isFound = bc.FindHeightInBlockchain([]byte("hash2"))
	assert.False(t,isFound)
}