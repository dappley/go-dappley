package core

import (
	"github.com/dappley/go-dappley/core/block"
	"github.com/dappley/go-dappley/logic/block_logic"
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

	b1 := block.NewBlockWithRawInfo(nil, blk.GetHash(), 1, 0, 1, nil)
	b3 := block.NewBlockWithRawInfo(nil, b1.GetHash(), 3, 0, 2, nil)
	b3.SetHash(block_logic.CalculateHash(b3))
	b6 := block.NewBlockWithRawInfo(nil, b3.GetHash(), 6, 0, 3, nil)
	b6.SetHash(block_logic.CalculateHash(b6))

	err = bc.AddBlockContextToTail(&BlockContext{Block: b1, UtxoIndex: NewUTXOIndex(nil), State: NewScState()})
	require.Nil(t, err)
	err = bc.AddBlockContextToTail(&BlockContext{Block: b3, UtxoIndex: NewUTXOIndex(nil), State: NewScState()})
	require.Nil(t, err)
	err = bc.AddBlockContextToTail(&BlockContext{Block: b6, UtxoIndex: NewUTXOIndex(nil), State: NewScState()})
	require.Nil(t, err)

	// create first fork of height 3
	b2 := block.NewBlockWithRawInfo(nil, b1.GetHash(), 2, 0, 2, nil)
	b2.SetHash(block_logic.CalculateHash(b2))

	b4 := block.NewBlockWithRawInfo(nil, b2.GetHash(), 4, 0, 3, nil)
	b4.SetHash(block_logic.CalculateHash(b4))

	b5 := block.NewBlockWithRawInfo(nil, b2.GetHash(), 5, 0, 3, nil)
	b5.SetHash(block_logic.CalculateHash(b5))

	b7 := block.NewBlockWithRawInfo(nil, b4.GetHash(), 7, 0, 4, nil)
	b7.SetHash(block_logic.CalculateHash(b7))

	/*
		              b1
		            b2  b3
		          b4 b5  b6
		        b7
			BlockChain:  Genesis - b1 - b3 - b6
	*/

	bp := NewBlockPool()
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
	b8 := block.NewBlockWithRawInfo(nil, []byte{9}, 8, 0, 4, nil)
	bp.CacheBlock(b8, 0)
	require.Equal(t, 2, testGetNumForkHeads(bp))

	numForks, longestFork := bcm.NumForks()
	require.EqualValues(t, 2, numForks)
	require.EqualValues(t, 3, longestFork)

	// create a new fork off b6
	b9 := block.NewBlockWithRawInfo(nil, b6.GetHash(), 9, 0, 4, nil)
	b9.SetHash(block_logic.CalculateHash(b9))

	bp.CacheBlock(b9, 0)
	require.Equal(t, 3, testGetNumForkHeads(bp))

	require.ElementsMatch(t,
		[]string{b2.GetHash().String(), b8.GetHash().String(), b9.GetHash().String()}, testGetForkHeadHashes(bp))

	numForks, longestFork = bcm.NumForks()
	require.EqualValues(t, 3, numForks)
	require.EqualValues(t, 3, longestFork)
}
