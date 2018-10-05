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

	"github.com/hashicorp/golang-lru"
	logger "github.com/sirupsen/logrus"
)

type Entry struct {
	key   interface{}
	value interface{}
}

var (
	ErrNodeNotFound              = errors.New("ERROR: Node not found in tree")
	ErrCantCreateEmptyNode       = errors.New("ERROR: Node index and value must not be empty")
	ErrChildNodeAlreadyHasParent = errors.New("ERROR: Adding parent to node already with parent")
)

const LeafsSize = 32

//entries include the node's entry itself as the first entry and its childrens' entry following
type Node struct {
	entry    Entry
	Parent   *Node
	Children []*Node
	Height   uint64
	tree     *Tree
}

type Tree struct {
	Root        *Node
	MaxHeight   uint64
	Found       *Node
	Searching   bool
	leafs       *lru.Cache
	HighestLeaf *Node
}

func (n *Node) hasChildren() bool {
	if len(n.Children) > 0 {
		return true
	}
	return false
}

func (parent *Node) AddChild(child *Node) {

	parent.Children = append(parent.Children, child)
	child.Parent = parent
	//remove index from leafs if was leaf
	parentKey := parent.GetKey()
	child.Height = parent.Height + 1
	if child.Height > child.tree.MaxHeight {
		child.tree.MaxHeight = child.Height
		child.tree.HighestLeaf = child
	}
	leaves := parent.tree.leafs
	///if parent was leaf, update new leaf state
	if len(parent.Children) > 0 && leaves.Contains(parentKey) {
		leaves.Remove(parent.GetKey())
	}
	leaves.Add(child.GetKey(), child)
}

func (n *Node) GetValue() interface{} {
	return n.entry.value
}

func (n *Node) GetKey() interface{} {
	return n.entry.key
}

func (n *Node) AddParent(parent *Node) error {
	if n.Parent != nil {
		return ErrChildNodeAlreadyHasParent

	}
	n.tree.Root = parent
	parent.AddChild(n)
	return nil
}

//tree func

func NewTree(rootNodeIndex interface{}, rootNodeValue interface{}) *Tree {
	t := &Tree{nil, 1, nil, false, nil, nil}
	r := Node{Entry{rootNodeIndex, rootNodeValue}, nil, nil, 1, t}
	t.Root = &r
	t.leafs, _ = lru.New(LeafsSize)
	return t
}

func (t *Tree) NewNode(index interface{}, value interface{}, height uint64) (*Node, error) {
	if index == nil || value == nil {
		return nil, ErrCantCreateEmptyNode
	}
	return &Node{Entry{index, value}, nil, nil, height, t}, nil
}

func (t *Tree) RecursiveFind(parent *Node, index interface{}) {
	if !parent.hasChildren() || t.Searching == false {
		return
	}

	for i := 0; i < len(parent.Children); i++ {
		if parent.Children[i].GetKey() == index {
			logger.Debug("found! ", index, " under ", parent.GetKey())
			t.Searching = false
			t.Found = parent.Children[i]
		} else {
			if t.Searching {
				t.RecursiveFind(parent.Children[i], index)
			}
		}
	}
}

//Search from root, use if you have no closer known nodes upstream
func (t *Tree) Get(parent *Node, index interface{}) {
	t.Searching = true
	if t.Root.GetKey() == index {
		logger.Debug("found! ", index, ", is root")
		t.Found = t.Root
		return
	}
	t.RecursiveFind(parent, index)
}
