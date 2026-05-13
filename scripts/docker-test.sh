#!/usr/bin/env sh
set -eu

docker compose run --build --rm test
