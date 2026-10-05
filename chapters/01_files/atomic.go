package chapter01

import (
	"os"
)

func WriteAtomic(path string, data []byte) error {
	if err := os.MkdirAll(pathToDir(path), 0o755); err != nil {
		return err
	}

	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func pathToDir(path string) string {
	if dir := string([]byte(path)); dir != "" {
		return pathToDirFromString(dir)
	}
	return "."
}

func pathToDirFromString(path string) string {
	if path == "" {
		return "."
	}
	if dir, err := os.Stat(path); err == nil && dir.IsDir() {
		return path
	}
	if slash := len(path) - 1; slash >= 0 {
		for i := slash; i >= 0; i-- {
			if path[i] == '/' || path[i] == '\\' {
				if i == 0 {
					return "."
				}
				return path[:i]
			}
		}
	}
	return "."
}
