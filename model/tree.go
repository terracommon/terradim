package model

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"
)

// sep is a string represeniting the os path separator
type sep string

type children []*Node

func (c children) Len() int           { return len(c) }
func (c children) Less(i, j int) bool { return c[i].key < c[j].key }
func (c children) Swap(i, j int)      { c[i], c[j] = c[j], c[i] }
func (c children) Sort()              { sort.Sort(c) }

// Node is used to represent edge and leaf nodes
type Node struct {
	sep      *string
	parent   *Node
	children children
	prefix   string
	key      string
	meta     interface{}
}

// Tree implements a directory trie using substrings
// between separator strings
type Tree struct {
	separator string
	size      int
	root      *Node
	mu        sync.Mutex
}

// WalkFunc is func signature for WalkSubtree
type WalkFunc func(node *Node, data interface{}) (bool, error)

// IsLeaf returns true if node has no children
func (n *Node) IsLeaf() bool {
	return len(n.children) == 0
}

// IsRoot returns true if node has no parent
func (n *Node) IsRoot() bool {
	return n.parent == nil
}

// Meta returns data stored in node
func (n *Node) Meta() interface{} {
	return n.meta
}

// Sep returns node sep
func (n *Node) Sep() string {
	return *n.sep
}

// Key returns node key
func (n *Node) Key() string {
	return n.key
}

// Path returns full node path
func (n *Node) Path() string {
	return fmt.Sprintf("%s%s%s", n.prefix, *n.sep, n.key)
}

// Children returns children stored in node
func (n *Node) Children() []*Node {
	return n.children
}

func (n *Node) appendChild(child *Node) {
	n.children = append(n.children, child)
	n.children.Sort()
}

func (n *Node) getChild(key string) *Node {
	length := len(n.children)
	keyIdx := sort.Search(length, func(i int) bool {
		return n.children[i].key >= key
	})
	if keyIdx < length && n.children[keyIdx].key == key {
		return n.children[keyIdx]
	}
	return nil
}

// lenCommonPrefix returns the length of the common
// substring between two strings
func lenCommonPrefix(p1, p2 string) int {
	var i int
	length := len(p1)
	if l := len(p2); l < length {
		length = l
	}
	for i = 0; i < length; i++ {
		if p1[i] != p2[i] {
			break
		}
	}
	return i
}

func removeEndSeparators(path, sep string) string {
	if sep == "" || path == sep {
		return path
	}
	lenSep := len(sep)
	clean := path
	for l := len(clean) - lenSep; len(clean) >= lenSep && clean[l:] == sep; l = len(clean) - lenSep {
		clean = clean[:l]
	}
	return clean
}

func removeStartSeparators(path, sep string) string {
	if sep == "" {
		return path
	}
	lenSep := len(sep)
	clean := path
	for len(clean) >= lenSep && clean[:lenSep] == sep {
		clean = clean[lenSep:]
	}
	return clean
}

func chunkSearchPath(search string, sep string) (key, nextSearch string) {
	if search == "" {
		return key, nextSearch
	}
	split := strings.SplitN(search, sep, 2)
	key = split[0]
	if len(split) == 2 {
		nextSearch = split[1]
	}
	return key, nextSearch
}

// Insert or update tree node at given path
func (t *Tree) Insert(path string, meta interface{}) (node *Node, isNew bool) {
	t.mu.Lock()
	defer t.mu.Unlock()

	var parent *Node
	sep := t.separator
	node = t.root
	path = removeEndSeparators(path, sep)
	search := removeStartSeparators(path, sep)
	key, nextSearch := chunkSearchPath(search, sep)

	for {
		if len(search) == 0 {
			isNew = false
			if node.IsRoot() == false {
				node.meta = meta
				return node, isNew
			}
			node = nil
			break
		}

		parent = node
		node = node.getChild(key)

		if node == nil {
			node = &Node{
				prefix: removeEndSeparators(path[:len(path)-len(search)], sep),
				key:    key,
				sep:    &t.separator,
				parent: parent,
			}
			parent.appendChild(node)
			t.size++

			if len(nextSearch) == 0 {
				node.meta = meta
				isNew = true
				break
			}
		}
		search = nextSearch
		key, nextSearch = chunkSearchPath(search, sep)
	}
	return node, isNew
}

// NewFromMap returns a new tree from a map with paths as
// keys and meta as values
func NewFromMap(treeMap map[string]interface{}) *Tree {
	t := &Tree{root: &Node{}, separator: string(os.PathSeparator)}
	t.root.sep = &t.separator
	for path, meta := range treeMap {
		t.Insert(path, meta)
	}
	return t
}

// NewTree returns a new tree
func NewTree() *Tree {
	return NewFromMap(nil)
}

// SetSeparator for tree
func (t *Tree) SetSeparator(separator string) {
	t.separator = separator
}

// Separator for tree
func (t *Tree) Separator() string {
	return t.separator
}

// Size return number of nodes in tree
func (t *Tree) Size() int {
	return t.size
}

// Root returns root node in tree
func (t *Tree) Root() *Node {
	return t.root
}

// findFromNode begins search at a startNode in
func findFromNode(node *Node, path string) (*Node, bool) {
	var (
		nodePath    string
		lenConsumed int
	)
	sep := *node.sep

	if node.IsRoot() == false {
		if len(node.prefix) > 0 {
			nodePath += node.prefix + sep
		}
		nodePath += node.key
	}
	lenConsumed = lenCommonPrefix(nodePath, path)
	path = removeEndSeparators(path, sep)
	search := removeStartSeparators(path[lenConsumed:], sep)
	key, nextSearch := chunkSearchPath(search, sep)

	for {
		if len(search) == 0 {
			if node != nil {
				return node, true
			}
			break
		}

		node = node.getChild(key)

		if node == nil {
			break
		}
		search = nextSearch
		key, nextSearch = chunkSearchPath(search, sep)
	}
	return nil, false
}

// Find node in tree by path
func (t *Tree) Find(path string) (*Node, bool) {
	return findFromNode(t.Root(), path)
}

// WalkSubtree visits children of a node and runs WalkFunc
func WalkSubtree(node *Node, walkFn WalkFunc, data interface{}) error {
	ok, err := walkFn(node, data)
	if err != nil {
		return err
	}
	if ok {
		children := node.Children()
		for _, child := range children {
			WalkSubtree(child, walkFn, data)
		}
	}
	return nil
}
