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
)

func setupIntTree() *Tree{
	tree:= NewTree(0,0)
	parent := tree.Root
	//add 100000 nodes unto the tree
	for i:=1;i<100000;i++  {
		newNode := node{[]Entry{Entry{i,i}},parent,nil, parent.Height+1}
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
		newNode := node{[]Entry{Entry{alphabetSlice[i],alphabetSlice[i]}},parent,nil, parent.Height+1}
		parent.Children = append(parent.Children, &newNode)
		//if is true, create a new branch, else build existing branch
		if(getBool()){
			parent = &newNode
			tree.MaxHeight++
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

func Test_appendTree(t *testing.T){

	mergeIndex := 99980
	mergeHeight := mergeIndex+1 //0 based index offset

	t1 := setupIntTree()
	t2 := setupAlphabetTree()
	t1.appendTree(t2, mergeIndex)

	t1.Get(t1.Root, "b")
	assert.Equal(t, "b", t1.Found.Entries[0].value )
	t1.Get(t1.Root, "y")
	assert.Equal(t, "y", t1.Found.Entries[0].value )


	assert.Equal(t, uint(mergeHeight)+t2.MaxHeight , t1.MaxHeight)
}

func getBool() bool {
	//rand.Seed(time.Now().UnixNano())
	//return rand.Intn(10) >= 5
	return true
}


