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

package blockchain

import (
	"reflect"
	"strconv"
	"testing"

	"github.com/dappley/go-dappley/common/hash"
	"github.com/dappley/go-dappley/core/block"
	logger "github.com/sirupsen/logrus"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dappley/go-dappley/common"
)

func TestLRUCacheWithIntKeyAndValue(t *testing.T) {
	bp := NewBlockPool(nil)
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
	bp := NewBlockPool(nil)

	parent := block.NewBlockWithRawInfo(hash.Hash("parent"), []byte{0}, 0, 0, 1, nil)
	blk := block.NewBlockWithRawInfo(hash.Hash("blk"), parent.GetHash(), 0, 0, 2, nil)
	child := block.NewBlockWithRawInfo(hash.Hash("child"), blk.GetHash(), 0, 0, 3, nil)

	// cache a blk
	bp.AddBlock(blk)
	readBlk, isFound := bp.blkCache.Get(blk.GetHash().String())
	require.Equal(t, blk, readBlk.(*common.TreeNode).GetValue().(*block.Block))
	require.True(t, isFound)
	require.ElementsMatch(t, []string{blk.GetHash().String()}, testGetForkHeadHashes(bp))

	// attach child to blk
	bp.AddBlock(child)
	require.ElementsMatch(t, []string{blk.GetHash().String()}, testGetForkHeadHashes(bp))

	// attach parent to blk
	bp.AddBlock(parent)
	require.ElementsMatch(t, []string{parent.GetHash().String()}, testGetForkHeadHashes(bp))

	// cache extraneous block
	unrelatedBlk := block.NewBlockWithRawInfo(hash.Hash("unrelated"), []byte{0}, 0, 0, 1, nil)
	bp.AddBlock(unrelatedBlk)
	require.ElementsMatch(t, []string{parent.GetHash().String(), unrelatedBlk.GetHash().String()}, testGetForkHeadHashes(bp))

	// remove parent
	bp.RemoveFork([]*block.Block{parent})
	require.ElementsMatch(t, []string{unrelatedBlk.GetHash().String()}, testGetForkHeadHashes(bp))

	// remove unrelated
	bp.RemoveFork([]*block.Block{unrelatedBlk})
	require.Nil(t, testGetForkHeadHashes(bp))
}

func TestBlockPool_GetForkHead(t *testing.T) {
	bp := NewBlockPool(nil)

	parent := block.NewBlockWithRawInfo(hash.Hash("parent"), []byte{0}, 0, 0, 1, nil)
	blk := block.NewBlockWithRawInfo(hash.Hash("blk"), parent.GetHash(), 0, 0, 2, nil)
	child := block.NewBlockWithRawInfo(hash.Hash("child"), blk.GetHash(), 0, 0, 3, nil)

	bp.AddBlock(blk)
	bp.AddBlock(child)
	assert.Equal(t, blk, bp.GetForkHead(child))
	bp.AddBlock(parent)
	assert.Equal(t, parent, bp.GetForkHead(blk))

	unrelatedBlk := block.NewBlockWithRawInfo(hash.Hash("unrelated"), []byte{0}, 0, 0, 1, nil)

	bp.AddBlock(unrelatedBlk)
	assert.Equal(t, unrelatedBlk, bp.GetForkHead(unrelatedBlk))

	nonexistentBlk := block.NewBlockWithRawInfo(hash.Hash("nonexistent"), []byte{0}, 0, 0, 1, nil)
	assert.Nil(t, bp.GetForkHead(nonexistentBlk))
}

