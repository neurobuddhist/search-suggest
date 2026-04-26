package suggest

import (
	"strings"

	dtrie "github.com/dghubble/trie"
)

type DghubbleTrieEngine struct {
	index *dtrie.RuneTrie
}

func NewDghubbleTrie(items []Item) *DghubbleTrieEngine {
	index := dtrie.NewRuneTrie()
	for _, entry := range makeEntries(items) {
		index.Put(entry.key, entry.item)
	}
	return &DghubbleTrieEngine{index: index}
}

func (e *DghubbleTrieEngine) Name() string {
	return "dghubble-rune-trie"
}

func (e *DghubbleTrieEngine) Suggest(text string, k int) []Item {
	prefix := normalize(text)
	top := newTopK(k)
	_ = e.index.Walk(func(key string, value interface{}) error {
		if !strings.HasPrefix(key, prefix) {
			return nil
		}

		item, ok := value.(Item)
		if ok {
			top.add(item)
		}
		return nil
	})
	return top.result()
}
