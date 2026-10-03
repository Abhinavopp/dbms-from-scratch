package chapter01

import (
	"os"
)

// AppendLog appends a line to the log file.
func AppendLog(path, line string) error {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.WriteString(line + "\n")
	return err
}
