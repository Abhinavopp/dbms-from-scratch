package chapter01

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// FileDB is a tiny durable store that serializes a value to disk.
type FileDB struct {
	Path string
}

func NewFileDB(path string) (*FileDB, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	return &FileDB{Path: path}, nil
}

func (db *FileDB) Save(value any) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return WriteAtomic(db.Path, data)
}

func (db *FileDB) Load(value any) error {
	data, err := os.ReadFile(db.Path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	return json.Unmarshal(data, value)
}
