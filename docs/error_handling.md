# Error Handling Taxonomy

This document defines how to model, propagate, and test errors in this codebase.

The goal is to keep client behavior predictable while preserving strong diagnostics for operators and developers.

## Error Categories

Use two categories only:

- Domain errors (business errors): expected outcomes caused by business rules or invalid domain inputs.
- Operation errors (technical errors): unexpected failures in an operation step (DB, transaction, I/O, query/scan, infrastructure).

## Domain Errors

Domain errors communicate "the request is valid, but business rules reject or cannot fulfill it."

Properties:

- Expected and stable from API/use-case perspective.
- Usually map to client-meaningful outcomes.
- Should be asserted by type (`errors.As`) and fields in tests.
- Should not be wrapped into `OperationError`.

Examples in this project:

- Users:
  - `DuplicateEmailError`
  - `NotFoundError`
  - `InvalidCredentialsError`
- Flags:
  - `DuplicateKeyError`
  - `NotFoundError`
  - `InvalidRuleError`
  - `RulesStrategyMismatchError`
  - `InvalidUserIDError`
- Experiments:
  - `DuplicateExperimentError`
  - `ExperimentNotFoundError`
  - `InvalidWeightsError`
  - `VariantNotFoundError`
  - `InvalidUserIDError`

Typical scenarios:

- Creating a user with an existing email -> `DuplicateEmailError`.
- Logging in with wrong password or unknown account -> `InvalidCredentialsError`.
- Creating an experiment with invalid variant weights -> `InvalidWeightsError`.
- Updating a non-existing flag -> `NotFoundError`.

## Operation Errors

Operation errors communicate "business rule is not the issue, but a concrete operation step failed."

Properties:

- Unexpected from business perspective.
- Must preserve structured context for diagnostics.
- Must wrap the root cause (`Cause`) and support `errors.Is` / `errors.As`.
- Should include a stable operation identifier (`Op`) and relevant contextual fields.

Current shape:

- Package-specific `OperationError` structs with:
  - `Op`
  - domain context fields (`Key`, `ID`, `Environment`, `ExperimentID`, `UserID`, etc.)
  - `Cause error`

Typical scenarios:

- DB query fails while reading variants -> `experiments.OperationError`.
- Transaction commit fails during flag rule replace -> `flags.OperationError`.
- User update store call fails -> `users.OperationError`.

## Wrapping Rules

Follow these rules consistently:

- Return domain errors directly when the failure is a business condition.
- Return `OperationError` when a technical operation step fails.
- Preserve cause chain (`Unwrap`) so `errors.Is(err, cause)` keeps working.
- Do not convert domain errors into generic internal messages inside service/repository layers.
- Keep external sanitization/mapping in transport boundary (GraphQL presenter).

## Testing Rules

### For Domain Errors

- Assert by concrete type and context fields.
- Avoid asserting only raw string output for primary behavior.

Pattern:

```go
var e *users.DuplicateEmailError
if !errors.As(err, &e) {
	t.Fatalf("expected *users.DuplicateEmailError, got %T", err)
}
if e.Email != "a@b.com" {
	t.Fatalf("unexpected email: %q", e.Email)
}
```

### For Operation Errors

- Assert both:
  - root cause wrapping (`errors.Is`)
  - structured context (`errors.As` -> `Op` + fields)

Pattern:

```go
if !errors.Is(err, wantErr) {
	t.Fatalf("expected wrapped cause %v, got %v", wantErr, err)
}
var opErr *flags.OperationError
if !errors.As(err, &opErr) {
	t.Fatalf("expected *flags.OperationError, got %T", err)
}
if opErr.Op != "flags.service.update_flag.store_update" {
	t.Fatalf("unexpected op: %q", opErr.Op)
}
if opErr.FlagID != "f1" || opErr.Key != "checkout" || opErr.Environment != "prod" {
	t.Fatalf("unexpected context fields: %+v", opErr)
}
```

### For `Error()` Determinism

- Keep targeted tests that assert deterministic `Error()` formatting for custom error structs.
- Prefer 2 scenarios in one test (fully populated + partial fields) to document formatting rules clearly.
- These tests are complementary to typed assertions; they do not replace them.

## What To Avoid

- Mixing domain and operation semantics in one returned error type.
- Relying only on `err.Error()` string matching in service tests when typed assertions are possible.
- Losing cause chain by creating a fresh error without `Unwrap`.

## Short Decision Table

- Business rule violation or domain validation failure -> domain error.
- Technical/store/DB/tx/network step failure -> `OperationError`.
- External API response shaping -> transport boundary (error presenter/middleware), not domain/service layer.
