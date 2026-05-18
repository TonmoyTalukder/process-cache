# ProcessCache

[![CI](https://github.com/tonmoytalukder/process-cache/actions/workflows/ci.yml/badge.svg)](https://github.com/tonmoytalukder/process-cache/actions/workflows/ci.yml)
[![pkg.go.dev](https://pkg.go.dev/badge/github.com/tonmoytalukder/process-cache.svg)](https://pkg.go.dev/github.com/tonmoytalukder/process-cache)
[![Go Report Card](https://goreportcard.com/badge/github.com/tonmoytalukder/process-cache)](https://goreportcard.com/report/github.com/tonmoytalukder/process-cache)

ProcessCache is a bounded, thread-safe, in-process LRU cache for Go applications that need fast local caching without Redis, Memcached, a database, an HTTP server, or any runtime sidecar.

```go
import processcache "github.com/tonmoytalukder/process-cache"
```

## Install

Requires Go 1.24 or newer.

```sh
go get github.com/tonmoytalukder/process-cache
```

## Stability

ProcessCache follows semantic versioning.

- `v0.x` means the API is usable but may still evolve based on early adoption feedback.
- `v1.0.0` will mark the start of stable compatibility expectations for the public API.

## Quick Start

```go
package main

import (
	"fmt"
	"time"

	processcache "github.com/tonmoytalukder/process-cache"
)

func main() {
	cache, err := processcache.NewMemoryCache(
		processcache.WithMaxSize(100*processcache.MB),
		processcache.WithCleanupInterval(5*time.Minute),
		processcache.WithTypeLimit("username:", 1*processcache.MB),
		processcache.WithTypeLimit("session:", 50*processcache.MB),
	)
	if err != nil {
		panic(err)
	}
	defer cache.Close()

	if !cache.Set("username:tonmoy", true, 5*time.Minute) {
		panic("cache rejected username entry")
	}

	exists, ok := processcache.GetAs[bool](cache, "username:tonmoy")
	fmt.Println(exists, ok)
}
```

## Features

- Bounded by an approximate global memory cap.
- Optional per-prefix quotas such as `username:` or `session:`.
- O(1) average key lookup with `map[string]*item`.
- O(1) global LRU eviction using one doubly-linked list.
- O(1) type-scoped LRU eviction using one doubly-linked list per configured prefix.
- Exact LRU promotion on successful reads.
- Lazy expiration on `Get` and `Exists`.
- Background expiration cleanup.
- Idempotent `Close`.
- Stats snapshots with copied type-size maps.
- Configurable `Sizer` and `Clock`.
- Zero runtime external dependencies.

## When To Use ProcessCache

Use ProcessCache when your Go service needs a small, fast, process-local cache and adding external infrastructure would create more operational cost than value. It is a good fit for data that is safe to recompute, reload, or fetch again after an eviction, expiration, deploy, or process restart.

ProcessCache is especially useful when you want:

- Lower latency for hot reads inside a single Go process.
- Fewer repeated database, API, or filesystem calls for short-lived data.
- A bounded memory footprint instead of an unbounded map.
- Deterministic LRU eviction under a global size budget.
- Per-prefix quotas so one key family cannot consume the whole cache.
- A zero-dependency cache that works in CLIs, workers, batch jobs, and HTTP services.
- A cache abstraction that can be injected into services and disabled with `nil`.

## Use Cases

- **Username, slug, or email availability checks:** Cache short-lived lookup results such as `username:tonmoy` to reduce repeated validation queries during signup or profile editing flows.
- **Session and token metadata:** Keep frequently accessed token introspection, session flags, or permission snapshots local to the process with a TTL.
- **Reference data:** Cache rarely changing data such as feature flags, plan limits, country lists, category trees, or configuration records after loading them from a database or API.
- **Expensive computation results:** Store deterministic function outputs, rendered fragments, parsed payloads, or validation results that are safe to recompute.
- **External API response shielding:** Reduce repeated calls to rate-limited or latency-sensitive services when responses can be reused briefly.
- **Background workers and queue consumers:** Reuse small lookup results across many jobs in the same worker process without running a shared cache service.
- **Development and internal tools:** Add bounded caching to scripts, CLIs, admin tools, and prototypes without introducing Redis or Memcached.

For the design story and tradeoffs behind the package, read the [ProcessCache case study](https://www.tonmoytalukder.com/case-study/process-cache).

## When Not To Use It

ProcessCache is intentionally local to one process. Use Redis, Memcached, or another distributed cache when you need shared cache state across multiple processes, persistence, cross-service invalidation, centralized memory management, atomic distributed operations, or cache data that must survive restarts.

## API

```go
type Cache interface {
	Get(key string) (any, bool)
	Set(key string, value any, ttl ...time.Duration) bool
	Delete(key string) bool
	Exists(key string) bool
	Clear()
	Len() int
	Stats() Stats
	Close() error
}
```

Create a cache with:

```go
cache, err := processcache.NewMemoryCache()
```

Or start from the exported config defaults:

```go
cfg := processcache.DefaultConfig()
cfg.MaxSize = 32 * processcache.MB
cfg.CleanupDisabled = true

cache, err := processcache.NewMemoryCacheFromConfig(cfg)
```

Use `GetAs` for typed reads:

```go
value, ok := processcache.GetAs[string](cache, "key")
```

Cached `nil` values are visible through `Get`, but `GetAs` treats them like a typed miss.

## Options

- `WithMaxSize(bytes int64)`
- `WithCleanupInterval(interval time.Duration)`
- `WithCleanupDisabled()`
- `WithTypeLimit(prefix string, bytes int64)`
- `WithTypeLimits(limits ...TypeLimit)`
- `WithSizer(sizer Sizer)`
- `WithClock(clock Clock)`
- `WithMetrics(enabled bool)`

Defaults:

- Max size: `100 * processcache.MB`
- Cleanup interval: `5 * time.Minute`
- Metrics: enabled
- Type limits: none

`Set` accepts at most one meaningful TTL value. If the TTL is omitted or `<= 0`, the entry does not expire. If more than one TTL is passed, only the first value is used.

The cache preserves exact LRU ordering, so operations synchronize through one internal mutex rather than a read-optimized approximate policy.

## Size Accounting

ProcessCache size accounting is approximate. The default sizer counts key length, common scalar sizes, strings, byte slices, and a conservative fixed overhead for unknown values. Go heap metadata, interface boxing, map growth, and GC behavior mean process memory can exceed the cache's internal byte count.

The built-in estimator is most accurate for strings, byte slices, booleans, numeric scalars, `time.Time`, and `time.Duration`. If you cache richer structs, slices, or maps and want tighter limits, provide `WithSizer`.

## Stats

```go
stats := cache.Stats()
fmt.Println(stats.Hits, stats.Misses, stats.CurrentSize)
```

`Stats.TypeSizes` and `Stats.TypeLimits` are copied before return, so callers cannot mutate internal cache state.

Configured prefixes remain present in `Stats.TypeSizes` even when their current size is zero.

Explicit `Delete` calls increment `Stats.Deletes`, including when the removed item is already expired. Overwrites increment `Stats.Sets` but not `Stats.Deletes` or `Stats.Evictions`.

## Optional Service Integration

Application services can depend directly on `processcache.Cache` and treat `nil` as "cache disabled":

```go
type UserService struct {
	cache processcache.Cache
}

func (s *UserService) IsUsernameTaken(username string) (bool, error) {
	key := "username:" + username
	if s.cache != nil {
		if value, ok := processcache.GetAs[bool](s.cache, key); ok {
			return value, nil
		}
	}

	// Query your source of truth here.
	exists := false

	if s.cache != nil {
		s.cache.Set(key, exists, 5*time.Minute)
	}
	return exists, nil
}
```

## Shutdown

Always close the cache when your application shuts down:

```go
defer cache.Close()
```

`Close` is idempotent and waits for the background sweeper to exit.

After `Close`, cache operations still work, but only lazy expiration runs; background cleanup is stopped.

All cache methods are safe for concurrent use.

## Redis Comparison

ProcessCache is not a distributed cache. It is intentionally local to one Go process. Use Redis or Memcached when you need shared cache state across processes, persistence, cross-service invalidation, or centralized memory management. Use ProcessCache when you need a zero-infrastructure in-process cache with predictable local LRU behavior.

## Development

```sh
./scripts/test.sh
./scripts/race.sh
./scripts/bench.sh
```

Docker:

```sh
./scripts/docker-test.sh
docker compose run --rm race
docker compose run --rm bench
```

## Project Docs

- [Case Study](https://www.tonmoytalukder.com/case-study/process-cache)
- [Algorithm Explained](docs/algorithm.md)
- [CHANGELOG](CHANGELOG.md)
- [CONTRIBUTING](CONTRIBUTING.md)
- [SECURITY](SECURITY.md)
- [LICENSE](LICENSE)
