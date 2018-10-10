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
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_AddParent(t *testing.T) {
	parentNode1, _ := NewNode("parent1", "parent1", 0)
	parentNode2, _ := NewNode("parent2", "parent2", 0)
	childNode, _ := NewNode("child2", "child2", 0)

	err1 := childNode.AddParent(parentNode1)
	assert.Equal(t, nil, err1)
	err2 := childNode.AddParent(parentNode2)
	assert.Equal(t, errors.New("ERROR: Adding parent to node already with parent"), err2)

	assert.Equal(t, parentNode1, childNode.Parent)

}

func Test_AddChild(t *testing.T) {
	parentNode, _ := NewNode("parent", "parent", 0)
	childNode1, _ := NewNode("child1", "child1", 0)
	childNode2, _ := NewNode("child2", "child2", 0)

	parentNode.AddChild(childNode1)
	parentNode.AddChild(childNode2)

	assert.Equal(t, parentNode, childNode1.Parent)
	assert.Equal(t, parentNode, childNode2.Parent)

	assert.Equal(t, uint64(0x1), childNode2.Height)
	assert.Equal(t, uint64(0x1), childNode2.Height)
	assert.Equal(t, uint64(0x0), parentNode.Height)

	assert.Equal(t, 2, len(parentNode.Children))
	assert.True(t, parentNode.containChild(childNode1))
	assert.True(t, parentNode.containChild(childNode2))
}

func Test_HasChild(t *testing.T) {
	parentNode1, _ := NewNode("parent1", "parent1", 0)
	parentNode2, _ := NewNode("parent2", "parent2", 0)
	childNode1, _ := NewNode("child1", "child1", 0)
	childNode2, _ := NewNode("child2", "child2", 0)
	parentNode1.AddChild(childNode1)
	childNode2.AddParent(parentNode2)

	assert.True(t, parentNode1.hasChildren())
	assert.True(t, parentNode2.hasChildren())

}

func Test_FindHeightestChild(t *testing.T) {
	node1Height0, _ := NewNode("node1Height0", "node1Height0", 0)
	node1Height1, _ := NewNode("node1Height1", "node1Height1", 1)
	node2Height1, _ := NewNode("node2Height1", "node2Height1", 1)
	node1Height2, _ := NewNode("node1Height2", "node1Height2", 2)
	node2Height2, _ := NewNode("node2Height2", "node2Height2", 2)
	node3Height2, _ := NewNode("node3Height2", "node3Height2", 2)
	node4Height2, _ := NewNode("node4Height2", "node4Height2", 2)
	node1Height3, _ := NewNode("node1Height3", "node1Height3", 3)

	node1Height0.AddChild(node1Height1)
	node1Height0.AddChild(node2Height1)
	node1Height1.AddChild(node1Height2)
	node1Height1.AddChild(node2Height2)
	node2Height1.AddChild(node3Height2)
	node2Height1.AddChild(node4Height2)
	node3Height2.AddChild(node1Height3)

	var heightest1 Node
	var heightest2 Node
	var heightest3 Node

	node1Height0.FindHeightestChild(&heightest1)
	node2Height1.FindHeightestChild(&heightest2)
	node1Height1.FindHeightestChild(&heightest3)

	assert.Equal(t, node1Height3, &heightest1)
	assert.Equal(t, node1Height3, &heightest2)
	assert.Equal(t, node1Height2, &heightest3)

}

func Test_GetParentNodesRange(t *testing.T) {
	node1, _ := NewNode("node1", "node1", 0)
	node2, _ := NewNode("node2", "node2", 1)
	node3, _ := NewNode("node3", "node3", 2)
	node4, _ := NewNode("node4", "node4", 3)
	node5, _ := NewNode("node5", "node5", 4)
	node6, _ := NewNode("node6", "node6", 5)
	node7, _ := NewNode("node7", "node7", 6)
	node8, _ := NewNode("node8", "node8", 7)

	node1.AddChild(node2)
	node2.AddChild(node3)
	node3.AddChild(node4)
	node4.AddChild(node5)
	node5.AddChild(node6)
	node6.AddChild(node7)
	node7.AddChild(node8)

	expect := []*Node{node6, node5, node4, node3}
	nodes := node6.GetParentNodesRange(node2)

	assert.Equal(t, expect, nodes)
}
