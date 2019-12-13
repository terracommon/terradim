package model

import (
	"testing"
)

func TestInsertRelative(t *testing.T) {
	tree := NewTree()
	node, ok := tree.Insert("", true)
	if size := tree.Size(); size < 0 {
		t.Fatalf("Tree should have size 0. Size: %v", size)
	}
	if ok {
		t.Fatalf("isNew should be false for blank insert")
	}
	if node != nil {
		t.Fatalf("Node should be nil. Node: %v", node)
	}

	node, ok = tree.Insert("foo", true)
	if size := tree.Size(); size != 1 {
		t.Fatalf("Tree should have size 1. Size: %v", size)
	}
	if ok != true {
		t.Fatalf("isNew should be true for new node")
	}
	if node == nil {
		t.Fatalf("Node should exist after insert")
	}

	node, ok = tree.Insert("foo/bar", true)
	if size := tree.Size(); size != 2 {
		t.Fatalf("Tree should have size 2. Size: %v", size)
	}
	if ok != true {
		t.Fatalf("isNew should be true for new node")
	}
	if node == nil {
		t.Fatalf("Node should exist after insert")
	}

	node, ok = tree.Insert("foo", false)
	if size := tree.Size(); size != 2 {
		t.Fatalf("Tree should have size 2. Size: %v", size)
	}
	if ok {
		t.Fatalf("isNew should be false for updated node")
	}
	if node.meta.(bool) == true {
		t.Fatalf("Meta for updated node should be false. Meta: %+v", node.meta)
	}
}

func TestInsertAbsolute(t *testing.T) {
	tree := NewTree()
	node, ok := tree.Insert(tree.Separator(), true)
	if size := tree.Size(); size < 0 {
		t.Fatalf("Tree should have size 0. Size: %v", size)
	}
	if ok {
		t.Fatalf("isNew should be false for blank insert")
	}
	if node != nil {
		t.Fatalf("Node should be nil. Node: %v", node)
	}

	node, ok = tree.Insert(tree.Separator()+tree.Separator(), true)
	if size := tree.Size(); size < 0 {
		t.Fatalf("Tree should have size 0. Size: %v", size)
	}
	if ok {
		t.Fatalf("isNew should be false for blank insert")
	}
	if node != nil {
		t.Fatalf("Node should be nil. Node: %v", node)
	}

	node, ok = tree.Insert("/foo", true)
	if size := tree.Size(); size != 1 {
		t.Fatalf("Tree should have size 1. Size: %v", size)
	}
	if ok != true {
		t.Fatalf("isNew should be true for new node")
	}
	if node == nil {
		t.Fatalf("Node should exist after insert")
	}

	node, ok = tree.Insert("/foo/bar", true)
	if size := tree.Size(); size != 2 {
		t.Fatalf("Tree should have size 1. Size: %v", size)
	}
	if ok != true {
		t.Fatalf("isNew should be true for new node")
	}
	if node == nil {
		t.Fatalf("Node should exist after insert")
	}

	node, ok = tree.Insert("/foo/bar", false)
	if size := tree.Size(); size != 2 {
		t.Fatalf("Tree should have size 2. Size: %v", size)
	}
	if ok {
		t.Fatalf("isNew should be false for updated node")
	}
	if node.meta.(bool) == true {
		t.Fatalf("Meta for updated node should be false. Meta: %v", node.meta)
	}
}

func TestFind(t *testing.T) {
	tree := NewTree()
	root := tree.Root()
	_, _ = tree.Insert("foo", true)
	node, ok := tree.Find("foo")
	if ok != true {
		t.Fatalf("First inserted node not found")
	}
	if node.parent != root {
		t.Fatalf("First node does not have root as parent")
	}
	if node.meta.(bool) != true {
		t.Fatalf("Meta value not set to true. Meta: %+v/n", node.meta)
	}

	_, _ = tree.Insert("foo/bar", true)
	node, ok = tree.Find("foo")
	if ok != true {
		t.Fatalf("Inserted node not found")
	}

	tree = NewTree()
	_, _ = tree.Insert("/foo/bar/bang", true)
	_, ok = tree.Find("foo")
	if ok != true {
		t.Fatalf("Inserted node not found")
	}
	_, ok = tree.Find("/foo/bar")
	if ok != true {
		t.Fatalf("Inserted node not found")
	}
	node, ok = tree.Find("/foo/bar/bang")
	if ok != true {
		t.Fatalf("Inserted node not found")
	}
	if node.meta.(bool) != true {
		t.Fatalf("Meta value not set to true. Meta: %+v/n", node.meta)
	}
}
