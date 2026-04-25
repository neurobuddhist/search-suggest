package suggest

import (
	"sort"
	"strings"
)

type SortedEngine struct {
	entries []entry
}

func NewSorted(items []Item) *SortedEngine {
	entries := makeEntries(items)
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].key != entries[j].key {
			return entries[i].key < entries[j].key
		}
		return ranksBefore(entries[i].item, entries[j].item)
	})
	return &SortedEngine{entries: entries}
}

func (e *SortedEngine) Name() string {
	return "sorted"
}

func (e *SortedEngine) Suggest(text string, k int) []Item {
	prefix := normalize(text)
	start := 0
	if prefix != "" {
		start = sort.Search(len(e.entries), func(i int) bool {
			return e.entries[i].key >= prefix
		})
	}

	top := newTopK(k)
	for i := start; i < len(e.entries); i++ {
		if !strings.HasPrefix(e.entries[i].key, prefix) {
			break
		}
		top.add(e.entries[i].item)
	}
	return top.result()
}
