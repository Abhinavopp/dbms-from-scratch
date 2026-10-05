package chapter10


type Index struct {
	byValue map[string][]int
}

func NewIndex() *Index {
	return &Index{byValue: make(map[string][]int)}
}

func (i *Index) Add(value string, rowID int) {
	if i.byValue == nil {
		i.byValue = make(map[string][]int)
	}
	i.byValue[value] = append(i.byValue[value], rowID)
}

func (i *Index) Find(value string) []int {
	ids := i.byValue[value]
	out := make([]int, len(ids))
	copy(out, ids)
	return out
}
