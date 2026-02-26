---
# Progress tracker – Feature Flag & Experiment Management API
# Aligned with .cursor/rules/development_workflow.mdc
---

# 🏗️ Progress Tracker – Feature Flag API

**Last updated**: 2026-02-26  
**Overall progress**: ██████░░░░ 60% (Phase 1 in progress)  
**Status**: APPROVED (Phase 1 – Flags + Users core done; integration tests use real Postgres; auth & logging pending)  
**Next step**: Authentication middleware (JWT), logging middleware  
**Blockers**: None

## 📋 Milestones (per development_workflow.mdc)

| Phase | Status | Progress | Key Deliverables |
|-------|--------|----------|------------------|
| Phase 1: Feature Flags & Users Core | 🔄 In Progress | ~60% | Flags + Users API, DB, repos, integration tests (real Postgres); auth/logging middleware pending |
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
- [x] Integration tests: GraphQL over HTTPS with **real PostgreSQL** (testutil.PostgresForIntegration, TruncateAll); TestFlagsAPI (createFlag → updateFlag → evaluateFlag) and TestUsersAPI (createUser → get → update → get → delete → get) against PostgresStore
- [x] flags.Service.CreateFlag / UpdateFlag / EvaluateFlag **business logic** (uniqueness, Update by key+env, EvaluateFlag with percentage rollout; default env "dev")
- [ ] Authentication middleware (JWT)
- [ ] Logging middleware

## 📈 Metrics

- Test coverage: unit tests for db (init), flags.Service and users.Service (domain mocks from internal/flags/mock and internal/users/mock), flags.PostgresStore and users.PostgresStore (all methods + error cases, build tag `integration`); integration tests (2) for HTTPS+GraphQL against **real Postgres** (testcontainers).
- Tests: internal/db (1), internal/flags (service unit + postgres integration), internal/users (service unit + postgres integration), test/integration (2: TestFlagsAPI, TestUsersAPI; tag `integration`). Default `go test ./...` skips integration; run with `-tags=integration` for full E2E.

## 📝 Changelog

**2026-02-26 (session 3)**: User management added. DB schema extended with `users` table. internal/users: entity, errors, store interface, PostgresStore (Create, GetByID, GetByEmail, Update, Delete), service (CreateUser, GetUser, GetUserByEmail, UpdateUser, DeleteUser). GraphQL: users.graphqls, gqlgen generated types, users resolvers; NewApp(tlsConfig, flagsStore, usersStore). **Single mock per domain**: internal/flags/mock and internal/users/mock (queue-based Store for unit tests); removed internal/testutil/flags_mock.go. Unit tests: flags and users service tests use package `*_test` with domain mocks. **Integration tests use real PostgreSQL**: testutil.PostgresForIntegration(t) and TruncateAll(t, db) in internal/testutil/postgres.go; TestFlagsAPI and TestUsersAPI run against PostgresStore and testcontainers Postgres (no mocks in integration). Progress doc updated.

**2026-02-26 (session 2)**: flags.Service business logic implemented: CreateFlag (uniqueness check, Create), UpdateFlag (GetByKey+default env "dev", Update), EvaluateFlag (userID validation, percentage rule with deterministic hash, attribute rule fallback, ErrInvalidRule). Added internal/flags/service_test.go: unit tests with mock store. PostgresStore tests tagged `//go:build integration`. Reformatted tests to strict arrange/act/assert. Updated .cursor/rules/testing_style.mdc.

**2026-02-26**: Phase 1 progress. Database layer: internal/db (Open, EnsureSchema, schema for feature_flags + flag_rules). PostgresStore implemented and tested. NewApp(tlsConfig, flagsStore) accepts Store. Service layer and resolvers wired to Store. Progress doc updated per development_workflow.mdc.

*(Earlier: HTTPS server, GraphQL resolvers, testutil GraphQL client + TLS, flags.Store interface and mocks.)*

## 🤝 Discussion / Questions

- None.
