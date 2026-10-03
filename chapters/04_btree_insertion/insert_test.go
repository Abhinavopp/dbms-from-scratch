package chapter04

import "testing"

func TestTreeInsert(t *testing.T) {
	tree := NewTree(3)
	for _, key := range []int{10, 20, 5, 6, 12, 30, 7, 17} {
		tree.Insert(key)
	}

	if !tree.Search(17) {
		t.Fatal("expected 17 to be present")
	}
	if tree.Search(999) {
		t.Fatal("expected 999 to be absent")
	}
}

func TestTreeSplit(t *testing.T) {
	tree := NewTree(3)
	for _, key := range []int{1, 2, 3, 4, 5, 6, 7, 8} {
		tree.Insert(key)
	}

	if tree.Root == nil || len(tree.Root.Children) == 0 {
		t.Fatal("expected the root to split into children after enough inserts")
	}
}
