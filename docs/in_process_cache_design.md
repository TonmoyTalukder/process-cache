# ProcessCache Design

## 1. Overview

**ProcessCache** is a distributable Go module for bounded, thread-safe, in-process LRU caching. It is designed for applications that need sub-millisecond cache access without adding Redis, Memcached, or another runtime service.

The module exposes one clean import path:

```go
import processcache "github.com/tonmoytalukder/process-cache"
```

The repository keeps one root Go file, `processcache.go`, as the public facade. All implementation files live under `internal/processcache`, so the root stays clean while the package remains importable from the module root.

## 2. Goal

Build **ProcessCache** as a distributable Go package. The repository and module path use the Go-friendly kebab-case name `process-cache`, while the public package identifier is `processcache`.

It is installable in any Go project with:

```sh
go get github.com/tonmoytalukder/process-cache
```

The package provides a bounded, thread-safe, in-process LRU cache with:

- Zero runtime infrastructure dependencies.
- Global memory cap.
- Optional key-prefix type quotas.
- LRU eviction.
- Lazy expiration on reads.
- Background expiration sweep.
- Clean shutdown.
- Stats snapshots.
- Nil-safe service integration.
- Public API stable enough to version with semantic tags.

## 3. Repository Layout

Target layout:

```text
process-cache/
├── processcache.go              # the only root .go file; public facade package
│
├── cmd/
│   ├── example/
│   │   └── main.go              # runnable usage demo
│   └── bench/
│       └── main.go              # optional manual benchmark runner
│
├── internal/
│   ├── processcache/
│   │   ├── cache.go             # internal Cache-compatible implementation types
│   │   ├── memory.go            # MemoryCache implementation
│   │   ├── options.go           # option application and validation
│   │   ├── config.go            # Config and TypeLimit implementation model
│   │   ├── item.go              # item model with global/type list elements
│   │   ├── eviction.go          # global and type-scoped eviction
│   │   ├── expiration.go        # lazy expiry and sweeper
│   │   ├── stats.go             # stats counters and snapshots
│   │   ├── sizer.go             # default sizing
│   │   ├── clock.go             # real clock
│   │   └── errors.go            # internal validation helpers
│   │
│   ├── config/
│   │   └── config.go            # optional config loader for examples/CLI only
│   └── pkg/
│       └── testclock/
│           └── clock.go         # private fake clock for tests
│
├── test/
│   ├── unit/
│   │   └── processcache/        # public behavior tests
│   ├── integration/
│   │   └── processcache/        # installed-module style tests
│   └── e2e/
│       └── consumer/            # tiny external Go module using go get/replace
│
├── docs/
│   └── in_process_cache_design.md
│
├── scripts/
│   ├── test.sh
│   ├── race.sh
│   ├── bench.sh
│   └── docker-test.sh
│
├── .github/
│   └── workflows/
│       └── ci.yml
│
├── Dockerfile                   # reproducible test/bench container
├── docker-compose.yml           # local Docker test runner
├── .dockerignore
├── .gitignore
├── LICENSE
├── README.md
├── go.mod
└── go.sum                       # generated only if dependencies are added
```

Root package rule:

- The package must install as `go get github.com/tonmoytalukder/process-cache`.
- Consumers import `github.com/tonmoytalukder/process-cache`, not a nested package.
- Go requires at least one root `.go` file for the module root to be importable as a package.
- The single root file, `processcache.go`, contains only public API declarations, type aliases, constants, constructors, and package docs.
- Implementation details stay under `internal/processcache`.
- Private support code stays under `internal`.
- Tests use a top-level `test` folder for unit, integration, and consumer verification.
- Runnable demos live under `cmd`.

## 4. Module And Import Path

Module:

```text
github.com/tonmoytalukder/process-cache
```

Primary import:

```go
import processcache "github.com/tonmoytalukder/process-cache"
```

This keeps both installation and import at the module root while allowing the implementation to remain organized under `internal/`.

## 5. Public API

