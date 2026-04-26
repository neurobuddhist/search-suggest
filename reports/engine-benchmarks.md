# Benchmark Report: In-memory Search Suggest Engines

## Заметки автора

> [!IMPORTANT]
> Это учебно-исследовательский бенчмарк, а не серьёзное исследование для production системы.
>
> Цель сравнить базовые in-memory подходы к prefix suggest.
>
> Бенчмарки проведены с использованием LLM. Выводы основаны на raw benchmark output в `reports/`.

## Summary

Benchmark сравнивает реализации search suggest на 100k synthetic phrases при `k=10`.

Главный результат: `ranked-trie` имеет самый быстрый query path на этом workload. Он хранит cached top-k suggestions рядом с prefix node, поэтому `broad` и `medium` prefix queries почти не зависят от числа кандидатов под subtree.

Score в проекте захардкожен, поэтому benchmark проверяет механику bounded top-k retrieval under prefix. 

`sorted` - самый сильный простой baseline в этом benchmark-е: мало retained memory, быстрый build и хорошее поведение на `narrow`/`missing`.

Обычный trie (`zyedidia-trie`) полезен как учебный и benchmark baseline после `sorted`, но для bounded top-k suggest он всё равно перечисляет кандидатов под prefix.

Итоговый выбор зависит от tradeoff-а: query latency vs build time vs memory.

---

## Glossary

| Термин | Значение в этом отчёте |
| --- | --- |
| `search suggest` / `autocomplete` | Подсказки по мере ввода текста пользователем. В этом проекте рассматривается только prefix-based suggest. |
| `prefix` | Начало строки, по которому ищутся подходящие suggestions. Например, для `go` candidates могут быть `golang`, `google`, `go tutorial`. |
| `suggestion` | Один результат подсказки: текст + score. |
| `candidate` | Строка из dataset, которая подходит под prefix и потенциально может попасть в top-k. |
| `k` | Максимальное число suggestions, которое нужно вернуть. В benchmark-е используется `k=10`. |
| `top-k` | Лучшие `k` suggestions по сортировке `score DESC`, затем `text ASC`. |
| `bounded top-k retrieval` | Контракт запроса: вернуть не все candidates под prefix, а только ограниченный top-k. |
| `score` | Числовой вес suggestion. В проекте score захардкожен и нужен только для демонстрации top-k mechanics. Качество ranking-а не измеряется. |
| `ranking` | Продуктовая/ML-задача сортировки suggestions по полезности для пользователя. В этом проекте полноценный ranking не реализован. |
| `tie-breaking` | Правило сортировки при одинаковом score. В benchmark-е используется `score DESC`, затем `text ASC`. |
| `query path` | Код, который выполняется во время `Suggest(prefix, k)`. Для read-heavy suggest это самая важная часть. |
| `build path` | Код построения индекса перед serving. Может быть дорогим, если это уменьшает query latency. |
| `read-heavy workload` | Нагрузка, где запросов на чтение сильно больше, чем обновлений индекса. |
| `linear scan` | Наивный baseline: пройти по всем строкам и проверить prefix. |
| `sorted` | Baseline на отсортированном массиве строк: binary search находит prefix range, затем range сканируется для top-k. |
| `trie` | Prefix tree: структура данных, где путь от root соответствует символам prefix. |
| `radix tree` | Сжатый trie, где ребро может хранить не один символ, а строковый fragment. |
| `ternary search trie` / `TST` | Trie-like структура, где каждый node хранит символ и три направления поиска: less/equal/greater. |
| `adaptive radix tree` / `ART` | Radix tree, который меняет representation node в зависимости от числа children. |
| `prefix traversal` / `prefix enumeration` | Поиск prefix node и обход всех descendants/candidates под ним. |
| `subtree` | Часть дерева под найденным prefix node. На broad prefix subtree может быть большим. |
| `cached top-k` | Предрассчитанный top-k suggestions, сохранённый прямо в node. Позволяет не обходить subtree во время query. |
| `materialization` | Создание/сбор полного списка candidates или result slice во время query. |
| `result ownership` | Контракт владения результатом: можно ли caller-у мутировать returned slice, или это read-only cached data. |
| `synthetic dataset` | Искусственно сгенерированный dataset. Он полезен для контролируемого сравнения, но не заменяет real query logs. |
| `workload` | Набор query cases, на которых измеряется engine: `broad`, `medium`, `narrow`, `missing`, `mixed`. |
| `broad prefix` | Короткий prefix с большим числом candidates. |
| `medium prefix` | Prefix со средним числом candidates, ближе к обычному пользовательскому вводу. |
| `narrow prefix` | Selective prefix, где candidates мало. |
| `missing prefix` | Prefix, которого нет в индексе. |
| `ns/op` | Nanoseconds per operation: средняя стоимость одной benchmark operation. |
| `B/op` | Bytes allocated per operation: allocation traffic за одну operation. |
| `allocs/op` | Количество allocations per operation. |
| `retained heap` | Память, которая остаётся занятой после build и GC. Отличается от `B/op`, который показывает allocation traffic. |
| `benchstat` | Инструмент для статистического сравнения Go benchmark outputs. В отчёте используются медианы, но не полноценный benchstat-анализ. |

