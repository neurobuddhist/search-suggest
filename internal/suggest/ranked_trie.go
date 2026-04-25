package suggest

type RankedTrieEngine struct {
	root   *rankedTrieNode
	cacheK int
}

type rankedTrieNode struct {
	children map[rune]*rankedTrieNode
	item     *Item
	top      []Item
}

func NewRankedTrie(items []Item, cacheK int) *RankedTrieEngine {
	if cacheK <= 0 {
		cacheK = 10
	}

	engine := &RankedTrieEngine{
		root:   &rankedTrieNode{},
		cacheK: cacheK,
	}
	for _, entry := range makeEntries(items) {
		engine.insert(entry)
	}
	return engine
}

func (e *RankedTrieEngine) Name() string {
	return "ranked-trie"
}

func (e *RankedTrieEngine) Suggest(text string, k int) []Item {
	node := e.root
	for _, r := range normalize(text) {
		next := node.children[r]
		if next == nil {
			return nil
		}
		node = next
	}

	if k <= e.cacheK {
		return cloneTop(node.top, k)
	}

	candidates := make([]Item, 0)
	collectRankedTrie(node, &candidates)
	return rankItems(candidates, k)
}

func (e *RankedTrieEngine) insert(entry entry) {
	node := e.root
	node.top = insertTopK(node.top, entry.item, e.cacheK)
	for _, r := range entry.key {
		if node.children == nil {
			node.children = make(map[rune]*rankedTrieNode)
		}
		if node.children[r] == nil {
			node.children[r] = &rankedTrieNode{}
		}
		node = node.children[r]
		node.top = insertTopK(node.top, entry.item, e.cacheK)
	}

	item := entry.item
	node.item = &item
}

func collectRankedTrie(node *rankedTrieNode, out *[]Item) {
	if node.item != nil {
		*out = append(*out, *node.item)
	}
	for _, child := range node.children {
		collectRankedTrie(child, out)
	}
}
