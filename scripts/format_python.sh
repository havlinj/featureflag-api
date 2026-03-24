#!/usr/bin/env bash
# Local-only Python formatting/lint helper (not used in CI).
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$ROOT_DIR"

VENV_DIR="${PY_FORMAT_VENV_DIR:-.venv}"
RUFF_BIN="$VENV_DIR/bin/ruff"

if [[ ! -x "$RUFF_BIN" ]]; then
  echo "== setting up local venv and installing ruff"
  python3 -m venv "$VENV_DIR"
  "$VENV_DIR/bin/pip" install ruff
fi

echo "== formatting Python files under scripts/"
"$RUFF_BIN" format scripts

echo "== applying safe lint fixes under scripts/"
"$RUFF_BIN" check --fix scripts

echo "== final lint verification"
"$RUFF_BIN" check scripts

echo "OK: Python formatting and lint checks completed."
