package chapter05


type Node struct {
	Keys     []int
	Children []*Node
	Leaf     bool
}

func newNode(leaf bool) *Node {
	return &Node{Leaf: leaf}
}


type Tree struct {
	Root  *Node
	Order int
}

func NewTree(order int) *Tree {
	if order < 2 {
		order = 2
	}
	return &Tree{Root: newNode(true), Order: order}
}

func (t *Tree) Insert(key int) {
	if t.Root == nil {
		t.Root = newNode(true)
	}
	t.Root.Keys = insertSorted(t.Root.Keys, key)
}

func splitRoot(node *Node) *Node {
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

func (t *Tree) Delete(key int) {
	if t.Root == nil {
		return
	}
	if deleteFromNode(t.Root, key) {
		if len(t.Root.Keys) == 0 && len(t.Root.Children) > 0 {
			t.Root = t.Root.Children[0]
		}
		if len(t.Root.Keys) == 0 && t.Root.Leaf {
			t.Root = nil
		}
	}
}

func deleteFromNode(node *Node, key int) bool {
	if node == nil {
		return false
	}
	for i, value := range node.Keys {
		if value == key {
			node.Keys = append(node.Keys[:i], node.Keys[i+1:]...)
			return true
		}
	}
	if node.Leaf {
		return false
	}
	idx := 0
	for idx < len(node.Keys) && key > node.Keys[idx] {
		idx++
	}
	if idx >= len(node.Children) {
		idx = len(node.Children) - 1
	}
	if deleteFromNode(node.Children[idx], key) {
		if len(node.Children[idx].Keys) == 0 && len(node.Children[idx].Children) > 0 {
			node.Children[idx] = node.Children[idx].Children[0]
		}
	}
	return false
}

func (t *Tree) Search(key int) bool {
	return searchNode(t.Root, key)
}

func searchNode(node *Node, key int) bool {
	if node == nil {
		return false
	}
	for _, value := range node.Keys {
		if value == key {
			return true
		}
	}
	for _, child := range node.Children {
		if searchNode(child, key) {
			return true
		}
	}
	return false
}
