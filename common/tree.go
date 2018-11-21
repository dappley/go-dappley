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

type Entry struct {
	key   interface{}
	value interface{}
}

var (
	ErrNodeNotFound              = errors.New("ERROR: Node not found in tree")
	ErrCantCreateEmptyNode       = errors.New("ERROR: Node index and value must not be empty")
	ErrChildNodeAlreadyHasParent = errors.New("ERROR: Adding parent to node already with parent")
)

//entries include the node's entry itself as the first entry and its childrens' entry following
type Tree struct {
	entry    Entry
	Parent   *Tree
	Children []*Tree
}

func NewTree(index interface{}, value interface{}) (*Tree, error) {
	if index == nil || value == nil {
		return nil, ErrCantCreateEmptyNode
	}
	return &Tree{Entry{index, value}, nil, nil}, nil
}

func (t *Tree) hasChildren() bool {
	if len(t.Children) > 0 {
		return true
	}
	return false
}

func (t *Tree) containChild(child *Tree) bool {
	for _, c := range t.Children {
		if c == child {
			return true
		}
	}
	return false
}

func (t *Tree) Delete() {
	if t.Parent != nil {
		for i := 0; i < len(t.Parent.Children); i++ {
			if t.Parent.Children[i].GetKey() == t.GetKey() {
				t.Parent.Children = append(t.Parent.Children[:i], t.Parent.Children[i+1:]...)
			}
		}
	}
	t.Parent = nil
}

func (t *Tree) GetParentTreesRange(head *Tree) []*Tree {
	var parentTrees []*Tree
	parentTrees = append(parentTrees, t)
	if t.GetKey() == head.GetKey() { //fork of length 1
		return parentTrees
	}
	if t.Parent != nil && head != nil {
		for parent := t.Parent; parent.GetKey() != head.GetKey(); parent = parent.Parent {
			parentTrees = append(parentTrees, parent)
		}
	} else {
		logger.Error("Fork tail or head is empty!")
		return nil
	}
	parentTrees = append(parentTrees, head)
	return parentTrees
}

func (t *Tree) FindHeightestChild(path *Tree, prevDeep, deepest int) (deep int, deepPath *Tree) {
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

func (t *Tree) AddChild(child *Tree) {
	t.Children = append(t.Children, child)
	child.Parent = t
}

func (t *Tree) AddParent(parent *Tree) error {
	if t.Parent != nil {
		return ErrChildNodeAlreadyHasParent
	}
	parent.AddChild(t)
	return nil
}

func (t *Tree) GetValue() interface{} {
	return t.entry.value
}

func (t *Tree) GetKey() interface{} {
	return t.entry.key
}
