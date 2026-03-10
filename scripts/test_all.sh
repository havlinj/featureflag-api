#!/usr/bin/env bash
# Runs the full local validation: check, unit tests, integration tests.
# Exits on first failure.
set -e
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$ROOT_DIR"

echo "=== 1/3 check"
"$SCRIPT_DIR/check.sh"
echo ""
echo "=== 2/3 unit tests"
"$SCRIPT_DIR/test_unit.sh"
echo ""
echo "=== 3/3 integration tests"
"$SCRIPT_DIR/test_integration.sh"
echo ""
echo "=== All passed."