### Interface

```go
package processcache

import "time"

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

Design notes:

- `Set` uses a variadic TTL for ergonomic optional expiration, and returns `bool` so callers know whether the item was accepted.
- `Delete` returns `bool` to say whether it removed anything.
- `Exists` is included for callers that only need presence checks.
- `Close` is added to stop the cleanup goroutine.
- `ttl` omitted or `ttl <= 0` means no expiration.
- A nil `Cache` is treated by services as "cache disabled".

### Constructor

```go
func NewMemoryCache(opts ...Option) (*MemoryCache, error)
```

Example:

```go
c, err := processcache.NewMemoryCache(
    processcache.WithMaxSize(100*processcache.MB),
    processcache.WithCleanupInterval(5*time.Minute),
    processcache.WithTypeLimit("username:", 1*processcache.MB),
    processcache.WithTypeLimit("session:", 50*processcache.MB),
)
if err != nil {
    return err
}
defer c.Close()
```

The constructor name is explicit so projects can add other cache implementations later without ambiguity.

### Generic Helper

```go
func GetAs[T any](c Cache, key string) (T, bool)
```

Rules:

- Nil cache returns zero value and `false`.
- Cache miss returns zero value and `false`.
- Type mismatch returns zero value and `false`.
- Successful type assertion returns the value and `true`.

## 6. Configuration

Public config:

```go
type Config struct {
    MaxSize         int64
    CleanupInterval time.Duration
    TypeLimits      []TypeLimit
    Sizer           Sizer
    Clock           Clock
    Metrics         bool
}

type TypeLimit struct {
    Prefix  string
    MaxSize int64
    Enabled bool
}
```

Options:

```go
type Option func(*Config) error

func WithMaxSize(bytes int64) Option
func WithCleanupInterval(interval time.Duration) Option
func WithCleanupDisabled() Option
func WithTypeLimit(prefix string, bytes int64) Option
func WithTypeLimits(limits ...TypeLimit) Option
func WithSizer(sizer Sizer) Option
func WithClock(clock Clock) Option
func WithMetrics(enabled bool) Option
```

Defaults:

```text
MaxSize:          100 MB
CleanupInterval: 5 minutes
Sizer:            DefaultSizer
Clock:            real time clock
Metrics:          true
TypeLimits:       none by default
```

Important package boundary decision:

The reusable package does not hard-code `username:` or any other app-specific prefix as a default type limit. Applications configure their own prefixes explicitly.

Validation rules:

- `MaxSize` must be greater than zero.
- `CleanupInterval` must be greater than zero unless cleanup is disabled.
- Type prefixes must be non-empty.
- Enabled type limits must be greater than zero.
- Duplicate prefixes return an error.
- Overlapping prefixes return an error by default, because `user:` and `user:profile:` create ambiguous accounting.

## 7. Internal Model

The public root package exposes the API through `processcache.go`. The implementation lives under `internal/processcache`, using unexported structs where possible.

```go
type MemoryCache struct {
    mu sync.RWMutex

    items     map[string]*item
    globalLRU *list.List

    maxSize     int64
    currentSize int64

    typeSizes  map[string]int64
    typeLimits map[string]TypeLimit
    typeLRUs   map[string]*list.List
    prefixes   []string

    sizer Sizer
    clock Clock

    cleanupInterval time.Duration
    cleanupDisabled bool
    stopCleanup     chan struct{}
    cleanupDone     chan struct{}
    closeOnce       sync.Once

    stats statsCounter
}

