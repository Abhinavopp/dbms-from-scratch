package chapter14

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"sync"

	parser "dbmsfromscratch/chapters/13_parser"
)

// Result is the outcome of a query execution.
type Result struct {
	Columns []string
	Rows    []map[string]string
}

// ColumnMetadata describes a table column for database exploration.
type ColumnMetadata struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

// TableMetadata describes a table including schema and indexes.
type TableMetadata struct {
	Name       string          `json:"name"`
	Columns    []ColumnMetadata `json:"columns"`
	PrimaryKey string          `json:"primaryKey"`
	Indexes    []string        `json:"indexes"`
}

type secondaryIndex struct {
	column string
	values map[string][]int
}

type table struct {
	columns   []parser.ColumnDefinition
	rows      []map[string]string
	primary   string
	indexes   map[string]*secondaryIndex
	indexDefs []parser.IndexDefinition
}

// Executor is an in-memory SQL executor for the educational database.
type Executor struct {
	mu     sync.RWMutex
	tables map[string]*table
}

type evalValue struct {
	kind string
	text string
	num  float64
	b    bool
}

func NewExecutor() *Executor {
	return &Executor{tables: make(map[string]*table)}
}

func (e *Executor) Tables() []TableMetadata {
	e.mu.RLock()
	defer e.mu.RUnlock()

	order := make([]string, 0, len(e.tables))
	for name := range e.tables {
		order = append(order, name)
	}
	sort.Strings(order)

	result := make([]TableMetadata, 0, len(order))
	for _, key := range order {
		t := e.tables[key]
		metadata := TableMetadata{Name: t.primary, Columns: make([]ColumnMetadata, 0, len(t.columns)), PrimaryKey: t.primary, Indexes: make([]string, 0, len(t.indexDefs))}
		for _, column := range t.columns {
			metadata.Columns = append(metadata.Columns, ColumnMetadata{Name: column.Name, Type: column.Type})
		}
		for _, definition := range t.indexDefs {
			if strings.EqualFold(definition.Column, t.primary) {
				continue
			}
			metadata.Indexes = append(metadata.Indexes, definition.Column)
		}
		metadata.Name = key
		result = append(result, metadata)
	}
	return result
}

func (e *Executor) Execute(sql string) (Result, error) {
	stmt, err := parser.NewParser(sql).Parse()
	if err != nil {
		return Result{}, err
	}

	switch stmt.Kind {
	case "SELECT":
		return e.selectRows(stmt)
	case "INSERT":
		return e.insertRows(stmt)
	case "CREATE":
		return e.createTable(stmt)
	case "DROP":
		return e.dropTable(stmt)
	case "ALTER":
		return e.alterTable(stmt)
	case "UPDATE":
		return e.updateRows(stmt)
	case "DELETE":
		return e.deleteRows(stmt)
	default:
		return Result{}, fmt.Errorf("unsupported statement kind: %s", stmt.Kind)
	}
}

func (e *Executor) createTable(stmt parser.Statement) (Result, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	key := normalize(stmt.Table)
	if _, exists := e.tables[key]; exists {
		if stmt.IfNotExists {
			return Result{}, nil
		}
		return Result{}, fmt.Errorf("table %q already exists", stmt.Table)
	}
	columns := make([]parser.ColumnDefinition, len(stmt.Definitions))
	seen := make(map[string]bool, len(stmt.Definitions))
	for i, column := range stmt.Definitions {
		columnKey := normalize(column.Name)
		if seen[columnKey] {
			return Result{}, fmt.Errorf("duplicate column %q", column.Name)
		}
		seen[columnKey] = true
		columns[i] = column
	}

	t := &table{columns: columns, indexes: make(map[string]*secondaryIndex)}
	if stmt.PrimaryKey != "" {
		index := columnIndex(t, stmt.PrimaryKey)
		if index < 0 {
			return Result{}, fmt.Errorf("primary key column %q does not exist", stmt.PrimaryKey)
		}
		t.primary = t.columns[index].Name
		t.indexDefs = append(t.indexDefs, parser.IndexDefinition{Column: t.primary})
	}
	for _, definition := range stmt.Indexes {
		index := columnIndex(t, definition.Column)
		if index < 0 {
			return Result{}, fmt.Errorf("index column %q does not exist", definition.Column)
		}
		column := t.columns[index].Name
		if containsIndex(t.indexDefs, column) {
			continue
		}
		definition.Column = column
		t.indexDefs = append(t.indexDefs, definition)
	}
	e.rebuildIndexes(t)
	e.tables[key] = t
	return Result{}, nil
}

