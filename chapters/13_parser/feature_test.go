package chapter13

import "testing"

func TestParserBuildsExpressionTreeWithPrecedence(t *testing.T) {
	stmt, err := NewParser("SELECT 2 + 3 * 4 AS total FROM numbers").Parse()
	if err != nil {
		t.Fatal(err)
	}
	expression := stmt.SelectItems[0].Expression
	if expression.Operator != "+" || expression.Left.Value != "2" ||
		expression.Right.Operator != "*" || expression.Right.Left.Value != "3" ||
		expression.Right.Right.Value != "4" {
		t.Fatalf("unexpected arithmetic precedence tree: %#v", expression)
	}

	stmt, err = NewParser("SELECT a OR b AND c AS result FROM flags").Parse()
	if err != nil {
		t.Fatal(err)
	}
	expression = stmt.SelectItems[0].Expression
	if expression.Operator != "OR" || expression.Right.Operator != "AND" {
		t.Fatalf("unexpected boolean precedence tree: %#v", expression)
	}
}

func TestParserQueryClausesAndStatements(t *testing.T) {
	stmt, err := NewParser(`SELECT name, age + 1 AS next_age FROM users INDEX BY age > 20 AND age < 40 LIMIT 3, 5 FILTER NOT (name = 'skip' OR age < 25)`).Parse()
	if err != nil {
		t.Fatal(err)
	}
	if len(stmt.SelectItems) != 2 || stmt.SelectItems[1].Alias != "next_age" ||
		stmt.IndexBy == nil || stmt.Limit == nil || stmt.Offset == nil || stmt.Filter == nil {
		t.Fatalf("query clauses or select expressions were not parsed: %+v", stmt)
	}

	create, err := NewParser("CREATE TABLE users (id INTEGER PRIMARY KEY, age INTEGER, INDEX idx_age (age))").Parse()
	if err != nil {
		t.Fatal(err)
	}
	if create.PrimaryKey != "id" || len(create.Indexes) != 1 || create.Indexes[0].Column != "age" {
		t.Fatalf("primary key/index declarations were not parsed: %+v", create)
	}

	insert, err := NewParser("INSERT INTO users (id, age) VALUES (1, 20), (2, 30)").Parse()
	if err != nil {
		t.Fatal(err)
	}
	if len(insert.ValueRows) != 2 || len(insert.ValueRows[1]) != 2 {
		t.Fatalf("multi-row INSERT was not parsed: %+v", insert.ValueRows)
	}

	update, err := NewParser("UPDATE users SET age = age + 1 WHERE id = 1").Parse()
	if err != nil {
		t.Fatal(err)
	}
	if update.Assignments[0].Expression == nil || update.Where == nil {
		t.Fatalf("UPDATE expressions were not parsed: %+v", update)
	}

	deletion, err := NewParser("DELETE FROM users WHERE age >= 30").Parse()
	if err != nil {
		t.Fatal(err)
	}
	if deletion.Kind != "DELETE" || deletion.Table != "users" || deletion.Where == nil {
		t.Fatalf("DELETE statement was not parsed: %+v", deletion)
	}
}

func TestParserRejectsInvalidExpressionAndLimitSyntax(t *testing.T) {
	for _, sql := range []string{
		"SELECT 1 + FROM numbers",
		"SELECT * FROM numbers LIMIT",
		"DELETE users",
		"UPDATE users SET age = 1 +",
	} {
		t.Run(sql, func(t *testing.T) {
			if _, err := NewParser(sql).Parse(); err == nil {
				t.Fatalf("expected %q to fail parsing", sql)
			}
		})
	}
}
