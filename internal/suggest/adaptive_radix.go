package suggest

import art "github.com/plar/go-adaptive-radix-tree/v2"

type AdaptiveRadixEngine struct {
	tree art.Tree
}

func NewAdaptiveRadix(items []Item) *AdaptiveRadixEngine {
	tree := art.New()
	for _, entry := range makeEntries(items) {
		tree.Insert(art.Key(entry.key), entry.item)
	}
	return &AdaptiveRadixEngine{tree: tree}
}

func (e *AdaptiveRadixEngine) Name() string {
	return "adaptive-radix"
}

func (e *AdaptiveRadixEngine) Suggest(text string, k int) []Item {
	top := newTopK(k)
	e.tree.ForEachPrefix(art.Key(normalize(text)), func(node art.Node) bool {
		item, ok := node.Value().(Item)
		if ok {
			top.add(item)
		}
		return true
	})
	return top.result()
}