type item struct {
    key        string
    value      any
    size       int64
    prefix     string
    expiration time.Time
    hasExpiry  bool
    globalElem *list.Element
    typeElem   *list.Element
}
```

List conventions:

- `globalLRU` front is most recently used globally.
- `globalLRU` back is least recently used globally.
- `typeLRUs[prefix]` front is most recently used within that prefix.
- `typeLRUs[prefix]` back is least recently used within that prefix.

Required invariants:

- Every map entry has one global LRU element.
- Every configured-prefix item has one type LRU element.
- Every LRU element points to an item still in the map.
- `currentSize` equals the sum of item sizes.
- `typeSizes[prefix]` equals the sum of item sizes for that prefix.
- Removing an item updates all size counters exactly once.
- Removing an item removes it from both its global list and type list, when present.

The last invariant directly prevents double-subtract bugs during item replacement.

## 8. Prefix Matching And Type Quotas

Prefix matching is deterministic:

```go
func (c *MemoryCache) prefixFor(key string) string
```

Rules:

- No configured prefix means the item belongs only to the global pool.
- Exactly one matching prefix means the item is accounted against that type.
- Overlapping prefixes are rejected during construction.
- Disabled type limits are ignored.

Type quota behavior:

- If an item has a configured prefix, evict from that prefix first.
- If the item still cannot fit globally, evict global LRU items.
- If the item is larger than its type limit, reject it.
- If the item is larger than global limit, reject it.
- Type-scoped eviction uses the prefix's own doubly-linked list, so it does not scan unrelated cache entries.

This keeps the case study behavior while making rejection explicit.

## 9. Size Accounting

Public interface:

```go
type Sizer interface {
    SizeOf(key string, value any) int64
}
```

Default behavior:

- Count key length.
- Count simple scalar values with a fast type switch.
- Count strings and byte slices by length.
- Use a conservative fixed overhead for unknown values.
- Avoid reflection on the hot `Set` path.

Constants:

```go
const (
    KB int64 = 1024
    MB int64 = 1024 * KB
)
```

Documentation must say that cache accounting is approximate. Go heap metadata, interface boxing, map growth, and GC behavior mean process memory can exceed the cache's internal byte count.

## 10. Algorithms Used

ProcessCache uses a small set of predictable algorithms and data structures:

| Area | Algorithm / Data Structure | Target Cost | Why |
| --- | --- | --- | --- |
| Key lookup | Hash table: `map[string]*item` | O(1) average | Direct key-to-entry lookup without scanning. |
| Global recency | Doubly-linked list | O(1) move, insert, delete, tail eviction | Supports LRU by moving hits to the front and evicting from the back. |
| Global LRU cache | Hash map + doubly-linked list | O(1) average `Get`, `Set`, `Delete`, global eviction | This is the standard LRU design referenced in the case study. |
| Type-scoped LRU | One doubly-linked list per configured prefix | O(1) type eviction | Avoids scanning the global list to find the oldest item for a prefix. |
| Type quota lookup | Prefix map plus validated prefix list | O(p) prefix match, where p is number of configured prefixes | Prefix count is expected to be small; overlapping prefixes are rejected. |
| TTL expiration | Timestamp comparison on read | O(1) per read | Expired items are never returned. |
| Background cleanup | Periodic full scan | O(n) per sweep | Removes expired items that are never read again. |
| Size accounting | Fast type switch | O(1) for common values | Avoids reflection on the hot write path. |
| Stats | Atomic counters plus locked snapshots | O(1) counter updates | Keeps metrics cheap without exposing internal maps. |

### Why Use A Doubly-Linked List?

The cache uses a doubly-linked list. That is what makes the LRU path O(1):

- On `Get`, the item is found in the hash map and its list node is moved to the front in O(1).
- On `Set`, the new item is inserted at the front in O(1).
- On `Delete`, the item node is removed from the list in O(1).
- On global eviction, the tail node is removed in O(1).

The subtle part is type-scoped eviction. A single global doubly-linked list makes global LRU O(1), but it does not make "evict the oldest `username:` item" O(1). If type eviction only scans the global list from the back until it finds a matching prefix, that path is O(n).

ProcessCache maintains both:

- one global doubly-linked list for global LRU order.
- one doubly-linked list per configured prefix for type-local LRU order.

Each item stores two list elements: one for the global list and one for its type list. That makes both global eviction and type-scoped eviction O(1), at the cost of slightly more bookkeeping.

## 11. Read Path

Target flow:

```text
Get(key)
  RLock
  item, ok := items[key]
  RUnlock

  if !ok:
      record miss
      return nil, false

  if expired:
      Lock
      if items[key] is still this item:
          remove item
      Unlock
      record expiration + miss
      return nil, false

  Lock
  if items[key] is still this item:
      move global LRU element to front
      if item has a configured prefix:
          move type LRU element to front
      value = item.value
  Unlock

  record hit
  return value, true
