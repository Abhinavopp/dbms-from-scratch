package chapter06

import (
	"encoding/json"
	"os"
)

type KV struct {
	data map[string]string
}

func NewKV() *KV {
	return &KV{data: make(map[string]string)}
}

func (k *KV) Set(key, value string) {
	if k.data == nil {
		k.data = make(map[string]string)
	}
	k.data[key] = value
}

func (k *KV) Get(key string) (string, bool) {
	value, ok := k.data[key]
	return value, ok
}

func (k *KV) Delete(key string) {
	delete(k.data, key)
}

func (k *KV) Save(path string) error {
	data, err := json.Marshal(k.data)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func Load(path string) (*KV, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return NewKV(), nil
		}
		return nil, err
	}
	store := NewKV()
	if err := json.Unmarshal(data, &store.data); err != nil {
		return nil, err
	}
	return store, nil
}
