package main

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	query "dbmsfromscratch/chapters/14_query"
)

type requestBody struct {
	SQL string `json:"sql"`
}

func httpServer(port string) error {
	executor := query.NewExecutor()
	mux := http.NewServeMux()

	mux.HandleFunc("/api/schema", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]any{"tables": executor.Tables()}); err != nil {
			http.Error(w, "unable to encode schema response", http.StatusInternalServerError)
		}
	})

	mux.HandleFunc("/api/query", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "unable to read request", http.StatusBadRequest)
			return
		}

		var payload requestBody
		if err := json.Unmarshal(body, &payload); err != nil {
			http.Error(w, "valid JSON body required", http.StatusBadRequest)
			return
		}

		result, err := executor.Execute(strings.TrimSpace(payload.SQL))
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			if encodeErr := json.NewEncoder(w).Encode(map[string]string{"error": err.Error()}); encodeErr != nil {
				http.Error(w, "unable to encode error response", http.StatusInternalServerError)
			}
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]any{
			"ok":      true,
			"columns": result.Columns,
			"rows":    result.Rows,
		}); err != nil {
			http.Error(w, "unable to encode response", http.StatusInternalServerError)
		}
	})

	root, err := os.Getwd()
	if err != nil {
		return err
	}
	webRoot := filepath.Join(root, "cmd", "db", "web")
	mux.Handle("/", http.FileServer(http.Dir(webRoot)))
	return http.ListenAndServe("0.0.0.0:"+port, mux)
}
