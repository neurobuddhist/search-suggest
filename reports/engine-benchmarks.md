# Benchmark Report: Search Suggest Engines

## Summary

`ranked-trie` имеет самый быстрый query path в этом benchmark-е. Он хранит cached top-k рядом с каждым prefix node, поэтому `broad` и `medium` prefix queries почти не зависят от числа кандидатов под subtree.

`sorted` остается лучшим простым baseline: мало памяти, понятный build, сильное поведение на `narrow` и `missing`. На `broad` и `medium` latency растет вместе с размером prefix range.

`zyedidia-trie`, `dghubble-rune-trie`, `go-radix`, `hashicorp-radix` и `adaptive-radix` полезны как исследовательские prefix index baselines. Они показывают важную границу: generic trie/radix обычно оптимизирует prefix enumeration, а не bounded top-k suggest.

`olympos.io/container/pruning-radix-trie` удален из проекта. По API он подходил под autocomplete, но dependency adoption слабый: пакет не в latest version своего module и на pkg.go.dev показывает `Imported by: 0`. Для pet project, который надо показывать на интервью, лучше не тащить такую зависимость в `go.mod`.

Production-выбор зависит от read/write ratio, memory budget, build time и требований к latency. Самый быстрый query path не всегда означает лучший общий выбор.

## Environment

| Field | Value |
| --- | --- |
| Date | 2026-04-25 |
| OS | Windows amd64 |
| CPU | 11th Gen Intel(R) Core(TM) i5-11600K @ 3.90GHz |
| Go | 1.25.4 |
| Dataset | 100k synthetic phrases |
| k | 10 |

Benchstat-friendly запуск:

```powershell
$env:GOCACHE = "$PWD\.gocache"

go test "./internal/suggest" `
  -run "^$" `
  -bench "Benchmark(TrieImplementations|RadixTrieImplementations|TopKImplementations|SelectedSuggestEngines)" `
  -benchmem `
  -benchtime=3s `
  -count=10 `
  > bench.txt

benchstat bench.txt
```

Один прогон benchmark-а - это smoke test. Для уверенных выводов нужен `-count=10`; `benchstat` показывает разброс и помогает не принимать шум за результат. Если разница между engines в десятки или сотни раз, общий вывод обычно устойчив даже без идеальной статистики, но `benchstat` все равно нужен для аккуратного отчета.

## Workload

| Case | Prefix examples | Meaning |
| --- | --- | --- |
| `broad` | `g`, `r`, `p`, `s` | короткий prefix, много кандидатов |
| `medium` | `go`, `golang`, `docker`, `search`, `vector` | обычный пользовательский prefix |
| `narrow` | `go tutorial 000000`, `docker docs 000052` | почти точечное совпадение |
| `missing` | `zz`, `unknown` | prefix отсутствует |
| `mixed` | fixed 10k prefix list | 20% broad, 60% medium, 10% narrow, 10% missing |

`broad` показывает тяжелый случай. `narrow` и `missing` показывают selective path. `mixed` ближе к среднему пользовательскому трафику, но все еще synthetic.

## Dependency Selection

| Package | Adoption signal | Role |
| --- | --- | --- |
| `github.com/zyedidia/generic` | GitHub показывает 1.4k stars | upstream trie/TST baseline |
| `github.com/dghubble/trie` | GitHub показывает 505 stars | popular trie baseline for `Get`; poor API fit for suggest |
| `github.com/armon/go-radix` | GitHub показывает 936 stars и `Used by 21.8k` | mutable radix baseline |
| `github.com/hashicorp/go-immutable-radix/v2` | GitHub показывает 1.1k stars | immutable radix baseline |
| `github.com/plar/go-adaptive-radix-tree/v2` | GitHub показывает 414 stars | ART baseline |
| custom `ranked-trie` | локальная реализация | top-k aware read-optimized engine |

Удалено:

| Package | Причина |
| --- | --- |
| `github.com/st1064870/generic` | no-name fork; заменен на upstream `github.com/zyedidia/generic` |
| `olympos.io/container/pruning-radix-trie` | weak adoption signal: pkg.go.dev показывает `Imported by: 0`; не держим в `go.mod` |

## Engines