func TestBlockPool_pruneOrphans(t *testing.T) {

	parent := block.NewBlockWithRawInfo(hash.Hash("parent"), []byte{0}, 0, 0, 1, nil)
	blk1 := block.NewBlockWithRawInfo(hash.Hash("blk"), parent.GetHash(), 0, 0, 3, nil)
	blk2 := block.NewBlockWithRawInfo(hash.Hash("child"), parent.GetHash(), 0, 0, 5, nil)

	bp := NewBlockPool(parent)
	bp.AddBlock(blk1)

	// manually add unlinked child blk
	blk2Node, _ := common.NewTreeNode(blk2)
	bp.blkCache.Add(getKey(blk2Node), blk2Node)
	bp.orphans[getKey(blk2Node)] = blk2Node

	unrelatedBlk := block.NewBlockWithRawInfo(hash.Hash("unrelated"), []byte{0}, 0, 0, 3, nil)
	bp.AddBlock(unrelatedBlk)

	blk1NodeInterface, _ := bp.blkCache.Get(blk1.GetHash().String())
	blk1Node := blk1NodeInterface.(*common.TreeNode)

	unrelatedNodeInterface, _ := bp.blkCache.Get(unrelatedBlk.GetHash().String())
	unrelatedNode := unrelatedNodeInterface.(*common.TreeNode)

	beforePrune := map[string]*common.TreeNode{
		blk2.GetHash().String():         blk2Node,
		unrelatedBlk.GetHash().String(): unrelatedNode,
	}
	assert.Equal(t, beforePrune, bp.orphans)

	bp.pruneOrphans()
	// unrelatedBlk still has valid height so it is not pruned
	assert.Equal(t, []*common.TreeNode{blk1Node, blk2Node}, bp.root.Children)
	assert.Equal(t, map[string]*common.TreeNode{unrelatedBlk.GetHash().String(): unrelatedNode}, bp.orphans)

	// replace unrelatedBlk with a block that has invalid height
	replacementBlk := block.NewBlockWithRawInfo(hash.Hash("unrelated"), []byte{0}, 0, 0, 1, nil)
	replacementNode, _ := common.NewTreeNode(replacementBlk)
	*unrelatedNode = *replacementNode

	bp.pruneOrphans()
	assert.Equal(t, map[string]*common.TreeNode{}, bp.orphans)
}

func TestBlockPool_isBlockValid(t *testing.T) {
	tests := []struct {
		name     string
		rootBlk  *block.Block
		blk      *block.Block
		expected bool
	}{
		{
			"Empty Block",
			createBlock(hash.Hash("child"), nil, 0),
			nil,
			false,
		},
		{
			"No rootBlkNode",
			nil,
			createBlock(hash.Hash("child"), hash.Hash("parent"), 0),
			true,
		},
		{
			"rootBlk is parent and input block is 1 block higher",
			createBlock(hash.Hash("1"), hash.Hash("0"), 0),
			createBlock(hash.Hash("2"), hash.Hash("1"), 1),
			true,
		},
		{
			"rootBlk is not parent and input block is 1 block higher",
			createBlock(hash.Hash("1"), hash.Hash("0"), 0),
			createBlock(hash.Hash("2"), hash.Hash("3"), 1),
			false,
		},
		{
			"input block is more than 1 block higher than rootBlk",
			createBlock(hash.Hash("1"), hash.Hash("0"), 0),
			createBlock(hash.Hash("2"), hash.Hash("3"), 2),
			true,
		},
		{
			"input block is same height as rootBlk",
			createBlock(hash.Hash("1"), hash.Hash("0"), 0),
			createBlock(hash.Hash("2"), hash.Hash("3"), 2),
			true,
		},
		{
			"input block is lower than rootBlk",
			createBlock(hash.Hash("1"), hash.Hash("0"), 5),
			createBlock(hash.Hash("2"), hash.Hash("3"), 2),
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bp := NewBlockPool(tt.rootBlk)
			assert.Equal(t, tt.expected, bp.isBlockValid(tt.blk))
		})
	}
}