func (e *Executor) dropTable(stmt parser.Statement) (Result, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	key := normalize(stmt.Table)
	if _, exists := e.tables[key]; !exists {
		if stmt.IfExists {
			return Result{}, nil
		}
		return Result{}, fmt.Errorf("table %q does not exist", stmt.Table)
	}
	delete(e.tables, key)
	return Result{}, nil
}

func (e *Executor) alterTable(stmt parser.Statement) (Result, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	key := normalize(stmt.Table)
	t, exists := e.tables[key]
	if !exists {
		return Result{}, fmt.Errorf("table %q does not exist", stmt.Table)
	}

	switch stmt.AlterAction {
	case "ADD COLUMN":
		if columnIndex(t, stmt.ColumnName) >= 0 {
			return Result{}, fmt.Errorf("column %q already exists", stmt.ColumnName)
		}
		t.columns = append(t.columns, parser.ColumnDefinition{Name: stmt.ColumnName, Type: stmt.ColumnType})
		for _, row := range t.rows {
			row[stmt.ColumnName] = ""
		}
	case "DROP COLUMN":
		index := columnIndex(t, stmt.ColumnName)
		if index < 0 {
			return Result{}, fmt.Errorf("column %q does not exist", stmt.ColumnName)
		}
		if strings.EqualFold(t.primary, t.columns[index].Name) {
			return Result{}, fmt.Errorf("cannot drop primary key column %q", stmt.ColumnName)
		}
		if len(t.columns) == 1 {
			return Result{}, fmt.Errorf("cannot drop the last column from table %q", stmt.Table)
		}
		dropped := t.columns[index].Name
		for _, row := range t.rows {
			deleteColumn(row, dropped)
		}
		t.columns = append(t.columns[:index], t.columns[index+1:]...)
		filtered := t.indexDefs[:0]
		for _, definition := range t.indexDefs {
			if !strings.EqualFold(definition.Column, dropped) {
				filtered = append(filtered, definition)
			}
		}
		t.indexDefs = filtered
		e.rebuildIndexes(t)
	case "RENAME COLUMN":
		index := columnIndex(t, stmt.ColumnName)
		if index < 0 {
			return Result{}, fmt.Errorf("column %q does not exist", stmt.ColumnName)
		}
		if columnIndex(t, stmt.NewName) >= 0 {
			return Result{}, fmt.Errorf("column %q already exists", stmt.NewName)
		}
		oldName := t.columns[index].Name
		t.columns[index].Name = stmt.NewName
		if strings.EqualFold(t.primary, oldName) {
			t.primary = stmt.NewName
		}
		for _, definition := range t.indexDefs {
			if strings.EqualFold(definition.Column, oldName) {
				definition.Column = stmt.NewName
			}
		}
		for _, row := range t.rows {
			value := row[oldName]
			delete(row, oldName)
			row[stmt.NewName] = value
		}
		e.rebuildIndexes(t)
	case "RENAME TABLE":
		newKey := normalize(stmt.NewName)
		if _, exists := e.tables[newKey]; exists {
			return Result{}, fmt.Errorf("table %q already exists", stmt.NewName)
		}
		delete(e.tables, key)
		e.tables[newKey] = t
	default:
		return Result{}, fmt.Errorf("unsupported ALTER TABLE operation %q", stmt.AlterAction)
	}
	return Result{}, nil
}