| Engine | Idea | Role |
| --- | --- | --- |
| `linear` | full scan по всем фразам | correctness/performance baseline |
| `sorted` | sorted strings + binary search range + bounded top-k accumulator | strong simple baseline |
| `zyedidia-trie` | `github.com/zyedidia/generic/trie`, ternary search trie | ordinary trie/TST baseline |
| `dghubble-rune-trie` | `github.com/dghubble/trie`, rune trie | popular trie baseline; not selected |
| `go-radix` / `radix` | `github.com/armon/go-radix`, mutable radix tree | selected generic radix baseline |
| `hashicorp-radix` | `github.com/hashicorp/go-immutable-radix/v2`, immutable radix tree | selected immutable radix baseline |
| `adaptive-radix` | `github.com/plar/go-adaptive-radix-tree/v2` | ART research baseline; not selected |
| `ranked-trie` | custom trie with cached top-k per node | fastest read-optimized engine in this benchmark |

## Fairness

Все engines используют один synthetic dataset и одинаковые prefix sets. `k` одинаковый: `10`.

Build benchmark и suggest benchmark разделены: build time не попадает в query latency. В query benchmark engine строится до `b.ResetTimer()`. Prefix list создается до `b.ResetTimer()`. В hot loop выполняется только `Suggest(prefix, k)` и запись результата в package-level sink:

```go
res := engine.Suggest(prefixes[i%len(prefixes)], k)
sinkSuggestions = res
sinkInt += len(res)
```

Mixed workload использует fixed seed и 10k заранее подготовленных prefix-ов. Данные не генерируются внутри hot loop.

Tie-breaking одинаковый для correctness-сравнения: выше `score`, затем лексикографически меньше `text`. `TestEnginesAgree` проверяет одинаковый top-k на тестовом наборе.

Ограничение fairness: allocation patterns являются частью реализации. Например, `zyedidia-trie` возвращает ключи через `KeysWithPrefix`, `dghubble-rune-trie` не дает эффективного prefix subtree API и adapter делает полный `Walk`, а `ranked-trie` возвращает cached top-k. Это не выравнивается искусственно, потому что именно API и поведение структуры влияют на пригодность для suggest.

## Build Results

| Engine | ns/op | B/op | allocs/op |
| --- | ---: | ---: | ---: |
| `linear` | 11,261,162 | 10,302,616 | 259 |
| `sorted` | 45,300,031 | 10,302,736 | 262 |
| `zyedidia-trie` | 106,797,375 | 30,736,480 | 319,532 |
| `dghubble-rune-trie` | 105,503,867 | 63,043,198 | 863,361 |
| `go-radix` | 60,324,292 | 29,104,538 | 508,838 |
| `hashicorp-radix` | 116,939,956 | 76,625,394 | 940,406 |
| `adaptive-radix` | 70,672,800 | 29,101,120 | 571,446 |
| `ranked-trie` | 170,845,417 | 86,100,912 | 1,243,726 |

`linear` почти бесплатен на build: он только нормализует и хранит данные. `sorted` дороже из-за сортировки, но остается дешевым по памяти. `ranked-trie` самый дорогой на build, потому что кладет top-k cache во многие узлы. Build time не равен query time: `ranked-trie` платит заранее, чтобы ускорить чтение.

`hashicorp-radix` строится заметно дороже `go-radix`: immutable структура и transaction path дают цену по allocation traffic. Это нормально для immutable radix, но для batch-built read-only suggest индекса такой build cost надо учитывать отдельно.

## Suggest Results

### Broad

| Engine | ns/op | B/op | allocs/op |
| --- | ---: | ---: | ---: |
| `linear` | 1,050,870 | 480 | 2 |
| `sorted` | 413,637 | 480 | 2 |
| `zyedidia-trie` | 6,798,259 | 1,766,943 | 17,252 |
| `dghubble-rune-trie` | 45,817,756 | 7,935,786 | 319,274 |
| `go-radix` | 2,266,603 | 480 | 2 |
| `hashicorp-radix` | 2,097,447 | 856 | 7 |
| `adaptive-radix` | 16,493,563 | 998,233 | 53,453 |
| `ranked-trie` | 108.6 | 240 | 1 |

