package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	query "dbmsfromscratch/chapters/14_query"
)

func TestQueryAPIUsesPersistentExecutor(t *testing.T) {
	path := filepath.Join(t.TempDir(), "database.json")
	executor, err := query.OpenExecutor(path)
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(newHandler(executor, t.TempDir()))
	defer server.Close()

	for _, sql := range []string{
		"CREATE TABLE notes (id INTEGER PRIMARY KEY, body TEXT)",
		`INSERT INTO notes VALUES (1, 'saved through HTTP')`,
	} {
		body, err := json.Marshal(requestBody{SQL: sql})
		if err != nil {
			t.Fatal(err)
		}
		response, err := http.Post(server.URL+"/api/query", "application/json", bytes.NewReader(body))
		if err != nil {
			t.Fatal(err)
		}
		response.Body.Close()
		if response.StatusCode != http.StatusOK {
			t.Fatalf("%s returned HTTP %d", sql, response.StatusCode)
		}
	}

	reopened, err := query.OpenExecutor(path)
	if err != nil {
		t.Fatal(err)
	}
	result, err := reopened.Execute("SELECT body FROM notes WHERE id = 1")
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Rows) != 1 || result.Rows[0]["body"] != "saved through HTTP" {
		t.Fatalf("expected HTTP-inserted row after reopening, got %+v", result.Rows)
	}
}
