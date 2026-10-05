package chapter14

import (
	"path/filepath"
	"testing"
)

func TestOpenExecutorPersistsParsedSQLChanges(t *testing.T) {
	path := filepath.Join(t.TempDir(), "data", "database.json")
	executor, err := OpenExecutor(path)
	if err != nil {
		t.Fatal(err)
	}

	statements := []string{
		"CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT, age INTEGER, INDEX age_idx (age))",
		`INSERT INTO users (id, name, age) VALUES (1, 'Ari', 20), (2, 'Bea', 30)`,
		"UPDATE users SET age = age + 1 WHERE id = 1",
		"ALTER TABLE users ADD COLUMN active BOOLEAN",
		"UPDATE users SET active = true WHERE id = 1",
		"DELETE FROM users WHERE id = 2",
		"CREATE TABLE temporary (id INTEGER)",
		"DROP TABLE temporary",
	}
	for _, sql := range statements {
		if _, err := executor.Execute(sql); err != nil {
			t.Fatalf("%s: %v", sql, err)
		}
	}

	reopened, err := OpenExecutor(path)
	if err != nil {
		t.Fatal(err)
	}
	result, err := reopened.Execute("SELECT id, name, age, active FROM users INDEX BY age >= 20")
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Rows) != 1 {
		t.Fatalf("expected one persisted row, got %+v", result.Rows)
	}
	want := map[string]string{"id": "1", "name": "Ari", "age": "21", "active": "true"}
	for column, value := range want {
		if result.Rows[0][column] != value {
			t.Errorf("expected %s=%q, got %q", column, value, result.Rows[0][column])
		}
	}
	if tables := reopened.Tables(); len(tables) != 1 || tables[0].Name != "users" {
		t.Fatalf("expected only persisted users table, got %+v", tables)
	}
}
