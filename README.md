# In-memory Search Suggest Engines in Go

Учебно-исследовательский проект про autocomplete/search suggest: сравнение простых in-memory подходов и trie с cached top-k suggestions на Go.

Демо: https://disk.yandex.ru/i/9_rvjOm52cxBOw
0:53 оговорка, не все префиксы, а всех кандидатов для префикса пользователя

## Заметки автора

> [!NOTE]
> Проект сделан как исследование Backend + IR: как базовые структуры данных ведут себя в задаче autocomplete/search suggest.
> 
> Сравниваются базовые подходы автодополнения: linear scan, sorted array, generic trie/radix и top-k-aware trie.

## API

```text
GET /api/suggest?text=go&engine=ranked-trie&k=8
GET /api/engines
```

В демо-интерфейсе оставлены только выбранные engines:

| Engine | Роль |
| --- | --- |
| `linear` | correctness baseline: полный перебор |
| `sorted` | сильный простой baseline: binary search по prefix range + top-k scan |
| `radix` | mutable radix baseline на `github.com/armon/go-radix` |
| `ranked-trie` | custom top-k-aware trie: быстрый read-heavy query path за счёт cached top-k на prefix node |

## Запуск

```powershell
$env:GOCACHE = "$PWD\.gocache"
go run ./cmd/server
```

Страничка на:

```text
http://localhost:8080
```

## Thread-Safety

Текущая модель проекта - build-on-start, read-only serving.

Индексы строятся один раз при старте приложения, после этого HTTP handlers только вызывают `Suggest`. Проект использует engines только в immutable/read-only режиме.

| Component | Thread-safety модель |
| --- | --- |
| `Registry` | map заполняется при старте и дальше только читается |
| `linear` / `sorted` | immutable slices после build |
| `radix` (`go-radix`) | build happens before serving; after publication only read operations are used |
| `ranked-trie` | custom trie публикуется после build и дальше не мутируется; maps/slices используются только для чтения |

В проекте нет online updates и in-place rebuild. Если добавлять live reload индекса, безопасная модель - copy-build-swap: новый индекс строится отдельно, затем публикуется целиком через `atomic.Value`, `sync.RWMutex` или другой явный snapshot swap. In-place mutation существующего trie/radix во время `Suggest` запрещена.

## Benchmarks

Быстрый smoke test для проверки, что benchmark-и запускаются локально:

```powershell
$env:GOCACHE = "$PWD\.gocache"
go test "./internal/suggest" -run "^$" -bench "BenchmarkSelectedSuggestEngines" -benchmem -benchtime=1s
```

Основной benchmark-прогон для отчёта:

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

Dataset synthetic, поэтому результаты не стоит переносить напрямую на production traffic. Benchmark-и показывают относительное поведение структур данных на контролируемой нагрузке: broad/medium/narrow/missing prefixes, mixed traffic и scale tests.

Один benchmark run используется только как smoke test. Таблицы в отчёте собраны по медиане из `-count=10`, чтобы уменьшить влияние шума планировщика, GC и фоновой нагрузки.

Полный отчёт на русском: [reports/engine-benchmarks.md](reports/engine-benchmarks.md).

Сырой вывод:

- [reports/benchmarks-2026-04-26.txt](reports/benchmarks-2026-04-26.txt)
- [reports/benchmarks-scale-mixed-2026-04-26.txt](reports/benchmarks-scale-mixed-2026-04-26.txt)
- [reports/retained-heap-2026-04-26.txt](reports/retained-heap-2026-04-26.txt)

## Scope and Limitations

Проект не является production search engine. Это учебно-исследовательский benchmark in-memory подходов к prefix-based search suggest.

Что осознанно не реализовано:

- ML ranking: score в проекте задан заранее и не обучается на real query/click logs.
- Typo correction: нет исправления опечаток, fuzzy matching и edit distance.
- Personalization: подсказки не зависят от пользователя, истории, региона или контекста.
- Online rebuild / incremental updates: индекс строится заранее и используется как read-only snapshot.
- Distributed serving: нет шардинга, репликации, multi-node deployment и consistency модели.
- Anti-fraud / abuse protection: нет защиты от накруток, спама и adversarial queries.

Ограничения benchmark-а:

- Dataset synthetic, без real query logs.
- Benchmark запущен на local machine, не в production-like окружении.
- Retained heap измеряется отдельным diagnostic test, а не напрямую через `testing.B`.
- Online incremental updates отдельно не benchmark-ились.

Проект сделан с использованием LLM, выводы основаны на raw benchmark output в `reports/`.
