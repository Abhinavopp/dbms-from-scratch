package chapter07

import "testing"

func TestFreeListAllocationAndReuse(t *testing.T) {
	list := NewFreeList()
	first := list.Allocate()
	second := list.Allocate()
	list.Free(first)
	if reused := list.Allocate(); reused != first {
		t.Fatalf("expected reuse of first page %d, got %d", first, reused)
	}
	if second != 2 {
		t.Fatalf("expected second to be 2, got %d", second)
	}
}