func (e *Executor) selectRows(stmt parser.Statement) (Result, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	t, exists := e.tables[normalize(stmt.Table)]
	if !exists {
		return Result{}, fmt.Errorf("table %q does not exist", stmt.Table)
	}
	items, columns, err := resolveSelectItems(t, stmt)
	if err != nil {
		return Result{}, err
	}
	for _, item := range items {
		if err := validateExpressionColumns(item.Expression, t, false); err != nil {
			return Result{}, err
		}
	}
	for _, expression := range []*parser.Expression{stmt.Where, stmt.Filter, stmt.IndexBy} {
		if err := validateExpressionColumns(expression, t, false); err != nil {
			return Result{}, err
		}
	}
	rowIDs := make([]int, len(t.rows))
	for i := range t.rows {
		rowIDs[i] = i
	}
	if stmt.IndexBy != nil {
		rowIDs, err = e.indexCandidates(t, stmt.IndexBy)
		if err != nil {
			return Result{}, err
		}
	}
	offset, err := evaluateLimit(stmt.Offset)
	if err != nil {
		return Result{}, fmt.Errorf("invalid LIMIT offset: %w", err)
	}
	limit, err := evaluateLimit(stmt.Limit)
	if err != nil {
		return Result{}, fmt.Errorf("invalid LIMIT value: %w", err)
	}
	if offset > len(rowIDs) {
		offset = len(rowIDs)
	}
	rowIDs = rowIDs[offset:]
	if stmt.Limit != nil && limit < len(rowIDs) {
		rowIDs = rowIDs[:limit]
	}

	result := Result{Columns: columns, Rows: make([]map[string]string, 0)}
	for _, rowID := range rowIDs {
		row := t.rows[rowID]
		matches := true
		for _, expression := range []*parser.Expression{stmt.Where, stmt.Filter} {
			if expression == nil {
				continue
			}
			value, err := evaluate(expression, row, t, false)
			if err != nil {
				return Result{}, err
			}
			if value.kind != "boolean" {
				return Result{}, fmt.Errorf("filter expression must evaluate to BOOLEAN")
			}
			if !value.b {
				matches = false
				break
			}
		}
		if !matches {
			continue
		}
		selected := make(map[string]string, len(items))
		for i, item := range items {
			value, err := evaluate(item.Expression, row, t, false)
			if err != nil {
				return Result{}, err
			}
			selected[columns[i]] = value.text
		}
		result.Rows = append(result.Rows, selected)
	}
	return result, nil
}

func (e *Executor) insertRows(stmt parser.Statement) (Result, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	t, exists := e.tables[normalize(stmt.Table)]
	if !exists {
		return Result{}, fmt.Errorf("table %q does not exist", stmt.Table)
	}
	rows := stmt.ValueRows
	if len(rows) == 0 {
		rows = [][]*parser.Expression{make([]*parser.Expression, len(stmt.Values))}
		for i, value := range stmt.Values {
			rows[0][i] = &parser.Expression{Kind: "string", Value: value}
		}
	}

	columns := stmt.InsertColumns
	if len(columns) == 0 {
		for _, column := range t.columns {
			columns = append(columns, column.Name)
		}
	}
	columnNames := make([]string, len(columns))
	for i, name := range columns {
		index := columnIndex(t, name)
		if index < 0 {
			return Result{}, fmt.Errorf("column %q does not exist in table %q", name, stmt.Table)
		}
		columnNames[i] = t.columns[index].Name
		for prior := 0; prior < i; prior++ {
			if strings.EqualFold(columnNames[prior], columnNames[i]) {
				return Result{}, fmt.Errorf("column %q specified more than once", name)
			}
		}
	}

	newRows := make([]map[string]string, 0, len(rows))
	primaryValues := make(map[string]bool)
	if t.primary != "" {
		for _, row := range t.rows {
			primaryValues[row[t.primary]] = true
		}
	}
	for _, expressions := range rows {
		if len(expressions) != len(columnNames) {
			return Result{}, fmt.Errorf("INSERT has %d columns but %d values", len(columnNames), len(expressions))
		}
		row := make(map[string]string, len(t.columns))
		for _, column := range t.columns {
			row[column.Name] = ""
		}
		for i, expression := range expressions {
			value, err := evaluate(expression, nil, t, true)
			if err != nil {
				return Result{}, fmt.Errorf("invalid INSERT value for column %q: %w", columnNames[i], err)
			}
			columnPosition := columnIndex(t, columnNames[i])
			if err := validateColumnValue(value, t.columns[columnPosition]); err != nil {
				return Result{}, fmt.Errorf("invalid INSERT value for column %q: %w", columnNames[i], err)
			}
			row[columnNames[i]] = value.text
		}
		if t.primary != "" {
			value := row[t.primary]
			if value == "" {
				return Result{}, fmt.Errorf("primary key column %q cannot be empty", t.primary)
			}
			if primaryValues[value] {
				return Result{}, fmt.Errorf("duplicate primary key value %q", value)
			}
			primaryValues[value] = true
		}
		newRows = append(newRows, row)
	}
	t.rows = append(t.rows, newRows...)
	e.rebuildIndexes(t)

	resultRows := make([]map[string]string, len(newRows))
	for i, row := range newRows {
		resultRows[i] = clone(row)
	}
	return Result{Columns: columnNames, Rows: resultRows}, nil
}