```

Why not hold a write lock for the whole `Get`:

The package reduces read contention while still using a write lock for LRU mutation.

## 12. Write Path

Target flow:

```text
Set(key, value, ttl...)
  reject empty key
  estimate item size
  calculate expiration

  Lock
  if existing item:
      remove it once

  reject if item size > global max
  find prefix
  reject if item size > type max

  while type quota exceeded:
      evictTypeLRU(prefix) in O(1) from that prefix's tail

  while global quota exceeded:
      evictLRU() in O(1) from global tail

  insert new item at global LRU front
  if item has a configured prefix:
      insert into that prefix's type LRU front
  update global and type sizes
  Unlock

  record set
  return true
```

The implementation uses one private removal function:

```go
func (c *MemoryCache) removeItemLocked(it *item)
```

That helper must be the only place that mutates map/list/size state for removal.

## 13. Eviction

### Global LRU

```text
evictLRULocked()
  tail := globalLRU.Back()
  if tail == nil: return false
  remove tail item
  return true
```

Cost: O(1).

### Type-Scoped LRU

```text
evictTypeLRULocked(prefix)
  typeList := typeLRUs[prefix]
  if typeList == nil: return false
  tail := typeList.Back()
  if tail == nil: return false
  remove tail item
  return true if removed
```

Cost: O(1).

This is why ProcessCache uses doubly-linked lists for both global and type-local LRU order. A single global list is enough for O(1) global eviction, but a per-type list is needed for O(1) type quota eviction.

Bookkeeping cost:

- Each item stores one global list element.
- Items with configured prefixes also store one type-list element.
- `Get`, `Set`, `Delete`, and eviction must keep both lists in sync.

## 14. Expiration And Shutdown

Lazy expiration:

- Every `Get` and `Exists` checks expiration.
- Expired items are removed before returning.
- Expired items are never returned.

Background sweep:

```text
startCleanup()
  ticker every CleanupInterval
  select:
    ticker.C -> cleanupExpiredLocked
    stopCleanup -> exit
```

Shutdown contract:

- `Close` is idempotent.
- `Close` stops the sweeper.
- `Close` waits for the goroutine to exit.
- `Close` is safe even when cleanup is disabled.

This is a mandatory package behavior so applications can shut down cleanly.

## 15. Stats

Public snapshot:

```go
type Stats struct {
    Hits        uint64
    Misses      uint64
    Sets        uint64
    Deletes     uint64
    Evictions   uint64
    Expirations uint64
    Rejections  uint64

    Len         int
    CurrentSize int64
    MaxSize     int64
    TypeSizes   map[string]int64
}
```

Rules:

- Counter fields can use atomics.
- Size and length fields are copied under lock.
- `TypeSizes` must be copied before returning.
- Stats do not expose mutable internal maps.

## 16. Errors

Public sentinel errors:

```go
var (
    ErrInvalidMaxSize         = errors.New("cache: max size must be greater than zero")
    ErrInvalidCleanupInterval = errors.New("cache: cleanup interval must be greater than zero")
    ErrInvalidTypePrefix      = errors.New("cache: type prefix must not be empty")
    ErrInvalidTypeLimit       = errors.New("cache: type limit must be greater than zero")
    ErrDuplicateTypePrefix    = errors.New("cache: duplicate type prefix")
    ErrOverlappingTypePrefix  = errors.New("cache: overlapping type prefixes")
    ErrNilSizer               = errors.New("cache: sizer must not be nil")
    ErrNilClock               = errors.New("cache: clock must not be nil")
)
```

Use wrapping for context:

```go
return fmt.Errorf("%w: %q", ErrDuplicateTypePrefix, prefix)
```

## 17. Installing And Using In Other Projects

Install the package with:

```sh
go get github.com/tonmoytalukder/process-cache
```

Then import it where the application composes dependencies:

```go
import (
    "time"

    processcache "github.com/tonmoytalukder/process-cache"
)

