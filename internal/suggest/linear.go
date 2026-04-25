package suggest

import "strings"

type LinearEngine struct {
	entries []entry
}

func NewLinear(items []Item) *LinearEngine {
	return &LinearEngine{entries: makeEntries(items)}
}

func (e *LinearEngine) Name() string {
	return "linear"
}

func (e *LinearEngine) Suggest(text string, k int) []Item {
	prefix := normalize(text)
	top := newTopK(k)
	for _, entry := range e.entries {
		if strings.HasPrefix(entry.key, prefix) {
			top.add(entry.item)
		}
	}
	return top.result()
}
