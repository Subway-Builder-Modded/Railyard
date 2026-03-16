#!/usr/bin/env sh
set -eu

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT_DIR"

echo "[pre-push] checking Go formatting..."
GO_FILES="$(git ls-files '*.go')"
if [ -n "$GO_FILES" ]; then
  # shellcheck disable=SC2086
  UNFORMATTED="$(gofmt -l $GO_FILES)"
  if [ -n "$UNFORMATTED" ]; then
    echo "[pre-push] gofmt required for:"
    echo "$UNFORMATTED"
    exit 1
  fi
fi

echo "[pre-push] running Go tests..."
go test ./...

echo "[pre-push] running Go coverage gate..."
"$ROOT_DIR/scripts/check-go-coverage.sh"

echo "[pre-push] running frontend lint/format/tests..."
cd "$ROOT_DIR/frontend"
pnpm run lint
pnpm run format:check
pnpm run test

echo "[pre-push] all checks passed"
