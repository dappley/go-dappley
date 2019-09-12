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
)

func Test_AddParent(t *testing.T) {
	parentNode1, _ := NewTreeNode("parent1", "parent1")
	parentNode2, _ := NewTreeNode("parent2", "parent2")
	childNode, _ := NewTreeNode("child2", "child2")

	err1 := childNode.SetParent(parentNode1)
	assert.Equal(t, nil, err1)
	err2 := childNode.SetParent(parentNode2)
	assert.Equal(t, ErrNodeAlreadyHasParent, err2)

	assert.Equal(t, parentNode1, childNode.Parent)
}

func Test_AddChild(t *testing.T) {
	parentNode, _ := NewTreeNode("parent", "parent")
	childNode1, _ := NewTreeNode("child1", "child1")
	childNode2, _ := NewTreeNode("child2", "child2")

	parentNode.AddChild(childNode1)
	parentNode.AddChild(childNode2)

	assert.Equal(t, parentNode, childNode1.Parent)
	assert.Equal(t, parentNode, childNode2.Parent)

	assert.Equal(t, 2, len(parentNode.Children))
	assert.True(t, parentNode.containChild(childNode1))
	assert.True(t, parentNode.containChild(childNode2))
}

func Test_HasChild(t *testing.T) {
	var nilTree TreeNode
	newTree, _ := NewTreeNode("new", "new")
	parentNode1, _ := NewTreeNode("parent1", "parent1")
	parentNode2, _ := NewTreeNode("parent2", "parent2")
	childNode1, _ := NewTreeNode("child1", "child1")
	childNode2, _ := NewTreeNode("child2", "child2")
	parentNode1.AddChild(childNode1)
	childNode2.SetParent(parentNode2)

	assert.Nil(t, nilTree.Children)
	assert.True(t, parentNode1.hasChildren())
	assert.True(t, parentNode2.hasChildren())
	assert.False(t, newTree.hasChildren())
	assert.False(t, nilTree.hasChildren())
}

func TestTreeNode_GetLongestPath(t *testing.T) {

}

func TestTree_Size(t *testing.T) {
	t0, _ := NewTreeNode("t0", "t0")
	t1, _ := NewTreeNode("t1", "t1")
	t2, _ := NewTreeNode("t2", "t2")
	t3, _ := NewTreeNode("t3", "t3")
	assert.EqualValues(t, 1, t1.Size())
	t1.AddChild(t0)
	assert.EqualValues(t, 2, t1.Size())
	t1.AddChild(t2)
	assert.EqualValues(t, 3, t1.Size())
	t2.AddChild(t3)
	assert.EqualValues(t, 4, t1.Size())
}

func TestTree_Height(t *testing.T) {
	t0, _ := NewTreeNode("t0", "t0")
	t1, _ := NewTreeNode("t1", "t1")
	t2, _ := NewTreeNode("t2", "t2")
	t3, _ := NewTreeNode("t3", "t3")

	assert.EqualValues(t, 1, t0.Height())
	t0.AddChild(t1)
	assert.EqualValues(t, 2, t0.Height())
	t0.AddChild(t2)
	assert.EqualValues(t, 2, t0.Height())

	/*
	      t0
	   t1   t2
	       t3
	*/

	t2.AddChild(t3)
	assert.EqualValues(t, 3, t0.Height())
}

func TestTree_NumLeaves(t *testing.T) {
	n1, _ := NewTreeNode("n1", "n1")
	n2, _ := NewTreeNode("n2", "n2")
	n3, _ := NewTreeNode("n3", "n3")
	n4, _ := NewTreeNode("n4", "n4")
	n5, _ := NewTreeNode("n5", "n5")
	n6, _ := NewTreeNode("n6", "n6")
	n7, _ := NewTreeNode("n7", "n7")
	n8, _ := NewTreeNode("n8", "n8")

	assert.EqualValues(t, 1, n1.NumLeaves())
	n1.AddChild(n2)
	assert.EqualValues(t, 1, n1.NumLeaves())
	n1.AddChild(n3)
	assert.EqualValues(t, 2, n1.NumLeaves())
	n2.AddChild(n4)
	assert.EqualValues(t, 2, n1.NumLeaves())
	n2.AddChild(n5)
	assert.EqualValues(t, 3, n1.NumLeaves())
	n3.AddChild(n6)
	assert.EqualValues(t, 3, n1.NumLeaves())
	n3.AddChild(n7)
	assert.EqualValues(t, 4, n1.NumLeaves())
	n7.AddChild(n8)

	/*
	         n1
	     n2     n3
	   n4 n5  n6  n7
	                n8
	*/

	assert.EqualValues(t, 4, n1.NumLeaves())
	assert.EqualValues(t, 8, n1.Size())
	assert.EqualValues(t, 4, n1.Height())
}
