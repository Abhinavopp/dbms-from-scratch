package chapter06

import (
	"os"
	"path/filepath"
	"testing"
)

func TestKVSetGetDelete(t *testing.T) {
	kv := NewKV()
	kv.Set("name", "alice")
	if got, ok := kv.Get("name"); !ok || got != "alice" {
		t.Fatalf("expected alice, got %q, %v", got, ok)
	}
	kv.Delete("name")
	if _, ok := kv.Get("name"); ok {
		t.Fatal("expected key to be deleted")
	}
}

func TestKVSaveAndLoad(t *testing.T) {
	path := filepath.Join(t.TempDir(), "store.json")
	kv := NewKV()
	kv.Set("role", "admin")
	if err := kv.Save(path); err != nil {
		t.Fatal(err)
	}

	loaded, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if got, ok := loaded.Get("role"); !ok || got != "admin" {
		t.Fatalf("expected admin after reload, got %q, %v", got, ok)
	}
	_ = os.WriteFile
}
