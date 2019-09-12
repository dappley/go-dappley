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

type TreeNode struct {
	key      interface{}
	value    interface{}
	Parent   *TreeNode
	Children []*TreeNode
}

//NewTreeNode creates a new tree node
func NewTreeNode(key interface{}, value interface{}) (*TreeNode, error) {
	if key == nil || value == nil {
		return nil, ErrCantCreateEmptyNode
	}
	return &TreeNode{key, value, nil, nil}, nil
}

//GetRoot returns the root of current tree node
func (t *TreeNode) GetRoot() *TreeNode {
	root := t
	parent := t.Parent
	for parent != nil {
		root = parent
		parent = parent.Parent
	}
	return root
}

//GetParentTreesRange returns all Treenodes between head -> current node
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

//FindHeightestChild find the deepest leaf
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

//AddChild adds a child to the tree node
func (t *TreeNode) AddChild(child *TreeNode) {
	t.Children = append(t.Children, child)
	child.Parent = t
}

//AddParent sets parent of the tree node
func (t *TreeNode) AddParent(parent *TreeNode) error {
	if t.Parent != nil {
		return ErrNodeAlreadyHasParent
	}
	parent.AddChild(t)
	return nil
}

//GetValue returns the value of current node
func (t *TreeNode) GetValue() interface{} {
	return t.value
}

//GetKey returns the key of the current node
func (t *TreeNode) GetKey() interface{} {
	return t.key
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

//hasChildren returns if current node has any children
func (t *TreeNode) hasChildren() bool {
	if len(t.Children) > 0 {
		return true
	}
	return false
}

//containChild returns if the input node is a child of current node
func (t *TreeNode) containChild(child *TreeNode) bool {
	for _, c := range t.Children {
		if c == child {
			return true
		}
	}
	return false
}