func TestBlockPool_removeTree(t *testing.T) {
	/*  BLOCK FORK STRUCTURE
	MAIN FORK:		     1           ORPHANS:(3 orphan forks)
				    2        3
				  8  9     4                              15
				10	     5 6 7              11          16
	                                      12  13					17
	                                            14
	*/

	tests := []struct {
		name                   string
		serializedBp           string
		rootBlkHash            string
		treeRoot               string
		expectedNumOfNodesLeft int
	}{
		{
			"Remove from main fork",
			"0^1, 1#2, 1#3, 3#4, 4#5, 4#6, 4#7, 2#8, 2#9, 8#10, 3^11, 11#12, 11#13, 13#14, 2^15, 15#16, 4^17",
			"1",
			"3",
			12,
		},
		{
			"Remove from orphan 1",
			"0^1, 1#2, 1#3, 3#4, 4#5, 4#6, 4#7, 2#8, 2#9, 8#10, 3^11, 11#12, 11#13, 13#14, 2^15, 15#16, 4^17",
			"1",
			"13",
			15,
		},
		{
			"Remove all from orphan 2",
			"0^1, 1#2, 1#3, 3#4, 4#5, 4#6, 4#7, 2#8, 2#9, 8#10, 3^11, 11#12, 11#13, 13#14, 2^15, 15#16, 4^17",
			"1",
			"15",
			15,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bp, _ := deserializeBlockPool(tt.serializedBp, tt.rootBlkHash)
			node, ok := bp.blkCache.Get(hash.Hash(tt.treeRoot).String())
			assert.True(t, ok)
			bp.removeTree(node.(*common.TreeNode))
			assert.Equal(t, tt.expectedNumOfNodesLeft, bp.blkCache.Len())
		})
	}
}

func TestBlockPool_removeNode(t *testing.T) {
	/*  BLOCK FORK STRUCTURE
	MAIN FORK:		     1           ORPHANS:(3 orphan forks)
				    2        3
				  8  9     4                              15
				10	     5 6 7              11          16
	                                      12  13					17
	                                            14
	*/

	tests := []struct {
		name                   string
		serializedBp           string
		rootBlkHash            string
		treeRoot               string
		expectedNumOfNodesLeft int
	}{
		{
			"Remove from main fork",
			"0^1, 1#2, 1#3, 3#4, 4#5, 4#6, 4#7, 2#8, 2#9, 8#10, 3^11, 11#12, 11#13, 13#14, 2^15, 15#16, 4^17",
			"1",
			"3",
			16,
		},
		{
			"Remove from orphan 1",
			"0^1, 1#2, 1#3, 3#4, 4#5, 4#6, 4#7, 2#8, 2#9, 8#10, 3^11, 11#12, 11#13, 13#14, 2^15, 15#16, 4^17",
			"1",
			"13",
			16,
		},
		{
			"Remove from orphan 2",
			"0^1, 1#2, 1#3, 3#4, 4#5, 4#6, 4#7, 2#8, 2#9, 8#10, 3^11, 11#12, 11#13, 13#14, 2^15, 15#16, 4^17",
			"1",
			"15",
			16,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bp, _ := deserializeBlockPool(tt.serializedBp, tt.rootBlkHash)
			node, ok := bp.blkCache.Get(hash.Hash(tt.treeRoot).String())
			assert.True(t, ok)
			bp.removeNode(node.(*common.TreeNode))
			assert.Equal(t, tt.expectedNumOfNodesLeft, bp.blkCache.Len())
		})
	}
}

