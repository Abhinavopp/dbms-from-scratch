package chapter12

import "sync"

// RWStore represents a simple concurrent access layer with a reader-writer lock.
type RWStore struct {
	mu   sync.RWMutex
	data map[string]string
}

func NewRWStore() *RWStore {
	return &RWStore{data: make(map[string]string)}
}

func (s *RWStore) Set(key, value string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[key] = value
}

func (s *RWStore) Get(key string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	value, ok := s.data[key]
	return value, ok
}

func (s *RWStore) Snapshot() map[string]string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make(map[string]string, len(s.data))
	for k, v := range s.data {
		out[k] = v
	}
	return out
}
