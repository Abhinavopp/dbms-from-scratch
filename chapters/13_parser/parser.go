package chapter13

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

type ColumnDefinition struct {
	Name string
	Type string
}

type IndexDefinition struct {
	Name   string
	Column string
}


type Expression struct {
	Kind     string
	Value    string
	Operator string
	Left     *Expression
	Right    *Expression
}


type Condition struct {
	Column   string
	Operator string
	Value    string
}

type Assignment struct {
	Column     string
	Value      string
	Expression *Expression
}


type SelectItem struct {
	Expression *Expression
	Alias      string
}

type Statement struct {
	Kind          string
	Table         string
	Columns       []string
	SelectItems   []SelectItem
	Values        []string
	ValueRows     [][]*Expression
	Assignments   []Assignment
	InsertColumns []string
	Definitions   []ColumnDefinition
	Indexes       []IndexDefinition
	PrimaryKey    string
	AlterAction   string
	ColumnName    string
	ColumnType    string
	NewName       string
	IfExists      bool
	IfNotExists   bool
	Condition     string
	Conditions    []Condition
	Where         *Expression
	IndexBy       *Expression
	Filter        *Expression
	Limit         *Expression
	Offset        *Expression
}

type token struct {
	text string
	kind byte
}

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
		if err := p.parseDelete(&stmt); err != nil {
			return Statement{}, err
		}
	default:
		return Statement{}, fmt.Errorf("expected SELECT, INSERT, UPDATE, DELETE, CREATE, DROP, or ALTER")
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
		expression, err := p.parseExpression()
		if err != nil {
			return err
		}
		assignment := Assignment{Column: column, Value: expressionValue(expression)}
		if expression.Kind != "string" && expression.Kind != "number" && expression.Kind != "boolean" {
			assignment.Expression = expression
		}
		stmt.Assignments = append(stmt.Assignments, assignment)
		if !p.take(",") {
			break
		}
	}
	if len(stmt.Assignments) == 0 {
		return fmt.Errorf("UPDATE requires at least one assignment")
	}
	if p.takeKeyword("WHERE") {
		stmt.Where, err = p.parseExpression()
		if err != nil {
			return err
		}
		stmt.Conditions = simpleConditions(stmt.Where)
	}
	return nil
}

