#!/usr/bin/env bash
# Seeds the first admin user. Reads FIRST_ADMIN_EMAIL and FIRST_ADMIN_PASSWORD from the environment.
# Requires: go (for hashing), psql (for DB). Connection via PGHOST, PGPORT, PGDATABASE, PGUSER, PGPASSWORD.
set -e
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
if [[ -z "$FIRST_ADMIN_EMAIL" || -z "$FIRST_ADMIN_PASSWORD" ]]; then
  echo "Set FIRST_ADMIN_EMAIL and FIRST_ADMIN_PASSWORD in the environment." >&2
  exit 1
fi
cd "$ROOT_DIR"
PASSHASH=$(go run ./cmd/seedpass "$FIRST_ADMIN_PASSWORD")
export email="$FIRST_ADMIN_EMAIL"
export passhash="$PASSHASH"
psql -v email="$FIRST_ADMIN_EMAIL" -v passhash="$PASSHASH" -f scripts/seed_admin.sql
