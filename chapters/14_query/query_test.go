package chapter14

import "testing"

func TestExecutorSelectAndInsert(t *testing.T) {
	exec := NewExecutor()
	_, err := exec.Execute("CREATE TABLE users (name TEXT, role TEXT)")
	if err != nil {
		t.Fatal(err)
	}
	_, err = exec.Execute("INSERT INTO users (name, role) VALUES ('alice', 'admin')")
	if err != nil {
		t.Fatal(err)
	}
	result, err := exec.Execute("SELECT name FROM users WHERE role = admin")
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Rows) != 1 {
		t.Fatalf("expected 1 result row, got %d", len(result.Rows))
	}
	if result.Rows[0]["name"] != "alice" {
		t.Fatalf("expected alice in result, got %q", result.Rows[0]["name"])
	}
	if _, ok := result.Rows[0]["role"]; ok {
		t.Fatal("expected selected column projection to omit role")
	}
}

func TestExecutorSupportsDynamicDDLAndSelect(t *testing.T) {
	exec := NewExecutor()
	statements := []string{
		"CREATE TABLE inventory (sku TEXT, description VARCHAR(100), price DECIMAL(8,2), active BOOLEAN)",
		`INSERT INTO inventory (sku, description, price, active) VALUES ('A-1', 'desk lamp', 19.50, true)`,
		`INSERT INTO inventory (sku, description, price, active) VALUES ('B-2', 'chair', 45.00, false)`,
	}
	for _, sql := range statements {
		if _, err := exec.Execute(sql); err != nil {
			t.Fatalf("%s: %v", sql, err)
		}
	}

	result, err := exec.Execute(`SELECT sku, description FROM inventory WHERE price >= 20 AND active = false`)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Rows) != 1 || result.Rows[0]["sku"] != "B-2" || result.Rows[0]["description"] != "chair" {
		t.Fatalf("unexpected SELECT result: %+v", result.Rows)
	}
	if _, exists := result.Rows[0]["price"]; exists {
		t.Fatal("SELECT projection should omit unrequested columns")
	}
}

func TestExecutorAlterAndDropTable(t *testing.T) {
	exec := NewExecutor()
	for _, sql := range []string{
		"CREATE TABLE gadgets (id INTEGER, name TEXT)",
		`INSERT INTO gadgets (id, name) VALUES (1, 'radio')`,
		"ALTER TABLE gadgets ADD COLUMN price DECIMAL(6,2)",
		"ALTER TABLE gadgets RENAME COLUMN name TO model",
	} {
		if _, err := exec.Execute(sql); err != nil {
			t.Fatalf("%s: %v", sql, err)
		}
	}

	result, err := exec.Execute("SELECT * FROM gadgets")
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Rows) != 1 || result.Rows[0]["model"] != "radio" || result.Rows[0]["price"] != "" {
		t.Fatalf("unexpected row after ADD/RENAME: %+v", result.Rows)
	}

	if _, err := exec.Execute("ALTER TABLE gadgets DROP COLUMN id"); err != nil {
		t.Fatal(err)
	}
	if _, err := exec.Execute("ALTER TABLE gadgets RENAME TO devices"); err != nil {
		t.Fatal(err)
	}
	if _, err := exec.Execute("SELECT * FROM gadgets"); err == nil {
		t.Fatal("expected the old table name to stop resolving")
	}
	if _, err := exec.Execute("SELECT * FROM devices"); err != nil {
		t.Fatal(err)
	}
	if _, err := exec.Execute("DROP TABLE devices"); err != nil {
		t.Fatal(err)
	}
	if _, err := exec.Execute("SELECT * FROM devices"); err == nil {
		t.Fatal("expected dropped table to stop resolving")
	}
}

func TestExecutorReportsSchemaErrorsWithoutMutatingTable(t *testing.T) {
	exec := NewExecutor()
	if _, err := exec.Execute("CREATE TABLE accounts (id INTEGER)"); err != nil {
		t.Fatal(err)
	}
	if _, err := exec.Execute("CREATE TABLE accounts (name TEXT)"); err == nil {
		t.Fatal("expected duplicate table creation to fail")
	}
	if _, err := exec.Execute("ALTER TABLE accounts DROP COLUMN id"); err == nil {
		t.Fatal("expected dropping the last column to fail")
	}
	if _, err := exec.Execute("SELECT missing FROM accounts"); err == nil {
		t.Fatal("expected unknown selected column to fail")
	}
	if _, err := exec.Execute("SELECT * FROM accounts"); err != nil {
		t.Fatalf("failed DDL should leave the original table usable: %v", err)
	}
}

