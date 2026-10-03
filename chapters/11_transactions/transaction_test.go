package chapter11

import "testing"

func TestTransactionCommit(t *testing.T) {
	tx := Begin()
	tx.Set("name", "alice")
	committed := tx.Commit()
	if committed["name"] != "alice" {
		t.Fatalf("expected committed value alice, got %q", committed["name"])
	}
}

func TestTransactionRollback(t *testing.T) {
	tx := Begin()
	tx.Set("role", "admin")
	tx.Rollback()
	if committed := tx.Commit(); committed != nil {
		t.Fatalf("expected nil after rollback, got %#v", committed)
	}
}
