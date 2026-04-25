package suggest

import ztrie "github.com/zyedidia/generic/trie"

type TrieEngine struct {
	index *ztrie.Trie[Item]
}

func NewTrie(items []Item) *TrieEngine {
	index := ztrie.New[Item]()
	for _, entry := range makeEntries(items) {
		index.Put(entry.key, entry.item)
	}
	return &TrieEngine{index: index}
}

func (e *TrieEngine) Name() string {
	return "zyedidia-trie"
}

func (e *TrieEngine) Suggest(text string, k int) []Item {
	top := newTopK(k)
	for _, key := range e.index.KeysWithPrefix(normalize(text)) {
		item, ok := e.index.Get(key)
		if ok {
			top.add(item)
		}
	}
	return top.result()
}
