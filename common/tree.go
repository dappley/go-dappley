package common

import (
	"errors"
	logger "github.com/sirupsen/logrus"
)

type Entry struct{
	key interface{}
	value interface{}
}


var (
	ErrNodeNotFound = errors.New("ERROR: Node not found in tree")
	ErrCantCreateEmptyNode = errors.New("ERROR: Node index and value must not be empty")

)

//entries include the node's entry itself as the first entry and its childrens' entry following
type node struct {
	Entries []Entry
	Parent *node
	Children []*node
	Height uint
}

type Tree struct {
	Root *node
	MaxHeight uint
	Found *node
	Searching bool
}
type Test struct {
	Num uint
}


func (n *node) hasChildren() bool{
	if len(n.Children) > 0 {
		return true
	}
	return false
}

func (t *Tree) NewNode(index interface{}, value interface{}) (*node, error){
	if index == nil || value == nil {
		return nil, ErrCantCreateEmptyNode
	}
	return &node{[]Entry{Entry{index,value,}}, nil, nil, 1}, nil
}

func NewTree(rootNodeIndex interface{}, rootNodeValue interface{}) *Tree{
	r := node{[]Entry{Entry{rootNodeIndex,rootNodeValue,}}, nil, nil, 1}
	return &Tree{&r, r.Height , nil, false}
}


func (t *Tree) RecursiveFind (parent *node, index interface{}) {
	if !parent.hasChildren() ||  t.Searching == false {
		logger.Debug(parent.Entries[0].key," has no children")
		return
	}
	for i:=0;i< len(parent.Children);i++  {
		if parent.Children[i].Entries[0].key == index{
			logger.Debug("found! ", index, " under ", parent.Entries[0].key)
			t.Searching = false
			t.Found = parent.Children[i]
		}else{
			if t.Searching {
				t.RecursiveFind(parent.Children[i], index)
			}
		}
	}
}

//Search from root, use if you have no closer known nodes upstream
func (t *Tree) Get(parent *node, index interface{}){
	t.Searching = true
	if t.Root.Entries[0].key == index{
		logger.Debug("found! ", index, ", is root")
		t.Found = t.Root
		return
	}
	t.RecursiveFind(parent, index)
}

func (t *Tree) SearchParentNodeAndAddChild( startNode *node, parentIndex interface{} , childIndex interface{}, childValue interface{}){
	child,_ := t.NewNode(childIndex, childValue)
	t.Get(t.Root, parentIndex)
	parent := t.Found
	parent.AddChild(child)
}

func (parent *node) AddChild(child *node){
	parent.Children = append(parent.Children, child)
	parent.Entries = append(parent.Entries, child.Entries[0])
	child.Parent = parent
}

//attach a tree's root node to a specific node of another tree through node index
func (t *Tree) appendTree(tree *Tree, mergeIndex interface{}) {
	t.Get(t.Root, mergeIndex)
	t.Found.AddChild(tree.Root)
	//is higher than original tree after appending
	if tree.MaxHeight + t.Found.Height > t.MaxHeight{
		t.MaxHeight = tree.MaxHeight + t.Found.Height
	}
}