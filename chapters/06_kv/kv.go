package chapter06

import (
	"encoding/json"
	"errors"
	"fmt"
)

const snapshotVersion = 1

type KV struct {
	root *node
	path string
}

type snapshot struct {
	Version int   `json:"version"`
	Root    *node `json:"root"`
}

func NewKV() *KV {
	return &KV{root: newNode(true)}
}

func (k *KV) Set(key, value string) error {
	if k.root == nil {
		k.root = newNode(true)
	}
	previousRoot := cloneNode(k.root)
	set(k, key, value)
	if err := k.persist(); err != nil {
		k.root = previousRoot
		return err
	}
	return nil
}

func (k *KV) Get(key string) (string, bool) {
	return get(k.root, key)
}

func (k *KV) Entries() map[string]string {
	entries := make(map[string]string)
	all := make([]entry, 0)
	collect(k.root, &all)
	for _, item := range all {
		entries[item.key] = item.value
	}
	return entries
}

func (k *KV) ReplaceAll(values map[string]string) error {
	previousRoot := cloneNode(k.root)
	k.root = newNode(true)
	for key, value := range values {
		set(k, key, value)
	}
	if err := k.persist(); err != nil {
		k.root = previousRoot
		return err
	}
	return nil
}

func (k *KV) Delete(key string) error {
	_, ok := k.Get(key)
	if !ok {
		return nil
	}

	previousRoot := cloneNode(k.root)
	entries := make([]entry, 0)
	collect(k.root, &entries)
	k.root = newNode(true)
	for _, item := range entries {
		if item.key != key {
			set(k, item.key, item.value)
		}
	}
	if err := k.persist(); err != nil {
		k.root = previousRoot
		return err
	}
	return nil
}

func (k *KV) Save(path string) error {
	if err := k.write(path); err != nil {
		return err
	}
	k.path = path
	return nil
}

func (k *KV) persist() error {
	if k.path == "" {
		return nil
	}
	return k.write(k.path)
}

func (k *KV) write(path string) error {
	data, err := json.Marshal(snapshot{Version: snapshotVersion, Root: k.root})
	if err != nil {
		return fmt.Errorf("encode KV snapshot: %w", err)
	}
	if err := writeSnapshot(path, data); err != nil {
		return fmt.Errorf("write KV snapshot %q: %w", path, err)
	}
	return nil
}

func Load(path string) (*KV, error) {
	data, err := readSnapshot(path)
	if err != nil {
		if errors.Is(err, errSnapshotNotFound) {
			store := NewKV()
			store.path = path
			return store, nil
		}
		return nil, fmt.Errorf("read KV snapshot %q: %w", path, err)
	}

	var saved snapshot
	if err := json.Unmarshal(data, &saved); err != nil {
		return nil, fmt.Errorf("decode KV snapshot %q: %w", path, err)
	}
	if saved.Version != snapshotVersion {
		return nil, fmt.Errorf("unsupported KV snapshot version %d", saved.Version)
	}
	if err := validateNode(saved.Root, nil, nil); err != nil {
		return nil, fmt.Errorf("invalid KV snapshot %q: %w", path, err)
	}
	return &KV{root: saved.Root, path: path}, nil
}

var errSnapshotNotFound = errors.New("snapshot not found")
