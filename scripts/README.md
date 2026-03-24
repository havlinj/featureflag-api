# Scripts

## Tests and CI

- **Quick local validation**: `./scripts/test_all_quick.sh` — check, unit tests, Go integration tests (no binary smoke).
- **Full suite (same as CI)**: `./scripts/test_all_full.sh` — check, unit, Go integration, build, binary smoke, bash integration tests. Requires Docker.
- **CI**: GitHub Actions (`.github/workflows/ci.yml`) runs the full suite on every push to `master` and on pull requests targeting `master`. Scripts are executable (execute bit in repo + `chmod +x` step in CI on Linux).
- **CI sync guard**: `python3 ./scripts/verify_ci_sync.py` verifies that scripts referenced by `test_all_full.sh` are also listed in CI workflow commands.

See `scripts/integration/README.md` for the list of bash integration tests.

## Coverage tooling

All coverage-related scripts and docs are in `scripts/coverage/`.

- Gate runner: `bash ./scripts/coverage/test_coverage.sh`
- Hotspot analysis: `python3 ./scripts/coverage/coverage_hotspots.py`
- Docs and usage: `scripts/coverage/README.md`


## Seed first admin

Before using the API for user management or protected operations, at least one admin must exist. Create the first admin by running the seed script **once per environment** (e.g. after deploy).

### Option 1: Wrapper script (recommended)

Set environment variables and run the script. The script hashes the password with bcrypt and inserts the user (idempotent: safe to run multiple times; existing email is skipped).

```bash
export FIRST_ADMIN_EMAIL="admin@example.com"
export FIRST_ADMIN_PASSWORD="your-secure-password"
# DB connection: set PGHOST, PGPORT, PGDATABASE, PGUSER, PGPASSWORD (or use .pgpass)
./scripts/seed_admin.sh
```

### Option 2: Manual SQL

1. Generate a bcrypt hash of the password:
   ```bash
   go run ./cmd/seedpass "your-secure-password"
   ```
2. Run the SQL with the email and hash (replace placeholders):
   ```bash
   psql ... -v email="admin@example.com" -v passhash='$2a$12$...' -f scripts/seed_admin.sql
   ```

### Security

- Never commit plaintext passwords.
- Use a strong password for the first admin.
- Restrict access to the script and env vars in production.
