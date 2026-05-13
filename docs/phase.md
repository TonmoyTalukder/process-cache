# ProcessCache Implementation Phases

This file splits the ProcessCache build into small, verifiable phases. Follow the phases in order. Do not skip verification commands unless the tool is unavailable.

## Phase 0: Read The Spec

Goal: understand the target before writing code.

Tasks:

- Read `docs/in_process_cache_design.md`.
- Read `AGENTS.md`.
- Confirm the public import path is `github.com/tonmoytalukder/process-cache`.
- Confirm the only root Go file is `processcache.go`.
- Confirm implementation files belong under `internal/processcache`.

Exit checks:

- No code changes yet.
- You can explain the package API, repository layout, algorithms, and test plan.

## Phase 1: Project Skeleton

Goal: create the repository structure and module metadata.

Tasks:

- Create `go.mod` with module path `github.com/tonmoytalukder/process-cache`.
- Set `go.mod` to Go `1.24` or newer.
- Create `processcache.go` as the only root `.go` file.
- Create directories:
  - `internal/processcache`
  - `internal/config`
  - `internal/pkg/testclock`
  - `cmd/example`
  - `cmd/bench`
  - `test/unit/processcache`
  - `test/integration/processcache`
  - `test/e2e/consumer`
  - `scripts`
  - `.github/workflows`
- Add `.gitignore`, `.dockerignore`, `Dockerfile`, `docker-compose.yml`, `README.md`, and `LICENSE`.
- Keep `AGENTS.md` and `CLAUDE.md` at the repository root.
- Keep this phase guide at `docs/phase.md`.

Exit checks:

```sh
go list ./...
go test ./...
```

## Phase 2: Public API Facade

Goal: define the public API in `processcache.go`.

Tasks:

- Define package `processcache`.
- Add package documentation.
- Expose `Cache`, `Config`, `TypeLimit`, `Stats`, `Option`, `Sizer`, and `Clock`.
- Expose constants `KB` and `MB`.
- Expose sentinel errors.
- Expose `NewMemoryCache(opts ...Option) (*MemoryCache, error)`.
- Expose options:
  - `WithMaxSize`
  - `WithCleanupInterval`
  - `WithCleanupDisabled`
  - `WithTypeLimit`
  - `WithTypeLimits`
  - `WithSizer`
  - `WithClock`
  - `WithMetrics`
- Expose `GetAs[T any](c Cache, key string) (T, bool)`.
- Delegate implementation to `internal/processcache`.

Exit checks:

```sh
gofmt -w processcache.go
go test ./...
```

## Phase 3: Config, Errors, Clock, And Sizer

Goal: implement constructor validation and supporting abstractions.

Tasks:

- Implement config defaults:
  - max size: `100 * MB`
  - cleanup interval: `5 * time.Minute`
  - metrics enabled
  - no default type limits
- Implement validation for:
  - invalid max size
  - invalid cleanup interval
  - nil sizer
  - nil clock
  - empty type prefix
  - invalid type limit
  - duplicate type prefix
  - overlapping type prefix
- Implement real clock.
- Implement fake clock in `internal/pkg/testclock`.
- Implement approximate default sizer using a fast type switch.

Exit checks:

```sh
go test ./...
```

## Phase 4: Core Memory Cache

Goal: implement basic cache behavior without expiration or stats polish.

Tasks:

- Implement `MemoryCache`.
- Use `map[string]*item` for key lookup.
- Use one global doubly-linked list for global LRU.
- Use one doubly-linked list per configured prefix for type-local LRU.
- Store global and type list elements on each item.
- Implement:
  - `Get`
  - `Set`
  - `Delete`
  - `Exists`
  - `Clear`
  - `Len`
  - `Close`
- Keep all list, map, and size accounting mutations under lock.
- Ensure item removal updates map, global list, type list, global size, and type size exactly once.

Exit checks:

```sh
go test ./...
```

## Phase 5: Eviction And Quotas

Goal: complete global and type-scoped eviction.

Tasks:

- Reject empty keys in `Set`.
- Reject items larger than global max size.
- Reject items larger than their configured type limit.
- Implement global LRU eviction from the global tail in O(1).
- Implement type LRU eviction from the prefix tail in O(1).
- Ensure configured prefixes do not starve each other.
- Ensure unconfigured keys participate only in global LRU.

Exit checks:

```sh
go test ./...
```

## Phase 6: Expiration And Shutdown

Goal: complete TTL behavior and cleanup lifecycle.

Tasks:

- Implement `ttl <= 0` as no expiration.
- Implement lazy expiration in `Get` and `Exists`.
- Implement background cleanup sweeper.
- Implement `WithCleanupDisabled`.
- Make `Close` idempotent.
- Ensure cleanup goroutine exits on `Close`.
- Ensure `Close` is safe when cleanup is disabled.

Exit checks:

```sh
go test ./...
go test -race ./...
```

## Phase 7: Stats

Goal: expose accurate snapshot metrics.

Tasks:

- Implement hit, miss, set, delete, eviction, expiration, and rejection counters.
- Implement `Stats()` snapshot with:
  - counters
  - `Len`
  - `CurrentSize`
  - `MaxSize`
  - copied `TypeSizes`
- Ensure returned maps cannot mutate internal state.

Exit checks:

```sh
go test ./...
```

## Phase 8: Tests And Benchmarks

Goal: make behavior safe and measurable.

Tasks:

- Add unit tests for all public API behavior.
- Add integration tests using the public import path.
- Add e2e consumer test with a local `replace`.
- Add race/concurrency tests.
- Add benchmarks:
  - get hit
  - get miss
  - set without eviction
  - global eviction
  - type eviction
  - mixed concurrent workload

Exit checks:

```sh
go test ./...
go test -race ./...
go test -bench=. -benchmem ./...
```

## Phase 9: Examples, README, Scripts, Docker, CI

Goal: make the project usable by other developers.

Tasks:

- Add `cmd/example/main.go`.
- Add optional `cmd/bench/main.go`.
- Write `README.md` with install, examples, config, stats, shutdown, Docker, and Redis comparison.
- Add scripts:
  - `scripts/test.sh`
  - `scripts/race.sh`
  - `scripts/bench.sh`
  - `scripts/docker-test.sh`
- Add `Dockerfile`, `docker-compose.yml`, and `.dockerignore`.
- Add `.github/workflows/ci.yml`.

Exit checks:

```sh
./scripts/test.sh
./scripts/race.sh
./scripts/bench.sh
```

If Docker is available:

```sh
./scripts/docker-test.sh
```

## Phase 10: Final Verification

Goal: ensure the complete codebase is ready.

Tasks:

- Run formatting.
- Run all tests.
- Run race tests.
- Run benchmarks.
- Verify root Go file count is exactly one.
- Verify no implementation Go files are in the root other than `processcache.go`.
- Verify public examples compile.

Exit checks:

```sh
gofmt -w processcache.go internal cmd test
go test ./...
go test -race ./...
go test -bench=. -benchmem ./...
find . -maxdepth 1 -name '*.go' -print
```

Expected root Go output:

```text
./processcache.go
```
