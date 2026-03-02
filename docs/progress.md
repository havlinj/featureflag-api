---
# Progress tracker – Feature Flag & Experiment Management API
# Aligned with .cursor/rules/development_workflow.mdc
---

# 🏗️ Progress Tracker – Feature Flag API

**Last updated**: 2026-03-02  
**Overall progress**: █████░░░░░ 50% (Phase 1 of 4 complete)  
**Status**: Phase 1 **reviewed and complete** – Flags (incl. rollout strategies, rules, evaluation, delete), Users, Auth, Logging; Phase 2 planned (local test scripts + binary smoke)  
**Next step**: Phase 2 – Local test scripts & binary smoke test  
**Blockers**: None

## 📋 Milestones (per development_workflow.mdc)

| Phase | Status | Progress | Key Deliverables |
|-------|--------|----------|------------------|
| Phase 1: Feature Flags & Users Core | ✅ Complete | 100% | Flags + Users API, rollout strategies (percentage + attribute, one per flag), rules CRUD, EvaluateFlag with context, DeleteFlag, DB, auth (JWT), logging, integration tests |
| Phase 2: Local Test Scripts & Binary Smoke Test | ⏳ Planned | 0% | Bash scripts (check, unit, integration, build, test_all, test_binary_smoke); one smoke test against real binary |
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

## 📈 Metrics

- Test coverage: unit tests for db, flags.Service (incl. all return paths: CreateFlag/UpdateFlag/DeleteFlag/EvaluateFlag errors, strategy mismatch, ReplaceRulesByFlagID), users.Service (incl. all return paths: GetUser, GetUserByEmail, UpdateUser, DeleteUser store errors and invalid role), auth (password, JWT, RequireRole), middleware (logging, auth; incl. Authorization not Bearer → 401); flags and users PostgresStore (build tag `integration`); integration tests for HTTPS+GraphQL against **real Postgres** (testcontainers). Mock errors in tests use descriptive labels (e.g. GetByKeyAndEnvironment failed, ReplaceRulesByFlagID failed on CreateFlag).
- Tests: internal/db, internal/flags (service_test, attribute_test), internal/users, internal/auth (auth_test, jwt_test, password_test), transport/graphql/middleware (logging_test, auth_test), test/integration (tag `integration`). Default `go test ./...` skips integration; run with `-tags=integration` for full E2E.
- Code style: gofmt run before task completion (see .cursor/rules/coding_style.mdc).

## 📝 Changelog

**2026-03-02 (session 5)**: Phase 1 declared **complete** after final review. development_workflow.mdc updated: new Phase 2 (Local Test Scripts & Binary Smoke Test) with Bash scripts and one binary smoke test; former Phase 2/3 renumbered to Phase 3/4. progress.md updated: milestones table aligned (Phase 2 = local scripts, Phase 3 = Experiments, Phase 4 = Audit); status set to “Phase 1 reviewed and complete”; next step Phase 2. docs/session_5.md created.

**2026-03-01 (session 4)**: Phase 1 completed. Rollout strategies: feature_flags.rollout_strategy (none/percentage/attribute), one strategy per flag; CreateFlag/UpdateFlag with rules and validation (ErrRulesStrategyMismatch); EvaluateFlag(key, context: { userId, email }); attribute rules (internal/flags/attribute.go: suffix, in, eq); DeleteFlag(key, environment); Store.Delete, ReplaceRulesByFlagID. Unit test coverage: all return paths in flags.Service, users.Service, middleware auth; mock error messages made descriptive. progress.md updated; gofmt verified.

**2026-02-28**: Auth and logging completed. JWT auth middleware; login mutation; users table password_hash; auth.PasswordMatches (renamed from ComparePassword), IssueToken, ParseAndValidate, RequireRole. Logging middleware (method, path, status, duration); refactored with loggingHandler. Optional password in CreateUserInput and UpdateUserInput so admin can set login password. Unit tests: auth (password, JWT, RequireRole), users (Login, CreateUser/UpdateUser with password), middleware (logging, auth). Integration tests: login returns token and token works for createFlag; protected mutation without auth returns error; admin-created user (dev/viewer) can login and role enforced via API. Testing style: use `result` for single bool return in tests; keep `ok` for multi-value returns (testing_style.mdc). Progress updated.

**2026-02-26 (session 3)**: User management added. DB schema extended with `users` table. internal/users: entity, errors, store interface, PostgresStore (Create, GetByID, GetByEmail, Update, Delete), service (CreateUser, GetUser, GetUserByEmail, UpdateUser, DeleteUser). GraphQL: users.graphqls, gqlgen generated types, users resolvers; NewApp(tlsConfig, flagsStore, usersStore). **Single mock per domain**: internal/flags/mock and internal/users/mock (queue-based Store for unit tests); removed internal/testutil/flags_mock.go. Unit tests: flags and users service tests use package `*_test` with domain mocks. **Integration tests use real PostgreSQL**: testutil.PostgresForIntegration(t) and TruncateAll(t, db) in internal/testutil/postgres.go; TestFlagsAPI and TestUsersAPI run against PostgresStore and testcontainers Postgres (no mocks in integration). Progress doc updated.

**2026-02-26 (session 2)**: flags.Service business logic implemented: CreateFlag (uniqueness check, Create), UpdateFlag (GetByKey+default env "dev", Update), EvaluateFlag (userID validation, percentage rule with deterministic hash, attribute rule fallback, ErrInvalidRule). Added internal/flags/service_test.go: unit tests with mock store. PostgresStore tests tagged `//go:build integration`. Reformatted tests to strict arrange/act/assert. Updated .cursor/rules/testing_style.mdc.

**2026-02-26**: Phase 1 progress. Database layer: internal/db (Open, EnsureSchema, schema for feature_flags + flag_rules). PostgresStore implemented and tested. NewApp(tlsConfig, flagsStore) accepts Store. Service layer and resolvers wired to Store. Progress doc updated per development_workflow.mdc.

*(Earlier: HTTPS server, GraphQL resolvers, testutil GraphQL client + TLS, flags.Store interface and mocks.)*

## 🤝 Discussion / Questions

- None.
