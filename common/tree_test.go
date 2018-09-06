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
	"testing"
	"github.com/stretchr/testify/assert"
	"strings"
	"math/rand"
	"time"
)

func setupIntTree() *Tree{
	tree:= NewTree(0,0)
	parent := tree.Root
	//add 100000 nodes unto the tree
	for i:=1;i<100000;i++  {

		newNode := Node{[]Entry{Entry{i,i}},parent,nil, parent.Height+1, tree}
		parent.Children = append(parent.Children, &newNode)
		//if is true, create a new branch, else build existing branch
		if(getBool()){
			parent = &newNode
			tree.MaxHeight++
		}
	}
	return tree
}


func setupAlphabetTree() *Tree{
	alphabets:= "abcdefghijklmnopqrstuvwxyz"
	alphabetSlice := strings.Split(alphabets, "")
	tree:= NewTree("a","a")
	parent := tree.Root
	//add 26 nodes unto the tree
	for i:=1;i< len(alphabetSlice);i++  {
		newNode,_ := tree.NewNode(alphabetSlice[i], alphabetSlice[i])
		parent.AddChild(newNode)
		if(getBool()){
			parent = newNode
		}
	}
	return tree
}
func Test_RecursiveFind(t *testing.T){
	tree := setupIntTree()
	//run find lots of times
	for i:=50000; i<50500;i++ {
		tree.Get(tree.Root, i)
		assert.Equal(t, i, tree.Found.Entries[0].value)
	}
}

func Test_SearchParentNodeAndAddChild(t *testing.T){
	tree := setupIntTree()
	//add child {asd:asd} to 90000 block
	tree.SearchParentNodeAndAddChild(
		tree.Root,90000, "asd", "asd")

	tree.Get(tree.Root, "asd")
	assert.Equal(t, "asd", tree.Found.Entries[0].value)
	assert.Equal(t, 90000, tree.Found.Parent.Entries[0].value)

}

func Test_TreeHeightAndGetNodesAfterAppendTree(t *testing.T){
	t1 := setupIntTree()
	t1heightB4Merge := t1.MaxHeight

	mergeHeight := int(t1.MaxHeight) - 10

	t2 := setupAlphabetTree()

	t1.appendTree(t2, mergeHeight)

	//test addition of new nodes from t2
	t1.Get(t1.Root, "b")
	assert.Equal(t, "b", t1.Found.Entries[0].value )
	t1.Get(t1.Root, "y")
	assert.Equal(t, "y", t1.Found.Entries[0].value )

	//test height after merging t2
	t2heightAfterMerge := uint(t1.Found.Height)+t2.MaxHeight
	if t2heightAfterMerge > t1heightB4Merge{
		assert.Equal(t, t1.MaxHeight, t2heightAfterMerge)
	}
	assert.Equal(t, t1.MaxHeight, t1heightB4Merge)

}

func Test_TreeLeafs(t *testing.T){
	tree:=setupAlphabetTree()
	//cached leaf nodes should not have any children
	for _,v := range tree.leafs.Keys(){
		val, _ := tree.leafs.Get(v)
		assert.Equal(t, 0, len(val.(*Node).Children))
	}
}

func getBool() bool {
	rand.Seed(time.Now().UnixNano())
	return rand.Intn(10) >= 5
}


