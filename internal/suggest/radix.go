package suggest

import aradix "github.com/armon/go-radix"

type RadixEngine struct {
	tree *aradix.Tree
}

func NewRadix(items []Item) *RadixEngine {
	tree := aradix.New()
	for _, entry := range makeEntries(items) {
		tree.Insert(entry.key, entry.item)
	}
	return &RadixEngine{tree: tree}
}

func (e *RadixEngine) Name() string {
	return "radix"
}

func (e *RadixEngine) Suggest(text string, k int) []Item {
	top := newTopK(k)
	e.tree.WalkPrefix(normalize(text), func(_ string, value interface{}) bool {
		item, ok := value.(Item)
		if ok {
			top.add(item)
		}
		return false
	})
	return top.result()
}
