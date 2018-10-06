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
}

func NewNode(index interface{}, value interface{}, height uint64) (*Node, error) {
	if index == nil || value == nil {
		return nil, ErrCantCreateEmptyNode
	}
	return &Node{Entry{index, value}, nil, nil, height}, nil
}

func (n *Node) hasChildren() bool {
	if len(n.Children) > 0 {
		return true
	}
	return false
}

func (n *Node) containChild(child *Node) bool {
	for _, c := range n.Children {
		if c == child {
			return true
		}
	}
	return false
}

func (parent *Node) AddChild(child *Node) {
	parent.Children = append(parent.Children, child)
	child.Parent = parent
	child.Height = parent.Height + 1
}

func (child *Node) AddParent(parent *Node) error {
	if child.Parent != nil {
		return ErrChildNodeAlreadyHasParent
	}
	parent.AddChild(child)
	return nil
}

func (n *Node) GetValue() interface{} {
	return n.entry.value
}

func (n *Node) GetKey() interface{} {
	return n.entry.key
}
