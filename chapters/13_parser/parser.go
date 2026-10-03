package chapter13

import (
	"fmt"
	"strings"
	"unicode"
)

// ColumnDefinition describes a column declared by CREATE TABLE or ALTER TABLE.
type ColumnDefinition struct {
	Name string
	Type string
}

// Condition is one comparison in a WHERE clause.
type Condition struct {
	Column   string
	Operator string
	Value    string
}

// Assignment is one column/value pair in an UPDATE statement.
type Assignment struct {
	Column string
	Value  string
}

// Statement is the parsed representation of the SQL subset supported here.
type Statement struct {
	Kind          string
	Table         string
	Columns       []string
	Values        []string
	Assignments   []Assignment
	InsertColumns []string
	Definitions   []ColumnDefinition
	AlterAction   string
	ColumnName    string
	ColumnType    string
	NewName       string
	IfExists      bool
	IfNotExists   bool
	Condition     string
	Conditions    []Condition
}

type token struct {
	text string
	kind byte
}

// Parser parses a single SQL statement.
type Parser struct {
	tokens   []token
	pos      int
	parseErr error
}

func NewParser(sql string) *Parser {
	tokens, err := tokenize(sql)
	return &Parser{tokens: tokens, parseErr: err}
}

func (p *Parser) Parse() (Statement, error) {
	if p.parseErr != nil {
		return Statement{}, p.parseErr
	}

	var stmt Statement
	switch {
	case p.takeKeyword("SELECT"):
		stmt.Kind = "SELECT"
		if err := p.parseSelect(&stmt); err != nil {
			return Statement{}, err
		}
	case p.takeKeyword("INSERT"):
		stmt.Kind = "INSERT"
		if err := p.parseInsert(&stmt); err != nil {
			return Statement{}, err
		}
	case p.takeKeyword("CREATE"):
		stmt.Kind = "CREATE"
		if err := p.parseCreate(&stmt); err != nil {
			return Statement{}, err
		}
	case p.takeKeyword("DROP"):
		stmt.Kind = "DROP"
		if err := p.parseDrop(&stmt); err != nil {
			return Statement{}, err
		}
	case p.takeKeyword("ALTER"):
		stmt.Kind = "ALTER"
		if err := p.parseAlter(&stmt); err != nil {
			return Statement{}, err
		}
	case p.takeKeyword("UPDATE"):
		stmt.Kind = "UPDATE"
		if err := p.parseUpdate(&stmt); err != nil {
			return Statement{}, err
		}
	case p.takeKeyword("DELETE"):
		stmt.Kind = "DELETE"
		return Statement{}, fmt.Errorf("DELETE is not supported by this parser yet")
	default:
		return Statement{}, fmt.Errorf("expected SELECT, INSERT, UPDATE, CREATE, DROP, or ALTER")
	}

	p.take(";")
	if p.current().kind != 0 {
		return Statement{}, fmt.Errorf("unexpected token %q", p.current().text)
	}
	return stmt, nil
}

func (p *Parser) parseUpdate(stmt *Statement) error {
	table, err := p.identifier()
	if err != nil {
		return err
	}
	stmt.Table = table
	if err := p.expectKeyword("SET"); err != nil {
		return err
	}

	for {
		column, err := p.identifier()
		if err != nil {
			return err
		}
		if err := p.expect("="); err != nil {
			return err
		}
		value, err := p.value()
		if err != nil {
			return err
		}
		stmt.Assignments = append(stmt.Assignments, Assignment{Column: column, Value: value})
		if !p.take(",") {
			break
		}
	}
	if len(stmt.Assignments) == 0 {
		return fmt.Errorf("UPDATE requires at least one assignment")
	}

	if p.takeKeyword("WHERE") {
		for {
			condition, err := p.parseCondition()
			if err != nil {
				return err
			}
			stmt.Conditions = append(stmt.Conditions, condition)
			if !p.takeKeyword("AND") {
				break
			}
		}
	}
	return nil
}

func (p *Parser) parseSelect(stmt *Statement) error {
	if p.take("*") {
		stmt.Columns = []string{"*"}
	} else {
		for {
			column, err := p.identifier()
			if err != nil {
				return err
			}
			stmt.Columns = append(stmt.Columns, column)
			if !p.take(",") {
				break
			}
		}
	}
	if err := p.expectKeyword("FROM"); err != nil {
		return err
	}
	table, err := p.identifier()
	if err != nil {
		return err
	}
	stmt.Table = table
	if p.takeKeyword("WHERE") {
		for {
			condition, err := p.parseCondition()
			if err != nil {
				return err
			}
			stmt.Conditions = append(stmt.Conditions, condition)
			if len(stmt.Condition) > 0 {
				stmt.Condition += " AND "
			}
			stmt.Condition += condition.Column + " " + condition.Operator + " " + condition.Value
			if !p.takeKeyword("AND") {
				break
			}
		}
	}
	return nil
}