`broad` показывает главный тезис проекта. Обычные trie/radix проходят много кандидатов под prefix. `ranked-trie` не материализует subtree и возвращает cached top-k, поэтому разница становится не микрооптимизацией, а другим классом query path.

### Medium

| Engine | ns/op | B/op | allocs/op |
| --- | ---: | ---: | ---: |
| `linear` | 851,015 | 480 | 2 |
| `sorted` | 154,354 | 480 | 2 |
| `zyedidia-trie` | 2,771,944 | 644,051 | 7,521 |
| `dghubble-rune-trie` | 46,655,563 | 7,935,784 | 319,274 |
| `go-radix` | 975,791 | 480 | 2 |
| `hashicorp-radix` | 875,971 | 856 | 7 |
| `adaptive-radix` | 15,192,246 | 998,232 | 53,453 |
| `ranked-trie` | 126.9 | 240 | 1 |

На обычном пользовательском prefix `sorted` уже сильно лучше linear, но все еще зависит от размера range. Generic trie/radix тоже обходят subtree. `ranked-trie` работает как bounded top-k suggest engine, а не как prefix enumeration engine.

### Narrow

| Engine | ns/op | B/op | allocs/op |
| --- | ---: | ---: | ---: |
| `linear` | 557,129 | 264 | 2 |
| `sorted` | 186.8 | 264 | 2 |
| `zyedidia-trie` | 335.8 | 280 | 3 |
| `dghubble-rune-trie` | 43,824,756 | 7,935,567 | 319,274 |
| `go-radix` | 173.6 | 264 | 2 |
| `hashicorp-radix` | 228.1 | 304 | 4 |
| `adaptive-radix` | 14,273,179 | 998,032 | 53,453 |
| `ranked-trie` | 162.9 | 24 | 1 |

На почти точечном совпадении `sorted`, `go-radix` и `ranked-trie` близки. Это важный tradeoff: если workload почти всегда selective, простой `sorted` может быть достаточно хорош.

### Missing

| Engine | ns/op | B/op | allocs/op |
| --- | ---: | ---: | ---: |
| `linear` | 582,861 | 240 | 1 |
| `sorted` | 132.3 | 240 | 1 |
| `zyedidia-trie` | 80.65 | 240 | 1 |
| `dghubble-rune-trie` | 44,453,040 | 7,935,541 | 319,273 |
| `go-radix` | 87.73 | 240 | 1 |
| `hashicorp-radix` | 82.21 | 240 | 1 |
| `adaptive-radix` | 15,071,630 | 997,992 | 53,452 |
| `ranked-trie` | 14.52 | 0 | 0 |

На отсутствующем prefix дерево может выйти рано, поэтому `zyedidia-trie`, `go-radix` и `hashicorp-radix` выглядят хорошо. `ranked-trie` все равно быстрее за счет короткого lookup path и отсутствия аллокаций.

### Mixed Traffic

| Engine | ns/op | B/op | allocs/op |
| --- | ---: | ---: | ---: |
| `linear` | 659,351 | 432 | 1 |
| `sorted` | 130,664 | 434 | 1 |
| `go-radix` | 917,878 | 433 | 1 |
| `hashicorp-radix` | 893,461 | 736 | 6 |
| `ranked-trie` | 121.8 | 194 | 0 |

Mixed workload показывает усредненную картину: `sorted` остается сильным baseline, `go-radix` и `hashicorp-radix` проигрывают на broad/medium части трафика, а `ranked-trie` почти не реагирует на размер prefix range.

## Trie Implementations

| Engine | Broad ns/op | Medium ns/op | Narrow ns/op | Missing ns/op | Вывод |
| --- | ---: | ---: | ---: | ---: | --- |
| `zyedidia-trie` | 6,798,259 | 2,771,944 | 335.8 | 80.65 | нормальный ordinary trie/TST baseline; broad/medium дорогие из-за materialized keys |
| `dghubble-rune-trie` | 45,817,756 | 46,655,563 | 43,824,756 | 44,453,040 | не подходит для suggest adapter: нет эффективного prefix subtree API, приходится делать full walk |