func NewAppCache() (processcache.Cache, error) {
    c, err := processcache.NewMemoryCache(
        processcache.WithMaxSize(100*processcache.MB),
        processcache.WithCleanupInterval(5*time.Minute),
        processcache.WithTypeLimit("username:", 1*processcache.MB),
        processcache.WithTypeLimit("session:", 50*processcache.MB),
    )
    if err != nil {
        return nil, err
    }
    return c, nil
}
```

Call `Close` from the application's graceful shutdown path:

```go
c, err := NewAppCache()
if err != nil {
    return err
}
defer c.Close()
```

### Depend On An Interface Type

Application services depend on a small interface instead of the concrete `*processcache.MemoryCache`. This lets Go infer compatibility structurally: any type with these methods can be injected.

```go
type KeyValueCache interface {
    Get(key string) (any, bool)
    Set(key string, value any, ttl ...time.Duration) bool
    Delete(key string) bool
}
```

Service constructor:

```go
type UserService struct {
    repo  UserRepository
    cache KeyValueCache
}

func NewUserService(repo UserRepository, c KeyValueCache) *UserService {
    return &UserService{
        repo:  repo,
        cache: c,
    }
}
```

Composition root:

```go
appCache, err := processcache.NewMemoryCache(
    processcache.WithMaxSize(100*processcache.MB),
    processcache.WithTypeLimit("username:", 1*processcache.MB),
)
if err != nil {
    return err
}
defer appCache.Close()

userService := NewUserService(userRepo, appCache)
```

This works because `*processcache.MemoryCache` implements the local `KeyValueCache` interface without explicit declarations.

### Type-Safe Reads

For typed cache values, callers can use the generic helper:

```go
exists, ok := processcache.GetAs[bool](appCache, "username:tonmoy")
if ok && exists {
    // cached username availability result
}
```

For projects that prefer local interfaces, keep the injected field as the small interface and use normal type assertions:

```go
value, ok := s.cache.Get("username:" + username)
if ok {
    exists, typed := value.(bool)
    if typed {
        return exists, nil
    }
}
```

### Optional Cache

Services can treat a nil cache as disabled:

```go
if s.cache != nil {
    s.cache.Set(key, value, 5*time.Minute)
}
```

This keeps tests simple because a service can be created with `nil` when the cache is irrelevant to the behavior being tested.

## 18. Usage Patterns

### Username Availability

```go
type UserService struct {
    repo  UserRepository
    cache processcache.Cache
}