func (p *Parser) parseSelect(stmt *Statement) error {
	if p.take("*") {
		stmt.Columns = []string{"*"}
		stmt.SelectItems = []SelectItem{{Expression: &Expression{Kind: "star", Value: "*"}}}
	} else {
		for {
			expression, err := p.parseExpression()
			if err != nil {
				return err
			}
			item := SelectItem{Expression: expression}
			if p.takeKeyword("AS") {
				item.Alias, err = p.identifier()
				if err != nil {
					return err
				}
			}
			stmt.SelectItems = append(stmt.SelectItems, item)
			if expression.Kind == "column" && item.Alias == "" {
				stmt.Columns = append(stmt.Columns, expression.Value)
			}
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

	for {
		switch {
		case p.takeKeyword("INDEX"):
			if err := p.expectKeyword("BY"); err != nil {
				return err
			}
			if stmt.IndexBy != nil {
				return fmt.Errorf("SELECT may contain only one INDEX BY clause")
			}
			stmt.IndexBy, err = p.parseExpression()
			if err != nil {
				return err
			}
		case p.takeKeyword("LIMIT"):
			if stmt.Limit != nil {
				return fmt.Errorf("SELECT may contain only one LIMIT clause")
			}
			first, parseErr := p.parseExpression()
			if parseErr != nil {
				return parseErr
			}
			if p.take(",") {
				stmt.Offset = first
				stmt.Limit, err = p.parseExpression()
				if err != nil {
					return err
				}
			} else {
				stmt.Limit = first
			}
		case p.takeKeyword("FILTER"):
			if stmt.Filter != nil {
				return fmt.Errorf("SELECT may contain only one FILTER clause")
			}
			stmt.Filter, err = p.parseExpression()
			if err != nil {
				return err
			}
		case p.takeKeyword("WHERE"):
			if stmt.Where != nil {
				return fmt.Errorf("SELECT may contain only one WHERE clause")
			}
			stmt.Where, err = p.parseExpression()
			if err != nil {
				return err
			}
			stmt.Conditions = simpleConditions(stmt.Where)
		default:
			stmt.Condition = conditionText(stmt.Conditions)
			return nil
		}
	}
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
	for {
		if err := p.expect("("); err != nil {
			return fmt.Errorf("VALUES must be followed by a parenthesized value list: %w", err)
		}
		row := make([]*Expression, 0)
		for {
			expression, err := p.parseExpression()
			if err != nil {
				return err
			}
			row = append(row, expression)
			if !p.take(",") {
				break
			}
		}
		if err := p.expect(")"); err != nil {
			return err
		}
		if len(row) == 0 {
			return fmt.Errorf("INSERT requires at least one value")
		}
		stmt.ValueRows = append(stmt.ValueRows, row)
		if !p.take(",") {
			break
		}
	}
	if len(stmt.ValueRows) == 0 {
		return fmt.Errorf("INSERT requires at least one value")
	}
	stmt.Values = make([]string, len(stmt.ValueRows[0]))
	for i, value := range stmt.ValueRows[0] {
		stmt.Values[i] = expressionValue(value)
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
		switch {
		case p.takeKeyword("PRIMARY"):
			if err := p.expectKeyword("KEY"); err != nil {
				return err
			}
			if stmt.PrimaryKey != "" {
				return fmt.Errorf("CREATE TABLE may declare only one primary key")
			}
			if err := p.expect("("); err != nil {
				return err
			}
			stmt.PrimaryKey, err = p.identifier()
			if err != nil {
				return err
			}
			if err := p.expect(")"); err != nil {
				return err
			}
		case p.takeKeyword("INDEX"), p.takeKeyword("KEY"):
			index, err := p.parseIndexDefinition()
			if err != nil {
				return err
			}
			stmt.Indexes = append(stmt.Indexes, index)
		default:
			name, err := p.identifier()
			if err != nil {
				return err
			}
			columnType, err := p.columnType()
			if err != nil {
				return err
			}
			stmt.Definitions = append(stmt.Definitions, ColumnDefinition{Name: name, Type: columnType})
			if err := p.parseInlineColumnConstraints(stmt, name); err != nil {
				return err
			}
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

func (p *Parser) parseInlineColumnConstraints(stmt *Statement, column string) error {
	for p.current().kind == 'w' {
		switch {
		case p.takeKeyword("PRIMARY"):
			if err := p.expectKeyword("KEY"); err != nil {
				return err
			}
			if stmt.PrimaryKey != "" {
				return fmt.Errorf("CREATE TABLE may declare only one primary key")
			}
			stmt.PrimaryKey = column
		case p.takeKeyword("INDEX"), p.takeKeyword("KEY"):
			stmt.Indexes = append(stmt.Indexes, IndexDefinition{Column: column})
		case p.takeKeyword("NOT"):
			if err := p.expectKeyword("NULL"); err != nil {
				return err
			}
		case p.takeKeyword("UNIQUE"):
			continue
		default:
			return nil
		}
	}
	return nil
}

func (p *Parser) parseIndexDefinition() (IndexDefinition, error) {
	var index IndexDefinition
	if p.current().text != "(" {
		index.Name = p.current().text
		if p.current().kind != 'w' && p.current().kind != 'i' {
			return IndexDefinition{}, fmt.Errorf("expected index name or (")
		}
		p.pos++
	}
	if p.takeKeyword("ON") {
		if _, err := p.identifier(); err != nil {
			return IndexDefinition{}, err
		}
	}
	if err := p.expect("("); err != nil {
		return IndexDefinition{}, err
	}
	column, err := p.identifier()
	if err != nil {
		return IndexDefinition{}, err
	}
	index.Column = column
	if err := p.expect(")"); err != nil {
		return IndexDefinition{}, err
	}
	return index, nil
}

func (p *Parser) parseDelete(stmt *Statement) error {
	if err := p.expectKeyword("FROM"); err != nil {
		return err
	}
	table, err := p.identifier()
	if err != nil {
		return err
	}
	stmt.Table = table
	if p.takeKeyword("WHERE") {
		stmt.Where, err = p.parseExpression()
		if err != nil {
			return err
		}
		stmt.Conditions = simpleConditions(stmt.Where)
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

func (p *Parser) parseExpression() (*Expression, error) {
	return p.parseOr()
}

func (p *Parser) parseOr() (*Expression, error) {
	left, err := p.parseAnd()
	if err != nil {
		return nil, err
	}
	for p.takeKeyword("OR") {
		right, err := p.parseAnd()
		if err != nil {
			return nil, err
		}
		left = binary("OR", left, right)
	}
	return left, nil
}

func (p *Parser) parseAnd() (*Expression, error) {
	left, err := p.parseNot()
	if err != nil {
		return nil, err
	}
	for p.takeKeyword("AND") {
		right, err := p.parseNot()
		if err != nil {
			return nil, err
		}
		left = binary("AND", left, right)
	}
	return left, nil
}

func (p *Parser) parseNot() (*Expression, error) {
	if p.takeKeyword("NOT") {
		child, err := p.parseNot()
		if err != nil {
			return nil, err
		}
		return &Expression{Kind: "unary", Operator: "NOT", Left: child}, nil
	}
	return p.parseComparison()
}

func (p *Parser) parseComparison() (*Expression, error) {
	left, err := p.parseAdditive()
	if err != nil {
		return nil, err
	}
	switch p.current().text {
	case "=", "!=", "<>", "<", ">", "<=", ">=":
		op := p.current().text
		p.pos++
		right, err := p.parseAdditive()
		if err != nil {
			return nil, err
		}
		left = binary(op, left, right)
		if isComparison(p.current().text) {
			return nil, fmt.Errorf("chained comparison operators are not supported")
		}
	}
	return left, nil
}

func (p *Parser) parseAdditive() (*Expression, error) {
	left, err := p.parseMultiplicative()
	if err != nil {
		return nil, err
	}
	for p.current().text == "+" || p.current().text == "-" {
		op := p.current().text
		p.pos++
		right, err := p.parseMultiplicative()
		if err != nil {
			return nil, err
		}
		left = binary(op, left, right)
	}
	return left, nil
}

func (p *Parser) parseMultiplicative() (*Expression, error) {
	left, err := p.parseUnary()
	if err != nil {
		return nil, err
	}
	for p.current().text == "*" || p.current().text == "/" {
		op := p.current().text
		p.pos++
		right, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		left = binary(op, left, right)
	}
	return left, nil
}

func (p *Parser) parseUnary() (*Expression, error) {
	if p.take("-") {
		child, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		return &Expression{Kind: "unary", Operator: "-", Left: child}, nil
	}
	return p.parsePrimary()
}

func (p *Parser) parsePrimary() (*Expression, error) {
	current := p.current()
	if p.take("(") {
		expression, err := p.parseExpression()
		if err != nil {
			return nil, err
		}
		if err := p.expect(")"); err != nil {
			return nil, err
		}
		return expression, nil
	}
	if current.kind == 's' {
		p.pos++
		return &Expression{Kind: "string", Value: current.text}, nil
	}
	if current.kind == 'i' {
		p.pos++
		return &Expression{Kind: "column", Value: current.text}, nil
	}
	if current.kind != 'w' {
		return nil, fmt.Errorf("expected expression, got %q", current.text)
	}
	p.pos++
	if _, err := strconv.ParseFloat(current.text, 64); err == nil {
		return &Expression{Kind: "number", Value: current.text}, nil
	}
	if strings.EqualFold(current.text, "TRUE") || strings.EqualFold(current.text, "FALSE") {
		return &Expression{Kind: "boolean", Value: strings.ToLower(current.text)}, nil
	}
	return &Expression{Kind: "column", Value: current.text}, nil
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
		case ',', '(', ')', '*', ';', '+', '-', '/':
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
				!strings.ContainsRune(",()*;+=/!<>\"'`", rune(sql[i])) {
				if sql[i] == '-' && (i == start || i+1 >= len(sql) || !isDigit(sql[i-1]) || !isDigit(sql[i+1])) {
					break
				}
				i++
			}
			if start == i {
				if sql[i] == '-' {
					tokens = append(tokens, token{text: "-", kind: '-'})
					i++
					continue
				}
				return nil, fmt.Errorf("unexpected character %q", sql[i])
			}
			tokens = append(tokens, token{text: sql[start:i], kind: 'w'})
		}
	}
	return tokens, nil
}

func binary(operator string, left, right *Expression) *Expression {
	return &Expression{Kind: "binary", Operator: strings.ToUpper(operator), Left: left, Right: right}
}

func isComparison(operator string) bool {
	switch operator {
	case "=", "!=", "<>", "<", ">", "<=", ">=":
		return true
	default:
		return false
	}
}

func isDigit(value byte) bool {
	return value >= '0' && value <= '9'
}

func expressionValue(expression *Expression) string {
	if expression == nil {
		return ""
	}
	if expression.Kind == "string" || expression.Kind == "number" || expression.Kind == "boolean" || expression.Kind == "column" {
		return expression.Value
	}
	return ""
}

func simpleConditions(expression *Expression) []Condition {
	if expression == nil {
		return nil
	}
	if expression.Kind == "binary" && expression.Operator == "AND" {
		return append(simpleConditions(expression.Left), simpleConditions(expression.Right)...)
	}
	if expression.Kind != "binary" || !isComparison(expression.Operator) ||
		expression.Left == nil || expression.Right == nil ||
		expression.Left.Kind != "column" {
		return nil
	}
	return []Condition{{Column: expression.Left.Value, Operator: expression.Operator, Value: expressionValue(expression.Right)}}
}

func conditionText(conditions []Condition) string {
	parts := make([]string, len(conditions))
	for i, condition := range conditions {
		parts[i] = condition.Column + " " + condition.Operator + " " + condition.Value
	}
	return strings.Join(parts, " AND ")
}
