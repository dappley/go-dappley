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
	"fmt"
	"strconv"
	"testing"

	errval "github.com/dappley/go-dappley/errors"
	logger "github.com/sirupsen/logrus"

	"github.com/stretchr/testify/assert"
)

func Test_SetParent(t *testing.T) {
	parentNode1, _ := NewTreeNode("parent1")
	parentNode2, _ := NewTreeNode("parent2")
	childNode, _ := NewTreeNode("child2")

	err1 := childNode.SetParent(parentNode1)
	assert.Equal(t, nil, err1)
	err2 := childNode.SetParent(parentNode2)
	assert.Equal(t, errval.NodeAlreadyHasParent, err2)

	assert.Equal(t, parentNode1, childNode.Parent)
}

func Test_AddChild(t *testing.T) {
	parentNode, _ := NewTreeNode("parent")
	childNode1, _ := NewTreeNode("child1")
	childNode2, _ := NewTreeNode("child2")

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
	newTree, _ := NewTreeNode("new")
	parentNode1, _ := NewTreeNode("parent1")
	parentNode2, _ := NewTreeNode("parent2")
	childNode1, _ := NewTreeNode("child1")
	childNode2, _ := NewTreeNode("child2")
	parentNode1.AddChild(childNode1)
	childNode2.SetParent(parentNode2)

	assert.Nil(t, nilTree.Children)
	assert.True(t, parentNode1.hasChildren())
	assert.True(t, parentNode2.hasChildren())
	assert.False(t, newTree.hasChildren())
	assert.False(t, nilTree.hasChildren())
}

func TestTreeNode_GetLongestPath(t *testing.T) {

	tests := []struct {
		name             string
		deserializedTree string
		expected         []int
	}{
		{"Empty Root", "", []int{}},
		{"Normal Case 1", "1", []int{1}},
		{"Normal Case 2", "1, 1#2", []int{2, 1}},
		{"Normal Case 3", "1, 1#2, 1#3", []int{2, 1}},
		{"Normal Case 4", "1, 1#2, 1#3, 2#4", []int{4, 2, 1}},
		{"Normal Case 5", "1, 1#2, 1#3, 2#4, 2#5", []int{4, 2, 1}},
		{"Normal Case 6", "1, 1#2, 1#3, 2#4, 2#5, 3#6", []int{4, 2, 1}},
		{"Normal Case 7", "1, 1#2, 1#3, 2#4, 2#5, 3#6, 3#7", []int{4, 2, 1}},
		{"Normal Case 8", "1, 1#2, 1#3, 2#4, 2#5, 3#6, 3#7, 7#8", []int{8, 7, 3, 1}},
		{"Normal Case 9", "1, 1#2, 1#3, 3#4, 4#5, 5#6", []int{6, 5, 4, 3, 1}},
		{"More than 2 children", "1, 1#2, 1#3, 1#4, 4#5, 4#6, 3#7, 2#8, 8#9, 9#10, 10#11", []int{11, 10, 9, 8, 2, 1}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root, _ := deserializeTree(tt.deserializedTree)
			nodes := root.GetLongestPath()
			assert.Equal(t, len(tt.expected), len(nodes))
			for i, node := range nodes {
				assert.Equal(t, tt.expected[i], node.value)
			}
		})
	}

}

