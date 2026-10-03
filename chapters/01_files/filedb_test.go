package chapter01

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFileDBPersistsValue(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "db.json")
	store, err := NewFileDB(path)
	if err != nil {
		t.Fatal(err)
	}

	payload := map[string]string{"name": "alice"}
	if err := store.Save(payload); err != nil {
		t.Fatal(err)
	}

	var loaded map[string]string
	if err := store.Load(&loaded); err != nil {
		t.Fatal(err)
	}
	if loaded["name"] != "alice" {
		t.Fatalf("expected alice, got %s", loaded["name"])
	}
}

func TestWriteAtomic(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	if err := WriteAtomic(path, []byte("ready")); err != nil {
		t.Fatal(err)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "ready" {
		t.Fatalf("expected ready, got %q", string(content))
	}
}

func TestAppendLog(t *testing.T) {
	path := filepath.Join(t.TempDir(), "events.log")
	if err := AppendLog(path, "first"); err != nil {
		t.Fatal(err)
	}
	if err := AppendLog(path, "second"); err != nil {
		t.Fatal(err)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := string(content); got != "first\nsecond\n" {
		t.Fatalf("unexpected log contents: %q", got)
	}
}
