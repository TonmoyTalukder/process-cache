# AGENTS.md

This repository contains **ProcessCache**, a Go module for bounded, thread-safe, in-process LRU caching.

Agents working in this repository must follow `docs/in_process_cache_design.md` and `docs/phase.md`.

## Project Rules

- Module path: `github.com/tonmoytalukder/process-cache`.
- Go version: `1.24` or newer.
- Public import path:

  ```go
  import processcache "github.com/tonmoytalukder/process-cache"
  ```

- The root may contain at most one `.go` file: `processcache.go`.
- `processcache.go` is a public facade only.
- Implementation code lives under `internal/processcache`.
- Test helpers live under `internal/pkg/testclock` or `test`.
- Do not create public implementation packages such as `pkg/cache` or `cache`.
- Do not add Redis, database, HTTP server, or sidecar runtime dependencies.
- Keep runtime dependencies at zero unless there is a documented reason.
- Docker is for development, tests, race checks, and benchmarks only.

## API Requirements

The public facade must expose:

- `Cache`
- `MemoryCache`
- `Config`
- `TypeLimit`
- `Stats`
- `Option`
- `Sizer`
- `Clock`
- `KB`
- `MB`
- sentinel errors
- `NewMemoryCache(opts ...Option) (*MemoryCache, error)`
- `GetAs[T any](c Cache, key string) (T, bool)`
- all options listed in the design doc

`Cache` must include:

```go
Get(key string) (any, bool)
Set(key string, value any, ttl ...time.Duration) bool
Delete(key string) bool
Exists(key string) bool
Clear()
Len() int
Stats() Stats
Close() error
```

## Algorithm Requirements

- Use `map[string]*item` for O(1) average key lookup.
- Use one global doubly-linked list for global LRU.
- Use one doubly-linked list per configured prefix for type-scoped LRU.
- Global LRU eviction must be O(1).
- Type-scoped LRU eviction must be O(1).
- Lazy expiration must run on `Get` and `Exists`.
- Background cleanup must stop on `Close`.
- `Close` must be idempotent.

## Implementation Workflow

Follow `docs/phase.md` in order.

After each phase, run the phase exit checks before continuing.

Required final checks:

```sh
gofmt -w processcache.go internal cmd test
go test ./...
go test -race ./...
go test -bench=. -benchmem ./...
```

If Docker is available:

```sh
./scripts/docker-test.sh
```

## File Editing Rules

- Keep changes scoped to ProcessCache.
- Do not rewrite `docs/in_process_cache_design.md` unless the implementation proves the design needs a factual correction.
- Do not leave TODO placeholders.
- Do not commit generated cache/build output.
- Keep README examples compileable.
- Keep scripts executable.

## Notes For Codex

- Prefer `rg` and `rg --files` for inspection.
- Use small, reviewable patches.
- Verify with real commands, not only reasoning.
- If a network or Docker command is blocked by sandboxing, report it clearly.

## Notes For Claude

- Treat this file and `docs/phase.md` as the execution contract.
- Complete one phase at a time.
- Do not skip tests after implementing behavior.
- Keep the final response concise: summarize files changed and commands run.