---

## Окружение

| Поле | Значение |
| --- | --- |
| Дата | 2026-04-26 |
| OS | Windows |
| CPU | Intel i5-11600K |
| Go | 1.25.4 |
| Dataset | 100k synthetic phrases |
| k | 10 |

Основной benchmark-прогон:

```powershell
$env:GOCACHE = "$PWD\.gocache"
go test "./internal/suggest" `
  -run "^$" `
  -bench "Benchmark(TrieImplementations|RadixTrieImplementations|TopKImplementations|SelectedSuggestEngines)" `
  -benchmem `
  -benchtime=3s `
  -count=10 `
  *> reports\benchmarks-2026-04-26.txt
```

Дополнительный прогон для `mixed` и `scale`:

```powershell
$env:GOCACHE = "$PWD\.gocache"
go test "./internal/suggest" `
  -run "^$" `
  -bench "Benchmark(SuggestMixedTraffic|ScaleBuild|Scale)$" `
  -benchmem `
  -benchtime=3s `
  -count=10 `
  *> reports\benchmarks-scale-mixed-2026-04-26.txt
```

Retained heap diagnostic:

```powershell
$env:GOCACHE = "$PWD\.gocache"
$env:SUGGEST_PRINT_RETAINED = "1"
go test "./internal/suggest" -run "TestRetainedHeapComparison" -count=1 -v *> reports\retained-heap-2026-04-26.txt
```

Таблицы ниже используют медиану из 10 benchmark runs (`-count=10`). Retained heap - отдельный diagnostic test после build и GC, поэтому он не является `benchstat`-метрикой.

---

## Workload

| Case | Смысл |
| --- | --- |
| `broad` | короткий prefix, много кандидатов |
| `medium` | обычный пользовательский prefix |
| `narrow` | почти точное совпадение |
| `missing` | prefix отсутствует |
| `mixed` | 20% broad, 60% medium, 10% narrow, 10% missing |

---

## Engines

| Engine | Идея | Роль |
| --- | --- | --- |
| `linear` | full scan | baseline для корректности и производительности |
| `sorted` | sorted strings + binary search range + top-k accumulator | сильный простой baseline |
| `zyedidia-trie` | ternary search trie | обычный trie baseline |
| `dghubble-rune-trie` | rune trie | trie baseline, слабый API fit для suggest |
| `radix` / `go-radix` | mutable radix tree | generic radix baseline |
| `hashicorp-radix` | immutable radix tree | immutable radix baseline |
| `adaptive-radix` | adaptive radix tree | ART research baseline |
| `ranked-trie` | custom trie с cached top-k в каждом node | самый быстрый read-optimized engine |

---

## Fairness

Build и suggest benchmarks разделены. В suggest benchmark engine строится до `b.ResetTimer()`.

Все engines используют:

- один dataset;
- одинаковые prefix sets;
- одинаковый `k=10`;
- fixed seed для mixed workload;
- package-level sink, чтобы компилятор не выбросил результат.

Hot loop:

```go
res := engine.Suggest(prefixes[i%len(prefixes)], k)
sinkSuggestions = res
sinkInt += len(res)
```

Корректность проверяется относительно `linear` на детерминированном dataset. Tie-breaking: `score DESC`, затем `text ASC`.

Важное ограничение: allocation patterns не выравниваются искусственно. Они являются частью tradeoff-а конкретной реализации и её API.

---

## Build Results

Median of 10 samples, `benchtime=3s`.

| Engine | ns/op | B/op | allocs/op |
| --- | ---: | ---: | ---: |
| `linear` | 9,257,214 | 10,302,746 | 259 |
| `sorted` | 38,390,319 | 10,302,738 | 262 |
| `zyedidia-trie` | 85,438,548 | 30,736,146 | 319,532 |
| `dghubble-rune-trie` | 84,489,307 | 63,043,192 | 863,361 |
| `radix` | 53,691,829 | 29,104,538 | 508,838 |
| `hashicorp-radix` | 97,013,157 | 76,618,992 | 940,416 |
| `adaptive-radix` | 59,347,471 | 29,101,118 | 571,446 |
| `ranked-trie` | 136,495,641 | 86,100,870 | 1,243,726 |

