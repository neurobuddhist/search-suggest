# In-memory Search Suggest Engines in Go

Учебно-исследовательский проект про autocomplete/search suggest: сравнение простых in-memory подходов и trie с cached top-k suggestions на Go.

## Заметки автора.

> [!NOTE]
> Проект сделан как исследование Backend + IR: как базовые структуры данных ведут себя в задаче autocomplete/search suggest.
> 
> Сравниваются базовые подходы автодополнения: linear scan, sorted array, generic trie/radix и top-k-aware trie.
> 
> Целей заменить Lucene/ElasticSearch/Manticore не стояло, я просто исследовал простые подходы в search suggestion.
> 
> Осознанно не реализован: ранкинг(score захардкожен), автоисправление, персонализация, перестройка индекса в онлайне, антифрод, синонимы, учитывание контекста.
> 
> Это не production код, фокус на автодополнении через простые структуры данных и их сравнение.
>
> Проект сделан с использованием LLM. Выводы по производительности основаны на raw benchmark output в `reports/`.

## API

```text
GET /api/suggest?text=go&engine=ranked-trie&k=8
GET /api/engines
```

В API оставлены только выбранные engines:

| Engine | Роль |
| --- | --- |
| `linear` | correctness baseline: полный перебор |
| `sorted` | сильный простой baseline: binary search по prefix range + top-k scan |
| `radix` | mutable radix baseline на `github.com/armon/go-radix` |
| `hashicorp-radix` | immutable radix baseline на `github.com/hashicorp/go-immutable-radix/v2` |
| `ranked-trie` | custom top-k-aware trie: быстрый read-heavy query path за счёт cached top-k на prefix node |

## Запуск

```powershell
$env:GOCACHE = "$PWD\.gocache"
go run ./cmd/server
```

Открой:

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
| `hashicorp-radix` | immutable tree; естественно подходит для snapshot-style reads |
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