func TestExecutorUpdateWithWhereAndMultipleAssignments(t *testing.T) {
	exec := NewExecutor()
	for _, sql := range []string{
		"CREATE TABLE employees (id INTEGER, name TEXT, department TEXT, salary DECIMAL(10,2), active BOOLEAN)",
		`INSERT INTO employees (id, name, department, salary, active) VALUES (101, 'Ava', 'Engineering', 105000, true)`,
		`INSERT INTO employees (id, name, department, salary, active) VALUES (102, 'Noah', 'Sales', 82000, true)`,
		`INSERT INTO employees (id, name, department, salary, active) VALUES (103, 'Mia', 'Engineering', 94000, false)`,
	} {
		if _, err := exec.Execute(sql); err != nil {
			t.Fatalf("%s: %v", sql, err)
		}
	}

	updated, err := exec.Execute(`UPDATE employees SET salary = 100000, department = 'Research' WHERE department = 'Engineering' AND active = true`)
	if err != nil {
		t.Fatal(err)
	}
	if len(updated.Rows) != 1 ||
		updated.Rows[0]["id"] != "101" ||
		updated.Rows[0]["salary"] != "100000" ||
		updated.Rows[0]["department"] != "Research" {
		t.Fatalf("unexpected UPDATE result: %+v", updated.Rows)
	}
	if len(updated.Columns) != 5 {
		t.Fatalf("expected full updated row columns, got %v", updated.Columns)
	}

	selected, err := exec.Execute("SELECT id, salary, department FROM employees WHERE id = 101")
	if err != nil {
		t.Fatal(err)
	}
	if len(selected.Rows) != 1 || selected.Rows[0]["salary"] != "100000" || selected.Rows[0]["department"] != "Research" {
		t.Fatalf("UPDATE was not reflected in table data: %+v", selected.Rows)
	}
	unchanged, err := exec.Execute("SELECT salary, department FROM employees WHERE id = 103")
	if err != nil {
		t.Fatal(err)
	}
	if len(unchanged.Rows) != 1 || unchanged.Rows[0]["salary"] != "94000" || unchanged.Rows[0]["department"] != "Engineering" {
		t.Fatalf("UPDATE modified a non-matching row: %+v", unchanged.Rows)
	}
}

func TestExecutorUpdateWithoutWhereUpdatesAllRows(t *testing.T) {
	exec := NewExecutor()
	for _, sql := range []string{
		"CREATE TABLE flags (id INTEGER, enabled BOOLEAN)",
		`INSERT INTO flags (id, enabled) VALUES (1, false)`,
		`INSERT INTO flags (id, enabled) VALUES (2, false)`,
	} {
		if _, err := exec.Execute(sql); err != nil {
			t.Fatalf("%s: %v", sql, err)
		}
	}

	result, err := exec.Execute("UPDATE flags SET enabled = true")
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Rows) != 2 {
		t.Fatalf("expected both rows to be returned as updated, got %d", len(result.Rows))
	}
	for _, row := range result.Rows {
		if row["enabled"] != "true" {
			t.Fatalf("expected all rows to be enabled, got %+v", row)
		}
	}
}

func TestExecutorUpdateNoMatchAndValidationAreSafe(t *testing.T) {
	exec := NewExecutor()
	for _, sql := range []string{
		"CREATE TABLE accounts (id INTEGER, status TEXT)",
		`INSERT INTO accounts (id, status) VALUES (1, 'pending')`,
	} {
		if _, err := exec.Execute(sql); err != nil {
			t.Fatalf("%s: %v", sql, err)
		}
	}

	noMatch, err := exec.Execute("UPDATE accounts SET status = 'active' WHERE id = 99")
	if err != nil {
		t.Fatal(err)
	}
	if len(noMatch.Rows) != 0 {
		t.Fatalf("expected no updated rows, got %+v", noMatch.Rows)
	}

	for _, sql := range []string{
		"UPDATE missing SET status = 'active'",
		"UPDATE accounts SET missing = 'active'",
		"UPDATE accounts SET status = 'active' WHERE missing = 1",
		"UPDATE accounts SET status = 'active', STATUS = 'disabled'",
	} {
		if _, err := exec.Execute(sql); err == nil {
			t.Fatalf("expected %q to fail", sql)
		}
	}

	unchanged, err := exec.Execute("SELECT status FROM accounts WHERE id = 1")
	if err != nil {
		t.Fatal(err)
	}
	if len(unchanged.Rows) != 1 || unchanged.Rows[0]["status"] != "pending" {
		t.Fatalf("invalid UPDATE partially changed data: %+v", unchanged.Rows)
	}
}

func TestExecutorKeywordsAreCaseInsensitive(t *testing.T) {
	exec := NewExecutor()
	for _, sql := range []string{
		"cReAtE tAbLe people (id INTEGER, name TEXT, active BOOLEAN)",
		"iNsErT iNtO people (id, name, active) vAlUeS (1, 'Ava', false)",
		"uPdAtE people sEt name = 'Ava Patel', active = true wHeRe id = 1 aNd active = false",
	} {
		if _, err := exec.Execute(sql); err != nil {
			t.Fatalf("%s: %v", sql, err)
		}
	}

	result, err := exec.Execute("sElEcT name, active fRoM people wHeRe id = 1")
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Rows) != 1 || result.Rows[0]["name"] != "Ava Patel" || result.Rows[0]["active"] != "true" {
		t.Fatalf("mixed-case SQL keywords did not execute correctly: %+v", result.Rows)
	}

	if _, err := exec.Execute("aLtEr tAbLe people aDd cOlUmN city TEXT"); err != nil {
		t.Fatal(err)
	}
	if _, err := exec.Execute("dRoP tAbLe iF eXiStS people"); err != nil {
		t.Fatal(err)
	}
}
