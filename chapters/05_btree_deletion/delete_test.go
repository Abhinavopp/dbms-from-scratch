package chapter05

import "testing"

func TestDeleteRemovesKey(t *testing.T) {
	tree := NewTree(3)
	for _, key := range []int{1, 2, 3, 4, 5, 6, 7, 8} {
		tree.Insert(key)
	}

	tree.Delete(4)
	if tree.Search(4) {
		t.Fatal("expected key 4 to be removed")
	}
	if !tree.Search(5) {
		t.Fatal("expected key 5 to remain")
	}
}

func TestDeleteLeavesTreeUsable(t *testing.T) {
	tree := NewTree(3)
	for _, key := range []int{10, 20, 30, 40} {
		tree.Insert(key)
	}

	tree.Delete(20)
	if tree.Search(20) {
		t.Fatal("expected 20 to be gone")
	}
	if !tree.Search(30) {
		t.Fatal("expected 30 to remain")
	}
}
