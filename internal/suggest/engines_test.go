package suggest

import (
	"fmt"
	"math/rand"
	"os"
	"reflect"
	"runtime"
	"strconv"
	"testing"
)

type Suggestion = Item

var sinkSuggestions []Suggestion
var sinkInt int
var sinkEngine Engine
var sinkItems []Item

const (
	benchmarkItemCount = 100_000
	benchmarkK         = 10
)

func TestEnginesAgree(t *testing.T) {
	items := []Item{
		{Text: "go", Score: 100},
		{Text: "go benchmark", Score: 90},
		{Text: "go context", Score: 80},
		{Text: "golang", Score: 70},
		{Text: "golang trie", Score: 60},
		{Text: "graph", Score: 50},
		{Text: "grpc", Score: 40},
		{Text: "redis", Score: 30},
	}

	linear := NewLinear(items)
	engines := []Engine{
		NewSorted(items),
		NewTrie(items),
		NewDghubbleTrie(items),
		NewRadix(items),
		NewHashicorpRadix(items),
		NewAdaptiveRadix(items),
		NewRankedTrie(items, benchmarkK),
	}

	for _, query := range []string{"", "g", "go", "gol", "gr", "redis", "missing"} {
		want := linear.Suggest(query, 5)
		for _, engine := range engines {
			got := engine.Suggest(query, 5)
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("%s query %q:\n got: %#v\nwant: %#v", engine.Name(), query, got, want)
			}
		}
	}
}

func TestEnginesNormalizeRankAndDedupe(t *testing.T) {
	items := []Item{
		{Text: " Go ", Score: 10},
		{Text: "go", Score: 20},
		{Text: "GO", Score: 15},
		{Text: "go docs", Score: 30},
		{Text: "go delve", Score: 30},
		{Text: "gopher", Score: 5},
	}
	want := []Item{
		{Text: "go delve", Score: 30},
		{Text: "go docs", Score: 30},
		{Text: "go", Score: 20},
	}

	for _, factory := range allFactories() {
		engine := factory.build(items)
		got := engine.Suggest(" GO ", 3)
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("%s:\n got: %#v\nwant: %#v", engine.Name(), got, want)
		}
	}
}

func TestRankedTrieFallsBackWhenKExceedsCache(t *testing.T) {
	items := []Item{
		{Text: "go one", Score: 10},
		{Text: "go two", Score: 20},
		{Text: "go three", Score: 30},
		{Text: "go four", Score: 40},
	}

	engine := NewRankedTrie(items, 2)
	got := engine.Suggest("go", 4)
	want := rankItems(items, 4)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got: %#v\nwant: %#v", got, want)
	}
}

func TestRetainedHeapComparison(t *testing.T) {
	if os.Getenv("SUGGEST_PRINT_RETAINED") != "1" {
		t.Skip("set SUGGEST_PRINT_RETAINED=1 to print retained heap table")
	}

	for _, size := range []int{10_000, benchmarkItemCount} {
		for _, factory := range retainedMemoryFactories() {
			retained := measureRetainedHeap(factory, size)
			t.Logf("%d\t%s\t%d\t%.2f", size, factory.name, retained, float64(retained)/float64(size))
		}
	}
}

func BenchmarkBuild(b *testing.B) {
	runBuildBenchmarks(b, allFactories(), benchmarkItemCount, "")
}

func BenchmarkSuggest(b *testing.B) {
	runSuggestBenchmarks(b, allFactories(), benchmarkItemCount, "")
}

func BenchmarkTrieImplementations(b *testing.B) {
	factories := []engineFactory{
		{name: "zyedidia-trie", build: func(items []Item) Engine { return NewTrie(items) }},
		{name: "dghubble-rune-trie", build: func(items []Item) Engine { return NewDghubbleTrie(items) }},
	}

	runBuildBenchmarks(b, factories, benchmarkItemCount, "build")
	runSuggestBenchmarks(b, factories, benchmarkItemCount, "suggest")
}

func BenchmarkRadixTrieImplementations(b *testing.B) {
	factories := []engineFactory{
		{name: "go-radix", build: func(items []Item) Engine { return NewRadix(items) }},
		{name: "hashicorp-radix", build: func(items []Item) Engine { return NewHashicorpRadix(items) }},
		{name: "adaptive-radix", build: func(items []Item) Engine { return NewAdaptiveRadix(items) }},
	}

	runBuildBenchmarks(b, factories, benchmarkItemCount, "build")
	runSuggestBenchmarks(b, factories, benchmarkItemCount, "suggest")
}