func (e *Executor) updateRows(stmt parser.Statement) (Result, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	t, exists := e.tables[normalize(stmt.Table)]
	if !exists {
		return Result{}, fmt.Errorf("table %q does not exist", stmt.Table)
	}
	if err := validateExpressionColumns(stmt.Where, t, false); err != nil {
		return Result{}, err
	}
	assignments := make([]struct {
		column string
		expr   *parser.Expression
	}, len(stmt.Assignments))
	seen := make(map[string]bool, len(stmt.Assignments))
	for i, assignment := range stmt.Assignments {
		index := columnIndex(t, assignment.Column)
		if index < 0 {
			return Result{}, fmt.Errorf("column %q does not exist in table %q", assignment.Column, stmt.Table)
		}
		column := t.columns[index].Name
		if seen[normalize(column)] {
			return Result{}, fmt.Errorf("column %q assigned more than once", assignment.Column)
		}
		seen[normalize(column)] = true
		expression := assignment.Expression
		if expression == nil {
			expression = literalForColumn(assignment.Value, t.columns[index].Type)
		}
		if err := validateExpressionColumns(expression, t, false); err != nil {
			return Result{}, err
		}
		assignments[i] = struct {
			column string
			expr   *parser.Expression
		}{column: column, expr: expression}
	}

	result := Result{Columns: make([]string, len(t.columns)), Rows: make([]map[string]string, 0)}
	for i, column := range t.columns {
		result.Columns[i] = column.Name
	}
	pending := make([]struct {
		row   map[string]string
		value map[string]string
	}, 0)
	primaryValues := make(map[string]bool)
	matchedRows := make([]bool, len(t.rows))
	for i, row := range t.rows {
		matches, err := matchesExpression(stmt.Where, row, t)
		if err != nil {
			return Result{}, err
		}
		matchedRows[i] = matches
		if t.primary != "" && !matches {
			primaryValues[row[t.primary]] = true
		}
	}
	for i, row := range t.rows {
		if !matchedRows[i] {
			continue
		}
		updated := clone(row)
		for _, assignment := range assignments {
			value, err := evaluate(assignment.expr, row, t, false)
			if err != nil {
				return Result{}, err
			}
			columnIndex := columnIndex(t, assignment.column)
			if columnIndex >= 0 {
				if err := validateColumnValue(value, t.columns[columnIndex]); err != nil {
					return Result{}, fmt.Errorf("invalid UPDATE value for column %q: %w", assignment.column, err)
				}
			}
			updated[assignment.column] = value.text
		}
		if t.primary != "" {
			if updated[t.primary] == "" {
				return Result{}, fmt.Errorf("primary key column %q cannot be empty", t.primary)
			}
			if primaryValues[updated[t.primary]] {
				return Result{}, fmt.Errorf("duplicate primary key value %q", updated[t.primary])
			}
			primaryValues[updated[t.primary]] = true
		}
		pending = append(pending, struct {
			row   map[string]string
			value map[string]string
		}{row: row, value: updated})
	}
	for _, item := range pending {
		for key := range item.row {
			delete(item.row, key)
		}
		for key, value := range item.value {
			item.row[key] = value
		}
		result.Rows = append(result.Rows, clone(item.row))
	}
	e.rebuildIndexes(t)
	return result, nil
}

