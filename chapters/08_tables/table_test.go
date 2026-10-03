package chapter08

import "testing"

func TestTableInsertAndGet(t *testing.T) {
	table := NewTable(Schema{Name: "users", Fields: []string{"name", "email"}})
	table.Insert(Record{ID: 1, Data: map[string]string{"name": "alice", "email": "alice@example.com"}})

	row, ok := table.Get(1)
	if !ok {
		t.Fatal("expected record to exist")
	}
	if row.Data["name"] != "alice" {
		t.Fatalf("expected alice, got %s", row.Data["name"])
	}
}
