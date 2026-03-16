#!/usr/bin/env sh
set -eu

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
COVER_DIR="$ROOT_DIR/.tmp"
COVER_FILE="$COVER_DIR/coverage.out"

# Minimum acceptable total coverage percentage.
# Override with: GO_COVER_MIN=50 ./scripts/check-go-coverage.sh
MIN_COVERAGE="${GO_COVER_MIN:-45}"
GO_COVER_PACKAGES="${GO_COVER_PACKAGES:-./...}"

mkdir -p "$COVER_DIR"

echo "[coverage] running go tests with coverage profile..."
go test $GO_COVER_PACKAGES -coverprofile="$COVER_FILE"

TOTAL_LINE="$(go tool cover -func="$COVER_FILE" | grep '^total:')"
TOTAL_COVERAGE="$(printf '%s' "$TOTAL_LINE" | awk '{print $3}' | tr -d '%')"

echo "[coverage] total: ${TOTAL_COVERAGE}% (minimum: ${MIN_COVERAGE}%)"

awk_check="$(awk -v total="$TOTAL_COVERAGE" -v min="$MIN_COVERAGE" 'BEGIN { if (total+0 < min+0) print "fail"; else print "pass"; }')"
if [ "$awk_check" = "fail" ]; then
  echo "[coverage] failed: total coverage below threshold"
  exit 1
fi

echo "[coverage] passed"
