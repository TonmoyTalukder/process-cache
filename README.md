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
- Lazy expiration on `Get` and `Exists`.
- Background expiration cleanup.
- Idempotent `Close`.
- Stats snapshots with copied type-size maps.
- Configurable `Sizer` and `Clock`.
- Zero runtime external dependencies.

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
cfg.NoCleanup = true

cache, err := processcache.NewMemoryCacheFromConfig(cfg)
```

Use `GetAs` for typed reads:

```go
value, ok := processcache.GetAs[string](cache, "key")
```

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

## Size Accounting

ProcessCache size accounting is approximate. The default sizer counts key length, common scalar sizes, strings, byte slices, and a conservative fixed overhead for unknown values. Go heap metadata, interface boxing, map growth, and GC behavior mean process memory can exceed the cache's internal byte count.

The built-in estimator is most accurate for strings, byte slices, booleans, numeric scalars, `time.Time`, and `time.Duration`. If you cache richer structs, slices, or maps and want tighter limits, provide `WithSizer`.

## Stats

```go
stats := cache.Stats()
fmt.Println(stats.Hits, stats.Misses, stats.CurrentSize)
```

`Stats.TypeSizes` is copied before return, so callers cannot mutate internal cache state.

Configured prefixes remain present in `Stats.TypeSizes` even when their current size is zero.

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

- [CHANGELOG](CHANGELOG.md)
- [CONTRIBUTING](CONTRIBUTING.md)
- [SECURITY](SECURITY.md)
- [LICENSE](LICENSE)
