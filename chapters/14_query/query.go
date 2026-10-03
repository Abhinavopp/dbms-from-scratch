package chapter14

import (
	"fmt"
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

type table struct {
	columns []parser.ColumnDefinition
	rows    []map[string]string
}

// Executor is an in-memory SQL executor for the educational database.
type Executor struct {
	mu     sync.RWMutex
	tables map[string]*table
}

func NewExecutor() *Executor {
	return &Executor{tables: make(map[string]*table)}
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
		return e.insertRow(stmt)
	case "CREATE":
		return e.createTable(stmt)
	case "DROP":
		return e.dropTable(stmt)
	case "ALTER":
		return e.alterTable(stmt)
	case "UPDATE":
		return e.updateRows(stmt)
	case "DELETE":
		return Result{}, fmt.Errorf("DELETE is not supported")
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
	e.tables[key] = &table{columns: columns, rows: []map[string]string{}}
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
		if len(t.columns) == 1 {
			return Result{}, fmt.Errorf("cannot drop the last column from table %q", stmt.Table)
		}
		for _, row := range t.rows {
			deleteColumn(row, t.columns[index].Name)
		}
		t.columns = append(t.columns[:index], t.columns[index+1:]...)
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
		for _, row := range t.rows {
			value := row[oldName]
			delete(row, oldName)
			row[stmt.NewName] = value
		}
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
	columns, err := resolveColumns(t, stmt.Columns)
	if err != nil {
		return Result{}, err
	}
	conditions, err := resolveConditions(t, stmt.Conditions)
	if err != nil {
		return Result{}, err
	}

	result := Result{
		Columns: columns,
		Rows:    make([]map[string]string, 0),
	}
	for _, row := range t.rows {
		matches := true
		for _, condition := range conditions {
			if !compare(row[condition.Column], condition.Operator, condition.Value) {
				matches = false
				break
			}
		}
		if !matches {
			continue
		}
		selected := make(map[string]string, len(columns))
		for _, column := range columns {
			selected[column] = row[column]
		}
		result.Rows = append(result.Rows, selected)
	}
	return result, nil
}

func (e *Executor) insertRow(stmt parser.Statement) (Result, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	t, exists := e.tables[normalize(stmt.Table)]
	if !exists {
		return Result{}, fmt.Errorf("table %q does not exist", stmt.Table)
	}

	columns := stmt.InsertColumns
	if len(columns) == 0 {
		if len(stmt.Values) != len(t.columns) {
			return Result{}, fmt.Errorf("table %q expects %d values, got %d", stmt.Table, len(t.columns), len(stmt.Values))
		}
		columns = make([]string, len(t.columns))
		for i, column := range t.columns {
			columns[i] = column.Name
		}
	} else if len(columns) != len(stmt.Values) {
		return Result{}, fmt.Errorf("INSERT has %d columns but %d values", len(columns), len(stmt.Values))
	}

	row := make(map[string]string, len(columns))
	resultColumns := make([]string, len(columns))
	for i, name := range columns {
		index := columnIndex(t, name)
		if index < 0 {
			return Result{}, fmt.Errorf("column %q does not exist in table %q", name, stmt.Table)
		}
		canonical := t.columns[index].Name
		if _, duplicate := row[canonical]; duplicate {
			return Result{}, fmt.Errorf("column %q specified more than once", name)
		}
		resultColumns[i] = canonical
		row[canonical] = stmt.Values[i]
	}
	for _, column := range t.columns {
		if _, exists := row[column.Name]; !exists {
			row[column.Name] = ""
		}
	}
	t.rows = append(t.rows, row)
	return Result{Columns: resultColumns, Rows: []map[string]string{clone(row)}}, nil
}

func (e *Executor) updateRows(stmt parser.Statement) (Result, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	t, exists := e.tables[normalize(stmt.Table)]
	if !exists {
		return Result{}, fmt.Errorf("table %q does not exist", stmt.Table)
	}

	conditions, err := resolveConditions(t, stmt.Conditions)
	if err != nil {
		return Result{}, err
	}

	assignments := make(map[string]string, len(stmt.Assignments))
	for _, assignment := range stmt.Assignments {
		index := columnIndex(t, assignment.Column)
		if index < 0 {
			return Result{}, fmt.Errorf("column %q does not exist in table %q", assignment.Column, stmt.Table)
		}
		column := t.columns[index].Name
		if _, duplicate := assignments[column]; duplicate {
			return Result{}, fmt.Errorf("column %q assigned more than once", assignment.Column)
		}
		assignments[column] = assignment.Value
	}
	if len(assignments) == 0 {
		return Result{}, fmt.Errorf("UPDATE requires at least one assignment")
	}

	result := Result{
		Columns: make([]string, len(t.columns)),
		Rows:    make([]map[string]string, 0),
	}
	for i, column := range t.columns {
		result.Columns[i] = column.Name
	}
	for _, row := range t.rows {
		matches := true
		for _, condition := range conditions {
			if !compare(row[condition.Column], condition.Operator, condition.Value) {
				matches = false
				break
			}
		}
		if !matches {
			continue
		}
		for column, value := range assignments {
			row[column] = value
		}
		result.Rows = append(result.Rows, clone(row))
	}
	return result, nil
}

func resolveColumns(t *table, requested []string) ([]string, error) {
	if len(requested) == 0 || (len(requested) == 1 && requested[0] == "*") {
		columns := make([]string, len(t.columns))
		for i, column := range t.columns {
			columns[i] = column.Name
		}
		return columns, nil
	}
	columns := make([]string, len(requested))
	for i, name := range requested {
		index := columnIndex(t, name)
		if index < 0 {
			return nil, fmt.Errorf("column %q does not exist", name)
		}
		columns[i] = t.columns[index].Name
	}
	return columns, nil
}

func resolveConditions(t *table, conditions []parser.Condition) ([]parser.Condition, error) {
	resolved := make([]parser.Condition, len(conditions))
	for i, condition := range conditions {
		index := columnIndex(t, condition.Column)
		if index < 0 {
			return nil, fmt.Errorf("column %q does not exist", condition.Column)
		}
		condition.Column = t.columns[index].Name
		resolved[i] = condition
	}
	return resolved, nil
}

func compare(left, operator, right string) bool {
	leftNumber, leftErr := strconv.ParseFloat(left, 64)
	rightNumber, rightErr := strconv.ParseFloat(right, 64)
	numeric := leftErr == nil && rightErr == nil
	if numeric {
		switch operator {
		case "=":
			return leftNumber == rightNumber
		case "!=", "<>":
			return leftNumber != rightNumber
		case ">":
			return leftNumber > rightNumber
		case ">=":
			return leftNumber >= rightNumber
		case "<":
			return leftNumber < rightNumber
		case "<=":
			return leftNumber <= rightNumber
		}
	}
	comparison := strings.Compare(left, right)
	switch operator {
	case "=":
		return left == right
	case "!=", "<>":
		return left != right
	case ">":
		return comparison > 0
	case ">=":
		return comparison >= 0
	case "<":
		return comparison < 0
	case "<=":
		return comparison <= 0
	default:
		return false
	}
}

func columnIndex(t *table, name string) int {
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