func BenchmarkTopKImplementations(b *testing.B) {
	factories := []engineFactory{
		{name: "sorted-range-topk", build: func(items []Item) Engine { return NewSorted(items) }},
		{name: "ranked-trie", build: func(items []Item) Engine { return NewRankedTrie(items, benchmarkK) }},
	}

	runBuildBenchmarks(b, factories, benchmarkItemCount, "build")
	runSuggestBenchmarks(b, factories, benchmarkItemCount, "suggest")
}

func BenchmarkSelectedSuggestEngines(b *testing.B) {
	factories := selectedFactories()

	runBuildBenchmarks(b, factories, benchmarkItemCount, "build")
	runSuggestBenchmarks(b, factories, benchmarkItemCount, "suggest")
}

func BenchmarkSuggestMixedTraffic(b *testing.B) {
	items := benchmarkItems(benchmarkItemCount)
	prefixes := mixedPrefixes(10_000)

	for _, factory := range selectedFactories() {
		engine := factory.build(items)
		b.Run(factory.name, func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				res := engine.Suggest(prefixes[i%len(prefixes)], benchmarkK)
				sinkSuggestions = res
				sinkInt += len(res)
			}
			runtime.KeepAlive(engine)
		})
	}
}

func BenchmarkScaleBuild(b *testing.B) {
	for _, size := range scaleSizes() {
		for _, factory := range scaleFactories() {
			items := benchmarkItems(size)
			b.Run(strconv.Itoa(size)+"/"+factory.name, func(b *testing.B) {
				b.ReportAllocs()
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					sinkEngine = factory.build(items)
				}
				runtime.KeepAlive(items)
			})
		}
	}
}

func BenchmarkScale(b *testing.B) {
	for _, size := range scaleSizes() {
		items := benchmarkItems(size)
		cases := []struct {
			name     string
			prefixes []string
		}{
			{name: "broad", prefixes: prefixCases()["broad"]},
			{name: "medium", prefixes: prefixCases()["medium"]},
		}

		for _, factory := range scaleFactories() {
			engine := factory.build(items)
			for _, tc := range cases {
				b.Run(strconv.Itoa(size)+"/"+factory.name+"/"+tc.name, func(b *testing.B) {
					b.ReportAllocs()
					b.ResetTimer()
					for i := 0; i < b.N; i++ {
						res := engine.Suggest(tc.prefixes[i%len(tc.prefixes)], benchmarkK)
						sinkSuggestions = res
						sinkInt += len(res)
					}
					runtime.KeepAlive(engine)
				})
			}
		}
	}
}

func runBuildBenchmarks(b *testing.B, factories []engineFactory, itemCount int, group string) {
	items := benchmarkItems(itemCount)

	for _, factory := range factories {
		name := factory.name
		if group != "" {
			name = group + "/" + name
		}

		b.Run(name, func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				sinkEngine = factory.build(items)
			}
			runtime.KeepAlive(items)
		})
	}
}

func runSuggestBenchmarks(b *testing.B, factories []engineFactory, itemCount int, group string) {
	items := benchmarkItems(itemCount)
	cases := prefixCases()

	for _, factory := range factories {
		engine := factory.build(items)
		for _, caseName := range []string{"broad", "medium", "narrow", "missing"} {
			name := caseName + "/" + factory.name
			if group != "" {
				name = group + "/" + name
			}

			prefixes := cases[caseName]
			b.Run(name, func(b *testing.B) {
				b.ReportAllocs()
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					res := engine.Suggest(prefixes[i%len(prefixes)], benchmarkK)
					sinkSuggestions = res
					sinkInt += len(res)
				}
				runtime.KeepAlive(engine)
			})
		}
	}
}

type engineFactory struct {
	name  string
	build func([]Item) Engine
}

func allFactories() []engineFactory {
	return []engineFactory{
		{name: "linear", build: func(items []Item) Engine { return NewLinear(items) }},
		{name: "sorted", build: func(items []Item) Engine { return NewSorted(items) }},
		{name: "zyedidia-trie", build: func(items []Item) Engine { return NewTrie(items) }},
		{name: "dghubble-rune-trie", build: func(items []Item) Engine { return NewDghubbleTrie(items) }},
		{name: "go-radix", build: func(items []Item) Engine { return NewRadix(items) }},
		{name: "hashicorp-radix", build: func(items []Item) Engine { return NewHashicorpRadix(items) }},
		{name: "adaptive-radix", build: func(items []Item) Engine { return NewAdaptiveRadix(items) }},
		{name: "ranked-trie", build: func(items []Item) Engine { return NewRankedTrie(items, benchmarkK) }},
	}
}

