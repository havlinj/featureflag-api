# Session 1 – Summary

**Date**: 2026-02-26

## 1. HTTPS server & GraphQL (app + transport)

- **internal/app**: `NewApp(tlsConfig, flagsStore)`, `Run(addr)`, `Shutdown(ctx)`. App builds GraphQL handler from gqlgen and runs HTTPS server when `tlsConfig` is set.
- **transport/graphql**: `Server` with `NewServer(handler, tlsConfig)`, `Run(addr)`, `Shutdown(ctx)`; TLS if config has certificates.
- **internal/testutil**: `NewTLSConfigForServer()` (self-signed cert), `NewClientForIntegration(baseURL)`, GraphQL client with `DoRequest(query, variables)` and queue-style usage.
- **Integration test** (`test/integration`, tag `integration`): Start app over HTTPS, call createFlag, updateFlag, evaluateFlag via GraphQL client; assert response has `data` or `errors`. Uses `&testutil.MockFlagsStore{}` as store.

## 2. Flags domain – service & persistence contract

- **internal/flags/entity.go**: `Flag` (ID, Key, Description, Enabled, Environment, CreatedAt `time.Time`), `Rule`, `RuleType` (percentage, attribute).
- **internal/flags/store.go**: Interface `Store` – `Create`, `GetByKeyAndEnvironment`, `Update`, `GetRulesByFlagID`. Comment describes how `flags.Service` will use it.
- **internal/flags/service.go**: `Service` with `Store`; `NewService(store)`. `CreateFlag`, `UpdateFlag`, `EvaluateFlag` call store but business logic still `panic("unimplemented")`.
- **transport/graphql**: Resolvers call `r.Flags.CreateFlag` / `UpdateFlag` / `EvaluateFlag` (no panic in resolver).

## 3. Mock store (testutil)

- **internal/testutil/flags_mock.go**: `MockFlagsStore` implements `flags.Store`. No in-memory DB; only records arguments and returns next enqueued result per method.
- **Queues**: `CreateReturns []FlagsCreateResult`, `GetByKeyAndEnvironmentReturns []FlagsGetByKeyResult`, `UpdateReturns []error`, `GetRulesByFlagIDReturns []FlagsGetRulesResult`. Each call consumes the next element; empty queue → `ErrNoMoreFlagsReturns`.
- **Result types** (value or error, Rust-style): `FlagsCreateResult`, `FlagsGetByKeyResult`, `FlagsGetRulesResult` defined at end of file. Call recording in `*Calls` slices for assertions.

## 4. Database – init & schema

- **internal/db**: `DB` with `Open(ctx, dsn)`, `EnsureSchema(ctx)` (idempotent), `Conn()`, `Close()`. Schema in `schema.go`: `feature_flags` (id UUID, key, description, enabled, environment, created_at, UNIQUE(key, environment)), `flag_rules` (id UUID, flag_id FK, type, value). Driver: pgx via stdlib.
- **Init test**: `TestDB_EnsureSchema_and_naked_write_read` – testcontainers Postgres, EnsureSchema, truncate, naked INSERT into `feature_flags` and `flag_rules`, then SELECT and assert. Refactored into: `arrangeDBWithSchema(t)`, `truncateTables(t, database)`, `insertFeatureFlagRow`, `insertFlagRuleRow`, `assertFeatureFlagReadBack`, `assertFlagRuleReadBack`.

## 5. PostgresStore (flags repository)

- **internal/flags/postgres.go**: `NewPostgresStore(conn *sql.DB)`. `Create` – INSERT with RETURNING, on unique violation return `ErrDuplicateKey` (pgx 23505). `GetByKeyAndEnvironment` – SELECT; `sql.ErrNoRows` → `(nil, nil)`. `Update` – by ID; 0 rows → `ErrNotFound`. `GetRulesByFlagID` – SELECT all rules for flag.
- **internal/flags/errors.go**: `ErrDuplicateKey`, `ErrNotFound`.
- **Tests** (internal/flags/postgres_test.go): Shared `testDB(t)` and `truncateFlags(t, database)`. Each scenario as its own test function (no subtests): Create happy + duplicate, GetByKey happy + not found, Update happy + not found, GetRulesByFlagID happy + no rules + invalid UUID. All use testcontainers Postgres.

## 6. Test style & refactors

- **DB test**: One main test; arrange/act/assert split into helpers with `t.Helper()`.
- **Flags tests**: Each case = separate `TestPostgresStore_*` function (e.g. `TestPostgresStore_Create_happy_path`, `TestPostgresStore_Create_duplicate_key_returns_ErrDuplicateKey`).
- **Containers**: Started by test code via `postgres.Run()`; lifetime = one test; cleanup via `t.Cleanup` or `defer cleanup()` (Terminate + Close).

## 7. App wiring

- **NewApp(tlsConfig, flagsStore flags.Store)** – store is required. Production: `db.Open` → `EnsureSchema` → `flags.NewPostgresStore(db.Conn())` → `app.NewApp(tlsConfig, store)`. Integration test: `app.NewApp(tlsConfig, &testutil.MockFlagsStore{})`.

## 8. Docs & workflow

- **docs/progress.md**: Updated per `.cursor/rules/development_workflow.mdc` – Changelog entry 2026-02-26, Phase 1 checkboxes (done: HTTPS, schema, Store, PostgresStore, tests, integration test; open: service logic, auth, logging, E2E with real DB), status APPROVED, next steps.

## Files touched (overview)

- **New**: internal/db/db.go, schema.go, db_test.go; internal/flags/service.go, store.go, entity.go, errors.go, postgres.go, postgres_test.go; internal/testutil/flags_mock.go, tls.go (TLS + client already present).
- **Modified**: internal/app/app.go, server.go; transport/graphql/server.go, resolver.go, flags.resolvers.go; internal/flags/entity.go (CreatedAt); test/integration/integration_api_test.go; docs/progress.md.
- **Removed**: internal/flags/mock_store.go (moved to testutil).

## Dependencies added

- `github.com/jackc/pgx/v5/stdlib`
- `github.com/testcontainers/testcontainers-go`, `.../modules/postgres`
