package suggest

import (
	"bytes"

	iradix "github.com/hashicorp/go-immutable-radix/v2"
)

type HashicorpRadixEngine struct {
	tree *iradix.Tree[Item]
}

func NewHashicorpRadix(items []Item) *HashicorpRadixEngine {
	tree := iradix.New[Item]()
	txn := tree.Txn()
	for _, entry := range makeEntries(items) {
		txn.Insert([]byte(entry.key), entry.item)
	}
	return &HashicorpRadixEngine{tree: txn.Commit()}
}

func (e *HashicorpRadixEngine) Name() string {
	return "hashicorp-radix"
}

func (e *HashicorpRadixEngine) Suggest(text string, k int) []Item {
	prefix := []byte(normalize(text))
	top := newTopK(k)

	iter := e.tree.Root().Iterator()
	iter.SeekPrefix(prefix)
	for key, item, ok := iter.Next(); ok; key, item, ok = iter.Next() {
		if !bytes.HasPrefix(key, prefix) {
			break
		}
		top.add(item)
	}
	return top.result()
}
