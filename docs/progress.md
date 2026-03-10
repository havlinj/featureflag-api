---
# Progress tracker – Feature Flag & Experiment Management API
# Aligned with .cursor/rules/development_workflow.mdc
---

# 🏗️ Progress Tracker – Feature Flag API

**Last updated**: 2026-03-10  
**Overall progress**: ███████░░░ 62% (Phase 1–2 of 4 complete)  
**Status**: Phase 1 and Phase 2 **complete and reviewed (APPROVED)** – Local Bash scripts, binary smoke, config refactor, bash integration tests, GitHub Actions CI  
**Next step**: Phase 3 – Experiments integration (work on feature branch, then merge to master)  
**Blockers**: None

## 📋 Milestones (per development_workflow.mdc)

| Phase | Status | Progress | Key Deliverables |
|-------|--------|----------|------------------|
| Phase 1: Feature Flags & Users Core | ✅ Complete | 100% | Flags + Users API, rollout strategies (percentage + attribute, one per flag), rules CRUD, EvaluateFlag with context, DeleteFlag, DB, auth (JWT), logging, integration tests |
| Phase 2: Local Test Scripts, Binary Smoke & CI | ✅ Complete (reviewed) | 100% | Bash scripts (check, unit, integration, build, test_all_quick, test_all_full, test_binary_smoke); scripts/integration/; internal/config; GitHub Actions CI (push/PR to master) |
| Phase 3: Experiments Integration | ⏳ Planned | 0% | Experiments module, schema, resolvers, assignments |
| Phase 4: Audit Logging | ⏳ Planned | 0% | Audit service, audit_logs table, hooks |

## 🔧 Phase 1 – Current state

- [x] HTTPS server + GraphQL (gqlgen) transport
- [x] DB schema: `users`, `feature_flags` (incl. `rollout_strategy`), `flag_rules` (internal/db, EnsureSchema)
- [x] DB init test (testcontainers, naked INSERT/SELECT)
- [x] flags.Store interface; PostgresStore (Create, GetByKeyAndEnvironment, Update, Delete, GetRulesByFlagID, ReplaceRulesByFlagID) with unit/integration tests
- [x] flags.Service + NewService(store); resolvers call service; **single mock per domain**: internal/flags/mock (queue-based Store for unit tests)
- [x] users.Store interface; PostgresStore (Create, GetByID, GetByEmail, Update, Delete); users.Service (CreateUser, GetUser, GetUserByEmail, Login, UpdateUser, DeleteUser); internal/users/mock for unit tests
- [x] GraphQL: users schema (users.graphqls), gqlgen generated types, users resolvers; NewApp(tlsConfig, flagsStore, usersStore)
- [x] Integration tests: GraphQL over HTTPS with **real PostgreSQL** (testutil.PostgresForIntegration, TruncateAll); TestFlagsAPI, TestUsersAPI, TestLogin_returnsToken_and_tokenWorksForProtectedMutation, TestProtectedMutation_withoutAuth_returnsError, TestAdminCreatedUser_canLogin_and_roleEnforced
- [x] flags.Service.CreateFlag / UpdateFlag / EvaluateFlag / DeleteFlag **business logic**: uniqueness, default env "dev", **rollout_strategy** (none / percentage / attribute, one per flag), rules validation (same type, ErrRulesStrategyMismatch), ReplaceRulesByFlagID; **EvaluateFlag(key, context: { userId, email })** with percentage (deterministic hash) and attribute rules (suffix, in, eq)
- [x] internal/flags/attribute.go: attribute rule evaluation (JSON condition, attributeValue, evaluateSuffix/In/eq)
- [x] Authentication middleware (JWT); login mutation; users.password_hash; auth.PasswordMatches, IssueToken, ParseAndValidate, RequireRole
- [x] Logging middleware (request method, path, status, duration); refactored with loggingHandler
- [x] Optional password in CreateUserInput and UpdateUserInput (admin can set login password when creating/updating users)
- [x] Unit tests: all return paths covered in flags.Service, users.Service, middleware auth; descriptive mock error messages in tests; gofmt enforced before task completion (coding_style.mdc)

## 🔧 Phase 2 – Current state