func (p *Parser) parseInsert(stmt *Statement) error {
	if err := p.expectKeyword("INTO"); err != nil {
		return err
	}
	table, err := p.identifier()
	if err != nil {
		return err
	}
	stmt.Table = table

	if p.take("(") {
		for {
			column, err := p.identifier()
			if err != nil {
				return err
			}
			stmt.InsertColumns = append(stmt.InsertColumns, column)
			if !p.take(",") {
				break
			}
		}
		if err := p.expect(")"); err != nil {
			return err
		}
	}
	if err := p.expectKeyword("VALUES"); err != nil {
		return err
	}

	parenthesized := p.take("(")
	if !parenthesized {
		return fmt.Errorf("VALUES must be followed by a parenthesized value list")
	}
	for p.current().kind != 0 && p.current().text != ";" {
		value, err := p.value()
		if err != nil {
			return err
		}
		stmt.Values = append(stmt.Values, value)
		if p.take(",") {
			continue
		}
		break
	}
	if err := p.expect(")"); err != nil {
		return err
	}
	if len(stmt.Values) == 0 {
		return fmt.Errorf("INSERT requires at least one value")
	}
	return nil
}

func (p *Parser) parseCreate(stmt *Statement) error {
	if err := p.expectKeyword("TABLE"); err != nil {
		return err
	}
	if p.takeKeyword("IF") {
		if err := p.expectKeyword("NOT"); err != nil {
			return err
		}
		if err := p.expectKeyword("EXISTS"); err != nil {
			return err
		}
		stmt.IfNotExists = true
	}
	table, err := p.identifier()
	if err != nil {
		return err
	}
	stmt.Table = table
	if err := p.expect("("); err != nil {
		return err
	}
	for {
		name, err := p.identifier()
		if err != nil {
			return err
		}
		columnType, err := p.columnType()
		if err != nil {
			return err
		}
		stmt.Definitions = append(stmt.Definitions, ColumnDefinition{Name: name, Type: columnType})
		if err := p.skipColumnConstraints(); err != nil {
			return err
		}
		if p.take(")") {
			break
		}
		if err := p.expect(","); err != nil {
			return err
		}
	}
	if len(stmt.Definitions) == 0 {
		return fmt.Errorf("CREATE TABLE requires at least one column")
	}
	return nil
}

func (p *Parser) parseDrop(stmt *Statement) error {
	if err := p.expectKeyword("TABLE"); err != nil {
		return err
	}
	if p.takeKeyword("IF") {
		if err := p.expectKeyword("EXISTS"); err != nil {
			return err
		}
		stmt.IfExists = true
	}
	table, err := p.identifier()
	if err != nil {
		return err
	}
	stmt.Table = table
	return nil
}

func (p *Parser) parseAlter(stmt *Statement) error {
	if err := p.expectKeyword("TABLE"); err != nil {
		return err
	}
	table, err := p.identifier()
	if err != nil {
		return err
	}
	stmt.Table = table

	switch {
	case p.takeKeyword("ADD"):
		stmt.AlterAction = "ADD COLUMN"
		p.takeKeyword("COLUMN")
		stmt.ColumnName, err = p.identifier()
		if err != nil {
			return err
		}
		stmt.ColumnType, err = p.columnType()
		if err != nil {
			return err
		}
		return p.skipColumnConstraints()
	case p.takeKeyword("DROP"):
		stmt.AlterAction = "DROP COLUMN"
		p.takeKeyword("COLUMN")
		stmt.ColumnName, err = p.identifier()
		return err
	case p.takeKeyword("RENAME"):
		if p.takeKeyword("COLUMN") {
			stmt.AlterAction = "RENAME COLUMN"
			stmt.ColumnName, err = p.identifier()
			if err != nil {
				return err
			}
			if err = p.expectKeyword("TO"); err != nil {
				return err
			}
			stmt.NewName, err = p.identifier()
			return err
		}
		stmt.AlterAction = "RENAME TABLE"
		if err = p.expectKeyword("TO"); err != nil {
			return err
		}
		stmt.NewName, err = p.identifier()
		return err
	default:
		return fmt.Errorf("expected ADD COLUMN, DROP COLUMN, or RENAME after ALTER TABLE")
	}
}