func TestTreeNode_Prune(t *testing.T) {

	tests := []struct {
		name                     string
		deserializedTree         string
		rootNodeId               int
		expectedNumOfRemovedNode int
	}{
		{"Normal Case 1", "1", 1, 0},
		{"Normal Case 2", "1, 1#2", 2, 1},
		{"Normal Case 3", "1, 1#2, 1#3", 2, 2},
		{"Normal Case 4", "1, 1#2, 1#3, 2#4", 2, 2},
		{"Normal Case 5", "1, 1#2, 1#3, 2#4, 2#5", 3, 4},
		{"Normal Case 6", "1, 1#2, 1#3, 2#4, 2#5, 3#6", 2, 3},
		{"Normal Case 7", "1, 1#2, 1#3, 2#4, 2#5, 3#6, 3#7", 2, 4},
		{"Normal Case 8", "1, 1#2, 1#3, 2#4, 2#5, 3#6, 3#7, 7#8", 7, 6},
		{"Normal Case 9", "1, 1#2, 1#3, 3#4, 4#5, 5#6", 6, 5},
		{"More than 2 children", "1, 1#2, 1#3, 1#4, 4#5, 4#6, 3#7, 2#8, 8#9, 9#10, 10#11", 6, 10},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, nodes := deserializeTree(tt.deserializedTree)
			node := nodes[tt.rootNodeId]
			count := 0
			fmt.Println(node.Parent != nil)
			node.Prune(
				func(node *TreeNode) {
					count++
				})
			assert.Equal(t, tt.expectedNumOfRemovedNode, count)
		})
	}
}

func TestTreeNode_RemoveAllDescendants(t *testing.T) {

	tests := []struct {
		name                     string
		deserializedTree         string
		rootNodeId               int
		expectedHeight           int64
		expectedNumOfRemovedNode int
	}{
		{"Normal Case 1", "1, 1#2, 1#3, 2#4, 2#5", 2, 2, 2},
		{"Normal Case 2", "1, 1#2, 1#3, 2#4, 2#5, 3#6", 1, 1, 5},
		{"Normal Case 3", "1, 1#2, 1#3, 2#4, 2#5, 3#6, 3#7", 3, 3, 2},
		{"Normal Case 4", "1, 1#2, 1#3, 2#4, 2#5, 3#6, 3#7, 7#8", 2, 4, 2},
		{"Normal Case 5", "1, 1#2, 1#3, 3#4, 4#5, 5#6", 5, 4, 1},
		{"More than 2 children", "1, 1#2, 1#3, 1#4, 4#5, 4#6, 3#7, 2#8, 8#9, 9#10, 10#11", 4, 6, 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root, nodes := deserializeTree(tt.deserializedTree)
			node := nodes[tt.rootNodeId]
			count := 0
			node.RemoveAllDescendants(
				func(node *TreeNode) {
					count++
				})
			assert.Equal(t, tt.expectedHeight, root.Height())
			assert.Equal(t, tt.expectedNumOfRemovedNode, count)
		})
	}
}

func TestTree_Size(t *testing.T) {
	t0, _ := NewTreeNode("t0")
	t1, _ := NewTreeNode("t1")
	t2, _ := NewTreeNode("t2")
	t3, _ := NewTreeNode("t3")
	assert.EqualValues(t, 1, t1.Size())
	t1.AddChild(t0)
	assert.EqualValues(t, 2, t1.Size())
	t1.AddChild(t2)
	assert.EqualValues(t, 3, t1.Size())
	t2.AddChild(t3)
	assert.EqualValues(t, 4, t1.Size())
}

func TestTree_Height(t *testing.T) {

	tests := []struct {
		name             string
		deserializedTree string
		expected         int64
	}{
		{"Empty Root", "", 0},
		{"Normal Case 1", "1", 1},
		{"Normal Case 2", "1, 1#2", 2},
		{"Normal Case 3", "1, 1#2, 1#3", 2},
		{"Normal Case 4", "1, 1#2, 1#3, 2#4", 3},
		{"Normal Case 5", "1, 1#2, 1#3, 2#4, 2#5", 3},
		{"Normal Case 6", "1, 1#2, 1#3, 2#4, 2#5, 3#6", 3},
		{"Normal Case 7", "1, 1#2, 1#3, 2#4, 2#5, 3#6, 3#7", 3},
		{"Normal Case 8", "1, 1#2, 1#3, 2#4, 2#5, 3#6, 3#7, 7#8", 4},
		{"Normal Case 9", "1, 1#2, 1#3, 3#4, 4#5, 5#6", 5},
		{"More than 2 children", "1, 1#2, 1#3, 1#4, 4#5, 4#6, 3#7, 2#8, 8#9, 9#10, 10#11", 6},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root, _ := deserializeTree(tt.deserializedTree)
			assert.Equal(t, tt.expected, root.Height())
		})
	}

}