func TestBlockPool_GetFork(t *testing.T) {
	serializedBp := "0^1, 1#2, 1#3, 3#4, 4#5, 4#6, 4#7, 2#8, 2#9, 8#10, 3^11, 11#12, 11#13, 13#14, 2^15, 15#16, 4^17"
	bp, _ := deserializeBlockPool(serializedBp, "1")
	/*  Test Block Pool Structure
		Blkgheight 				MAIN FORK:		     	ORPHANS:(3 orphan forks)
	         0							1
			 1						2        3
			 2					  8  9     4                              15
			 3					10	     5 6 7              11          16
			 4											  12  13						17
			 5													14
	*/
	tests := []struct {
		name              string
		inputBlkHash      string
		expectedBlkHashes []string
	}{
		{
			name:              "GetFork from blk 1",
			inputBlkHash:      "1",
			expectedBlkHashes: []string{"10", "8", "2", "1"},
		},
		{
			name:              "GetFork from blk 3",
			inputBlkHash:      "3",
			expectedBlkHashes: []string{"5", "4", "3"},
		},
		{
			name:              "GetFork from blk 11",
			inputBlkHash:      "11",
			expectedBlkHashes: []string{"14", "13", "11"},
		},
		{
			name:              "GetFork from blk 16",
			inputBlkHash:      "16",
			expectedBlkHashes: []string{"16"},
		},
		{
			name:              "GetFork from nonexistent block",
			inputBlkHash:      "99",
			expectedBlkHashes: []string{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := bp.GetFork(hash.Hash(tt.inputBlkHash))
			if len(tt.expectedBlkHashes) == 0 {
				assert.Equal(t, 0, len(result))
			} else {
				for i, blk := range result {
					assert.Equal(t, hash.Hash(tt.expectedBlkHashes[i]).String(), blk.GetHash().String())
				}
			}
		})
	}
}

func TestBlockPool_findLongestChain(t *testing.T) {
	serializedBp := "0^1, 1#2, 1#3, 3#4, 4#5, 4#6, 4#7, 2#8, 2#9, 8#10, 3^11, 11#12, 11#13, 13#14, 2^15, 15#16, 4^17"
	bp, _ := deserializeBlockPool(serializedBp, "1")
	/*  Test Block Pool Structure
		Blkgheight 				MAIN FORK:		     	ORPHANS:(3 orphan forks)
	         0							1
			 1						2        3
			 2					  8  9     4                              15
			 3					10	     5 6 7              11          16
			 4											  12  13						17
			 5													14
	*/
	tests := []struct {
		name            string
		inputBlkHash    string
		expectedBlkHash string
	}{
		{
			name:            "main fork root",
			inputBlkHash:    "1",
			expectedBlkHash: "1",
		},
		{
			name:            "main fork left branch",
			inputBlkHash:    "2",
			expectedBlkHash: "2",
		},
		{
			name:            "orphan 1 right branch",
			inputBlkHash:    "13",
			expectedBlkHash: "13",
		},
		{
			name:            "orphan 2 leaf",
			inputBlkHash:    "16",
			expectedBlkHash: "16",
		},
		{
			name:            "orphan 3",
			inputBlkHash:    "17",
			expectedBlkHash: "17",
		},
		{
			name:            "invalid hash",
			inputBlkHash:    "invalid",
			expectedBlkHash: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := bp.findLongestChain(hash.Hash(tt.inputBlkHash))
			if tt.expectedBlkHash != "" {
				assert.Equal(t, hash.Hash(tt.expectedBlkHash).String(), result.GetValue().(*block.Block).GetHash().String())
			} else {
				assert.Nil(t, result)
			}
		})
	}
}

