package chapter14

import "testing"

func TestExecutorExpressionsAndUpdateCalculations(t *testing.T) {
	exec := NewExecutor()
	for _, sql := range []string{
		"CREATE TABLE employees (id INTEGER PRIMARY KEY, name TEXT, age INTEGER, salary DECIMAL(10,2), active BOOLEAN)",
		`INSERT INTO employees (id, name, age, salary, active) VALUES (1, 'Abhinav', 25, 100, true), (2, 'Mia', 19, 200, false), (3, 'Noah', 35, 300, true)`,
	} {
		if _, err := exec.Execute(sql); err != nil {
			t.Fatalf("%s: %v", sql, err)
		}
	}

	result, err := exec.Execute(`SELECT name, age + 2 * 3 AS adjusted, salary / 2 AS half_salary FROM employees FILTER age >= 20 AND (active OR NOT active)`)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Rows) != 2 || result.Rows[0]["adjusted"] != "31" ||
		result.Rows[0]["name"] != "Abhinav" || result.Rows[0]["half_salary"] != "50" ||
		result.Rows[1]["adjusted"] != "41" {
		t.Fatalf("unexpected expression/filter results: %+v", result.Rows)
	}
	arithmetic, err := exec.Execute("SELECT age - 5 / 5 AS adjusted FROM employees WHERE id = 1")
	if err != nil {
		t.Fatal(err)
	}
	if len(arithmetic.Rows) != 1 || arithmetic.Rows[0]["adjusted"] != "24" {
		t.Fatalf("subtraction/division precedence was incorrect: %+v", arithmetic.Rows)
	}

	updated, err := exec.Execute("UPDATE employees SET salary = salary + 50, age = -age WHERE id = 1")
	if err != nil {
		t.Fatal(err)
	}
	if len(updated.Rows) != 1 || updated.Rows[0]["salary"] != "150" || updated.Rows[0]["age"] != "-25" {
		t.Fatalf("UPDATE expressions were not evaluated against the existing row: %+v", updated.Rows)
	}
}

func TestExecutorIndexByRangeReverseAndFilter(t *testing.T) {
	exec := NewExecutor()
	for _, sql := range []string{
		"CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT, age INTEGER, INDEX age_idx (age))",
		`INSERT INTO users (id, name, age) VALUES (1, 'Abhinav', 18), (2, 'Bea', 21), (3, 'Cal', 30), (4, 'Dee', 42)`,
	} {
		if _, err := exec.Execute(sql); err != nil {
			t.Fatalf("%s: %v", sql, err)
		}
	}

	tests := []struct {
		query string
		want  []string
	}{
		{"SELECT name FROM users INDEX BY age = 30", []string{"Cal"}},
		{"SELECT name FROM users INDEX BY age > 20 AND age < 40", []string{"Bea", "Cal"}},
		{"SELECT name FROM users INDEX BY age > 20", []string{"Bea", "Cal", "Dee"}},
		{"SELECT name FROM users INDEX BY 40 > age AND 20 < age", []string{"Bea", "Cal"}},
		{`SELECT name FROM users INDEX BY age > 20 FILTER name != 'Cal'`, []string{"Bea", "Dee"}},
		{`SELECT name FROM users INDEX BY age > 20 FILTER name = "Cal"`, []string{"Cal"}},
	}
	for _, test := range tests {
		result, err := exec.Execute(test.query)
		if err != nil {
			t.Fatalf("%s: %v", test.query, err)
		}
		if len(result.Rows) != len(test.want) {
			t.Fatalf("%s: expected %d rows, got %+v", test.query, len(test.want), result.Rows)
		}
		for i, value := range test.want {
			if result.Rows[i]["name"] != value {
				t.Fatalf("%s: expected row %d to be %q, got %+v", test.query, i, value, result.Rows)
			}
		}
	}
}

