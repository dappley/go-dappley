package core

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/dappley/go-dappley/common"
	"github.com/dappley/go-dappley/storage"
)

func TestBlockChainManager_NumForks(t *testing.T) {
	// create BlockChain
	bc := CreateBlockchain(NewAddress(""), storage.NewRamStorage(), nil, 100, nil, 100)
	blk, err := bc.GetTailBlock()
	assert.Nil(t, err)
	b1 := &Block{header: &BlockHeader{height: 1, prevHash: blk.GetHash()}}
	b1.header.hash = b1.CalculateHash()
	b3 := &Block{header: &BlockHeader{height: 2, prevHash: b1.GetHash()}}
	b3.header.hash = b3.CalculateHash()
	b6 := &Block{header: &BlockHeader{height: 3, prevHash: b3.GetHash()}}
	b6.header.hash = b6.CalculateHash()

	err = bc.AddBlockContextToTail(&BlockContext{Block: b1, UtxoIndex: NewUTXOIndex(nil), State: NewScState()})
	assert.Nil(t, err)
	err = bc.AddBlockContextToTail(&BlockContext{Block: b3, UtxoIndex: NewUTXOIndex(nil), State: NewScState()})
	assert.Nil(t, err)
	err = bc.AddBlockContextToTail(&BlockContext{Block: b6, UtxoIndex: NewUTXOIndex(nil), State: NewScState()})
	assert.Nil(t, err)

	// create first fork of height 4
	b2 := &Block{header: &BlockHeader{height: 2, prevHash: b1.GetHash()}}
	b2.header.hash = b2.CalculateHash()
	b4 := &Block{header: &BlockHeader{height: 3, prevHash: b2.GetHash()}}
	b4.header.hash = b4.CalculateHash()
	b5 := &Block{header: &BlockHeader{height: 3, prevHash: b2.GetHash()}}
	b5.header.hash = b5.CalculateHash()
	b7 := &Block{header: &BlockHeader{height: 4, prevHash: b4.GetHash()}}
	b7.header.hash = b7.CalculateHash()

	/*
		              b1
		            b2  b3
		          b4 b5  b6
		        b7
			BlockChain:  Genesis - b1 - b3 - b6
	*/

	bcm := NewBlockChainManager()
	bcm.Setblockchain(bc)
	bp := NewBlockPool(100)
	bcm.SetblockPool(bp)

	t2, _ := common.NewTree(b2.GetHash().String(), b2)
	t4, _ := common.NewTree(b4.GetHash().String(), b4)
	t5, _ := common.NewTree(b5.GetHash().String(), b5)
	t7, _ := common.NewTree(b7.GetHash().String(), b7)

	t2.AddChild(t4)
	t2.AddChild(t5)
	t4.AddChild(t7)
	bp.CacheBlock(t2, 0)

	// adding block that is not connected to BlockChain should be ignored
	b8 := &Block{header: &BlockHeader{height: 4, prevHash: []byte{9}}}
	t8, _ := common.NewTree(b8.GetHash().String(), b8)
	bp.CacheBlock(t8, 0)

	numForks, longestFork := bcm.NumForks()
	assert.EqualValues(t, 2, numForks)
	assert.EqualValues(t, 3, longestFork)

	// create a new fork off b6
	b9 := &Block{header: &BlockHeader{height: 4, prevHash: b6.GetHash()}}
	b9.header.hash = b9.CalculateHash()
	t9, _ := common.NewTree(b9.GetHash().String(), b9)
	bp.CacheBlock(t9, 0)

	numForks, longestFork = bcm.NumForks()
	assert.EqualValues(t, 3, numForks)
	assert.EqualValues(t, 3, longestFork)
}
