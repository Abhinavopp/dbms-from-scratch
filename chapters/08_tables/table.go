package chapter08

// Schema defines the logical fields of a table.
type Schema struct {
	Name   string
	Fields []string
}

// Record is a row identified by an integer primary key.
type Record struct {
	ID   int
	Data map[string]string
}

// Table is the first row-oriented layer built on top of the KV store.
type Table struct {
	Schema Schema
	rows   map[int]Record
}

func NewTable(schema Schema) *Table {
	return &Table{Schema: schema, rows: make(map[int]Record)}
}

func (t *Table) Insert(record Record) {
	if record.ID == 0 {
		record.ID = len(t.rows) + 1
	}
	if t.rows == nil {
		t.rows = make(map[int]Record)
	}
	t.rows[record.ID] = record
}

func (t *Table) Get(id int) (Record, bool) {
	record, ok := t.rows[id]
	return record, ok
}

func (t *Table) All() []Record {
	out := make([]Record, 0, len(t.rows))
	for _, row := range t.rows {
		out = append(out, row)
	}
	return out
}