func (e *Executor) deleteRows(stmt parser.Statement) (Result, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	t, exists := e.tables[normalize(stmt.Table)]
	if !exists {
		return Result{}, fmt.Errorf("table %q does not exist", stmt.Table)
	}
	if err := validateExpressionColumns(stmt.Where, t, false); err != nil {
		return Result{}, err
	}
	result := Result{Columns: make([]string, len(t.columns)), Rows: make([]map[string]string, 0)}
	for i, column := range t.columns {
		result.Columns[i] = column.Name
	}
	kept := make([]map[string]string, 0, len(t.rows))
	for _, row := range t.rows {
		matches, err := matchesExpression(stmt.Where, row, t)
		if err != nil {
			return Result{}, err
		}
		if matches {
			result.Rows = append(result.Rows, clone(row))
		} else {
			kept = append(kept, row)
		}
	}
	t.rows = kept
	e.rebuildIndexes(t)
	return result, nil
}

func resolveSelectItems(t *table, stmt parser.Statement) ([]parser.SelectItem, []string, error) {
	items := stmt.SelectItems
	if len(items) == 0 {
		for _, name := range stmt.Columns {
			if name == "*" {
				items = []parser.SelectItem{{Expression: &parser.Expression{Kind: "star", Value: "*"}}}
				break
			}
			items = append(items, parser.SelectItem{Expression: &parser.Expression{Kind: "column", Value: name}})
		}
	}
	if len(items) == 1 && items[0].Expression != nil && items[0].Expression.Kind == "star" {
		columns := make([]string, len(t.columns))
		expanded := make([]parser.SelectItem, len(t.columns))
		for i, column := range t.columns {
			columns[i] = column.Name
			expanded[i] = parser.SelectItem{Expression: &parser.Expression{Kind: "column", Value: column.Name}}
		}
		return expanded, columns, nil
	}
	output := make([]string, len(items))
	for i, item := range items {
		if item.Expression == nil {
			return nil, nil, fmt.Errorf("SELECT item has no expression")
		}
		if item.Expression.Kind == "column" {
			index := columnIndex(t, item.Expression.Value)
			if index < 0 {
				return nil, nil, fmt.Errorf("column %q does not exist", item.Expression.Value)
			}
			if item.Alias == "" {
				output[i] = t.columns[index].Name
			} else {
				output[i] = item.Alias
			}
			continue
		}
		if item.Alias == "" {
			return nil, nil, fmt.Errorf("SELECT expressions require an AS alias")
		}
		output[i] = item.Alias
	}
	return items, output, nil
}

func (e *Executor) indexCandidates(t *table, expression *parser.Expression) ([]int, error) {
	column, checks, ok := indexChecks(expression, t)
	if !ok {
		return nil, fmt.Errorf("invalid INDEX BY condition: expected comparisons joined by AND on one indexed column")
	}
	index := columnIndex(t, column)
	if index < 0 {
		return nil, fmt.Errorf("INDEX BY column %q does not exist", column)
	}
	canonical := t.columns[index].Name
	scan := t.indexes[normalize(canonical)]
	if scan == nil {
		return nil, fmt.Errorf("no index exists for INDEX BY column %q", canonical)
	}
	var ids []int
	for key, rowIDs := range scan.values {
		matches := true
		for _, check := range checks {
			right, err := evaluate(check.value, nil, t, true)
			if err != nil {
				return nil, fmt.Errorf("invalid INDEX BY value: %w", err)
			}
			keyValue, err := valueForColumn(key, t.columns[index].Type)
			if err != nil {
				return nil, err
			}
			matched, err := compareValues(keyValue, check.operator, right)
			if err != nil {
				return nil, err
			}
			if !matched {
				matches = false
				break
			}
		}
		if matches {
			ids = append(ids, rowIDs...)
		}
	}
	sort.Ints(ids)
	return ids, nil
}

type indexCheck struct {
	operator string
	value    *parser.Expression
}

