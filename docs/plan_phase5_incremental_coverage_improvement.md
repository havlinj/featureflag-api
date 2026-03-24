# PLAN: Phase 5 Incremental Coverage Improvement

**Status**: NEEDS_APPROVAL  
**Date**: 2026-03-24  
**Scope**: Continue iterative coverage improvement by adding high-value tests that improve reliability and reduce blind spots.

---

## 1. Problem Definition

**What is being changed?**  
We will add a focused set of tests for currently under-covered fail/error/edge branches in service and repository layers.

**Why is the change needed?**  
Phase 5 coverage gates are not green. Current coverage analysis shows repeated blind spots in:
- guard/error branches (`nil`, `err != nil`),
- transaction unhappy paths (`BeginTx/Commit/Rollback`),
- not-found and typed error mapping paths.

The goal is to increase confidence in production behavior first, and coverage as a consequence.

---

## 2. Scope

**In scope:**
- Service-layer behavior contracts for flags/users/experiments.
- Transactional unhappy paths with real meaning (rollback/commit failure handling).
- Repository edge paths with realistic DB semantics.
- Resolver-specific behavior only where transport behavior differs from service behavior.
- A limited, prioritized test set (12 scenarios) with explicit Arrange/Act/Assert intent.

**Out of scope:**
- Mechanical tests for pure mapping/getter code with no behavior risk.
- Artificial mocks only to touch lines that cannot happen in real runtime.
- Bulk duplication of equivalent nil checks in multiple packages.
- Refactoring production code solely to simplify coverage.

---

## 3. Proposed Strategy

### 3.1 Principles (must hold for every added test)

1. Test a user-visible or ops-visible behavior contract.
2. Prefer one representative test per behavior family over many near-duplicates.
3. Favor integration tests when transactionality/persistence semantics matter.
4. Keep resolver unit tests only for resolver-specific behavior (auth/null shaping).
5. Assert typed errors (`errors.As`) and contextual fields where applicable.

### 3.2 Prioritization Model

Scoring used to prioritize scenarios:
- **Risk**: data integrity, auth/security, or API contract break risk.
- **Reach**: how often this path pattern repeats in uncovered code.
- **Yield**: expected coverage gain per test.
- **Cost**: implementation and maintenance complexity.

Priority order:
1. Service fail-path contracts
2. Transaction unhappy paths
3. Repository edge semantics
4. Resolver-specific transport behavior

---

## 4. Concrete Test Plan (12 scenarios)

## Priority A - Service contracts (highest ROI)

1. **`flags.Service.EvaluateFlagInEnvironment` returns `OperationError` when `GetByKeyAndEnvironment` fails**  
   - Arrange: mock store returns DB error.  
   - Act: call evaluate.  
   - Assert: `errors.As` to `*flags.OperationError`, assert `Op`, `Key`, `Environment`.

2. **`flags.Service.EvaluateFlagInEnvironment` returns `(false, nil)` for disabled or missing flag**  
   - Arrange: case table: `flag=nil`; `flag.Enabled=false`.  
   - Act: call evaluate.  
   - Assert: result `false`, error `nil`.

3. **`flags.Service.applyRulesUpdateWithStore` differentiates `rules=nil` vs `rules=[]`**  
   - Arrange: existing flag with strategy/rules.  
   - Act: update once with `nil`, once with empty slice.  
   - Assert: `nil` means no rules change; empty slice clears rules and sets strategy `none`.

4. **`users.Service.Login` always returns `InvalidCredentialsError` for auth failures**  
   - Arrange: table cases: user not found, wrong password, password hash missing.  
   - Act: login.  
   - Assert: always same typed domain error (no auth info leakage).

5. **`users.Service.deleteUserWithStore` maps `NotFoundError` to `(false, nil)`**  
   - Arrange: store delete returns `*users.NotFoundError`.  
   - Act: delete user.  
   - Assert: no error, deleted=false (idempotent delete contract).

6. **`experiments.Service.GetAssignment` returns typed errors for invalid input and missing variants**  
   - Arrange: case table for empty `userID` and zero variants.  
   - Act: get assignment.  
   - Assert: `*experiments.InvalidUserIDError` / `*experiments.VariantNotFoundError`.

