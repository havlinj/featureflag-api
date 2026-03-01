---
# Progress tracker – Feature Flag & Experiment Management API
# Aligned with .cursor/rules/development_workflow.mdc
---

# 🏗️ Progress Tracker – Feature Flag API

**Last updated**: 2026-02-28  
**Overall progress**: ████████░░ 85% (Phase 1 near complete)  
**Status**: APPROVED (Phase 1 – Flags + Users + Auth + Logging done; optional password in CreateUser/UpdateUser)  
**Next step**: Phase 1 review / Phase 2 planning  
**Blockers**: None

## 📋 Milestones (per development_workflow.mdc)

| Phase | Status | Progress | Key Deliverables |
|-------|--------|----------|------------------|
| Phase 1: Feature Flags & Users Core | 🔄 In Progress | ~85% | Flags + Users API, DB, repos, auth (JWT) + logging middleware, login, optional password in CreateUser/UpdateUser, integration tests |
| Phase 2: Experiments Integration | ⏳ Planned | 0% | Experiments module, schema, resolvers, assignments |
| Phase 3: Audit Logging | ⏳ Planned | 0% | Audit service, audit_logs table, hooks |

## 🔧 Phase 1 – Current state

- [x] HTTPS server + GraphQL (gqlgen) transport
- [x] DB schema: `users`, `feature_flags`, `flag_rules` (internal/db, EnsureSchema)
- [x] DB init test (testcontainers, naked INSERT/SELECT)
- [x] flags.Store interface; PostgresStore (Create, GetByKeyAndEnvironment, Update, GetRulesByFlagID) with unit tests
- [x] flags.Service + NewService(store); resolvers call service; **single mock per domain**: internal/flags/mock (queue-based Store for unit tests)
- [x] users.Store interface; PostgresStore (Create, GetByID, GetByEmail, Update, Delete); users.Service (CreateUser, GetUser, GetUserByEmail, UpdateUser, DeleteUser); internal/users/mock for unit tests
- [x] GraphQL: users schema (users.graphqls), gqlgen generated types, users resolvers; NewApp(tlsConfig, flagsStore, usersStore)
- [x] Integration tests: GraphQL over HTTPS with **real PostgreSQL** (testutil.PostgresForIntegration, TruncateAll); TestFlagsAPI, TestUsersAPI, TestLogin_returnsToken_and_tokenWorksForProtectedMutation, TestProtectedMutation_withoutAuth_returnsError, TestAdminCreatedUser_canLogin_and_roleEnforced
- [x] flags.Service.CreateFlag / UpdateFlag / EvaluateFlag **business logic** (uniqueness, Update by key+env, EvaluateFlag with percentage rollout; default env "dev")
- [x] Authentication middleware (JWT); login mutation; users.password_hash; auth.PasswordMatches, IssueToken, ParseAndValidate, RequireRole
- [x] Logging middleware (request method, path, status, duration); refactored with loggingHandler
- [x] Optional password in CreateUserInput and UpdateUserInput (admin can set login password when creating/updating users)

## 📈 Metrics

- Test coverage: unit tests for db, flags.Service, users.Service (incl. Login, CreateUser/UpdateUser with password), auth (password, JWT, RequireRole), middleware (logging, auth); flags and users PostgresStore (build tag `integration`); integration tests (5) for HTTPS+GraphQL against **real Postgres** (testcontainers).
- Tests: internal/db, internal/flags, internal/users, internal/auth (auth_test, jwt_test, password_test), transport/graphql/middleware (logging_test, auth_test), test/integration (TestFlagsAPI, TestUsersAPI, TestLogin_returnsToken_..., TestProtectedMutation_withoutAuth_..., TestAdminCreatedUser_...; tag `integration`). Default `go test ./...` skips integration; run with `-tags=integration` for full E2E.

## 📝 Changelog

**2026-02-28**: Auth and logging completed. JWT auth middleware; login mutation; users table password_hash; auth.PasswordMatches (renamed from ComparePassword), IssueToken, ParseAndValidate, RequireRole. Logging middleware (method, path, status, duration); refactored with loggingHandler. Optional password in CreateUserInput and UpdateUserInput so admin can set login password. Unit tests: auth (password, JWT, RequireRole), users (Login, CreateUser/UpdateUser with password), middleware (logging, auth). Integration tests: login returns token and token works for createFlag; protected mutation without auth returns error; admin-created user (dev/viewer) can login and role enforced via API. Testing style: use `result` for single bool return in tests; keep `ok` for multi-value returns (testing_style.mdc). Progress updated.

**2026-02-26 (session 3)**: User management added. DB schema extended with `users` table. internal/users: entity, errors, store interface, PostgresStore (Create, GetByID, GetByEmail, Update, Delete), service (CreateUser, GetUser, GetUserByEmail, UpdateUser, DeleteUser). GraphQL: users.graphqls, gqlgen generated types, users resolvers; NewApp(tlsConfig, flagsStore, usersStore). **Single mock per domain**: internal/flags/mock and internal/users/mock (queue-based Store for unit tests); removed internal/testutil/flags_mock.go. Unit tests: flags and users service tests use package `*_test` with domain mocks. **Integration tests use real PostgreSQL**: testutil.PostgresForIntegration(t) and TruncateAll(t, db) in internal/testutil/postgres.go; TestFlagsAPI and TestUsersAPI run against PostgresStore and testcontainers Postgres (no mocks in integration). Progress doc updated.

**2026-02-26 (session 2)**: flags.Service business logic implemented: CreateFlag (uniqueness check, Create), UpdateFlag (GetByKey+default env "dev", Update), EvaluateFlag (userID validation, percentage rule with deterministic hash, attribute rule fallback, ErrInvalidRule). Added internal/flags/service_test.go: unit tests with mock store. PostgresStore tests tagged `//go:build integration`. Reformatted tests to strict arrange/act/assert. Updated .cursor/rules/testing_style.mdc.

**2026-02-26**: Phase 1 progress. Database layer: internal/db (Open, EnsureSchema, schema for feature_flags + flag_rules). PostgresStore implemented and tested. NewApp(tlsConfig, flagsStore) accepts Store. Service layer and resolvers wired to Store. Progress doc updated per development_workflow.mdc.

*(Earlier: HTTPS server, GraphQL resolvers, testutil GraphQL client + TLS, flags.Store interface and mocks.)*

## 🤝 Discussion / Questions

- None.