func TestBlockPool_SetRootBlock(t *testing.T) {
	/*  Test Block Pool Structure
		Blkgheight 				MAIN FORK:		     	ORPHANS:(3 orphan forks)
	         0							1
			 1						2        3
			 2					  8  9     4                              15
			 3					10	     5 6 7              11          16
			 4											  12  13						17
			 5													14
	*/
	tests := []struct {
		name                 string
		serializedBp         string
		rootBlkHash          string
		newRootBlkHash       string
		expectedNumOfNodes   int
		expectedNumOfOrphans int
	}{
		/*  Expected Result
		Blkgheight 				MAIN FORK:		     	ORPHANS:
			 0
			 1						       3
			 2					         4
			 3						   5 6 7              11
			 4											12  13						17
			 5												  14
		*/
		{
			"Set rootBlkHash to a descendant in main fork upper section",
			"0^1, 1#2, 1#3, 3#4, 4#5, 4#6, 4#7, 2#8, 2#9, 8#10, 3^11, 11#12, 11#13, 13#14, 2^15, 15#16, 4^17",
			"1",
			"3",
			10,
			2,
		},

		/*  Expected Result
		Blkgheight 				MAIN FORK:		     	ORPHANS:
			 0
			 1
			 2					           4
			 3						     5 6 7
			 4											    						17
			 5
		*/
		{
			"Set rootBlkHash to a descendant in main fork middle section",
			"0^1, 1#2, 1#3, 3#4, 4#5, 4#6, 4#7, 2#8, 2#9, 8#10, 3^11, 11#12, 11#13, 13#14, 2^15, 15#16, 4^17",
			"1",
			"4",
			5,
			1,
		},

		/*  Expected Result
		Blkgheight 				MAIN FORK:		     	ORPHANS:
			 0
			 1
			 2
			 3					10
			 4
			 5
		*/
		{
			"Set rootBlkHash to a descendant in main fork bottom section",
			"0^1, 1#2, 1#3, 3#4, 4#5, 4#6, 4#7, 2#8, 2#9, 8#10, 3^11, 11#12, 11#13, 13#14, 2^15, 15#16, 4^17",
			"1",
			"10",
			1,
			0,
		},

		/*  Expected Result
		Blkgheight 				MAIN FORK:		     	ORPHANS:(3 orphan forks)
			 0
			 1
			 2
			 3								                11
			 4											  12  13
			 5													14
		*/
		{
			"Set rootBlkHash to the root in orphan fork 1",
			"0^1, 1#2, 1#3, 3#4, 4#5, 4#6, 4#7, 2#8, 2#9, 8#10, 3^11, 11#12, 11#13, 13#14, 2^15, 15#16, 4^17",
			"1",
			"11",
			4,
			0,
		},

		/*  Expected Result
		Blkgheight 				MAIN FORK:		     	ORPHANS:(3 orphan forks)
			 0
			 1
			 2
			 3
			 4											      13
			 5													14
		*/
		{
			"Set rootBlkHash to a node in orphan fork 1",
			"0^1, 1#2, 1#3, 3#4, 4#5, 4#6, 4#7, 2#8, 2#9, 8#10, 3^11, 11#12, 11#13, 13#14, 2^15, 15#16, 4^17",
			"1",
			"13",
			2,
			0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bp, _ := deserializeBlockPool(tt.serializedBp, tt.rootBlkHash)
			node, ok := bp.blkCache.Get(hash.Hash(tt.newRootBlkHash).String())
			assert.True(t, ok)
			bp.SetRootBlock(node.(*common.TreeNode).GetValue().(*block.Block))
			assert.Equal(t, tt.expectedNumOfNodes, bp.blkCache.Len())
			assert.Equal(t, node, bp.root)
		})
	}
}

func TestBlockPool_link(t *testing.T) {
	root, _ := common.NewTreeNode(createBlock(hash.Hash("root"), []byte{0}, 1))
	orphan1, _ := common.NewTreeNode(createBlock(hash.Hash("orphan1"), hash.Hash("root"), 3))
	orphan2, _ := common.NewTreeNode(createBlock(hash.Hash("orphan2"), hash.Hash("root"), 5))
	orphan3, _ := common.NewTreeNode(createBlock(hash.Hash("orphan3"), hash.Hash("nonexistent"), 7))

	bp := NewBlockPool(nil)
	bp.orphans[orphan1.GetValue().(*block.Block).GetHash().String()] = orphan1
	bp.orphans[orphan2.GetValue().(*block.Block).GetHash().String()] = orphan2
	bp.orphans[orphan3.GetValue().(*block.Block).GetHash().String()] = orphan3

	expectedOrphans := map[string]*common.TreeNode{
		root.GetValue().(*block.Block).GetHash().String():    root,
		orphan3.GetValue().(*block.Block).GetHash().String(): orphan3,
	}
	bp.link(root)
	assert.Equal(t, expectedOrphans, bp.orphans)
	assert.True(t, reflect.DeepEqual([]*common.TreeNode{orphan1, orphan2}, root.Children))
	assert.Equal(t, root, orphan1.Parent)
	assert.Equal(t, root, orphan2.Parent)
	assert.Nil(t, orphan3.Parent)
}

