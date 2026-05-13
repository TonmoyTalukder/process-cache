#!/usr/bin/env sh
set -eu

export GOCACHE="${GOCACHE:-$(pwd)/.gocache}"

go test -bench=. -benchmem ./...