func TestTree_NumLeaves(t *testing.T) {

	tests := []struct {
		name             string
		deserializedTree string
		expected         int64
	}{
		{"Empty Root", "", 0},
		{"Normal Case 1", "1", 1},
		{"Normal Case 2", "1, 1#2", 1},
		{"Normal Case 3", "1, 1#2, 1#3", 2},
		{"Normal Case 4", "1, 1#2, 1#3, 2#4", 2},
		{"Normal Case 5", "1, 1#2, 1#3, 2#4, 2#5", 3},
		{"Normal Case 6", "1, 1#2, 1#3, 2#4, 2#5, 3#6", 3},
		{"Normal Case 7", "1, 1#2, 1#3, 2#4, 2#5, 3#6, 3#7", 4},
		{"Normal Case 8", "1, 1#2, 1#3, 2#4, 2#5, 3#6, 3#7, 7#8", 4},
		{"Normal Case 9", "1, 1#2, 1#3, 3#4, 4#5, 5#6", 2},
		{"More than 2 children", "1, 1#2, 1#3, 1#4, 4#5, 4#6, 3#7, 2#8, 8#9, 9#10, 10#11", 4},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root, _ := deserializeTree(tt.deserializedTree)
			assert.Equal(t, tt.expected, root.NumLeaves())
		})
	}
}

//deserializeTree creates a tree structure by deserializing the input string. return the root of the tree
func deserializeTree(s string) (root *TreeNode, nodes map[int]*TreeNode) {
	/* "1, 1#2, 1#3, 3#4" describes a tree like following"
				1
			   2 3
	              4
	*/
	if s == "" {
		return nil, nil
	}

	var parentNode *TreeNode
	currStr := ""
	nodes = make(map[int]*TreeNode)

	for _, c := range s {
		switch c {
		case ',':
			num, err := strconv.Atoi(currStr)
			if err != nil {
				logger.WithError(err).Panic("deserialize tree failed while converting string to int")
			}
			node, _ := NewTreeNode(num)
			if parentNode == nil {
				root = node
			} else {
				parentNode.Children = append(parentNode.Children, node)
				node.SetParent(parentNode)
			}
			nodes[num] = node
			currStr = ""
			if parentNode == nil {
				logger.WithFields(logger.Fields{
					"root": num,
				}).Debug("Add a new node as root")
			} else {
				logger.WithFields(logger.Fields{
					"node":       num,
					"parentNode": parentNode.value,
				}).Debug("Add a new node")
			}

		case '#':
			num, err := strconv.Atoi(currStr)
			if err != nil {
				logger.WithError(err).Panic("deserialize tree failed while converting string to int")
			}
			if _, isFound := nodes[num]; !isFound {
				logger.WithFields(logger.Fields{
					"node": num,
				}).Panic("deserialize tree failed: the parent node is not found")
			}
			parentNode = nodes[num]
			currStr = ""
		case ' ':
			continue
		default:
			currStr = currStr + string(c)
		}
	}

	num, err := strconv.Atoi(currStr)
	if err != nil {
		logger.WithError(err).Panic("deserialize tree failed while converting string to int")
	}
	node, _ := NewTreeNode(num)
	if parentNode == nil {
		root = node
	} else {
		parentNode.Children = append(parentNode.Children, node)
		node.SetParent(parentNode)
	}
	nodes[num] = node
	currStr = ""
	if parentNode == nil {
		logger.WithFields(logger.Fields{
			"root": num,
		}).Debug("Add a new node as root")
	} else {
		logger.WithFields(logger.Fields{
			"node":       num,
			"parentNode": parentNode.value,
		}).Debug("Add a new node")
	}
	return root, nodes
}
