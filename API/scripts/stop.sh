#!/usr/bin/env sh
# Stop containers without deleting named PostgreSQL volumes.
set -eu

ROOT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
cd "$ROOT_DIR"

docker compose --profile cloudflare down
