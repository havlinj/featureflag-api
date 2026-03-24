---
# Progress tracker – Feature Flag & Experiment Management API
# Aligned with .cursor/rules/development_workflow.mdc
---

# 🏗️ Progress Tracker – Feature Flag API

**Last updated**: 2026-03-23  
**Overall progress**: ██████████░ ~**95%** (Phase 1–4 complete; **Phase 5 in progress** — hardening & coverage policy **not finished**; coverage gates **not green**)  
**Status**: Phase 1–4 **complete and reviewed (APPROVED)**; Phase 5 **active** (hardening + **coverage measurement / policy tuning** started; targets not met)  
**Next step**: Continue Phase 5 — raise coverage toward **90%** global and clear **function-floor** violations; use **`coverage.html`** / `-func` to guide tests (see `docs/session_7.md`)  
**Blockers**: None

## 📋 Milestones (per development_workflow.mdc)

| Phase | Status | Progress | Key Deliverables |
|-------|--------|----------|------------------|
| Phase 1: Feature Flags & Users Core | ✅ Complete | 100% | Flags + Users API, rollout strategies (percentage + attribute, one per flag), rules CRUD, EvaluateFlag with context, DeleteFlag, DB, auth (JWT), logging, integration tests |
| Phase 2: Local Test Scripts, Binary Smoke & CI | ✅ Complete (reviewed) | 100% | Bash scripts (check, unit, integration, build, test_all_quick, test_all_full, test_binary_smoke); scripts/integration/; internal/config; GitHub Actions CI (push/PR to master) |
| Phase 3: Experiments Integration | ✅ Complete (reviewed) | 100% | Experiments service, GraphQL schema + resolvers (createExperiment, experiment, getAssignment), DB (experiments, experiment_variants, experiment_assignments), deterministic assignment, integration + resolver unit tests |
| Phase 4: Audit Logging | ✅ Complete (reviewed) | 100% | Audit module, audit_logs table, atomic writes (fail-closed), audit read API, integration + resolver tests |
| Phase 5: Production Hardening & Quality Reiteration | ⏳ In progress | ~25% | Hardening checklist partial; **coverage script + multi-tier gates + `coverage_filter` landed**; **global ~72% vs 90% target**, function gate still failing on many core functions |

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

## 🔧 Phase 3 – Current state

- [x] internal/experiments: Store interface, PostgresStore (CreateExperiment, GetExperimentByKeyAndEnvironment, GetExperimentByID, CreateVariant, GetVariantsByExperimentID, GetAssignment, UpsertAssignment)
- [x] internal/experiments: Service (CreateExperiment, GetExperiment, GetAssignment), deterministic assignment (hash userID+experimentID → bucket, weights sum 100), structured errors (InvalidWeightsError, ExperimentNotFoundError, DuplicateExperimentError, VariantNotFoundError, InvalidUserIDError)
- [x] internal/experiments/mock: queue-based Store for unit tests
- [x] DB schema: experiments, experiment_variants, experiment_assignments (migrations; TruncateAll in testutil includes them)
- [x] GraphQL schema: experiments.graphqls (Experiment, ExperimentVariant, CreateExperimentInput, ExperimentVariantInput); Query (experiment, getAssignment), Mutation (createExperiment)
- [x] Resolvers: CreateExperiment (auth admin/developer, nil guard), Experiment (auth admin/developer/viewer, ExperimentNotFoundError → null), GetAssignment (auth, nil guard); ExperimentsService interface for DI and tests
- [x] Wiring: NewApp(..., experimentsStore), cmd/main.go and test/integration/util.go create experiments PostgresStore and pass to app
- [x] Integration tests: test/integration/integration_experiments_test.go (createExperiment, experiment query found/not found, getAssignment + determinism, userByEmail for userID)
- [x] Unit tests: experiments service (all paths), experiments PostgresStore (integration tag), experiments resolvers (auth, nil service, not-found→null, delegation, service errors)

## Phase 3 – REVIEW (final)

**Status: APPROVED**

1. **Architecture**  
   Layering is respected: transport/graphql resolvers only call auth.RequireRole and ExperimentsService; no business logic in resolvers. internal/experiments has no dependency on transport or graph. App and main only wire stores and construct services. ExperimentsService interface keeps resolver testable without concrete *experiments.Service.

2. **Simplicity**  
   Resolvers are thin (auth check, nil guard, delegate; Experiment maps ExperimentNotFoundError to (nil, nil) for GraphQL null). No over-engineering; interface is minimal (three methods). Wiring in app/main/util is consistent with flags and users.