## Priority B - Transaction unhappy paths (critical integrity)

7. **`experiments.Service.CreateExperiment` rolls back on audit write failure**  
   - Arrange: audit-enabled service; audit store create fails after experiment write path starts.  
   - Act: create experiment.  
   - Assert: returns error; experiment is not persisted.

8. **`flags.Service.UpdateFlag` rolls back on audit write failure**  
   - Arrange: audit-enabled service with tx-aware store; audit create fails.  
   - Act: update flag.  
   - Assert: returns error; original flag state remains unchanged in DB.

9. **`users.Service.DeleteUser` does not write audit and does not commit when entity not deleted**  
   - Arrange: delete returns not found path (`deleted=false`).  
   - Act: delete user in audit mode.  
   - Assert: returns `(false,nil)` and audit store create was not called.

## Priority C - Repository edge semantics

10. **`flags.PostgresStore.ReplaceRulesByFlagID` transactional branch rolls back on insert failure**  
    - Arrange: standalone store (`begin!=nil`), replacement includes invalid value causing insert error.  
    - Act: replace rules.  
    - Assert: operation fails and previously stored rule set remains unchanged.

11. **`audit.PostgresStore.List` enforces pagination bounds and offset validation**  
    - Arrange: list with `limit<=0`, `limit>max`, `offset<0`.  
    - Act: call list.  
    - Assert: default/capped limits; negative offset returns expected error.

## Priority D - Resolver-specific behavior only

12. **`experiments` resolver behavior: service not configured and not-found -> null**  
    - Arrange: resolver with nil experiments service; resolver with service returning `ExperimentNotFoundError`.  
    - Act: call `CreateExperiment`/`Experiment`.  
    - Assert: explicit config error for nil service; null response for not-found.

---

## 5. Affected Files (planned)

- `internal/flags/service_test.go`
- `internal/users/service_test.go`
- `internal/experiments/service_test.go`
- `internal/flags/postgres_test.go` (integration-tag branch)
- `internal/audit/postgres_test.go` (integration-tag branch)
- `transport/graphql/experiments_resolvers_test.go`
- `docs/progress.md` (after implementation/review only)

---

## 6. Execution Sequence

1. Implement Priority A tests (1-6), run unit tests.
2. Implement Priority B tests (7-9), run targeted unit/integration tests.
3. Implement Priority C tests (10-11), run integration-tag tests.
4. Implement Priority D test (12), run resolver tests.
5. Run `./scripts/coverage/test_coverage.sh` and evaluate global/function gates.
6. Update `docs/progress.md` with measured impact and remaining hotspots.

---

## 7. Follow-up Refactoring Option (post-test iteration)

After Priority A and B are completed and re-measured, we may run a small design spike focused on one hotspot flow (for example tx+audit orchestration in one service) to reduce branch complexity and improve testability.

Rules for this follow-up:
- It is not part of the current test implementation scope.
- It should be proposed only after coverage evidence confirms persistent hotspots.
- It should be incremental (one flow first), with explicit before/after impact on complexity and tests.
- It must preserve architecture boundaries and avoid abstraction-only refactors.

---

## 8. Test Quality Rules for this plan

- Use table-driven tests where it reduces duplication.
- Keep Arrange / Act / Assert separation explicit.
- Prefer behavior assertions over internal-call-count assertions.
- Use typed error checks (`errors.As`) with field assertions for custom errors.
- No assertions on fragile string fragments when typed checks are possible.

---

## 9. Risks and Mitigations

- **Risk:** Overfitting to implementation details.  
  **Mitigation:** assert outcomes/contracts, not call internals.

- **Risk:** Flaky transaction tests.  
  **Mitigation:** use deterministic setup/teardown and DB assertions, no sleeps.

- **Risk:** Coverage rises but business confidence does not.  
  **Mitigation:** every test must answer a production-relevant failure mode.

---

## 10. Success Criteria

- Added tests are reviewable as behavior-focused, not synthetic.
- Priority scenarios A and B are fully covered.
- Coverage gates improve measurably without policy relaxation.
- No regressions in existing unit/integration suites.

---

*End of plan. Ready for APPROVAL before implementation.*