func TestBlockPool_linkOrphan(t *testing.T) {
	root, _ := common.NewTreeNode(createBlock(hash.Hash("root"), []byte{0}, 1))
	orphan1, _ := common.NewTreeNode(createBlock(hash.Hash("orphan1"), hash.Hash("root"), 3))
	orphan2, _ := common.NewTreeNode(createBlock(hash.Hash("orphan2"), hash.Hash("root"), 5))
	orphan3, _ := common.NewTreeNode(createBlock(hash.Hash("orphan3"), hash.Hash("nonexistent"), 7))

	bp := NewBlockPool(nil)
	bp.orphans[orphan1.GetValue().(*block.Block).GetHash().String()] = orphan1
	bp.orphans[orphan2.GetValue().(*block.Block).GetHash().String()] = orphan2
	bp.orphans[orphan3.GetValue().(*block.Block).GetHash().String()] = orphan3

	expectedOrphans := map[string]*common.TreeNode{
		orphan3.GetValue().(*block.Block).GetHash().String(): orphan3,
	}
	bp.linkOrphan(root)
	assert.Equal(t, expectedOrphans, bp.orphans)
	assert.True(t, reflect.DeepEqual([]*common.TreeNode{orphan1, orphan2}, root.Children))
}

func TestBlockPool_linkParent(t *testing.T) {
	parent, _ := common.NewTreeNode(createBlock(hash.Hash("parent"), []byte{0}, 1))
	node, _ := common.NewTreeNode(createBlock(hash.Hash("node"), hash.Hash("parent"), 3))
	orphan, _ := common.NewTreeNode(createBlock(hash.Hash("orphan"), hash.Hash("nonexistent"), 3))

	bp := NewBlockPool(nil)
	bp.blkCache.Add(getKey(parent), parent)

	bp.linkParent(node)
	assert.Equal(t, parent, node.Parent)
	assert.Equal(t, map[string]*common.TreeNode{}, bp.orphans)

	expectedOrphans := map[string]*common.TreeNode{
		orphan.GetValue().(*block.Block).GetHash().String(): orphan,
	}
	bp.linkParent(orphan)
	assert.Nil(t, orphan.Parent)
	assert.Equal(t, expectedOrphans, bp.orphans)
}

func TestBlockPool_GetRootBlk(t *testing.T) {
	root, _ := common.NewTreeNode(createBlock(hash.Hash("root"), []byte{0}, 1))

	bp := NewBlockPool(nil)
	assert.Nil(t, bp.getRootBlk())

	bp.SetRootBlock(root.GetValue().(*block.Block))
	assert.Equal(t, root.GetValue().(*block.Block), bp.getRootBlk())
}

func TestBlockPool_GetBlocksFromTrees(t *testing.T) {
	treeSlice := make([]*common.TreeNode, 3)
	treeSlice[0], _ = common.NewTreeNode(createBlock(hash.Hash("node1"), []byte{0}, 1))
	treeSlice[1], _ = common.NewTreeNode(createBlock(hash.Hash("node2"), hash.Hash("node1"), 3))
	treeSlice[2], _ = common.NewTreeNode(createBlock(hash.Hash("node3"), hash.Hash("node2"), 5))

	blockSlice := getBlocksFromTrees(treeSlice)

	assert.Equal(t, treeSlice[0].GetValue().(*block.Block), blockSlice[0])
	assert.Equal(t, treeSlice[1].GetValue().(*block.Block), blockSlice[1])
	assert.Equal(t, treeSlice[2].GetValue().(*block.Block), blockSlice[2])
}

