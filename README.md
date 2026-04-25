# Search Suggest Lab

Мини pet project про in-memory autocomplete на Go.

Главный тезис: search suggest - это не просто prefix lookup. Обычный trie/radix отвечает на вопрос "какие строки имеют prefix?", а suggest отвечает на вопрос "какие top-k строк с этим prefix нужно вернуть прямо сейчас?".

## API

```text
GET /api/suggest?text=go&engine=ranked-trie&k=8
GET /api/engines
```

В API оставлены только выбранные engines:

| Engine | Роль |
| --- | --- |
| `linear` | correctness/performance baseline |
| `sorted` | сильный простой baseline |
| `radix` | mutable radix baseline на `github.com/armon/go-radix` |
| `hashicorp-radix` | immutable radix baseline на `github.com/hashicorp/go-immutable-radix/v2` |
| `ranked-trie` | самый быстрый read-heavy suggest engine в benchmark-е |

`zyedidia-trie`, `dghubble-rune-trie` и `adaptive-radix` остаются в исследовательских benchmark-ах, но не показываются как основной suggest API path.

## Dependency Policy

Low-adoption trie/autocomplete зависимости удалены из проекта. В частности, `github.com/st1064870/generic` заменен на upstream `github.com/zyedidia/generic`, а `olympos.io/container/pruning-radix-trie` удален из `go.mod`.

В benchmark-ах оставлены библиотеки с понятным adoption-сигналом:

| Package | Почему оставлен |
| --- | --- |
| `github.com/zyedidia/generic` | upstream generic data structures, trie/TST implementation |
| `github.com/dghubble/trie` | популярный trie для быстрых `Get`; benchmark показывает, почему его API плохо ложится на suggest |
| `github.com/armon/go-radix` | популярный mutable radix baseline |
| `github.com/hashicorp/go-immutable-radix/v2` | популярный immutable radix baseline |
| `github.com/plar/go-adaptive-radix-tree/v2` | ART baseline с prefix iteration |

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

Индексы строятся один раз при старте приложения, после этого HTTP handlers только вызывают `Suggest`. При таком инварианте выбранные engines безопасны для конкурентных read-only запросов:

| Component | Thread-safety модель |
| --- | --- |
| `Registry` | map заполняется при старте и дальше только читается |
| `linear` / `sorted` | immutable slices после build |
| `radix` (`go-radix`) | используется только для чтения после build; конкурентные mutation не поддерживаются |
| `hashicorp-radix` | immutable tree, хорошо ложится на concurrent reads |
| `ranked-trie` | custom trie не меняется после build; maps/slices только читаются |

В проекте нет online updates и in-place rebuild. Если добавлять live reload индекса, новый индекс нужно строить отдельно и публиковать целиком через `atomic.Value`, `sync.RWMutex` или другой явный swap-механизм. Мутировать существующий trie/radix во время `Suggest` нельзя.

## Benchmark-и

Smoke test:

```powershell
$env:GOCACHE = "$PWD\.gocache"
go test "./internal/suggest" -run "^$" -bench "BenchmarkSelectedSuggestEngines" -benchmem -benchtime=1s
```

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

Один прогон benchmark-а - это smoke test. Для уверенных выводов нужен `-count=10` и `benchstat`.

Полный отчёт на русском: [reports/engine-benchmarks.md](reports/engine-benchmarks.md).

Сырой вывод последнего smoke-прогона: [reports/benchmarks-2026-04-25.txt](reports/benchmarks-2026-04-25.txt).
