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
)

var (
	ErrCantCreateEmptyNode  = errors.New("tree: node index and value must not be empty")
	ErrNodeAlreadyHasParent = errors.New("tree: node already has a parent")
)

type Entry struct {
	key   interface{}
	value interface{}
}

//entries include the node's entry itself as the first entry and its childrens' entry following
type TreeNode struct {
	entry    Entry
	Parent   *TreeNode
	Children []*TreeNode
}

func NewTreeNode(index interface{}, value interface{}) (*TreeNode, error) {
	if index == nil || value == nil {
		return nil, ErrCantCreateEmptyNode
	}
	return &TreeNode{Entry{index, value}, nil, nil}, nil
}

func (t *TreeNode) hasChildren() bool {
	if len(t.Children) > 0 {
		return true
	}
	return false
}

func (t *TreeNode) containChild(child *TreeNode) bool {
	for _, c := range t.Children {
		if c == child {
			return true
		}
	}
	return false
}

func (t *TreeNode) Delete() {
	if t.Parent != nil {
		for i := 0; i < len(t.Parent.Children); i++ {
			if t.Parent.Children[i].GetKey() == t.GetKey() {
				t.Parent.Children = append(t.Parent.Children[:i], t.Parent.Children[i+1:]...)
			}
		}
	}
	t.Parent = nil
}

func (t *TreeNode) GetRoot() *TreeNode {
	root := t
	parent := t.Parent
	for parent != nil {
		root = parent
		parent = parent.Parent
	}
	return root
}

func (t *TreeNode) GetParentTreesRange(head *TreeNode) []*TreeNode {
	var parentTrees []*TreeNode
	parentTrees = append(parentTrees, t)
	if t.GetKey() == head.GetKey() { //fork of length 1
		return parentTrees
	}
	if t.Parent != nil && head != nil {
		for parent := t.Parent; parent.GetKey() != head.GetKey(); parent = parent.Parent {
			parentTrees = append(parentTrees, parent)
		}
	} else {
		logger.Error("TreeNode: fork tail or head is empty!")
		return nil
	}
	parentTrees = append(parentTrees, head)
	return parentTrees
}

func (t *TreeNode) FindHeightestChild(path *TreeNode, prevDeep, deepest int) (deep int, deepPath *TreeNode) {
	if t.hasChildren() {
		for _, child := range t.Children {
			correntDeepest, correntPath := child.FindHeightestChild(path, prevDeep+1, deepest)
			if correntDeepest > deepest {
				path = correntPath
				deepest = correntDeepest
			}
		}
	} else {
		path = t
		deepest = prevDeep
	}
	return deepest, path
}

func (t *TreeNode) AddChild(child *TreeNode) {
	t.Children = append(t.Children, child)
	child.Parent = t
}

func (t *TreeNode) AddParent(parent *TreeNode) error {
	if t.Parent != nil {
		return ErrNodeAlreadyHasParent
	}
	parent.AddChild(t)
	return nil
}

func (t *TreeNode) GetValue() interface{} {
	return t.entry.value
}

func (t *TreeNode) GetKey() interface{} {
	return t.entry.key
}

// NumLeaves returns the number of leaves in the tree t
func (t *TreeNode) NumLeaves() int64 {
	if !t.hasChildren() {
		return 1
	}
	var numLeaves int64 = 0
	for _, child := range t.Children {
		numLeaves += child.NumLeaves()
	}

	return numLeaves
}

// Size returns the number of nodes in the tree
func (t *TreeNode) Size() int64 {
	var size int64 = 1
	for _, child := range t.Children {
		size += child.Size()
	}

	return size
}

// Height returns the length of the deepest path counting nodes not edges
func (t *TreeNode) Height() int64 {
	var length int64 = 0
	for _, child := range t.Children {
		t := child.Height()
		if t > length {
			length = t
		}
	}

	return length + 1
}