- [x] scripts/check.sh — gofmt -l + go vet
- [x] scripts/test_unit.sh — unit tests (internal + transport)
- [x] scripts/test_integration.sh — integration tests (-tags=integration)
- [x] scripts/build.sh — binary to ./bin/featureflag-api
- [x] scripts/test_all_quick.sh — check + unit + Go integration (quick local validation; no binary smoke)
- [x] scripts/test_all_full.sh — full suite as in CI (check, unit, Go integration, build, binary smoke, bash integration tests)
- [x] scripts/test_binary_smoke.sh — build, Postgres, binary, login → createFlag → evaluateFlag, tear down
- [x] cmd/main.go runs from env (DATABASE_DSN / PG*, JWT_SECRET, LISTEN_ADDR, optional TLS_CERT_FILE/TLS_KEY_FILE)
- [x] internal/config: GetDSN, GetListenAddr, GetJWTSecret, LoadTLSConfig (unit tested, GUNIT form)
- [x] scripts/integration/: test_missing_jwt_secret, test_invalid_dsn, test_default_listen_addr, test_tls_config (each config function covered by at least one bash integration test)
- [x] .github/workflows/ci.yml — on push to master and PR to master: check, unit, Go integration, build, binary smoke, bash integration tests (runner: ubuntu-latest, Go + Docker)

## Phase 2 – REVIEW (final)

**Status: APPROVED**

1. **Architecture**  
   Layering is respected: `cmd/main.go` only wires config, db, app; config logic lives in `internal/config` with injectable getenv/loader. No business logic in transport; bash scripts only orchestrate. No hidden coupling.

2. **Simplicity**  
   Scripts are straightforward (check, unit, integration, build, smoke, bash integration). Config package is minimal (GetDSN, GetListenAddr, GetJWTSecret, LoadTLSConfig). No over-engineering; `tools/tools.go` is a documented, standard workaround for tool deps.

3. **Readability**  
   Clear naming (`test_all_quick.sh` vs `test_all_full.sh`, config function names). Scripts and CI steps are self-explanatory. Comments where needed (e.g. `tools/tools.go`, script headers).

4. **Test quality**  
   Unit tests for `internal/config` cover every branch (GUNIT form). Bash integration tests each cover one config function in the real binary (missing JWT, invalid DSN, default listen addr, TLS). Smoke test covers happy path. Go integration tests unchanged and still in place.

