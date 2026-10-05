package chapter02

import "sort"

type Entry struct {
	Key   string
	Value string
}

type SortedIndex struct {
	entries []Entry
}

func NewSortedIndex() *SortedIndex {
	return &SortedIndex{}
}

func (s *SortedIndex) Put(key, value string) {
	idx := sort.Search(len(s.entries), func(i int) bool {
		return s.entries[i].Key >= key
	})
	if idx < len(s.entries) && s.entries[idx].Key == key {
		s.entries[idx].Value = value
		return
	}
	s.entries = append(s.entries, Entry{})
	copy(s.entries[idx+1:], s.entries[idx:])
	s.entries[idx] = Entry{Key: key, Value: value}
}

func (s *SortedIndex) Get(key string) (string, bool) {
	idx := sort.Search(len(s.entries), func(i int) bool {
		return s.entries[i].Key >= key
	})
	if idx < len(s.entries) && s.entries[idx].Key == key {
		return s.entries[idx].Value, true
	}
	return "", false
}

func (s *SortedIndex) Range(start, end string) []Entry {
	left := sort.Search(len(s.entries), func(i int) bool {
		return s.entries[i].Key >= start
	})
	right := sort.Search(len(s.entries), func(i int) bool {
		return s.entries[i].Key > end
	})
	if left >= right {
		return nil
	}
	out := make([]Entry, right-left)
	copy(out, s.entries[left:right])
	return out
}
