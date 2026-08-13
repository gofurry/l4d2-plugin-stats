#!/usr/bin/env sh
set -eu

SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
REPO_ROOT=$(CDPATH= cd -- "$SCRIPT_DIR/.." && pwd)
DASHBOARD_ROOT="$REPO_ROOT/dashboard"
OUTPUT_DIR="$REPO_ROOT/dist"
VERSION=${L4D2_STATS_VERSION:-1.3.0}

cd "$DASHBOARD_ROOT/frontend"
pnpm install --frozen-lockfile
pnpm test
pnpm typecheck
pnpm lint
pnpm build

cd "$DASHBOARD_ROOT"
go tool sqlc generate
go test ./...
mkdir -p "$OUTPUT_DIR"
if [ ! -f "$OUTPUT_DIR/config.yaml" ]; then
  cp "$DASHBOARD_ROOT/config.example.yaml" "$OUTPUT_DIR/config.yaml"
fi
CGO_ENABLED=0 go build -trimpath \
  -ldflags "-s -w -X github.com/gofurry/l4d2-plugin-stats/dashboard/internal/cli.Version=$VERSION" \
  -o "$OUTPUT_DIR/l4d2-stats" \
  ./cmd/l4d2-stats

printf 'Dashboard build completed: %s\n' "$OUTPUT_DIR/l4d2-stats"