Коротко:

- `linear` дешевле всего строится;
- `sorted` платит за сортировку, но остаётся экономным по памяти;
- `radix` строится быстрее и дешевле, чем `hashicorp-radix`;
- `ranked-trie` дороже всего строится, потому что заранее считает top-k cache.

---

## Suggest Results

Median of 10 samples, `benchtime=3s`.

### Broad

| Engine | ns/op | B/op | allocs/op |
| --- | ---: | ---: | ---: |
| `linear` | 789,293 | 480 | 2 |
| `sorted` | 245,392 | 480 | 2 |
| `zyedidia-trie` | 5,876,984 | 1,763,522 | 17,222 |
| `dghubble-rune-trie` | 40,275,144 | 7,935,783 | 319,274 |
| `radix` | 1,873,149 | 480 | 2 |
| `hashicorp-radix` | 1,662,804 | 856 | 7 |
| `adaptive-radix` | 14,200,517 | 998,232 | 53,453 |
| `ranked-trie` | 94.6 | 240 | 1 |

На broad prefix обычные trie/radix обходят много кандидатов. `ranked-trie` возвращает cached top-k и не материализует subtree.

### Medium

| Engine | ns/op | B/op | allocs/op |
| --- | ---: | ---: | ---: |
| `linear` | 558,487 | 480 | 2 |
| `sorted` | 101,534 | 480 | 2 |
| `zyedidia-trie` | 2,374,201 | 643,864 | 7,520 |
| `dghubble-rune-trie` | 41,146,329 | 7,935,783 | 319,274 |
| `radix` | 661,046 | 480 | 2 |
| `hashicorp-radix` | 598,050 | 856 | 7 |
| `adaptive-radix` | 12,459,941 | 998,232 | 53,453 |
| `ranked-trie` | 117.05 | 240 | 1 |

На обычном пользовательском prefix `sorted` - сильный простой baseline. `ranked-trie` всё равно на порядки быстрее, потому что query cost ограничен lookup-ом prefix и возвратом cached top-k.

### Narrow

| Engine | ns/op | B/op | allocs/op |
| --- | ---: | ---: | ---: |
| `linear` | 383,106 | 264 | 2 |
| `sorted` | 164.2 | 264 | 2 |
| `zyedidia-trie` | 313.85 | 280 | 3 |
| `dghubble-rune-trie` | 39,383,230 | 7,935,568 | 319,274 |
| `radix` | 161.15 | 264 | 2 |
| `hashicorp-radix` | 206.3 | 304 | 4 |
| `adaptive-radix` | 11,350,978 | 998,032 | 53,453 |
| `ranked-trie` | 155 | 24 | 1 |

На selective prefix `sorted`, `radix` и `ranked-trie` близки. Если workload в основном selective, простой `sorted` может быть достаточно хорош.

### Missing

| Engine | ns/op | B/op | allocs/op |
| --- | ---: | ---: | ---: |
| `linear` | 373,707 | 240 | 1 |
| `sorted` | 119.45 | 240 | 1 |
| `zyedidia-trie` | 69.48 | 240 | 1 |
| `dghubble-rune-trie` | 40,021,579 | 7,935,544 | 319,273 |
| `radix` | 71.33 | 240 | 1 |
| `hashicorp-radix` | 72.14 | 240 | 1 |
| `adaptive-radix` | 11,741,697 | 997,993 | 53,452 |
| `ranked-trie` | 13.43 | 0 | 0 |

Missing prefix дешёв для деревьев: lookup быстро выходит. `ranked-trie` остаётся самым быстрым и без аллокаций.

### Mixed Traffic

| Engine | ns/op | B/op | allocs/op |
| --- | ---: | ---: | ---: |
| `linear` | 529,503 | 433 | 1 |
| `sorted` | 103,649 | 434 | 1 |
| `radix` | 634,086 | 433 | 1 |
| `hashicorp-radix` | 638,328 | 734 | 6 |
| `ranked-trie` | 114.05 | 194 | 0 |

Mixed traffic показывает практическую картину: `sorted` - лучший простой baseline, generic radix страдает на broad/medium части трафика, `ranked-trie` почти не зависит от размера prefix range.

---

## Retained Memory

`B/op` в build benchmark показывает allocation traffic. Retained heap показывает, сколько памяти остаётся после build и GC.

Diagnostic run: `reports/retained-heap-2026-04-26.txt`.

Для 100k фраз:

