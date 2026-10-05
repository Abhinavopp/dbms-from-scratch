package chapter09


type Row struct {
	ID    int
	Value string
}

func Scan(rows []Row, start, end int) []Row {
	out := make([]Row, 0)
	for _, row := range rows {
		if row.ID >= start && row.ID <= end {
			out = append(out, row)
		}
	}
	return out
}
