# Integration tests (bash, real binary)

Each script in this directory is one integration test. They run the built application binary with real OS (env, optional Docker) and assert behaviour.

- **Sad path / config tests**: no Docker required; they only need the binary built (run `../build.sh` first or the script builds it). They assert exit codes or minimal behaviour.
- **Tests that need a running server** (e.g. default listen address, TLS): start Postgres and the binary like the parent `test_binary_smoke.sh`, then assert via curl.

**Integration tests in this directory:**

| Script | Covers (config used in main) | Description |
|--------|------------------------------|-------------|
| `test_missing_jwt_secret.sh` | `GetJWTSecret` | Binary exits non-zero when JWT_SECRET is unset. |
| `test_weak_jwt_secret.sh` | `GetJWTSecret` | Binary exits non-zero when JWT_SECRET is too short. |
| `test_invalid_dsn.sh` | `GetDSN` | Binary exits non-zero when database is unreachable. |
| `test_default_listen_addr.sh` | `GetListenAddr` | When LISTEN_ADDR is unset, server listens on :8080. |
| `test_tls_config.sh` | `LoadTLSConfig` | When TLS_CERT_FILE and TLS_KEY_FILE are set, server serves HTTPS. |
| `test_invalid_tls_files.sh` | `LoadTLSConfig` | Binary exits non-zero when TLS_CERT_FILE/TLS_KEY_FILE are invalid. |

Run from project root, or from `scripts/`:

```bash
./scripts/integration/test_missing_jwt_secret.sh
./scripts/integration/test_weak_jwt_secret.sh
./scripts/integration/test_invalid_dsn.sh
./scripts/integration/test_default_listen_addr.sh
./scripts/integration/test_tls_config.sh
./scripts/integration/test_invalid_tls_files.sh
```

Requirements: `go`, and for tests that start the server also `docker`, `curl`, `openssl` (for `test_tls_config.sh`).