func (s *UserService) IsUsernameTaken(ctx context.Context, username string) (bool, error) {
    key := "username:" + username

    if value, ok := processcache.GetAs[bool](s.cache, key); ok {
        return value, nil
    }

    exists, err := s.repo.UsernameExists(ctx, username)
    if err != nil {
        return false, err
    }

    if s.cache != nil {
        s.cache.Set(key, exists, 5*time.Minute)
    }

    return exists, nil
}
```

### Session Invalidation

```go
func (s *SessionService) Logout(ctx context.Context, sessionID string) error {
    if err := s.repo.InvalidateSession(ctx, sessionID); err != nil {
        return err
    }

    if s.cache != nil {
        s.cache.Delete("session:" + sessionID)
    }

    return nil
}
```

## 19. Testing Plan

Use a top-level test structure that separates package behavior, integration-style usage, and consumer verification.

### Unit Tests: `test/unit/processcache`

Core behavior:

- constructor applies defaults.
- invalid options return sentinel errors.
- `Set` and `Get` work for live values.
- `Get` returns miss for unknown keys.
- `Delete` removes values and reports existence.
- `Exists` respects expiration.
- `Clear` resets map, list, size counters, and length.

LRU behavior:

- reading a key moves it to the front.
- global eviction removes the least recently used key.
- updating an existing key refreshes LRU position and size accounting.
- updating an existing key does not double-subtract size.

Type quota behavior:

- configured prefix evicts only items with that prefix while possible.
- one prefix cannot starve another configured prefix.
- unconfigured prefixes participate in global LRU.
- item larger than type limit is rejected.
- item larger than global limit is rejected.

Expiration:

- expired values are not returned.
- lazy expiration removes expired values.
- background sweeper removes expired unread values.
- non-expiring values survive cleanup.

Stats:

- hits and misses increment correctly.
- sets, deletes, evictions, expirations, and rejections increment correctly.
- returned `TypeSizes` map cannot mutate internal state.

### Integration Tests: `test/integration/processcache`

Use the public package as a consumer would:

- import `github.com/tonmoytalukder/process-cache`.
- configure multiple type limits.
- run mixed `Get`, `Set`, `Delete`, and expiry flows.
- verify stats and cleanup shutdown.

### E2E Tests: `test/e2e/consumer`

Create a tiny throwaway Go module that imports this package through a local `replace` directive. This verifies the package can be consumed outside its own module.

### Race Tests

Required command:

```sh
go test -race ./...
```

Race scenarios:

- many goroutines call `Set`.
- many goroutines call `Get`.
- mixed `Get`, `Set`, `Delete`.
- `Close` while reads/writes are active.

### Benchmarks

Benchmarks:

- `Get` hit.
- `Get` miss.
- `Set` without eviction.
- `Set` with global eviction.
- `Set` with type-scoped eviction.
- mixed concurrent workload.

Sizes:

- 100 items.
- 1,000 items.
- 10,000 items.
- 100,000 items if practical.

## 20. Scripts

Mirror the repeatable workflow style used by larger backend projects:

```sh
./scripts/test.sh
./scripts/race.sh
./scripts/bench.sh
./scripts/docker-test.sh
```

Script behavior:

- `test.sh`: `go test ./...`
- `race.sh`: `go test -race ./...`
- `bench.sh`: `go test -bench=. -benchmem ./...`
- `docker-test.sh`: `docker compose run --rm test`

## 21. Docker

ProcessCache uses Docker for development and verification, not runtime. It is an in-process Go package, not a network service. Docker is useful for reproducible development, testing, and benchmarking across machines.

Docker responsibilities:

- provide a consistent Go toolchain for contributors.
- run unit, integration, race, and benchmark commands in a clean environment.
- verify the package without relying on a developer's local Go installation.
- keep runtime dependency expectations honest: the package does not require Redis, a database, or any sidecar service.

### Dockerfile

```dockerfile
FROM golang:1.24-alpine AS test

WORKDIR /src

RUN apk add --no-cache git ca-certificates

COPY go.mod ./
RUN go mod download

COPY . .

CMD ["go", "test", "./..."]
```

The Dockerfile is for test execution, not for publishing a long-running ProcessCache container.

### Docker Compose

```yaml
services:
  test:
    build:
      context: .
      target: test
    command: go test ./...

  race:
    build:
      context: .
      target: test
    command: go test -race ./...

  bench:
    build:
      context: .
      target: test
    command: go test -bench=. -benchmem ./...
```

Local commands:

```sh
docker compose run --rm test
docker compose run --rm race
docker compose run --rm bench
```

### `.dockerignore`

```text
.git
.github
.gocache
tmp
coverage.out
dist
```

## 22. Documentation

### README

Include:

- install command.
- minimal example.
- type-limit example.
- nil-safe service integration example.
- stats example.
- expiration behavior.
- eviction behavior.
- concurrency guarantee.
- shutdown requirement.
- when to use this package versus Redis.
- warning that this is per-process, not cross-instance shared state.

### `processcache.go`

The single root file includes package docs and the public facade:

- what the package does.
- default limits.
- TTL rules.
- memory accounting approximation.
- `Close` requirement.
- type aliases for public structs where useful.
- public constructors and helpers that delegate to `internal/processcache`.

## 23. CI And Release

GitHub Actions:

```yaml
name: ci

