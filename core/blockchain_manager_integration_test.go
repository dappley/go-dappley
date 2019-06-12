// +build integration

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
	b1 := &Block{header: &BlockHeader{height: 1, prevHash: blk.GetHash(), nonce: 1}}
	b1.header.hash = b1.CalculateHash()
	b3 := &Block{header: &BlockHeader{height: 2, prevHash: b1.GetHash(), nonce: 3}}
	b3.header.hash = b3.CalculateHash()
	b6 := &Block{header: &BlockHeader{height: 3, prevHash: b3.GetHash(), nonce: 6}}
	b6.header.hash = b6.CalculateHash()

	err = bc.AddBlockContextToTail(&BlockContext{Block: b1, UtxoIndex: NewUTXOIndex(nil), State: NewScState()})
	assert.Nil(t, err)
	err = bc.AddBlockContextToTail(&BlockContext{Block: b3, UtxoIndex: NewUTXOIndex(nil), State: NewScState()})
	assert.Nil(t, err)
	err = bc.AddBlockContextToTail(&BlockContext{Block: b6, UtxoIndex: NewUTXOIndex(nil), State: NewScState()})
	assert.Nil(t, err)

	// create first fork of height 3
	b2 := &Block{header: &BlockHeader{height: 2, prevHash: b1.GetHash(), nonce: 2}}
	b2.header.hash = b2.CalculateHash()
	b4 := &Block{header: &BlockHeader{height: 3, prevHash: b2.GetHash(), nonce: 4}}
	b4.header.hash = b4.CalculateHash()
	b5 := &Block{header: &BlockHeader{height: 3, prevHash: b2.GetHash(), nonce: 5}}
	b5.header.hash = b5.CalculateHash()
	b7 := &Block{header: &BlockHeader{height: 4, prevHash: b4.GetHash(), nonce: 7}}
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

	bp.CacheBlock(b2, 0)
	assert.Equal(t, 1, bp.numForkHeads())
	bp.CacheBlock(b4, 0)
	assert.Equal(t, 1, bp.numForkHeads())
	bp.CacheBlock(b5, 0)
	assert.Equal(t, 1, bp.numForkHeads())
	bp.CacheBlock(b7, 0)
	assert.Equal(t, 1, bp.numForkHeads())

	// adding block that is not connected to BlockChain should be ignored
	b8 := &Block{header: &BlockHeader{height: 4, prevHash: []byte{9}, nonce: 8}}
	bp.CacheBlock(b8, 0)
	assert.Equal(t, 2, bp.numForkHeads())

	numForks, longestFork := bcm.NumForks()
	assert.EqualValues(t, 2, numForks)
	assert.EqualValues(t, 3, longestFork)

	// create a new fork off b6
	b9 := &Block{header: &BlockHeader{height: 4, prevHash: b6.GetHash(), nonce: 9}}
	b9.header.hash = b9.CalculateHash()
	bp.CacheBlock(b9, 0)
	assert.Equal(t, 3, bp.numForkHeads())

	bp.ForkHeadRange(func(blkHash string, tree *common.Tree) {
		assert.Contains(t, []string{b2.GetHash().String(), b8.GetHash().String(), b9.GetHash().String()}, blkHash)
	})

	numForks, longestFork = bcm.NumForks()
	assert.EqualValues(t, 3, numForks)
	assert.EqualValues(t, 3, longestFork)
}
