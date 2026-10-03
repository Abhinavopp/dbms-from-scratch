package chapter12

import (
	"sync"
	"testing"
)

func TestConcurrentReadWrite(t *testing.T) {
	store := NewRWStore()
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		for i := 0; i < 50; i++ {
			store.Set("value", "written")
		}
	}()
	go func() {
		defer wg.Done()
		for i := 0; i < 50; i++ {
			_, _ = store.Get("value")
		}
	}()
	wg.Wait()
	if got, ok := store.Get("value"); !ok || got != "written" {
		t.Fatalf("expected store to retain written value, got %q, %v", got, ok)
	}
}