`dghubble/trie` не плохая библиотека. Ее README прямо позиционирует use case вокруг быстрых `Get` после upfront `Put`. Для suggest нужен другой contract: bounded top-k под prefix. Поэтому она остается в исследовательской группе, но не выбирается для API.

## Radix Implementations

| Engine | Broad ns/op | Medium ns/op | Narrow ns/op | Missing ns/op | Вывод |
| --- | ---: | ---: | ---: | ---: | --- |
| `go-radix` | 2,266,603 | 975,791 | 173.6 | 87.73 | лучший mutable generic radix baseline |
| `hashicorp-radix` | 2,097,447 | 875,971 | 228.1 | 82.21 | похожая query latency, дороже build/memory из-за immutable design |
| `adaptive-radix` | 16,493,563 | 15,192,246 | 14,273,179 | 15,071,630 | observed behavior в этом adapter не подходит для suggest workload |

`go-radix` и `hashicorp-radix` близки по query на broad/medium. `hashicorp-radix` чуть быстрее на этих smoke числах, но платит заметно большим build allocation traffic и retained heap.

`adaptive-radix` формулирую аккуратно: в этой версии и с этим adapter latency и allocations почти не зависят от selectivity prefix. Это не оценка библиотеки в целом, а вывод для данного query contract.

## Top-K Implementations

| Engine | Broad ns/op | Medium ns/op | Narrow ns/op | Missing ns/op | Вывод |
| --- | ---: | ---: | ---: | ---: | --- |
| `sorted-range-topk` | 473,346 | 150,955 | 185.5 | 130.5 | сильный простой baseline |
| `ranked-trie` | 104.0 | 130.3 | 161.9 | 14.99 | самый быстрый query path в этом benchmark-е |

В Go-наборе зависимостей после фильтра по adoption не осталось library top-k autocomplete engine, который я бы оставил в `go.mod`. Поэтому top-k comparison теперь честнее: простой `sorted-range-topk` против custom `ranked-trie`.

## Retained Memory

Команда:

```powershell
$env:GOCACHE = "$PWD\.gocache"
$env:SUGGEST_PRINT_RETAINED = "1"
go test "./internal/suggest" -run "TestRetainedHeapComparison" -count=1 -v
```

Для 100k фраз:

| Engine | Retained heap | Bytes per phrase |
| --- | ---: | ---: |
| `sorted` | 8,984,160 | 89.84 |
| `go-radix` | 18,740,688 | 187.41 |
| `hashicorp-radix` | 48,887,112 | 488.87 |
| `ranked-trie` | 76,656,624 | 766.57 |

`B/op` в build benchmark показывает allocation traffic: сколько байт было выделено за операцию построения. Retained heap показывает другое: сколько памяти осталось жить после построения индекса и GC. Для capacity planning retained heap важнее, чем только `B/op`.

`sorted` самый экономный. `go-radix` примерно в два раза дороже `sorted`. `hashicorp-radix` заметно дороже `go-radix`. `ranked-trie` самый дорогой: он покупает минимальную query latency за счет памяти.

## Scale Results

`BenchmarkScale` по умолчанию запускает 10k и 100k. 1M включается явно:

```powershell
$env:SUGGEST_BENCH_1M = "1"
```

| Dataset size | Engine | broad ns/op | medium ns/op | build ns/op | retained memory |
| ---: | --- | ---: | ---: | ---: | ---: |
| 10,000 | `sorted` | 23,383 | 10,653 | 3,546,612 | 871,296 |
| 10,000 | `go-radix` | 52,704 | 21,418 | 4,630,619 | 1,903,520 |
| 10,000 | `hashicorp-radix` | 49,318 | 21,329 | 7,249,685 | 4,862,800 |
| 10,000 | `ranked-trie` | 104.0 | 134.5 | 13,612,255 | 8,606,288 |
| 100,000 | `sorted` | 315,840 | 120,789 | 43,989,064 | 8,984,160 |
| 100,000 | `go-radix` | 2,502,490 | 999,270 | 62,886,615 | 18,740,688 |
| 100,000 | `hashicorp-radix` | 2,024,497 | 862,246 | 118,359,867 | 48,887,112 |
| 100,000 | `ranked-trie` | 103.3 | 127.2 | 170,055,729 | 76,656,624 |
| 1,000,000 | `sorted` | TODO | TODO | TODO | TODO |
| 1,000,000 | `go-radix` | TODO | TODO | TODO | TODO |
| 1,000,000 | `hashicorp-radix` | TODO | TODO | TODO | TODO |
| 1,000,000 | `ranked-trie` | TODO | TODO | TODO | TODO |