on:
  push:
  pull_request:

jobs:
  test:
    runs-on: ubuntu-latest
    strategy:
      matrix:
        go: ["1.22", "1.23", "1.24"]
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: ${{ matrix.go }}
      - run: go test ./...
      - run: go test -race ./...
      - run: go test -bench=. -benchmem ./...
      - run: docker build --target test -t process-cache:test .
```

Release process:

1. Run `./scripts/test.sh`.
2. Run `./scripts/race.sh`.
3. Run `./scripts/bench.sh`.
4. Run `./scripts/docker-test.sh`.
5. Confirm README examples compile.
6. Tag:

   ```sh
   git tag v0.1.0
   git push origin v0.1.0
   ```

7. Verify:

   ```sh
   go list -m github.com/tonmoytalukder/process-cache@v0.1.0
   ```

## 24. Implementation Roadmap

### Milestone 1: Project Skeleton

- Create `go.mod`.
- Create `processcache.go`, `cmd`, `internal/processcache`, `test`, `docs`, and `scripts`.
- Add README and license placeholders.
- Add Dockerfile, docker-compose.yml, and .dockerignore.
- Add CI skeleton.

Exit criteria:

- `go test ./...` passes.
- `go list ./...` shows the expected packages.

### Milestone 2: Public Cache API

- Add `Cache` interface.
- Add `Config`, `TypeLimit`, and options.
- Add sentinel errors.
- Add `GetAs`.
- Add package docs.

Exit criteria:

- constructor tests pass.
- public examples compile.

### Milestone 3: Memory Cache Core

- Implement map plus LRU list.
- Implement `Get`, `Set`, `Delete`, `Exists`, `Clear`, `Len`.
- Implement default sizer.
- Fix size accounting around updates and removals.

Exit criteria:

- core unit tests pass.
- LRU unit tests pass.

### Milestone 4: Type Quotas

- Add prefix validation.
- Add per-type size tracking.
- Add type-scoped LRU eviction.
- Add rejection for oversized items.

Exit criteria:

- type quota tests pass.
- one cache type cannot starve another configured type.

### Milestone 5: Expiration And Cleanup

- Add TTL support.
- Add lazy expiration.
- Add background sweeper.
- Add idempotent `Close`.
- Add fake clock support.

Exit criteria:

- expiration tests pass without long sleeps.
- cleanup goroutine shutdown is tested.

### Milestone 6: Stats And Hardening

- Add stats counters.
- Add stats snapshot.
- Add race tests.
- Add benchmarks.

Exit criteria:

- `go test ./...` passes.
- `go test -race ./...` passes.
- benchmark results are recorded.

### Milestone 7: Consumer Verification

- Add `test/e2e/consumer`.
- Verify import through local module replacement.
- Add docs showing how external projects install and inject the cache.

Exit criteria:

- consumer test proves external import shape works.
- docs show installation, construction, interface-based injection, and shutdown.

### Milestone 8: v0.1.0 Release

- Finalize README.
- Finalize license.
- Run all scripts.
- Run Docker test flow.
- Tag `v0.1.0`.

Exit criteria:

- package can be installed with `go get github.com/tonmoytalukder/process-cache`.

## 25. Non-Goals For v0.1.0

- Redis protocol compatibility.
- Cross-process cache coherence.
- Persistent storage.
- Exact Go heap accounting.
- Sharded cache implementation.
- Prometheus exporter.
- HTTP server.
- Ent, database, or transport layers.

ProcessCache remains a library. It does not include app-specific layers, transport handlers, database access, or a long-running Dockerized service.
