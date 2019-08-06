package core

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/dappley/go-dappley/core/account"
	"github.com/dappley/go-dappley/storage"
)

func TestBlockChainManager_NumForks(t *testing.T) {
	// create BlockChain
	bc := CreateBlockchain(account.NewAddress(""), storage.NewRamStorage(), nil, NewTransactionPool(nil, 100), nil, 100)
	blk, err := bc.GetTailBlock()
	require.Nil(t, err)
	b1 := &Block{header: &BlockHeader{height: 1, prevHash: blk.GetHash(), nonce: 1}}
	b1.header.hash = b1.CalculateHash()
	b3 := &Block{header: &BlockHeader{height: 2, prevHash: b1.GetHash(), nonce: 3}}
	b3.header.hash = b3.CalculateHash()
	b6 := &Block{header: &BlockHeader{height: 3, prevHash: b3.GetHash(), nonce: 6}}
	b6.header.hash = b6.CalculateHash()

	err = bc.AddBlockContextToTail(&BlockContext{Block: b1, UtxoIndex: NewUTXOIndex(nil), State: NewScState()})
	require.Nil(t, err)
	err = bc.AddBlockContextToTail(&BlockContext{Block: b3, UtxoIndex: NewUTXOIndex(nil), State: NewScState()})
	require.Nil(t, err)
	err = bc.AddBlockContextToTail(&BlockContext{Block: b6, UtxoIndex: NewUTXOIndex(nil), State: NewScState()})
	require.Nil(t, err)

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

	bp := NewBlockPool(100)
	bcm := NewBlockChainManager(bc, bp, nil)

	bp.CacheBlock(b2, 0)
	require.Equal(t, 1, testGetNumForkHeads(bp))
	bp.CacheBlock(b4, 0)
	require.Equal(t, 1, testGetNumForkHeads(bp))
	bp.CacheBlock(b5, 0)
	require.Equal(t, 1, testGetNumForkHeads(bp))
	bp.CacheBlock(b7, 0)
	require.Equal(t, 1, testGetNumForkHeads(bp))

	// adding block that is not connected to BlockChain should be ignored
	b8 := &Block{header: &BlockHeader{height: 4, prevHash: []byte{9}, nonce: 8}}
	bp.CacheBlock(b8, 0)
	require.Equal(t, 2, testGetNumForkHeads(bp))

	numForks, longestFork := bcm.NumForks()
	require.EqualValues(t, 2, numForks)
	require.EqualValues(t, 3, longestFork)

	// create a new fork off b6
	b9 := &Block{header: &BlockHeader{height: 4, prevHash: b6.GetHash(), nonce: 9}}
	b9.header.hash = b9.CalculateHash()
	bp.CacheBlock(b9, 0)
	require.Equal(t, 3, testGetNumForkHeads(bp))

	require.ElementsMatch(t,
		[]string{b2.GetHash().String(), b8.GetHash().String(), b9.GetHash().String()}, testGetForkHeadHashes(bp))

	numForks, longestFork = bcm.NumForks()
	require.EqualValues(t, 3, numForks)
	require.EqualValues(t, 3, longestFork)
}