func indexChecks(expression *parser.Expression, t *table) (string, []indexCheck, bool) {
	if expression == nil {
		return "", nil, false
	}
	if expression.Kind == "binary" && expression.Operator == "AND" {
		leftColumn, leftChecks, leftOK := indexChecks(expression.Left, t)
		rightColumn, rightChecks, rightOK := indexChecks(expression.Right, t)
		if !leftOK || !rightOK || !strings.EqualFold(leftColumn, rightColumn) {
			return "", nil, false
		}
		return leftColumn, append(leftChecks, rightChecks...), true
	}
	if expression.Kind != "binary" || !isComparison(expression.Operator) {
		return "", nil, false
	}
	left, right := expression.Left, expression.Right
	if left != nil && left.Kind == "column" && right != nil &&
		(right.Kind != "column" || columnIndex(t, right.Value) < 0) {
		return left.Value, []indexCheck{{operator: expression.Operator, value: right}}, true
	}
	if right != nil && right.Kind == "column" && left != nil &&
		(left.Kind != "column" || columnIndex(t, left.Value) < 0) {
		return right.Value, []indexCheck{{operator: reverseComparison(expression.Operator), value: left}}, true
	}
	return "", nil, false
}

func reverseComparison(operator string) string {
	switch operator {
	case "<":
		return ">"
	case ">":
		return "<"
	case "<=":
		return ">="
	case ">=":
		return "<="
	default:
		return operator
	}
}

func (e *Executor) rebuildIndexes(t *table) {
	t.indexes = make(map[string]*secondaryIndex, len(t.indexDefs))
	for _, definition := range t.indexDefs {
		index := &secondaryIndex{column: definition.Column, values: make(map[string][]int)}
		t.indexes[normalize(definition.Column)] = index
		for rowID, row := range t.rows {
			value := row[definition.Column]
			index.values[value] = append(index.values[value], rowID)
		}
	}
}

func containsIndex(definitions []parser.IndexDefinition, column string) bool {
	for _, definition := range definitions {
		if strings.EqualFold(definition.Column, column) {
			return true
		}
	}
	return false
}

func matchesExpression(expression *parser.Expression, row map[string]string, t *table) (bool, error) {
	if expression == nil {
		return true, nil
	}
	value, err := evaluate(expression, row, t, false)
	if err != nil {
		return false, err
	}
	if value.kind != "boolean" {
		return false, fmt.Errorf("condition must evaluate to BOOLEAN")
	}
	return value.b, nil
}

func validateExpressionColumns(expression *parser.Expression, t *table, allowUnknown bool) error {
	if expression == nil {
		return nil
	}
	switch expression.Kind {
	case "column":
		if columnIndex(t, expression.Value) < 0 && !allowUnknown {
			return fmt.Errorf("column %q does not exist", expression.Value)
		}
	case "unary":
		return validateExpressionColumns(expression.Left, t, false)
	case "binary":
		if err := validateExpressionColumns(expression.Left, t, false); err != nil {
			return err
		}
		allowRightLiteral := isComparison(expression.Operator) &&
			expression.Right != nil && expression.Right.Kind == "column"
		return validateExpressionColumns(expression.Right, t, allowRightLiteral)
	}
	return nil
}

