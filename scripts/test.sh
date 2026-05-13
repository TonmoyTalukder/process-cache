#!/usr/bin/env sh
set -eu

export GOCACHE="${GOCACHE:-$(pwd)/.gocache}"

go test ./...
(
  cd test/e2e/consumer
  go test ./...
)