func testGetForkHeadHashes(bp *BlockPool) []string {
	var hashes []string
	bp.ForkHeadRange(func(blkHash string, tree *common.TreeNode) {
		hashes = append(hashes, blkHash)
	})
	return hashes
}

func createBlock(currentHash hash.Hash, prevHash hash.Hash, height uint64) *block.Block {
	return block.NewBlockWithRawInfo(currentHash, prevHash, 0, 0, height, nil)
}

//deserializeBlockPool creates a block pool by deserializing the input string. return the root of the tree
func deserializeBlockPool(s string, rootBlkHash string) (*BlockPool, map[string]*block.Block) {
	/* "0^1, 1#2, 1#3, 3#4, 0^5, 1^6" describes a block pool like following"
				1      5
			   2 3			6
	              4
	*/
	if s == "" {
		return NewBlockPool(nil), nil
	}

	s += ","

	rootBlk := createBlock(hash.Hash(rootBlkHash), nil, 0)
	bp := NewBlockPool(rootBlk)

	var parentBlk *block.Block
	currStr := ""
	blkHeight := 0
	blocks := make(map[string]*block.Block)
	blocks[hash.Hash(rootBlkHash).String()] = rootBlk

	for _, c := range s {
		switch c {
		case ',':

			if currStr == rootBlkHash {
				currStr = ""
				continue
			}

			var blk *block.Block
			if parentBlk == nil {
				blk = createBlock(hash.Hash(currStr), nil, uint64(blkHeight))
			} else {
				blk = createBlock(hash.Hash(currStr), parentBlk.GetHash(), parentBlk.GetHeight()+1)
			}
			bp.AddBlock(blk)
			blocks[hash.Hash(currStr).String()] = blk
			if parentBlk == nil {
				logger.WithFields(logger.Fields{
					"hash": hash.Hash(currStr).String(),
				}).Debug("Add a new head block")
			} else {
				logger.WithFields(logger.Fields{
					"hash":   hash.Hash(currStr).String(),
					"parent": parentBlk.GetHash().String(),
				}).Debug("Add a new block")
			}
			currStr = ""
			parentBlk = nil
			blkHeight = 0
		case '#':
			if _, isFound := blocks[hash.Hash(currStr).String()]; !isFound {
				logger.WithFields(logger.Fields{
					"hash": hash.Hash(currStr).String(),
				}).Panic("deserialize tree failed: the parent node is not found")
			}
			parentBlk = blocks[hash.Hash(currStr).String()]
			currStr = ""
		case '^':
			num, err := strconv.Atoi(currStr)
			if err != nil {
				logger.WithError(err).Panic("deserialize block pool failed while converting string to int")
			}
			blkHeight = num
			currStr = ""
		case ' ':
			continue
		default:
			currStr = currStr + string(c)
		}
	}

	return bp, blocks
}

func TestGetKey(t *testing.T) {
	node1, _ := common.NewTreeNode(createBlock(hash.Hash("hello"), hash.Hash("world"), 0))
	node2, _ := common.NewTreeNode(createBlock(hash.Hash("123"), hash.Hash("456"), 0))
	node3, _ := common.NewTreeNode(createBlock(nil, nil, 0))
	node4, _ := common.NewTreeNode(createBlock(hash.Hash(nil), hash.Hash(nil), 0))

	assert.Equal(t, "68656c6c6f", getKey(node1))
	assert.Equal(t, "313233", getKey(node2))
	assert.Equal(t, "", getKey(node3))
	assert.Equal(t, "", getKey(node4))
	assert.Equal(t, "", getKey(nil))
}
