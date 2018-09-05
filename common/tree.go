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
	"errors"
	logger "github.com/sirupsen/logrus"
	"github.com/hashicorp/golang-lru"
)

type Entry struct{
	key interface{}
	value interface{}
}

var (
	ErrNodeNotFound = errors.New("ERROR: Node not found in tree")
	ErrCantCreateEmptyNode = errors.New("ERROR: Node index and value must not be empty")
)
const LeafsSize = 32

//entries include the node's entry itself as the first entry and its childrens' entry following
type Node struct {
	Entries []Entry
	Parent *Node
	Children []*Node
	Height uint
	tree *Tree
}

type Tree struct {
	Root *Node
	MaxHeight uint
	Found *Node
	Searching bool
	leafs *lru.Cache
}
type Test struct {
	Num uint
}


func (n *Node) hasChildren() bool{
	if len(n.Children) > 0 {
		return true
	}
	return false
}

func (t *Tree) NewNode(index interface{}, value interface{}) (*Node, error){
	if index == nil || value == nil {
		return nil, ErrCantCreateEmptyNode
	}
	return &Node{[]Entry{Entry{index,value}}, nil, nil, 1, t}, nil
}

func NewTree(rootNodeIndex interface{}, rootNodeValue interface{}) *Tree{
	t := &Tree{nil, 0 , nil, false, nil}
	r := Node{[]Entry{Entry{rootNodeIndex,rootNodeValue}}, nil, nil, 1, t}
	t.Root = &r
	t.leafs,_ = lru.New(LeafsSize)
	return t
}


func (t *Tree) RecursiveFind (parent *Node, index interface{}) {
	if !parent.hasChildren() || t.Searching == false{
		return
	}

	for i:=0;i< len(parent.Children);i++  {
		if parent.Children[i].Entries[0].key == index{
			logger.Debug("found! ", index, " under ", parent.Entries[0].key)
			t.Searching = false
			t.Found = parent.Children[i]
		}else{
			if t.Searching {
				t.RecursiveFind(parent.Children[i], index)
			}
		}
	}
}

//Search from root, use if you have no closer known nodes upstream
func (t *Tree) Get(parent *Node, index interface{}){
	t.Searching = true
	if t.Root.Entries[0].key == index{
		logger.Debug("found! ", index, ", is root")
		t.Found = t.Root
		return
	}
	t.RecursiveFind(parent, index)
}

func (t *Tree) SearchParentNodeAndAddChild( startNode *Node, parentIndex interface{} , childIndex interface{}, childValue interface{}){
	child,_ := t.NewNode(childIndex, childValue)
	t.Get(t.Root, parentIndex)
	parent := t.Found
	parent.AddChild(child)
}

func (parent *Node) AddChild(child *Node){
	parent.Children = append(parent.Children, child)
	parent.Entries = append(parent.Entries, child.Entries[0])
	child.Parent = parent
	//remove index from leafs if was leaf
	if len(parent.Children) == 1 && parent.tree.leafs.Contains(parent.Entries[0].key){
		parent.tree.leafs.Remove(parent.Entries[0].key)
	}
	parent.tree.leafs.Add(child.Entries[0].key, child)
}

//attach a tree's root node to a specific node of another tree through node index
func (t *Tree) appendTree(tree *Tree, mergeIndex interface{}) {

	logger.Debug("index search: ", mergeIndex, t.MaxHeight)
	t.Get(t.Root, mergeIndex)
	t.Found.AddChild(tree.Root)
	t.setHeightPostMerge(tree, mergeIndex)
}

func (t *Tree) setHeightPostMerge(tree *Tree, mergeIndex interface{}) {
	//if is new tree is higher than original tree after appending
	if tree.MaxHeight + t.Found.Height > t.MaxHeight{
		t.MaxHeight = tree.MaxHeight + t.Found.Height
	}
}