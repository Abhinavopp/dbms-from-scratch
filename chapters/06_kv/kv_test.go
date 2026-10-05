package chapter06

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

func TestKVSetGetDelete(t *testing.T) {
	kv := NewKV()
	if err := kv.Set("name", "alice"); err != nil {
		t.Fatal(err)
	}
	if got, ok := kv.Get("name"); !ok || got != "alice" {
		t.Fatalf("expected alice, got %q, %v", got, ok)
	}
	if err := kv.Delete("name"); err != nil {
		t.Fatal(err)
	}
	if _, ok := kv.Get("name"); ok {
		t.Fatal("expected key to be deleted")
	}
}

func TestKVSaveAndLoad(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "store.json")
	kv := NewKV()
	for i := 0; i < 500; i++ {
		if err := kv.Set("key-"+strconv.Itoa(i), "value-"+strconv.Itoa(i)); err != nil {
			t.Fatal(err)
		}
	}
	if err := kv.Set("role", "admin"); err != nil {
		t.Fatal(err)
	}
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
	for i := 0; i < 500; i++ {
		key := "key-" + strconv.Itoa(i)
		if got, ok := loaded.Get(key); !ok || got != "value-"+strconv.Itoa(i) {
			t.Fatalf("expected value for %q, got %q, %v", key, got, ok)
		}
	}
}

func TestLoadedKVWritesThroughToDisk(t *testing.T) {
	path := filepath.Join(t.TempDir(), "store.json")
	store, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Set("before", "one"); err != nil {
		t.Fatal(err)
	}
	if err := store.Set("after", "two"); err != nil {
		t.Fatal(err)
	}
	if err := store.Delete("before"); err != nil {
		t.Fatal(err)
	}

	reloaded, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := reloaded.Get("before"); ok {
		t.Fatal("deleted key should remain absent after reload")
	}
	if got, ok := reloaded.Get("after"); !ok || got != "two" {
		t.Fatalf("expected persisted value two, got %q, %v", got, ok)
	}
}

func TestLoadRejectsInvalidSnapshots(t *testing.T) {
	path := filepath.Join(t.TempDir(), "invalid.json")
	if err := os.WriteFile(path, []byte(`{"version":99,"root":{}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path); err == nil {
		t.Fatal("expected unsupported snapshot version error")
	}
}
