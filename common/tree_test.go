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

package common

import (
	"math/rand"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func setupIntTree() (*Tree, int) {
	tree := NewTree(0, 0)
	parent := tree.Root
	len := 10000
	//add 10000 nodes unto the tree
	for i := 1; i < len; i++ {
		newNode := Node{[]Entry{Entry{i, i}}, parent, nil, parent.Height + 1, tree}
		parent.Children = append(parent.Children, &newNode)
		//if is true, create a new branch, else build existing branch
		if getBool() {
			parent = &newNode
			tree.MaxHeight++
		}
	}
	return tree, len
}

func setupAlphabetTree() (*Tree, int) {
	alphabets := "abcdefghijklmnopqrstuvwxyz"
	alphabetSlice := strings.Split(alphabets, "")
	tree := NewTree("a", "a")
	parent := tree.Root
	//add 26 nodes unto the tree
	for i := 1; i < len(alphabetSlice); i++ {
		newNode, _ := tree.NewNode(alphabetSlice[i], alphabetSlice[i], parent.Height+1)
		parent.AddChild(newNode)
		if getBool() {
			parent = newNode
		}
	}
	return tree, len(alphabetSlice)
}

func Test_RecursiveFind(t *testing.T) {
	tree, _ := setupIntTree()
	//run find lots of times
	for i := 5000; i < 5050; i++ {
		tree.Get(tree.Root, i)
		assert.Equal(t, i, tree.Found.Entries[0].value)
	}
}

func Test_TreeLeafs(t *testing.T) {
	//logger.SetLevel(logger.DebugLevel)
	tree, _ := setupAlphabetTree()
	//cached leaf nodes should not have any children
	for _, v := range tree.leafs.Keys() {
		val, _ := tree.leafs.Get(v)
		assert.Equal(t, 0, len(val.(*Node).Children))
		assert.Equal(t, true, val.(*Node).Height != 1)
	}
}

func Test_TreeHighestLeaf(t *testing.T) {
	//logger.SetLevel(logger.DebugLevel)
	tree, _ := setupAlphabetTree()
	//cached leaf nodes should not have any children
	assert.Equal(t, tree.MaxHeight, tree.HighestLeaf.Height)
}

func Test_AddParent(t *testing.T) {
	tree, _ := setupAlphabetTree()
	nodeToAdd, _ := tree.NewNode("asd", "asd", 0)
	//check root case
	child := tree.Root
	child.AddParent(nodeToAdd)
	assert.Equal(t, nodeToAdd.Entries[0].key, tree.Root.Entries[0].key)

	//check invalid case
	tree.Get(tree.Root, "u")
	nodeToAdd2, _ := tree.NewNode(11, 11, 0)
	err := tree.Found.AddParent(nodeToAdd2)
	assert.Equal(t, err, ErrChildNodeAlreadyHasParent)

}

func getBool() bool {
	rand.Seed(time.Now().UnixNano())
	return rand.Intn(10) >= 5
}