Scale показывает форму роста. `sorted` растет вместе с размером prefix range. `go-radix` и `hashicorp-radix` тоже обходят subtree, поэтому на broad prefix растут заметно. `ranked-trie` почти не зависит от числа кандидатов в broad/medium, потому что использует top-k cache. Цена `ranked-trie` видна в build time и retained memory.

## Analysis

`linear` - хороший correctness/performance baseline. Build почти бесплатный, query `O(N)`, масштабируется плохо. Он нужен для сравнения, но не как реальный suggest engine.

`sorted` - сильный простой baseline. Build дороже linear из-за сортировки. Query хорош на `narrow` и `missing`, а `broad`/`medium` зависят от размера prefix range. Памяти нужно мало. Это хороший кандидат, если нужна простота и приемлемая latency.

`zyedidia-trie` быстро выходит к prefix node, но затем материализует ключи под subtree. Поэтому `broad` и `medium` дорогие. Обычный trie решает prefix enumeration, а не top-k suggest.

`dghubble-rune-trie` оптимизирован под быстрые `Get`, но не под autocomplete suggest. В этом adapter он делает полный `Walk` и фильтрацию по prefix, поэтому latency почти не зависит от selectivity и остается высокой.

`go-radix` - лучший generic mutable radix baseline из проверенных. Он сжимает prefix structure и обычно лучше plain trie, но без top-k cache все равно обходит subtree для broad prefix.

`hashicorp-radix` - сильный immutable radix baseline. Query похож на `go-radix`, иногда немного быстрее на broad/medium в smoke run, но build и retained memory заметно дороже. Его сильная сторона - immutable snapshot semantics, а не минимальный memory footprint для этого pet project.

`adaptive-radix` по измерениям в этой версии и с этим adapter не подходит для suggest workload. Наблюдаемое поведение: latency и allocations почти не зависят от selectivity prefix. Формулировка важна: это observed behavior in this benchmark, а не "библиотека плохая".

`ranked-trie` имеет самый быстрый query path в этом benchmark-е. Cached top-k в каждом node меняет класс сложности query: `broad` и `medium` почти не зависят от числа кандидатов. Цена - build time, retained memory и большое количество allocation during build. Это лучший вариант для read-heavy in-memory suggest, если индекс можно перестраивать batch/offline.

## Final Decision

| Use case | Recommended engine | Why |
| --- | --- | --- |
| correctness baseline | `linear` | проще всего проверить корректность |
| simple baseline | `sorted` | мало памяти, понятная реализация, сильные narrow/missing |
| generic prefix tree baseline | `go-radix` | лучший mutable radix baseline из проверенных |
| immutable radix baseline | `hashicorp-radix` | полезен как проверенный immutable вариант, но дороже по памяти/build |
| fastest read-heavy suggest | `ranked-trie` | минимальная query latency при высокой цене памяти/build |

## Limitations

- Synthetic dataset.
- Local machine benchmark.
- Один smoke-прогон в таблицах; для финального отчета нужен `-count=10` и `benchstat`.
- Нет real query logs.
- Нет personalization.
- Нет typo correction.
- Нет ML ranking.
- Нет distributed serving.
- Нет benchmark-а online incremental updates.
- Dependency popularity не является security proof; это только pragmatic supply-chain filter.

## Main Conclusion

Правильная структура данных определяется контрактом запроса.

Для search suggest контракт - bounded top-k retrieval under a prefix. Generic trie/radix структуры оптимизируют prefix enumeration. Top-k aware структуры оптимизируют фактический suggest workload.

Поэтому `ranked-trie` доминирует на `broad` и `medium` prefix queries, а `sorted` остается сильным простым baseline. Generic trie/radix остаются полезными baseline-ами, но без top-k awareness они решают не тот query contract.
