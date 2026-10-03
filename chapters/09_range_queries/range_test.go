package chapter09

import "testing"

func TestScanRange(t *testing.T) {
	rows := []Row{{ID: 1, Value: "one"}, {ID: 2, Value: "two"}, {ID: 3, Value: "three"}, {ID: 4, Value: "four"}}
	got := Scan(rows, 2, 4)
	if len(got) != 3 {
		t.Fatalf("expected 3 items, got %d", len(got))
	}
	if got[0].Value != "two" || got[2].Value != "four" {
		t.Fatalf("unexpected range result: %#v", got)
	}
}