func evaluate(expression *parser.Expression, row map[string]string, t *table, unknownAsString bool) (evalValue, error) {
	if expression == nil {
		return evalValue{}, fmt.Errorf("missing expression")
	}
	switch expression.Kind {
	case "string":
		return evalValue{kind: "string", text: expression.Value}, nil
	case "number":
		number, err := strconv.ParseFloat(expression.Value, 64)
		if err != nil {
			return evalValue{}, fmt.Errorf("invalid number %q", expression.Value)
		}
		return evalValue{kind: "number", text: expression.Value, num: number}, nil
	case "boolean":
		value := strings.EqualFold(expression.Value, "true")
		return evalValue{kind: "boolean", text: strconv.FormatBool(value), b: value}, nil
	case "column":
		index := columnIndex(t, expression.Value)
		if index < 0 {
			if unknownAsString {
				return evalValue{kind: "string", text: expression.Value}, nil
			}
			return evalValue{}, fmt.Errorf("column %q does not exist", expression.Value)
		}
		if row == nil {
			return evalValue{}, fmt.Errorf("column %q cannot be evaluated without a row", expression.Value)
		}
		return valueForColumn(row[t.columns[index].Name], t.columns[index].Type)
	case "unary":
		value, err := evaluate(expression.Left, row, t, unknownAsString)
		if err != nil {
			return evalValue{}, err
		}
		switch strings.ToUpper(expression.Operator) {
		case "NOT":
			if value.kind != "boolean" {
				return evalValue{}, fmt.Errorf("NOT requires a BOOLEAN operand")
			}
			return evalValue{kind: "boolean", b: !value.b, text: strconv.FormatBool(!value.b)}, nil
		case "-":
			if value.kind != "number" {
				return evalValue{}, fmt.Errorf("unary - requires a numeric operand")
			}
			value.num = -value.num
			value.text = formatNumber(value.num)
			return value, nil
		default:
			return evalValue{}, fmt.Errorf("invalid unary operator %q", expression.Operator)
		}
	case "binary":
		operator := strings.ToUpper(expression.Operator)
		left, err := evaluate(expression.Left, row, t, false)
		if err != nil {
			return evalValue{}, err
		}
		if operator == "AND" && left.kind == "boolean" && !left.b {
			return left, nil
		}
		if operator == "OR" && left.kind == "boolean" && left.b {
			return left, nil
		}
		right, err := evaluate(expression.Right, row, t, isComparison(operator))
		if err != nil {
			return evalValue{}, err
		}
		switch operator {
		case "AND", "OR":
			if left.kind != "boolean" || right.kind != "boolean" {
				return evalValue{}, fmt.Errorf("%s requires BOOLEAN operands", operator)
			}
			result := left.b && right.b
			if operator == "OR" {
				result = left.b || right.b
			}
			return evalValue{kind: "boolean", b: result, text: strconv.FormatBool(result)}, nil
		case "=", "!=", "<>", "<", ">", "<=", ">=":
			result, err := compareValues(left, operator, right)
			if err != nil {
				return evalValue{}, err
			}
			return evalValue{kind: "boolean", b: result, text: strconv.FormatBool(result)}, nil
		case "+", "-", "*", "/":
			if left.kind != "number" || right.kind != "number" {
				return evalValue{}, fmt.Errorf("operator %s requires numeric operands", operator)
			}
			var number float64
			switch operator {
			case "+":
				number = left.num + right.num
			case "-":
				number = left.num - right.num
			case "*":
				number = left.num * right.num
			case "/":
				if right.num == 0 {
					return evalValue{}, fmt.Errorf("division by zero")
				}
				number = left.num / right.num
			}
			if math.IsInf(number, 0) || math.IsNaN(number) {
				return evalValue{}, fmt.Errorf("arithmetic result is not finite")
			}
			return evalValue{kind: "number", num: number, text: formatNumber(number)}, nil
		default:
			return evalValue{}, fmt.Errorf("invalid binary operator %q", expression.Operator)
		}
	default:
		return evalValue{}, fmt.Errorf("invalid expression kind %q", expression.Kind)
	}
}

func valueForColumn(value, columnType string) (evalValue, error) {
	typeName := strings.ToLower(columnType)
	switch {
	case strings.Contains(typeName, "bool"):
		if value == "" {
			return evalValue{kind: "boolean", text: "", b: false}, nil
		}
		boolean, err := strconv.ParseBool(value)
		if err != nil {
			return evalValue{}, fmt.Errorf("type error: %q is not BOOLEAN", value)
		}
		return evalValue{kind: "boolean", text: strconv.FormatBool(boolean), b: boolean}, nil
	case strings.Contains(typeName, "int"), strings.Contains(typeName, "real"),
		strings.Contains(typeName, "float"), strings.Contains(typeName, "double"),
		strings.Contains(typeName, "decimal"), strings.Contains(typeName, "numeric"):
		if value == "" {
			return evalValue{kind: "number", text: "", num: 0}, nil
		}
		number, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return evalValue{}, fmt.Errorf("type error: %q is not numeric", value)
		}
		return evalValue{kind: "number", text: value, num: number}, nil
	default:
		return evalValue{kind: "string", text: value}, nil
	}
}

