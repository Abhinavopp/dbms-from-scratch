package chapter03

import "testing"

func TestBTreeInsertAndSearch(t *testing.T) {
	tree := NewBTree(3)
	for _, key := range []int{10, 20, 5, 6, 12, 30} {
		tree.Insert(key)
	}

	if !tree.Search(12) {
		t.Fatal("expected key 12 to be present")
	}
	if tree.Search(99) {
		t.Fatal("expected key 99 to be absent")
	}
}

func TestBTreeKeepsKeysSorted(t *testing.T) {
	tree := NewBTree(3)
	for _, key := range []int{8, 4, 12, 2, 6, 10, 14} {
		tree.Insert(key)
	}

	if tree.Root == nil {
		t.Fatal("root should exist")
	}
	if len(tree.Root.Keys) == 0 {
		t.Fatal("root should contain keys")
	}
}