func selectedFactories() []engineFactory {
	return []engineFactory{
		{name: "linear", build: func(items []Item) Engine { return NewLinear(items) }},
		{name: "sorted", build: func(items []Item) Engine { return NewSorted(items) }},
		{name: "radix", build: func(items []Item) Engine { return NewRadix(items) }},
		{name: "hashicorp-radix", build: func(items []Item) Engine { return NewHashicorpRadix(items) }},
		{name: "ranked-trie", build: func(items []Item) Engine { return NewRankedTrie(items, benchmarkK) }},
	}
}

func scaleFactories() []engineFactory {
	return []engineFactory{
		{name: "sorted", build: func(items []Item) Engine { return NewSorted(items) }},
		{name: "go-radix", build: func(items []Item) Engine { return NewRadix(items) }},
		{name: "hashicorp-radix", build: func(items []Item) Engine { return NewHashicorpRadix(items) }},
		{name: "ranked-trie", build: func(items []Item) Engine { return NewRankedTrie(items, benchmarkK) }},
	}
}

func retainedMemoryFactories() []engineFactory {
	return []engineFactory{
		{name: "sorted", build: func(items []Item) Engine { return NewSorted(items) }},
		{name: "go-radix", build: func(items []Item) Engine { return NewRadix(items) }},
		{name: "hashicorp-radix", build: func(items []Item) Engine { return NewHashicorpRadix(items) }},
		{name: "ranked-trie", build: func(items []Item) Engine { return NewRankedTrie(items, benchmarkK) }},
	}
}

func prefixCases() map[string][]string {
	return map[string][]string{
		"broad":   {"g", "r", "p", "s"},
		"medium":  {"go", "golang", "docker", "search", "vector"},
		"narrow":  {"go tutorial 000000", "docker docs 000052", "search github 000072"},
		"missing": {"zz", "unknown"},
	}
}

func mixedPrefixes(n int) []string {
	cases := prefixCases()
	prefixes := make([]string, 0, n)
	add := func(caseName string, count int) {
		values := cases[caseName]
		for i := 0; i < count; i++ {
			prefixes = append(prefixes, values[i%len(values)])
		}
	}

	add("broad", n*20/100)
	add("medium", n*60/100)
	add("narrow", n*10/100)
	add("missing", n-len(prefixes))

	rng := rand.New(rand.NewSource(42))
	rng.Shuffle(len(prefixes), func(i, j int) {
		prefixes[i], prefixes[j] = prefixes[j], prefixes[i]
	})
	return prefixes
}

func scaleSizes() []int {
	sizes := []int{10_000, 100_000}
	if os.Getenv("SUGGEST_BENCH_1M") == "1" {
		sizes = append(sizes, 1_000_000)
	}
	return sizes
}

func measureRetainedHeap(factory engineFactory, itemCount int) int64 {
	sinkItems = nil
	sinkEngine = nil
	sinkSuggestions = nil
	runtime.GC()
	var before runtime.MemStats
	runtime.ReadMemStats(&before)

	items := benchmarkItems(itemCount)
	engine := factory.build(items)
	sinkItems = items
	sinkEngine = engine

	runtime.GC()
	var after runtime.MemStats
	runtime.ReadMemStats(&after)

	runtime.KeepAlive(items)
	runtime.KeepAlive(engine)

	return int64(after.HeapAlloc) - int64(before.HeapAlloc)
}

func benchmarkItems(n int) []Item {
	stems := []string{
		"go",
		"golang",
		"grpc",
		"graphql",
		"docker",
		"kubernetes",
		"postgres",
		"redis",
		"search",
		"autocomplete",
		"trie",
		"radix",
		"ranking",
		"vector",
		"python",
		"react",
	}
	intents := []string{
		"tutorial",
		"example",
		"benchmark",
		"docs",
		"github",
		"course",
		"best practices",
		"performance",
		"interview",
		"pricing",
	}

	items := make([]Item, 0, n)
	for i := 0; i < n; i++ {
		stem := stems[i%len(stems)]
		intent := intents[(i/len(stems))%len(intents)]
		items = append(items, Item{
			Text:  fmt.Sprintf("%s %s %06d", stem, intent, i),
			Score: 1_000_000 - (i*37)%100_000,
		})
	}
	return items
}
