package chapter10

import "testing"

func TestSecondaryIndexLookup(t *testing.T) {
	idx := NewIndex()
	idx.Add("admin", 1)
	idx.Add("admin", 7)
	idx.Add("user", 4)

	rows := idx.Find("admin")
	if len(rows) != 2 || rows[0] != 1 || rows[1] != 7 {
		t.Fatalf("unexpected admin rows: %#v", rows)
	}
}