3. **Readability**  
   Clear naming (CreateExperiment, GetExperiment, GetAssignment; ExperimentsService). Experiment “not found → null” behaviour is explicit in code and covered by tests. Checkboxes in Phase 3 section match deliverables.

4. **Test quality**  
   Service layer: unit tests with mock store cover all return paths (weights validation, duplicate, not found, assignment determinism). Resolver unit tests cover auth (no claims, viewer forbidden on create), nil Experiments, not-found→null, delegation success, and service error pass-through. Integration test covers full API flow (create, get, getAssignment, determinism). Deterministic assignment is asserted in service and integration tests.

5. **Alignment with plan**  
   All Phase 3 deliverables from development_workflow.mdc are present: experiments service layer (experiments, variants, user assignments), updated GraphQL schema and resolvers, DB tables (experiments, experiment_variants, experiment_assignments), integration tests API→service→DB, deterministic rollout and assignment behaviour. No scope creep.

**Conclusion:** Phase 3 implementation matches the approved design. Ready to close; next step is Phase 4 (Audit logging).

## 🔧 Phase 4 – Current state

- [x] DB schema updated with `audit_logs` table (`id`, `entity`, `entity_id`, `action`, `actor_id`, `created_at`)
- [x] Truncation helpers updated to include `audit_logs` for integration tests
- [x] New `internal/audit` module: `Entry`, `Store`, `PostgresStore`, thin `Service`
- [x] Critical mutations are audited with fail-closed policy (flags/users/experiments scope agreed in phase planning)
- [x] Atomic business + audit writes in one transaction for audited operations
- [x] Context actor propagation from GraphQL resolver layer (`auth.WithActorID`) into service layer
- [x] GraphQL audit read API delivered (`auditLog`, `auditLogs`) with admin RBAC
- [x] Pagination validation in API: negative `offset` returns explicit error
- [x] Resolver and service internals refactored to reduce duplication and improve encapsulation consistency
- [x] Audit metadata constants and shared audit tx helper introduced for maintainability
- [x] Unit, resolver, repository integration, and GraphQL integration tests updated and passing
- [x] Go runtime/toolchain baseline updated to 1.25 and gqlgen regenerated/validated

## Phase 4 – REVIEW (final)

**Status: APPROVED**

1. **Architecture**  
   Layering is preserved: GraphQL resolvers remain thin adapters, business logic stays in services, and persistence stays in repository/store implementations. Audit write orchestration is centralized without leaking transport concerns into domain logic.

2. **Simplicity**  
   The solution avoids heavy abstractions; shared helpers were introduced only where repetition created real maintenance risk (audit tx/write orchestration and resolver prechecks).

3. **Readability**  
   Naming and flow are clearer after encapsulation of dependencies and removal of duplicated inline audit wiring. Test fixtures now use constructor-based setup instead of direct dependency field access.

4. **Test quality**  
   Coverage includes unit and integration scenarios for audit writes, rollback/error branches, pagination validation, resolver authorization behavior, and end-to-end GraphQL audit behavior. Full standard and integration suites were executed successfully.

5. **Alignment with plan**  
   Phase 4 deliverables from `development_workflow.mdc` were delivered: audit module + table, service hooks for critical operations, audit read API, and validation through integration tests. Post-phase technical hardening (encapsulation and Go/gqlgen upgrade) was completed with passing tests.

**Conclusion:** Phase 4 implementation and follow-up hardening are complete and approved.

## 📈 Metrics

- **Enforced coverage policy (Phase 5, local script)**: `./scripts/coverage/test_coverage.sh` — global target **90%**, per-file floors for core roles, function floor **50%** on core files; violations summary + optional `coverage_filter`. As of session 7, **global ~72–73%** (gate **FAIL**); per-file gate **PASS**; function gate **FAIL** (~17 functions remain after filters). HTML: `go tool cover -html=coverage.out -o coverage.html`.
- Test coverage: unit tests for db, flags.Service (incl. all return paths), users.Service, experiments.Service (CreateExperiment, GetExperiment, GetAssignment, weight validation, duplicate, not found, assignment determinism), auth, middleware; flags, users, and experiments PostgresStore (build tag `integration`); **experiments resolvers** (auth, nil service, not-found→null, delegation, service errors); integration tests for HTTPS+GraphQL against **real Postgres** (testcontainers), including test/integration/integration_experiments_test.go (createExperiment, experiment query, getAssignment, determinism). Mock errors in tests use descriptive labels.
- Tests: internal/db, internal/flags, internal/users, internal/experiments (service_test, postgres_test, errors_test), internal/auth, transport/graphql (experiments_resolvers_test), transport/graphql/middleware, test/integration (flags, users, auth, experiments; tag `integration`). Default `go test ./...` skips integration; run with `-tags=integration` for full E2E.
- Code style: gofmt run before task completion (see .cursor/rules/coding_style.mdc).