func (p *Parser) parseCondition() (Condition, error) {
	column, err := p.identifier()
	if err != nil {
		return Condition{}, err
	}
	operator := p.current().text
	switch operator {
	case "=", "!=", "<>", ">", ">=", "<", "<=":
		p.pos++
	default:
		return Condition{}, fmt.Errorf("expected a comparison operator after %q", column)
	}
	value, err := p.value()
	if err != nil {
		return Condition{}, err
	}
	return Condition{Column: column, Operator: operator, Value: value}, nil
}

func (p *Parser) columnType() (string, error) {
	first, err := p.identifier()
	if err != nil {
		return "", fmt.Errorf("expected a column type: %w", err)
	}
	parts := []string{first}
	if p.current().kind == 'w' && (strings.EqualFold(p.current().text, "precision") ||
		strings.EqualFold(p.current().text, "varying")) {
		parts = append(parts, p.current().text)
		p.pos++
	}
	if p.take("(") {
		parts[0] += "("
		for {
			if p.current().kind == 0 {
				return "", fmt.Errorf("unterminated column type parameters")
			}
			if p.take(")") {
				parts[0] += ")"
				break
			}
			parts[0] += p.current().text
			p.pos++
		}
	}
	return strings.Join(parts, " "), nil
}

func (p *Parser) skipColumnConstraints() error {
	depth := 0
	for p.current().kind != 0 {
		if depth == 0 && (p.current().text == "," || p.current().text == ")") {
			return nil
		}
		switch p.current().text {
		case "(":
			depth++
		case ")":
			depth--
		}
		p.pos++
	}
	if depth != 0 {
		return fmt.Errorf("unterminated column constraint")
	}
	return nil
}

func (p *Parser) identifier() (string, error) {
	current := p.current()
	if current.kind != 'w' && current.kind != 'i' {
		return "", fmt.Errorf("expected identifier, got %q", current.text)
	}
	p.pos++
	return current.text, nil
}

func (p *Parser) value() (string, error) {
	current := p.current()
	if current.kind != 'w' && current.kind != 's' && current.kind != 'i' {
		return "", fmt.Errorf("expected a value, got %q", current.text)
	}
	p.pos++
	return current.text, nil
}

func (p *Parser) current() token {
	if p.pos >= len(p.tokens) {
		return token{}
	}
	return p.tokens[p.pos]
}

func (p *Parser) take(text string) bool {
	if p.current().text != text {
		return false
	}
	p.pos++
	return true
}

func (p *Parser) takeKeyword(keyword string) bool {
	if p.current().kind != 'w' || !strings.EqualFold(p.current().text, keyword) {
		return false
	}
	p.pos++
	return true
}

func (p *Parser) expect(text string) error {
	if !p.take(text) {
		return fmt.Errorf("expected %q, got %q", text, p.current().text)
	}
	return nil
}

func (p *Parser) expectKeyword(keyword string) error {
	if !p.takeKeyword(keyword) {
		return fmt.Errorf("expected %s, got %q", keyword, p.current().text)
	}
	return nil
}

func tokenize(sql string) ([]token, error) {
	var tokens []token
	for i := 0; i < len(sql); {
		r := rune(sql[i])
		if unicode.IsSpace(r) {
			i++
			continue
		}
		switch sql[i] {
		case ',', '(', ')', '*', ';':
			tokens = append(tokens, token{text: string(sql[i]), kind: sql[i]})
			i++
		case '=', '!', '<', '>':
			start := i
			i++
			if i < len(sql) && (sql[i] == '=' || (sql[start] == '<' && sql[i] == '>')) {
				i++
			}
			op := sql[start:i]
			if op == "!" {
				return nil, fmt.Errorf("invalid operator !")
			}
			tokens = append(tokens, token{text: op, kind: 'o'})
		case '\'', '"', '`':
			quote := sql[i]
			i++
			var value strings.Builder
			closed := false
			for i < len(sql) {
				if sql[i] == quote {
					if i+1 < len(sql) && sql[i+1] == quote {
						value.WriteByte(quote)
						i += 2
						continue
					}
					i++
					closed = true
					break
				}
				value.WriteByte(sql[i])
				i++
			}
			if !closed {
				return nil, fmt.Errorf("unterminated quoted value")
			}
			kind := byte('s')
			if quote != '\'' {
				kind = 'i'
			}
			tokens = append(tokens, token{text: value.String(), kind: kind})
		default:
			start := i
			for i < len(sql) && !unicode.IsSpace(rune(sql[i])) &&
				!strings.ContainsRune(",()*;=!<>\"'`", rune(sql[i])) {
				i++
			}
			if start == i {
				return nil, fmt.Errorf("unexpected character %q", sql[i])
			}
			tokens = append(tokens, token{text: sql[start:i], kind: 'w'})
		}
	}
	return tokens, nil
}
