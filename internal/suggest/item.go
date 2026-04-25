package suggest

import (
	"sort"
	"strings"
)

// Item is a searchable phrase with a simple popularity score.
type Item struct {
	Text  string `json:"text"`
	Score int    `json:"score"`
}

// Engine returns top-k suggestions for a text prefix.
type Engine interface {
	Name() string
	Suggest(text string, k int) []Item
}

type entry struct {
	item Item
	key  string
}

func makeEntries(items []Item) []entry {
	byKey := make(map[string]Item, len(items))
	for _, item := range items {
		item.Text = strings.TrimSpace(item.Text)
		if item.Text == "" {
			continue
		}

		key := normalize(item.Text)
		prev, ok := byKey[key]
		if !ok || ranksBefore(item, prev) {
			byKey[key] = item
		}
	}

	entries := make([]entry, 0, len(byKey))
	for key, item := range byKey {
		entries = append(entries, entry{item: item, key: key})
	}
	return entries
}

func normalize(text string) string {
	return strings.ToLower(strings.TrimSpace(text))
}

func rankItems(items []Item, k int) []Item {
	if k <= 0 || len(items) == 0 {
		return nil
	}

	ranked := append([]Item(nil), items...)
	sort.Slice(ranked, func(i, j int) bool {
		return ranksBefore(ranked[i], ranked[j])
	})

	if len(ranked) > k {
		ranked = ranked[:k]
	}
	return ranked
}

func ranksBefore(a, b Item) bool {
	if a.Score != b.Score {
		return a.Score > b.Score
	}
	return normalize(a.Text) < normalize(b.Text)
}

func cloneTop(items []Item, k int) []Item {
	if k <= 0 || len(items) == 0 {
		return nil
	}
	if len(items) < k {
		k = len(items)
	}
	out := make([]Item, k)
	copy(out, items[:k])
	return out
}

type topK struct {
	items []Item
	k     int
}

func newTopK(k int) topK {
	if k <= 0 {
		return topK{}
	}
	return topK{
		items: make([]Item, 0, k),
		k:     k,
	}
}

func (t *topK) add(item Item) {
	if t.k <= 0 {
		return
	}
	t.items = insertTopK(t.items, item, t.k)
}

func (t *topK) result() []Item {
	return cloneTop(t.items, t.k)
}

func insertTopK(top []Item, item Item, k int) []Item {
	if k <= 0 {
		return top
	}

	pos := sort.Search(len(top), func(i int) bool {
		return ranksBefore(item, top[i])
	})

	if len(top) < k {
		top = append(top, Item{})
		copy(top[pos+1:], top[pos:])
		top[pos] = item
		return top
	}
	if pos < k {
		copy(top[pos+1:], top[pos:k-1])
		top[pos] = item
	}
	return top
}
