# ProcessCache

ProcessCache is a bounded, thread-safe, in-process LRU cache for Go applications that need fast local caching without Redis, Memcached, a database, an HTTP server, or any runtime sidecar.

```go
import processcache "github.com/tonmoytalukder/process-cache"
```

## Install

Requires Go 1.24 or newer.

```sh
go get github.com/tonmoytalukder/process-cache
```

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

	cache.Set("username:tonmoy", true, 5*time.Minute)

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

## Size Accounting

ProcessCache size accounting is approximate. The default sizer counts key length, common scalar sizes, strings, byte slices, and a conservative fixed overhead for unknown values. Go heap metadata, interface boxing, map growth, and GC behavior mean process memory can exceed the cache's internal byte count.

## Stats

```go
stats := cache.Stats()
fmt.Println(stats.Hits, stats.Misses, stats.CurrentSize)
```

`Stats.TypeSizes` is copied before return, so callers cannot mutate internal cache state.

## Optional Service Integration

Application services can depend on a small local interface and treat `nil` as "cache disabled":

```go
type KeyValueCache interface {
	Get(key string) (any, bool)
	Set(key string, value any, ttl ...time.Duration) bool
	Delete(key string) bool
}

type UserService struct {
	cache KeyValueCache
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