## 🔁 Phase 5 – Candidate scope (from final review)

### Coverage & CI gates (started — **not done**)

- [x] **`scripts/coverage/test_coverage.sh`**: unit + integration coverage over production `COVERAGE_PKGS`; **global**, **per-file**, and **function-level** gates (configurable constants at top of script)
- [x] **`scripts/coverage/coverage_filter/`**: post-process function-gate violations (skip `graph/**/*.go`; thin-delegate heuristic for trivial `return other(...)` wrappers)
- [ ] **Meet global gate** (target **90%**; current runs ~**72–73%**, small run-to-run jitter ~0.2pp is normal)
- [ ] **Meet function-level gate** (core `service` / `postgres` / `*resolvers` functions ≥ **50%**; many mappers, audit paths, flag evaluation branches still cold)
- [ ] Optional: wire `test_coverage.sh` into **CI** (or `test_all_full.sh`) once gates pass or policy is explicitly relaxed
- [ ] **Race detector** in CI (still open from original Phase 5 list)

### Other Phase 5 items (largely untouched)

- [ ] Fix multi-environment correctness gap in flags update/evaluate flow (`dev` hardcoding)
- [ ] Reduce transport-model coupling in domain services
- [ ] Ensure experiment write atomicity independent of audit wiring
- [ ] Strengthen DB integrity constraints for experiment assignments
- [ ] Improve security defaults (TLS/DSN posture) and auth error shaping
- [ ] Harden JWT validation policy and login abuse controls
- [ ] Add server/runtime hardening (timeouts/limits) and GraphQL operation safeguards
- [ ] Remove flaky fixed sleeps in integration/bootstrap scripts via readiness checks

## 📝 Changelog

**2026-03-23 (session 7)**: Documented **Phase 5 coverage workstream** in `.cursor/rules/development_workflow.mdc` (subsection *Phase 5 – Coverage measurement & tuning*): `scripts/coverage/test_coverage.sh` gates, `scripts/coverage/coverage_filter/`, workflow with `coverage.html`, exit criteria. **progress.md**: Phase 5 status → *in progress*; split checkboxes (coverage started vs rest of hardening); milestones table updated; next step points to raising coverage and using HTML report. Added **`docs/session_7.md`** as resume handoff (filters, honest impact, suggested next steps). Note: global coverage ~72–73% vs 90% target; function gate still lists ~17 functions after filters; thin-delegate filter removes ~1 wrapper only; `generated=0` for `graph/` until those files are in the measured set.

**2026-03-23**: Introduced **Phase 5 – Production Hardening & Quality Reiteration** as a follow-up iteration after fulfilling original phase goals. Scope includes architecture hardening, security and runtime resilience improvements, data integrity constraints, and stronger CI/testing quality gates. This phase explicitly allows justified deviations from original assumptions when they materially improve safety and production quality.

**2026-03-23**: Phase 4 (Audit Logging) closed after final REVIEW (APPROVED). Delivered audit module and persistence, `audit_logs` schema integration, fail-closed atomic business+audit writes for critical operations, GraphQL audit read API with admin RBAC, explicit negative-offset validation, and expanded test coverage (unit/resolver/repository integration/E2E integration). Follow-up refactor pass completed: shared audit tx helper, typed audit metadata constants, resolver precheck cleanup, dependency encapsulation across services/resolvers/app wiring, and resolver test fixtures moved to constructor-based setup. Runtime stack upgraded to Go 1.25 and gqlgen regenerated; `go test ./...` and `go test -tags=integration ./...` pass.

**2026-03-15**: Phase 3 (Experiments Integration) closed after final REVIEW (APPROVED). Delivered: experiments service layer (Store, PostgresStore, Service with CreateExperiment, GetExperiment, GetAssignment; deterministic assignment; structured errors); GraphQL schema (experiments.graphqls) and resolvers (CreateExperiment, experiment, getAssignment) with auth (admin/developer for create, admin/developer/viewer for queries) and ExperimentNotFoundError→null; wiring in app (NewApp accepts experimentsStore), cmd/main.go, test/integration/util.go; integration tests (integration_experiments_test.go) and full resolver unit tests (experiments_resolvers_test.go). progress.md updated: Phase 3 checkboxes, REVIEW section, milestones table, metrics; next step Phase 4 (Audit logging).

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
