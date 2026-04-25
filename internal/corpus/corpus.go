package corpus

import "autocomplete/internal/suggest"

func Items() []suggest.Item {
	items := []suggest.Item{
		{Text: "go autocomplete", Score: 1200},
		{Text: "go benchmark", Score: 1120},
		{Text: "go context", Score: 1080},
		{Text: "go http server", Score: 1060},
		{Text: "go interface examples", Score: 1025},
		{Text: "golang trie implementation", Score: 1000},
		{Text: "golang radix tree", Score: 980},
		{Text: "search suggest architecture", Score: 960},
		{Text: "autocomplete ranking", Score: 940},
		{Text: "prefix search", Score: 920},
		{Text: "top k suggestions", Score: 900},
		{Text: "typeahead search", Score: 890},
		{Text: "binary search prefix range", Score: 870},
		{Text: "in memory index", Score: 850},
		{Text: "radix tree vs trie", Score: 830},
	}

	topics := []string{
		"go",
		"golang",
		"grpc",
		"graphql",
		"docker",
		"kubernetes",
		"postgres",
		"redis",
		"elasticsearch",
		"react",
		"typescript",
		"python",
		"java",
		"rust",
		"search",
		"autocomplete",
		"trie",
		"radix tree",
		"binary search",
		"ranking",
		"bm25",
		"vector search",
		"nearest neighbors",
		"machine learning",
		"iphone",
		"macbook",
		"coffee",
		"pizza",
		"flight",
		"hotel",
		"weather",
		"maps",
		"movies",
		"music",
		"books",
		"news",
		"finance",
	}
	intents := []string{
		"",
		" tutorial",
		" example",
		" benchmark",
		" github",
		" docs",
		" course",
		" interview questions",
		" best practices",
		" performance",
		" vs alternatives",
		" pricing",
		" near me",
		" 2026",
	}

	for i, topic := range topics {
		for j, intent := range intents {
			items = append(items, suggest.Item{
				Text:  topic + intent,
				Score: 780 - i*7 - j*3,
			})
		}
	}
	return items
}
