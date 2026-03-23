#!/usr/bin/env bash
# Runs coverage for core packages and enforces a minimum threshold.
set -e
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$ROOT_DIR"

MIN_COVERAGE=80

echo "== coverage (./internal/... ./transport/...)"
go test -covermode=atomic -coverprofile=coverage.out ./internal/... ./transport/...

COVERAGE_TOTAL=$(go tool cover -func=coverage.out | awk '/^total:/ {gsub("%","",$3); print $3}')
if awk "BEGIN {exit !($COVERAGE_TOTAL >= $MIN_COVERAGE)}"; then
  echo "Coverage ${COVERAGE_TOTAL}% >= ${MIN_COVERAGE}%"
else
  echo "Coverage ${COVERAGE_TOTAL}% < ${MIN_COVERAGE}%"
  exit 1
fi
