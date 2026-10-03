package chapter07

// FreeList reuses page ids instead of allocating new ones forever.
type FreeList struct {
	free   []int
	nextID int
}

func NewFreeList() *FreeList {
	return &FreeList{nextID: 1}
}

func (f *FreeList) Allocate() int {
	if len(f.free) > 0 {
		id := f.free[len(f.free)-1]
		f.free = f.free[:len(f.free)-1]
		return id
	}
	id := f.nextID
	f.nextID++
	return id
}

func (f *FreeList) Free(id int) {
	f.free = append(f.free, id)
}