func literalForColumn(value, columnType string) *parser.Expression {
	typeName := strings.ToLower(columnType)
	switch {
	case strings.Contains(typeName, "bool"):
		if strings.EqualFold(value, "true") || strings.EqualFold(value, "false") {
			return &parser.Expression{Kind: "boolean", Value: strings.ToLower(value)}
		}
	case strings.Contains(typeName, "int"), strings.Contains(typeName, "real"),
		strings.Contains(typeName, "float"), strings.Contains(typeName, "double"),
		strings.Contains(typeName, "decimal"), strings.Contains(typeName, "numeric"):
		if _, err := strconv.ParseFloat(value, 64); err == nil {
			return &parser.Expression{Kind: "number", Value: value}
		}
	}
	return &parser.Expression{Kind: "string", Value: value}
}

func validateColumnValue(value evalValue, column parser.ColumnDefinition) error {
	typeName := strings.ToLower(column.Type)
	switch {
	case strings.Contains(typeName, "bool"):
		if value.kind != "boolean" {
			return fmt.Errorf("type error: BOOLEAN column requires a boolean value")
		}
	case strings.Contains(typeName, "int"), strings.Contains(typeName, "real"),
		strings.Contains(typeName, "float"), strings.Contains(typeName, "double"),
		strings.Contains(typeName, "decimal"), strings.Contains(typeName, "numeric"):
		if value.kind != "number" {
			return fmt.Errorf("type error: numeric column requires a numeric value")
		}
	}
	return nil
}

func compareValues(left evalValue, operator string, right evalValue) (bool, error) {
	if left.kind != right.kind {
		return false, fmt.Errorf("type error: cannot compare %s with %s", left.kind, right.kind)
	}
	var comparison int
	switch left.kind {
	case "number":
		switch {
		case left.num < right.num:
			comparison = -1
		case left.num > right.num:
			comparison = 1
		}
	case "string":
		comparison = strings.Compare(left.text, right.text)
	case "boolean":
		if left.b != right.b {
			if !left.b {
				comparison = -1
			} else {
				comparison = 1
			}
		}
	default:
		return false, fmt.Errorf("cannot compare values of type %s", left.kind)
	}
	switch operator {
	case "=":
		return comparison == 0, nil
	case "!=", "<>":
		return comparison != 0, nil
	case "<":
		return comparison < 0, nil
	case ">":
		return comparison > 0, nil
	case "<=":
		return comparison <= 0, nil
	case ">=":
		return comparison >= 0, nil
	default:
		return false, fmt.Errorf("invalid comparison operator %q", operator)
	}
}

func evaluateLimit(expression *parser.Expression) (int, error) {
	if expression == nil {
		return 0, nil
	}
	value, err := evaluate(expression, nil, nil, false)
	if err != nil {
		return 0, err
	}
	if value.kind != "number" || value.num < 0 || math.Trunc(value.num) != value.num || value.num > float64(math.MaxInt) {
		return 0, fmt.Errorf("value must be a non-negative integer")
	}
	return int(value.num), nil
}

func formatNumber(value float64) string {
	return strconv.FormatFloat(value, 'f', -1, 64)
}

func columnIndex(t *table, name string) int {
	if t == nil {
		return -1
	}
	for i, column := range t.columns {
		if strings.EqualFold(column.Name, name) {
			return i
		}
	}
	return -1
}

func deleteColumn(row map[string]string, name string) {
	for key := range row {
		if strings.EqualFold(key, name) {
			delete(row, key)
		}
	}
}

func normalize(name string) string {
	return strings.ToLower(name)
}

func clone(row map[string]string) map[string]string {
	result := make(map[string]string, len(row))
	for key, value := range row {
		result[key] = value
	}
	return result
}

func isComparison(operator string) bool {
	switch operator {
	case "=", "!=", "<>", "<", ">", "<=", ">=":
		return true
	default:
		return false
	}
}
