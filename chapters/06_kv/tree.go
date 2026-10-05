package chapter06

import "fmt"

const minDegree = 8

type node struct {
	Keys     []string `json:"keys"`
	Values   []string `json:"values"`
	Children []*node  `json:"children,omitempty"`
	Leaf     bool     `json:"leaf"`
}

type entry struct {
	key   string
	value string
}

func newNode(leaf bool) *node {
	return &node{Leaf: leaf}
}

func get(current *node, key string) (string, bool) {
	if current == nil {
		return "", false
	}
	i := 0
	for i < len(current.Keys) && key > current.Keys[i] {
		i++
	}
	if i < len(current.Keys) && key == current.Keys[i] {
		return current.Values[i], true
	}
	if current.Leaf {
		return "", false
	}
	return get(current.Children[i], key)
}

func set(tree *KV, key, value string) {
	if i := findKey(tree.root, key); i != nil {
		i.node.Values[i.keyIndex] = value
		return
	}
	if len(tree.root.Keys) == 2*minDegree-1 {
		root := newNode(false)
		root.Children = []*node{tree.root}
		splitChild(root, 0)
		tree.root = root
	}
	insertNonFull(tree.root, key, value)
}

type keyLocation struct {
	node     *node
	keyIndex int
}

func findKey(current *node, key string) *keyLocation {
	if current == nil {
		return nil
	}
	i := 0
	for i < len(current.Keys) && key > current.Keys[i] {
		i++
	}
	if i < len(current.Keys) && key == current.Keys[i] {
		return &keyLocation{node: current, keyIndex: i}
	}
	if current.Leaf {
		return nil
	}
	return findKey(current.Children[i], key)
}

func insertNonFull(current *node, key, value string) {
	i := len(current.Keys) - 1
	if current.Leaf {
		current.Keys = append(current.Keys, "")
		current.Values = append(current.Values, "")
		for i >= 0 && key < current.Keys[i] {
			current.Keys[i+1] = current.Keys[i]
			current.Values[i+1] = current.Values[i]
			i--
		}
		current.Keys[i+1] = key
		current.Values[i+1] = value
		return
	}

	for i >= 0 && key < current.Keys[i] {
		i--
	}
	i++
	if len(current.Children[i].Keys) == 2*minDegree-1 {
		splitChild(current, i)
		if key > current.Keys[i] {
			i++
		}
	}
	insertNonFull(current.Children[i], key, value)
}

func splitChild(parent *node, index int) {
	full := parent.Children[index]
	right := newNode(full.Leaf)
	median := minDegree - 1

	right.Keys = append(right.Keys, full.Keys[minDegree:]...)
	right.Values = append(right.Values, full.Values[minDegree:]...)
	if !full.Leaf {
		right.Children = append(right.Children, full.Children[minDegree:]...)
		full.Children = full.Children[:minDegree]
	}
	promotedKey, promotedValue := full.Keys[median], full.Values[median]
	full.Keys = full.Keys[:median]
	full.Values = full.Values[:median]

	parent.Keys = append(parent.Keys, "")
	parent.Values = append(parent.Values, "")
	copy(parent.Keys[index+1:], parent.Keys[index:])
	copy(parent.Values[index+1:], parent.Values[index:])
	parent.Keys[index], parent.Values[index] = promotedKey, promotedValue

	parent.Children = append(parent.Children, nil)
	copy(parent.Children[index+2:], parent.Children[index+1:])
	parent.Children[index+1] = right
}

func collect(current *node, entries *[]entry) {
	if current == nil {
		return
	}
	for i, key := range current.Keys {
		if !current.Leaf {
			collect(current.Children[i], entries)
		}
		*entries = append(*entries, entry{key: key, value: current.Values[i]})
	}
	if !current.Leaf {
		collect(current.Children[len(current.Children)-1], entries)
	}
}

func cloneNode(current *node) *node {
	if current == nil {
		return nil
	}
	cloned := &node{
		Keys:   append([]string(nil), current.Keys...),
		Values: append([]string(nil), current.Values...),
		Leaf:   current.Leaf,
	}
	for _, child := range current.Children {
		cloned.Children = append(cloned.Children, cloneNode(child))
	}
	return cloned
}

func validateNode(current *node, lower, upper *string) error {
	if current == nil {
		return fmt.Errorf("root node is missing")
	}
	if len(current.Keys) != len(current.Values) {
		return fmt.Errorf("node has %d keys and %d values", len(current.Keys), len(current.Values))
	}
	for i := 1; i < len(current.Keys); i++ {
		if current.Keys[i-1] >= current.Keys[i] {
			return fmt.Errorf("node keys are not strictly ordered")
		}
	}
	for _, key := range current.Keys {
		if lower != nil && key <= *lower || upper != nil && key >= *upper {
			return fmt.Errorf("node key %q is outside its parent range", key)
		}
	}
	if current.Leaf {
		if len(current.Children) != 0 {
			return fmt.Errorf("leaf node has children")
		}
		return nil
	}
	if len(current.Children) != len(current.Keys)+1 {
		return fmt.Errorf("internal node has %d keys and %d children", len(current.Keys), len(current.Children))
	}
	for i, child := range current.Children {
		childLower, childUpper := lower, upper
		if i > 0 {
			childLower = &current.Keys[i-1]
		}
		if i < len(current.Keys) {
			childUpper = &current.Keys[i]
		}
		if err := validateNode(child, childLower, childUpper); err != nil {
			return err
		}
	}
	return nil
}
