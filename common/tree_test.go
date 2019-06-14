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
	parentNode1, _ := NewTree("parent1", "parent1")
	parentNode2, _ := NewTree("parent2", "parent2")
	childNode, _ := NewTree("child2", "child2")

	err1 := childNode.AddParent(parentNode1)
	assert.Equal(t, nil, err1)
	err2 := childNode.AddParent(parentNode2)
	assert.Equal(t, ErrNodeAlreadyHasParent, err2)

	assert.Equal(t, parentNode1, childNode.Parent)
}

func Test_AddChild(t *testing.T) {
	parentNode, _ := NewTree("parent", "parent")
	childNode1, _ := NewTree("child1", "child1")
	childNode2, _ := NewTree("child2", "child2")

	parentNode.AddChild(childNode1)
	parentNode.AddChild(childNode2)

	assert.Equal(t, parentNode, childNode1.Parent)
	assert.Equal(t, parentNode, childNode2.Parent)

	assert.Equal(t, 2, len(parentNode.Children))
	assert.True(t, parentNode.containChild(childNode1))
	assert.True(t, parentNode.containChild(childNode2))
}

func Test_HasChild(t *testing.T) {
	parentNode1, _ := NewTree("parent1", "parent1")
	parentNode2, _ := NewTree("parent2", "parent2")
	childNode1, _ := NewTree("child1", "child1")
	childNode2, _ := NewTree("child2", "child2")
	parentNode1.AddChild(childNode1)
	childNode2.AddParent(parentNode2)

	assert.True(t, parentNode1.hasChildren())
	assert.True(t, parentNode2.hasChildren())
}

func Test_FindHeightestChild(t *testing.T) {
	node1Height0, _ := NewTree("node1Height0", "node1Height0")
	node1Height1, _ := NewTree("node1Height1", "node1Height1")
	node2Height1, _ := NewTree("node2Height1", "node2Height1")
	node1Height2, _ := NewTree("node1Height2", "node1Height2")
	node2Height2, _ := NewTree("node2Height2", "node2Height2")
	node3Height2, _ := NewTree("node3Height2", "node3Height2")
	node4Height2, _ := NewTree("node4Height2", "node4Height2")
	node1Height3, _ := NewTree("node1Height3", "node1Height3")

	node1Height0.AddChild(node1Height1)
	node1Height0.AddChild(node2Height1)
	node1Height1.AddChild(node1Height2)
	node1Height1.AddChild(node2Height2)
	node2Height1.AddChild(node3Height2)
	node2Height1.AddChild(node4Height2)
	node3Height2.AddChild(node1Height3)

	var heightest1 *Tree
	var heightest2 *Tree
	var heightest3 *Tree

	_, heightest1 = node1Height0.FindHeightestChild(heightest1, 0, 0)
	_, heightest2 = node2Height1.FindHeightestChild(heightest2, 0, 0)
	_, heightest3 = node1Height1.FindHeightestChild(heightest3, 0, 0)

	assert.Equal(t, node1Height3, heightest1)
	assert.Equal(t, node1Height3, heightest2)
	assert.Equal(t, node1Height2, heightest3)
}

func Test_GetParentNodesRange(t *testing.T) {
	tree1, _ := NewTree("node1", "node1")
	tree2, _ := NewTree("node2", "node2")
	tree3, _ := NewTree("node3", "node3")
	tree4, _ := NewTree("node4", "node4")
	tree5, _ := NewTree("node5", "node5")
	tree6, _ := NewTree("node6", "node6")
	tree7, _ := NewTree("node7", "node7")
	tree8, _ := NewTree("node8", "node8")

	tree1.AddChild(tree2)
	tree2.AddChild(tree3)
	tree3.AddChild(tree4)
	tree4.AddChild(tree5)
	tree5.AddChild(tree6)
	tree6.AddChild(tree7)
	tree7.AddChild(tree8)

	expect := []*Tree{tree6, tree5, tree4, tree3, tree2}
	trees := tree6.GetParentTreesRange(tree2)

	assert.Equal(t, expect, trees)
}

func TestTree_Size(t *testing.T) {
	t0, _ := NewTree("t0", "t0")
	t1, _ := NewTree("t1", "t1")
	t2, _ := NewTree("t2", "t2")
	t3, _ := NewTree("t3", "t3")
	assert.EqualValues(t, 1, t1.Size())
	t1.AddChild(t0)
	assert.EqualValues(t, 2, t1.Size())
	t1.AddChild(t2)
	assert.EqualValues(t, 3, t1.Size())
	t2.AddChild(t3)
	assert.EqualValues(t, 4, t1.Size())
}

func TestTree_Height(t *testing.T) {
	t0, _ := NewTree("t0", "t0")
	t1, _ := NewTree("t1", "t1")
	t2, _ := NewTree("t2", "t2")
	t3, _ := NewTree("t3", "t3")

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
	n1, _ := NewTree("n1", "n1")
	n2, _ := NewTree("n2", "n2")
	n3, _ := NewTree("n3", "n3")
	n4, _ := NewTree("n4", "n4")
	n5, _ := NewTree("n5", "n5")
	n6, _ := NewTree("n6", "n6")
	n7, _ := NewTree("n7", "n7")
	n8, _ := NewTree("n8", "n8")

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