| Engine | Retained heap | Bytes per phrase |
| --- | ---: | ---: |
| `sorted` | 8,984,176 | 89.84 |
| `radix` | 18,735,552 | 187.36 |
| `hashicorp-radix` | 48,890,928 | 488.91 |
| `ranked-trie` | 76,656,624 | 766.57 |

Коротко:

- `sorted` самый экономный;
- `radix` примерно в 2 раза дороже `sorted`;
- `hashicorp-radix` заметно тяжелее из-за immutable overhead;
- `ranked-trie` самый дорогой: он покупает query latency памятью.

---

## Scale Results

Значения бенчмарков приведены как медианы по 10 запускам. Retained memory измерена отдельным diagnostic run для retained heap.

| Dataset size | Engine | broad ns/op | medium ns/op | build ns/op | retained memory |
| ---: | --- | ---: | ---: | ---: | ---: |
| 10,000 | `sorted` | 21,100 | 10,099 | 3,354,300 | 866,304 |
| 10,000 | `radix` | 47,086 | 19,793 | 4,122,562 | 1,893,408 |
| 10,000 | `hashicorp-radix` | 44,942 | 18,976 | 6,506,407 | 4,878,024 |
| 10,000 | `ranked-trie` | 93.86 | 114.9 | 12,212,201 | 8,606,272 |
| 100,000 | `sorted` | 262,777 | 94,810 | 37,043,186 | 8,984,176 |
| 100,000 | `radix` | 1,865,869 | 758,119 | 50,900,325 | 18,735,552 |
| 100,000 | `hashicorp-radix` | 1,806,559 | 686,643 | 96,719,544 | 48,890,928 |
| 100,000 | `ranked-trie` | 99.48 | 120.75 | 137,527,725 | 76,656,624 |

Коротко:

- `sorted` растёт вместе с prefix range;
- `radix` и `hashicorp-radix` тоже растут, потому что обходят subtree;
- `ranked-trie` почти не меняется на broad/medium query latency;
- цена `ranked-trie` видна в build time и retained memory.

---

## Analysis

`linear` нужен только как baseline. Build дешёвый, query - `O(N)`.

`sorted` - самая сильная простая реализация. Она экономна по памяти и очень быстра на selective prefix. Слабое место - broad/medium prefix, где приходится сканировать matching range.

`zyedidia-trie` полезен как обычный trie baseline после `sorted`, но он перечисляет ключи под prefix и потом добирает top-k. Для broad top-k suggest это не тот query contract.

`dghubble-rune-trie` не выбран: adapter вынужден делать full walk для suggest-like queries.

`radix` / `go-radix` полезен как mutable radix baseline, но тоже перечисляет subtree.

`hashicorp-radix` даёт immutable snapshot semantics, но в этом benchmark-е стоит дороже по build allocation traffic и retained memory.

`adaptive-radix` не выбран для этого workload. В этом adapter latency и allocations высокие и почти не зависят от selectivity prefix.

`ranked-trie` - самый быстрый read-optimized engine. Он заранее считает top-k в узлах, поэтому query не обходит subtree. Цена - build time и retained memory.

---

## Final Decision

| Use case | Engine | Причина |
| --- | --- | --- |
| correctness baseline | `linear` | самая простая reference-реализация |
| simple production-like baseline | `sorted` | мало памяти, простота, сильный selective-prefix performance |
| trie research baseline | `zyedidia-trie` | нужен в benchmark-е для честной линейки `sorted -> trie -> radix` |
| generic mutable radix baseline | `radix` | лучший generic radix вариант из проверенных |
| immutable radix baseline | `hashicorp-radix` | полезен для snapshot semantics, но тяжелее |
| fastest read-heavy suggest | `ranked-trie` | минимальная query latency, максимальная цена по memory/build |

В HTTP API сейчас оставлены только selected engines. Обычные trie baselines остаются в benchmark/test коде, чтобы API не раздувался исследовательскими вариантами.

---

## Limitations

- Synthetic dataset.
- Local machine benchmark.
- Нет real query logs.
- Нет personalization.
- Нет typo correction.
- Нет ML ranking.
- Нет distributed serving.
- Нет benchmark-а online incremental updates.
- Retained heap измеряется отдельным diagnostic test, а не `testing.B`.

---

## Main Conclusion

Правильная структура данных определяется контрактом запроса.

Для search suggest контракт - bounded top-k retrieval under a prefix. Generic trie/radix структуры оптимизируют prefix enumeration. Top-k aware структуры оптимизируют фактический suggest workload.

Поэтому `ranked-trie` доминирует на broad и medium prefix queries, а `sorted` остаётся самым сильным простым baseline.