5. **Alignment with plan**  
   All Phase 2 deliverables from `development_workflow.mdc` are present: check.sh, test_unit.sh, test_integration.sh, build.sh, test_all_quick.sh, test_all_full.sh, test_binary_smoke.sh, scripts/integration/*, GitHub Actions on push/PR to master. Module rename to `github.com/havlinj/featureflag-api` and `replace => .` are consistent; doc and script renames (test_all_quick / test_all_full) are reflected.

**Conclusion:** Phase 2 implementation matches the approved design. No scope creep. Ready to close; next step is Phase 3 (Experiments) or feature-branch workflow as planned.

## 📈 Metrics

- Test coverage: unit tests for db, flags.Service (incl. all return paths: CreateFlag/UpdateFlag/DeleteFlag/EvaluateFlag errors, strategy mismatch, ReplaceRulesByFlagID), users.Service (incl. all return paths: GetUser, GetUserByEmail, UpdateUser, DeleteUser store errors and invalid role), auth (password, JWT, RequireRole), middleware (logging, auth; incl. Authorization not Bearer → 401); flags and users PostgresStore (build tag `integration`); integration tests for HTTPS+GraphQL against **real Postgres** (testcontainers). Mock errors in tests use descriptive labels (e.g. GetByKeyAndEnvironment failed, ReplaceRulesByFlagID failed on CreateFlag).
- Tests: internal/db, internal/flags (service_test, attribute_test), internal/users, internal/auth (auth_test, jwt_test, password_test), transport/graphql/middleware (logging_test, auth_test), test/integration (tag `integration`). Default `go test ./...` skips integration; run with `-tags=integration` for full E2E.
- Code style: gofmt run before task completion (see .cursor/rules/coding_style.mdc).

## 📝 Changelog

**2026-03-10 (hotfix)**: CI/smoke fix so Actions pass on Linux. Smoke and integration scripts failed with “No public port 5432 published” and “Permission denied” on `build.sh`. Changes: (1) Docker runs for Postgres now publish port 5432 (`-p 5432:5432`) in `test_binary_smoke.sh`, `test_default_listen_addr.sh`, `test_tls_config.sh`. (2) Scripts are invoked directly (no `bash` wrapper); execute bit set in Git (`git update-index --chmod=+x`) and CI step “Make scripts executable” (`chmod +x scripts/*.sh scripts/integration/*.sh`) after checkout so scripts run on the Linux runner.

**2026-03-10**: Phase 2 closed after final REVIEW (APPROVED). Review checklist: architecture, simplicity, readability, test quality, alignment with plan — all satisfied. progress.md updated with Phase 2 – REVIEW section and status APPROVED; next step Phase 3. (Earlier same day: Phase 2 extended with CI.) GitHub Actions workflow (`.github/workflows/ci.yml`) runs on push to master and on pull_request to master: check, unit tests, Go integration tests (testcontainers), build, binary smoke test, and all bash integration tests (scripts/integration/*). Added `scripts/test_all_full.sh` for full suite (same as CI) and `scripts/test_all_quick.sh` for quick local validation only. development_workflow.mdc Phase 2 rewritten to include CI from the start; progress.md updated. Planned workflow for next phases: work on feature branch, then merge to master (CI runs on both).

**2026-03-02 (session 5)**: Phase 1 declared **complete** after final review. development_workflow.mdc updated: new Phase 2 (Local Test Scripts & Binary Smoke Test) with Bash scripts and one binary smoke test; former Phase 2/3 renumbered to Phase 3/4. progress.md updated: milestones table aligned (Phase 2 = local scripts, Phase 3 = Experiments, Phase 4 = Audit); status set to “Phase 1 reviewed and complete”; next step Phase 2. docs/session_5.md created.

**2026-03-01 (session 4)**: Phase 1 completed. Rollout strategies: feature_flags.rollout_strategy (none/percentage/attribute), one strategy per flag; CreateFlag/UpdateFlag with rules and validation (ErrRulesStrategyMismatch); EvaluateFlag(key, context: { userId, email }); attribute rules (internal/flags/attribute.go: suffix, in, eq); DeleteFlag(key, environment); Store.Delete, ReplaceRulesByFlagID. Unit test coverage: all return paths in flags.Service, users.Service, middleware auth; mock error messages made descriptive. progress.md updated; gofmt verified.

**2026-02-28**: Auth and logging completed. JWT auth middleware; login mutation; users table password_hash; auth.PasswordMatches (renamed from ComparePassword), IssueToken, ParseAndValidate, RequireRole. Logging middleware (method, path, status, duration); refactored with loggingHandler. Optional password in CreateUserInput and UpdateUserInput so admin can set login password. Unit tests: auth (password, JWT, RequireRole), users (Login, CreateUser/UpdateUser with password), middleware (logging, auth). Integration tests: login returns token and token works for createFlag; protected mutation without auth returns error; admin-created user (dev/viewer) can login and role enforced via API. Testing style: use `result` for single bool return in tests; keep `ok` for multi-value returns (testing_style.mdc). Progress updated.

**2026-02-26 (session 3)**: User management added. DB schema extended with `users` table. internal/users: entity, errors, store interface, PostgresStore (Create, GetByID, GetByEmail, Update, Delete), service (CreateUser, GetUser, GetUserByEmail, UpdateUser, DeleteUser). GraphQL: users.graphqls, gqlgen generated types, users resolvers; NewApp(tlsConfig, flagsStore, usersStore). **Single mock per domain**: internal/flags/mock and internal/users/mock (queue-based Store for unit tests); removed internal/testutil/flags_mock.go. Unit tests: flags and users service tests use package `*_test` with domain mocks. **Integration tests use real PostgreSQL**: testutil.PostgresForIntegration(t) and TruncateAll(t, db) in internal/testutil/postgres.go; TestFlagsAPI and TestUsersAPI run against PostgresStore and testcontainers Postgres (no mocks in integration). Progress doc updated.

**2026-02-26 (session 2)**: flags.Service business logic implemented: CreateFlag (uniqueness check, Create), UpdateFlag (GetByKey+default env "dev", Update), EvaluateFlag (userID validation, percentage rule with deterministic hash, attribute rule fallback, ErrInvalidRule). Added internal/flags/service_test.go: unit tests with mock store. PostgresStore tests tagged `//go:build integration`. Reformatted tests to strict arrange/act/assert. Updated .cursor/rules/testing_style.mdc.

**2026-02-26**: Phase 1 progress. Database layer: internal/db (Open, EnsureSchema, schema for feature_flags + flag_rules). PostgresStore implemented and tested. NewApp(tlsConfig, flagsStore) accepts Store. Service layer and resolvers wired to Store. Progress doc updated per development_workflow.mdc.

*(Earlier: HTTPS server, GraphQL resolvers, testutil GraphQL client + TLS, flags.Store interface and mocks.)*

## 🤝 Discussion / Questions

- None.
