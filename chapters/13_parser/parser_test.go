package chapter13

import "testing"

func TestParserSelect(t *testing.T) {
	stmt, err := NewParser("SELECT name, email FROM users WHERE role = admin").Parse()
	if err != nil {
		t.Fatal(err)
	}
	if stmt.Kind != "SELECT" {
		t.Fatalf("expected SELECT, got %s", stmt.Kind)
	}
	if stmt.Table != "users" {
		t.Fatalf("expected table users, got %s", stmt.Table)
	}
	if len(stmt.Columns) != 2 {
		t.Fatalf("expected 2 columns, got %d", len(stmt.Columns))
	}
}

func TestParserInsert(t *testing.T) {
	stmt, err := NewParser("INSERT INTO users (name, role) VALUES ('alice', 'admin')").Parse()
	if err != nil {
		t.Fatal(err)
	}
	if stmt.Kind != "INSERT" {
		t.Fatalf("expected INSERT, got %s", stmt.Kind)
	}
	if stmt.Table != "users" {
		t.Fatalf("expected table users, got %s", stmt.Table)
	}
	if len(stmt.InsertColumns) != 2 || len(stmt.Values) != 2 {
		t.Fatalf("expected two insert columns and values, got %+v", stmt)
	}
}

func TestParserDDLStatements(t *testing.T) {
	tests := []struct {
		sql         string
		kind        string
		table       string
		alterAction string
	}{
		{"CREATE TABLE products (id INTEGER, name VARCHAR(80))", "CREATE", "products", ""},
		{"DROP TABLE IF EXISTS products", "DROP", "products", ""},
		{"ALTER TABLE products ADD COLUMN price DECIMAL(8,2)", "ALTER", "products", "ADD COLUMN"},
		{"ALTER TABLE products DROP COLUMN price", "ALTER", "products", "DROP COLUMN"},
		{"ALTER TABLE products RENAME COLUMN name TO title", "ALTER", "products", "RENAME COLUMN"},
		{"ALTER TABLE products RENAME TO inventory", "ALTER", "products", "RENAME TABLE"},
	}

	for _, test := range tests {
		t.Run(test.sql, func(t *testing.T) {
			stmt, err := NewParser(test.sql).Parse()
			if err != nil {
				t.Fatal(err)
			}
			if stmt.Kind != test.kind || stmt.Table != test.table || stmt.AlterAction != test.alterAction {
				t.Fatalf("unexpected parsed statement: %+v", stmt)
			}
		})
	}
}

func TestParserSelectConditionsAndInsertColumns(t *testing.T) {
	stmt, err := NewParser(`SELECT id, name FROM products WHERE price >= 10 AND name != 'out of stock';`).Parse()
	if err != nil {
		t.Fatal(err)
	}
	if len(stmt.Conditions) != 2 || stmt.Conditions[0].Operator != ">=" || stmt.Conditions[1].Value != "out of stock" {
		t.Fatalf("unexpected conditions: %+v", stmt.Conditions)
	}

	insert, err := NewParser(`INSERT INTO products (id, name) VALUES (7, 'desk lamp')`).Parse()
	if err != nil {
		t.Fatal(err)
	}
	if len(insert.InsertColumns) != 2 || len(insert.Values) != 2 || insert.Values[1] != "desk lamp" {
		t.Fatalf("unexpected insert statement: %+v", insert)
	}
}

func TestParserUpdate(t *testing.T) {
	stmt, err := NewParser(`UPDATE employees SET salary = 95000, department = 'Research' WHERE id >= 10 AND active = true;`).Parse()
	if err != nil {
		t.Fatal(err)
	}
	if stmt.Kind != "UPDATE" || stmt.Table != "employees" {
		t.Fatalf("unexpected update target: %+v", stmt)
	}
	if len(stmt.Assignments) != 2 ||
		stmt.Assignments[0] != (Assignment{Column: "salary", Value: "95000"}) ||
		stmt.Assignments[1] != (Assignment{Column: "department", Value: "Research"}) {
		t.Fatalf("unexpected assignments: %+v", stmt.Assignments)
	}
	if len(stmt.Conditions) != 2 ||
		stmt.Conditions[0] != (Condition{Column: "id", Operator: ">=", Value: "10"}) ||
		stmt.Conditions[1] != (Condition{Column: "active", Operator: "=", Value: "true"}) {
		t.Fatalf("unexpected conditions: %+v", stmt.Conditions)
	}
}

func TestParserKeywordsAreCaseInsensitive(t *testing.T) {
	tests := []struct {
		sql  string
		kind string
	}{
		{"sElEcT * fRoM users wHeRe id = 1 aNd active = true", "SELECT"},
		{"iNsErT iNtO users (id, name) vAlUeS (1, 'Ava')", "INSERT"},
		{"cReAtE tAbLe users (id INTEGER, name TEXT)", "CREATE"},
		{"dRoP tAbLe iF eXiStS users", "DROP"},
		{"aLtEr tAbLe users aDd cOlUmN email TEXT", "ALTER"},
		{"uPdAtE users sEt name = 'Ava' wHeRe id = 1 aNd active = true", "UPDATE"},
	}

	for _, test := range tests {
		t.Run(test.kind, func(t *testing.T) {
			stmt, err := NewParser(test.sql).Parse()
			if err != nil {
				t.Fatal(err)
			}
			if stmt.Kind != test.kind {
				t.Fatalf("expected statement kind %s, got %s", test.kind, stmt.Kind)
			}
		})
	}
}

func TestParserRejectsMalformedStatements(t *testing.T) {
	for _, sql := range []string{
		"CREATE TABLE products (id)",
		"ALTER TABLE products ADD",
		"SELECT * products",
		`SELECT * FROM products WHERE name = 'unfinished`,
		"UPDATE employees SET WHERE id = 1",
		"UPDATE employees SET salary 100 WHERE id = 1",
		"UPDATE employees SET salary = 100 WHERE id =",
	} {
		t.Run(sql, func(t *testing.T) {
			if _, err := NewParser(sql).Parse(); err == nil {
				t.Fatalf("expected %q to fail parsing", sql)
			}
		})
	}
}
