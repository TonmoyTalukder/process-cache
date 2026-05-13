# Contributing

Thanks for contributing to ProcessCache.

## Local checks

Run the same core checks expected before release:

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

## Development notes

- Follow [AGENTS.md](AGENTS.md) and [docs/phase.md](docs/phase.md).
- Keep the root package limited to `processcache.go`.
- Put implementation under `internal/processcache`.
- Keep README examples compileable.
- Prefer small, reviewable patches with tests for behavior changes.
