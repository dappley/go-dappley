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
