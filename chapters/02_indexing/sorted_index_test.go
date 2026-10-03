package chapter02

import "testing"

func TestSortedIndexPutAndGet(t *testing.T) {
	idx := NewSortedIndex()
	idx.Put("b", "second")
	idx.Put("a", "first")
	idx.Put("c", "third")

	if got, ok := idx.Get("b"); !ok || got != "second" {
		t.Fatalf("expected b=second, got %q, %v", got, ok)
	}
	if _, ok := idx.Get("missing"); ok {
		t.Fatal("expected missing key to be absent")
	}
}

func TestSortedIndexRange(t *testing.T) {
	idx := NewSortedIndex()
	for _, entry := range []Entry{{"b", "2"}, {"a", "1"}, {"c", "3"}, {"d", "4"}} {
		idx.Put(entry.Key, entry.Value)
	}

	got := idx.Range("b", "d")
	if len(got) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(got))
	}
	if got[0].Key != "b" || got[1].Key != "c" || got[2].Key != "d" {
		t.Fatalf("range result was unexpected: %#v", got)
	}
}
