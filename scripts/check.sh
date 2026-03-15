#!/usr/bin/env bash
# Runs lint/format/vet checks. Exits with 1 if any check fails.
set -e
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$ROOT_DIR"

echo "== gofmt -l (unformatted files)"
UNFORMATTED=$(gofmt -l . 2>/dev/null | grep -v '^$' || true)
if [[ -n "$UNFORMATTED" ]]; then
  echo "The following files are not formatted with gofmt:"
  echo "$UNFORMATTED"
  exit 1
fi
echo "OK"

echo "== go vet ./..."
go vet ./...
echo "OK"
