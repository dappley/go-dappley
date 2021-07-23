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
	errorValues "github.com/dappley/go-dappley/errors"
)

type TreeNode struct {
	value    interface{}
	Parent   *TreeNode
	Children []*TreeNode
}

//NewTreeNode creates a new tree node
func NewTreeNode(value interface{}) (*TreeNode, error) {
	if value == nil {
		return nil, errorValues.ErrCantCreateEmptyNode
	}
	return &TreeNode{value, nil, nil}, nil
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

//GetLongestPath returns the path from the deepest leaf node to the current node
func (t *TreeNode) GetLongestPath() []*TreeNode {
	if t == nil {
		return nil
	}

	if !t.hasChildren() {
		return []*TreeNode{t}
	}

	longest := 0
	var path []*TreeNode
	for _, child := range t.Children {
		currentPath := child.GetLongestPath()
		if len(currentPath) > longest {
			path = currentPath
			longest = len(currentPath)
		}
	}
	return append(path, t)
}

//Prune removes all nodes that are not current node's descendants, and call onDeleteCallback function when a node is deleted
func (t *TreeNode) Prune(onDeleteCallbackFn func(node *TreeNode)) {
	if t == nil {
		return
	}

	if t.Parent == nil {
		return
	}

	for _, sibling := range t.Parent.Children {
		if sibling == nil {
			continue
		}

		if sibling.GetValue() == t.GetValue() {
			continue
		}

		sibling.Parent = nil
		onDeleteCallbackFn(sibling)
		sibling.RemoveAllDescendants(onDeleteCallbackFn)
	}

	t.Parent.Children = nil
	t.Parent.Prune(onDeleteCallbackFn)
	onDeleteCallbackFn(t.Parent)
	t.Parent = nil
}

//RemoveAllDescendants remove all descendants of the current node, and call onDeleteCallback function when a node is deleted
func (t *TreeNode) RemoveAllDescendants(onDeleteCallback func(node *TreeNode)) {

	if t == nil {
		return
	}
	for _, child := range t.Children {
		child.Parent = nil
		onDeleteCallback(child)
		child.RemoveAllDescendants(onDeleteCallback)
	}
	t.Children = nil
}

//AddChild adds a child to the tree node
func (t *TreeNode) AddChild(child *TreeNode) {
	child.Parent = t
	for _, c := range t.Children {
		if child.value == c.value {
			return
		}
	}
	t.Children = append(t.Children, child)
}

//SetParent sets parent of the tree node
func (t *TreeNode) SetParent(parent *TreeNode) error {
	if t.Parent != nil {
		return errorValues.ErrNodeAlreadyHasParent
	}
	parent.AddChild(t)
	return nil
}

//GetValue returns the value of current node
func (t *TreeNode) GetValue() interface{} {
	return t.value
}

// NumLeaves returns the number of leaves in the tree t
func (t *TreeNode) NumLeaves() int64 {
	if t == nil {
		return 0
	}
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
	if t == nil {
		return 0
	}
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
