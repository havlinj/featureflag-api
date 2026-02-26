---
# Progress tracker – Feature Flag & Experiment Management API
# Aligned with .cursor/rules/development_workflow.mdc
---

# 🏗️ Progress Tracker – Feature Flag API

**Last updated**: 2026-02-26  
**Overall progress**: █████░░░░░ 50% (Phase 1 in progress)  
**Status**: APPROVED (Phase 1 – DB, repository, and flags service logic done; auth & E2E pending)  
**Next step**: Authentication middleware (JWT), logging middleware; then 2–3 E2E integration tests (API → service → real DB)  
**Blockers**: None

## 📋 Milestones (per development_workflow.mdc)

| Phase | Status | Progress | Key Deliverables |
|-------|--------|----------|------------------|
| Phase 1: Feature Flags Core | 🔄 In Progress | ~50% | Flags API, DB, repos, auth/logging middleware, integration tests |
| Phase 2: Experiments Integration | ⏳ Planned | 0% | Experiments module, schema, resolvers, assignments |
| Phase 3: Audit Logging | ⏳ Planned | 0% | Audit service, audit_logs table, hooks |

## 🔧 Phase 1 – Current state

- [x] HTTPS server + GraphQL (gqlgen) transport
- [x] Minimal DB schema: `feature_flags`, `flag_rules` (internal/db, EnsureSchema)
- [x] DB init test (testcontainers, naked INSERT/SELECT)
- [x] flags.Store interface; PostgresStore (Create, GetByKeyAndEnvironment, Update, GetRulesByFlagID) with unit tests
- [x] flags.Service + NewService(store); resolvers call service (store injectable; mock in testutil)
- [x] Integration test: GraphQL over HTTPS (createFlag, updateFlag, evaluateFlag) with mock store
- [x] flags.Service.CreateFlag / UpdateFlag / EvaluateFlag **business logic** (uniqueness, Update by key+env, EvaluateFlag with percentage rollout; default env "dev")
- [ ] Authentication middleware (JWT)
- [ ] Logging middleware
- [ ] 2–3 end-to-end integration tests (API → service → real DB)

## 📈 Metrics

- Test coverage: unit tests for db (init), flags.Service (CreateFlag, UpdateFlag, EvaluateFlag – all use cases, mock store), flags.PostgresStore (all methods + error cases); integration test for HTTPS+GraphQL with mock.
- Tests: internal/db (1), internal/flags (service: 20 unit tests; postgres: 9, build tag `integration`), test/integration (1 with tag). Default `go test ./...` runs fast (no Postgres container).

## 📝 Changelog

**2026-02-26**: Phase 1 progress. Database layer: internal/db (Open, EnsureSchema, schema for feature_flags + flag_rules). PostgresStore implemented and tested (Create, GetByKeyAndEnvironment, Update, GetRulesByFlagID; happy path + ErrDuplicateKey, ErrNotFound, invalid UUID). Refactored db_test into arrange/insert/assert helpers; split PostgresStore tests into separate test functions. NewApp(tlsConfig, flagsStore) now accepts Store (production: NewPostgresStore(db.Conn()); tests: MockFlagsStore). Service layer and resolvers wired to Store; business logic in flags.Service still unimplemented (panic). Progress doc updated per development_workflow.mdc.

**2026-02-26 (session 2)**: flags.Service business logic implemented: CreateFlag (uniqueness check, Create), UpdateFlag (GetByKey+default env "dev", Update), EvaluateFlag (userID validation, percentage rule with deterministic hash, attribute rule fallback, ErrInvalidRule). Added internal/flags/service_test.go: 20 unit tests (mock store, arrange/act/assert, all use cases). PostgresStore tests tagged `//go:build integration` so default `go test` is fast (~4 ms for flags). Reformatted flags and postgres tests to strict arrange / act / assert with blank lines around act. Updated .cursor/rules/testing_style.mdc.

*(Earlier iterations: HTTPS server, GraphQL resolvers, testutil GraphQL client + TLS, flags.Store interface + MockFlagsStore in testutil with queue-of-results pattern.)*

## 🤝 Discussion / Questions

- None.