func TestExecutorLimitAndOffset(t *testing.T) {
	exec := NewExecutor()
	for _, sql := range []string{
		"CREATE TABLE numbers (id INTEGER, value TEXT)",
		`INSERT INTO numbers VALUES (1, 'a'), (2, 'b'), (3, 'c'), (4, 'd')`,
	} {
		if _, err := exec.Execute(sql); err != nil {
			t.Fatalf("%s: %v", sql, err)
		}
	}
	result, err := exec.Execute("SELECT value FROM numbers LIMIT 2")
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Rows) != 2 || result.Rows[1]["value"] != "b" {
		t.Fatalf("LIMIT did not bound rows: %+v", result.Rows)
	}
	result, err = exec.Execute("SELECT value FROM numbers LIMIT 1, 2")
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Rows) != 2 || result.Rows[0]["value"] != "b" || result.Rows[1]["value"] != "c" {
		t.Fatalf("LIMIT offset was not applied: %+v", result.Rows)
	}
}

func TestExecutorDeleteAndPrimaryKeyEnforcement(t *testing.T) {
	exec := NewExecutor()
	for _, sql := range []string{
		"CREATE TABLE accounts (id INTEGER, status TEXT, PRIMARY KEY (id))",
		`INSERT INTO accounts VALUES (1, 'active'), (2, 'pending')`,
	} {
		if _, err := exec.Execute(sql); err != nil {
			t.Fatalf("%s: %v", sql, err)
		}
	}
	if _, err := exec.Execute("INSERT INTO accounts VALUES (1, 'duplicate')"); err == nil {
		t.Fatal("expected duplicate primary key to fail")
	}
	if _, err := exec.Execute("UPDATE accounts SET id = 2 WHERE id = 1"); err == nil {
		t.Fatal("expected update to violate the primary key")
	}
	deleted, err := exec.Execute("DELETE FROM accounts WHERE status = 'pending'")
	if err != nil {
		t.Fatal(err)
	}
	if len(deleted.Rows) != 1 || deleted.Rows[0]["id"] != "2" {
		t.Fatalf("unexpected DELETE result: %+v", deleted.Rows)
	}
	remaining, err := exec.Execute("SELECT * FROM accounts")
	if err != nil {
		t.Fatal(err)
	}
	if len(remaining.Rows) != 1 || remaining.Rows[0]["id"] != "1" {
		t.Fatalf("DELETE did not remove the matching row: %+v", remaining.Rows)
	}
}

func TestExecutorReportsQueryErrors(t *testing.T) {
	exec := NewExecutor()
	if _, err := exec.Execute("CREATE TABLE metrics (id INTEGER, name TEXT, INDEX (id))"); err != nil {
		t.Fatal(err)
	}
	if _, err := exec.Execute("INSERT INTO metrics VALUES (1, 'one')"); err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		operator string
		value    string
		wantRows int
	}{
		{"=", "1", 1}, {"<", "1", 0}, {">", "1", 0},
		{"<=", "1", 1}, {">=", "1", 1}, {"!=", "2", 1},
	} {
		result, err := exec.Execute("SELECT id FROM metrics WHERE id " + test.operator + " " + test.value)
		if err != nil {
			t.Fatalf("operator %s: %v", test.operator, err)
		}
		if len(result.Rows) != test.wantRows {
			t.Fatalf("operator %s: expected %d rows, got %+v", test.operator, test.wantRows, result.Rows)
		}
	}
	for _, sql := range []string{
		"SELECT * FROM absent",
		"SELECT absent FROM metrics",
		"SELECT * FROM metrics LIMIT -1",
		"SELECT * FROM metrics LIMIT not_a_number",
		"SELECT * FROM metrics INDEX BY name = 'one'",
		"SELECT * FROM metrics FILTER id + 'bad'",
		"UPDATE metrics SET id = id / 0",
		"UPDATE metrics SET id = absent WHERE id = 99",
	} {
		t.Run(sql, func(t *testing.T) {
			if _, err := exec.Execute(sql); err == nil {
				t.Fatalf("expected %q to return an error", sql)
			}
		})
	}
}
