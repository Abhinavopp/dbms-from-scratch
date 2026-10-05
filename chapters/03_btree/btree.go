package chapter03


type Node struct {
	Keys     []int
	Children []*Node
	Leaf     bool
}

func newNode(leaf bool) *Node {
	return &Node{Leaf: leaf}
}

type BTree struct {
	Root  *Node
	Order int
}

func NewBTree(order int) *BTree {
	if order < 2 {
		order = 2
	}
	return &BTree{Root: newNode(true), Order: order}
}

func (t *BTree) Insert(key int) {
	if t.Root == nil {
		t.Root = newNode(true)
	}
	if t.Root.Leaf {
		t.Root.Keys = insertSorted(t.Root.Keys, key)
		if len(t.Root.Keys) > t.Order-1 {
			t.Root = splitRoot(t.Root)
		}
		return
	}
	insertIntoNode(t.Root, key)
	if len(t.Root.Keys) > t.Order-1 {
		t.Root = splitRoot(t.Root)
	}
}

func insertIntoNode(node *Node, key int) {
	if node == nil {
		return
	}
	if node.Leaf {
		node.Keys = insertSorted(node.Keys, key)
		return
	}

	idx := 0
	for idx < len(node.Keys) && key > node.Keys[idx] {
		idx++
	}
	if idx < len(node.Children) {
		insertIntoNode(node.Children[idx], key)
	}
	if len(node.Children[idx].Keys) > nodeChildrenLimit(node.Children[idx]) {
		node.Children[idx] = splitNode(node.Children[idx])
	}
}

func splitRoot(node *Node) *Node {
	if len(node.Keys) <= 1 {
		return node
	}
	mid := len(node.Keys) / 2
	left := newNode(true)
	left.Keys = append(left.Keys, node.Keys[:mid]...)
	right := newNode(true)
	right.Keys = append(right.Keys, node.Keys[mid+1:]...)
	root := newNode(false)
	root.Keys = []int{node.Keys[mid]}
	root.Children = []*Node{left, right}
	return root
}

func splitNode(node *Node) *Node {
	if node == nil || len(node.Keys) <= 1 {
		return node
	}
	mid := len(node.Keys) / 2
	left := newNode(true)
	left.Keys = append(left.Keys, node.Keys[:mid]...)
	right := newNode(true)
	right.Keys = append(right.Keys, node.Keys[mid+1:]...)
	root := newNode(false)
	root.Keys = []int{node.Keys[mid]}
	root.Children = []*Node{left, right}
	return root
}

func nodeChildrenLimit(node *Node) int {
	if node == nil {
		return 1
	}
	return len(node.Keys)
}

func insertSorted(values []int, key int) []int {
	idx := 0
	for idx < len(values) && values[idx] < key {
		idx++
	}
	values = append(values, 0)
	copy(values[idx+1:], values[idx:])
	values[idx] = key
	return values
}

func (t *BTree) Search(key int) bool {
	return searchNode(t.Root, key)
}

func searchNode(node *Node, key int) bool {
	if node == nil {
		return false
	}
	for i, value := range node.Keys {
		if value == key {
			return true
		}
		if value > key {
			if node.Leaf {
				return false
			}
			return searchNode(node.Children[i], key)
		}
	}
	if node.Leaf {
		return false
	}
	if len(node.Children) == 0 {
		return false
	}
	return searchNode(node.Children[len(node.Children)-1], key)
}
