# Plan: One rollout strategy per flag + API changes

**Status:** NEEDS_APPROVAL  
**Scope:** Phase 1 – enforce single strategy per flag and plan API for rules + evaluation context.

---

## 1. Product rule (agreed)

- Each feature flag has **exactly one** rollout strategy: either **percentage** or **attribute** (or none = no rules).
- If percentage is chosen, **only** percentage rules apply; attributes are ignored.
- If attribute is chosen, **only** attribute rules apply; percentages are ignored.
- No mixing: no "30% of users and then add more by attribute".

---

## 2. Data model changes

### 2.1 Add `rollout_strategy` to `feature_flags`

- **Column:** `rollout_strategy TEXT NOT NULL DEFAULT 'none' CHECK (rollout_strategy IN ('none', 'percentage', 'attribute'))`.
- **Meaning:**
  - `none` – no rollout rules (flag enabled = everyone sees it when enabled).
  - `percentage` – only percentage rules are stored and evaluated.
  - `attribute` – only attribute rules are stored and evaluated.
- **Migration:** Add column to existing schema (e.g. in `internal/db/schema.go` or new migration step); existing flags without rules become `none`; existing flags with rules need a one-time backfill (e.g. set from first rule's type, or default `percentage` if any percentage rule exists else `attribute`).

### 2.2 Validation rules (backend)

Validation runs whenever the backend creates, updates, or replaces rules for a flag (e.g. in CreateFlag with `rules`, UpdateFlag with `rules`, or future “set flag rules” / “add rule” operations). The goal is: **the flag’s `rollout_strategy` and the types of its rules must always stay in sync.**

#### When adding or replacing rules

1. **Read current state**  
   Load the flag and its current `rollout_strategy`. Optionally load existing rules (if any) for this flag.

2. **Decide allowed rule type**  
   - If the flag has **no rules yet** (`rollout_strategy = 'none'` or no rows in `flag_rules`):  
     The **first** rule (or the set of rules being written) determines the strategy. All rules in this request must be of the **same** type (all percentage or all attribute). After saving, set the flag’s `rollout_strategy` to that type (e.g. `'percentage'` or `'attribute'`).  
   - If the flag **already has** `rollout_strategy = 'percentage'`:  
     Only **percentage** rules are allowed. Reject the request if any rule has type `attribute`. When saving, only store percentage rules; optionally delete any existing attribute rules if we are doing a full replace.  
   - If the flag **already has** `rollout_strategy = 'attribute'`:  
     Only **attribute** rules are allowed. Reject the request if any rule has type `percentage`. When saving, only store attribute rules; optionally delete any existing percentage rules if we are doing a full replace.

3. **Validate the incoming rules**  
   - If the request contains **mixed** rule types (e.g. one percentage and one attribute), **reject** the request with a clear error (e.g. “all rules must use the same strategy: percentage or attribute”).  
   - If the request’s rule type does not match the flag’s current `rollout_strategy` (when it is already set to percentage or attribute), **reject** the request (e.g. “flag uses percentage strategy; attribute rules are not allowed”).

4. **Persist**  
   Save the rules and, when the flag had no strategy before, update the flag’s `rollout_strategy` to the chosen type.

#### When removing rules

- When the last rule for a flag is removed (e.g. client sends “replace rules with empty list” or “delete rule” and no rules remain), set the flag’s `rollout_strategy` back to `'none'`. That way the flag is again “no rules, strategy none” and the next added rule can set either percentage or attribute.

**Ways to remove rules (so the developer can do it easily):**

1. **Replace list (recommended for Phase 1)**  
   `UpdateFlag` accepts optional `rules: [RuleInput!]`. If the client sends `rules: []` (empty array), the backend replaces all rules for that flag with the empty list: delete all rows in `flag_rules` for the flag and set `rollout_strategy = 'none'`. So "remove all rules" = call updateFlag with the same flag key/enabled/description and `rules: []`. To remove only some rules, the client sends the new desired list (e.g. current rules minus one); the backend replaces the full set. No dedicated "delete rule" API is required for Phase 1.

2. **Explicit delete (optional, later)**  
   A mutation such as `deleteFlagRule(flagKey: String!, ruleId: ID!): FeatureFlag!` or `removeFlagRules(flagKey: String!, ruleIds: [ID!]!): FeatureFlag!` would remove specific rules by ID. After the last rule is removed, set `rollout_strategy = 'none'`. This is convenient when the client has rule IDs (e.g. from a previous query that returns `FeatureFlag.rules`) and wants to remove one rule without resending the full list. Can be added in Phase 1 or later.

**Recommendation:** Implement (1) in Phase 1: "replace rules" with an empty or smaller list. Option (2) can follow if we expose `rules` on `FeatureFlag` and clients need to delete by ID.

#### Summary table

| Flag’s current `rollout_strategy` | Incoming rules                    | Action                                                                 |
|----------------------------------|------------------------------------|------------------------------------------------------------------------|
| `none`                           | Empty                              | OK; leave strategy `none`.                                             |
| `none`                           | All same type (e.g. all percentage) | OK; save rules and set strategy to that type (e.g. `percentage`).      |
| `none`                           | Mixed types                        | **Reject**: all rules must be same type.                                |
| `percentage`                     | Only percentage rules              | OK; save (replace or add as designed).                                  |
| `percentage`                     | Any attribute rule                 | **Reject**: flag uses percentage strategy.                             |
| `attribute`                      | Only attribute rules               | OK; save (replace or add as designed).                                  |
| `attribute`                      | Any percentage rule                | **Reject**: flag uses attribute strategy.                             |

#### How the developer is informed when rules are rejected

When validation rejects the request (mixed rule types or rule type does not match the flag’s strategy), the backend must return a **clear, machine- and human-readable error** so the developer knows that the rules were not applied and why.

1. **Service layer**  
   Introduce a dedicated domain error, e.g. `ErrRulesStrategyMismatch` (or `ErrInvalidRulesForStrategy`), so callers can detect this case. The error message should be descriptive, for example:
   - *"all rules must use the same strategy: percentage or attribute"* (when the request contains mixed types),
   - *"flag uses percentage strategy; only percentage rules are allowed"* (when the flag already has percentage and the request sends attribute rules),
   - *"flag uses attribute strategy; only attribute rules are allowed"* (when the flag already has attribute and the request sends percentage rules).

2. **GraphQL response**  
   Mutations (e.g. `createFlag`, `updateFlag`) that accept rules return the same shape as today (e.g. `FeatureFlag` or error). On validation failure:
   - The resolver does **not** return data for the mutation payload (or returns `null` for the field).
   - The GraphQL response includes an entry in the top-level `errors` array. Each error has a `message` with the text above. Optionally, use the `extensions` field (e.g. `extensions.code = "RULES_STRATEGY_MISMATCH"`) so clients can handle this case by code instead of parsing the message.

3. **HTTP status (optional)**  
   If the transport layer maps domain errors to HTTP status codes, use **400 Bad Request** (or **422 Unprocessable Entity**) for strategy-mismatch errors, so the client can distinguish them from 401/403 and 500.

4. **Summary**  
   The developer finds out that the rules were rejected by:
   - Inspecting the GraphQL `errors` array in the response,
   - Reading the `message` (and optionally `extensions.code`),
   - Optionally checking HTTP status 400/422.

No silent failure: if validation fails, the mutation is treated as failed and no rules are persisted; the response must contain an error entry with a clear reason.

---

## 3. API changes

### 3.1 GraphQL schema

**A) Feature flag type and inputs**

- Add enum: `RolloutStrategy = NONE | PERCENTAGE | ATTRIBUTE`.
- Extend `FeatureFlag`: `rolloutStrategy: RolloutStrategy!`.
- Extend `CreateFlagInput`: optional `rolloutStrategy: RolloutStrategy` (default NONE). If client sends rules (see below), strategy can be inferred or must match.
- Extend `UpdateFlagInput`: optional `rolloutStrategy: RolloutStrategy` when updating rules (so client can switch strategy only when no rules, or we replace rules and set strategy).

**B) Rules on the flag**

- **Option B1 – Rules as part of flag payload (simpler for Phase 1):**
  - `RuleInput`: `{ type: RolloutRuleType!, value: String! }` where `RolloutRuleType = PERCENTAGE | ATTRIBUTE`, and `value` is e.g. `"30"` or JSON for attribute.
  - `CreateFlagInput`: optional `rules: [RuleInput!]`. If present, all must be same type; that sets `rollout_strategy`. If `rolloutStrategy` also provided, must match.
  - `UpdateFlagInput`: optional `rules: [RuleInput!]`. If provided, replaces all rules for the flag; all must match flag's current `rollout_strategy` (or set new strategy if flag had no rules).
  - Response: `FeatureFlag` can expose `rules: [Rule!]` so client sees current rules (optional for Phase 1, or add in same step).
- **Option B2 – Dedicated rule mutations:**
  - `addFlagRule(flagKey: String!, environment: String!, rule: RuleInput!): FeatureFlag!`
  - `removeFlagRule(flagKey: String!, ruleId: ID!): FeatureFlag!` or `setFlagRules(flagKey: String!, rules: [RuleInput!]!): FeatureFlag!`
  - Same validation: rule type must match flag's `rollout_strategy` (or set strategy on first add).

Recommendation: **B1** for Phase 1 – optional `rules` in create/update keeps one place to define the flag and its strategy + rules; we can add B2 later if needed.

**C) Evaluation query (already agreed – Approach A)**

- Replace current `evaluateFlag(key: String!, userId: ID!): Boolean!` with:
  - `evaluateFlag(key: String!, context: EvaluationContextInput!): Boolean!`
- `EvaluationContextInput`: `{ userId: ID!, email: String }` (and optional fields later).
- Backend: load flag; if `rollout_strategy == percentage`, evaluate only percentage rules with `context.userId`; if `attribute`, evaluate only attribute rules with `context`; if `none`, no rules → return `flag.enabled`.

---

## 4. Evaluation logic (backend)

- Load flag; if disabled or not found → false.
- If `flag.rollout_strategy == 'none'` and no rules → return true (current behaviour).
- If `flag.rollout_strategy == 'percentage'`: load rules, filter to percentage only (or rely on DB having only percentage rules), evaluate with `userId`; first rule that applies wins, or single percentage rule as today.
- If `flag.rollout_strategy == 'attribute'`: load rules, filter to attribute only, evaluate with full context (userId, email, …); first matching rule → true; no match → false.
- Do **not** mix: never evaluate percentage and attribute together for the same flag.

---

## 5. Affected files (overview)

| Layer        | Files to touch |
|-------------|----------------|
| DB          | `internal/db/schema.go` (add `rollout_strategy`); migration/backfill if needed |
| Entity      | `internal/flags/entity.go` (add `RolloutStrategy` to `Flag`) |
| Errors      | `internal/flags/errors.go` (add `ErrRulesStrategyMismatch` or similar for validation rejections) |
| Store        | `internal/flags/store.go` (add `Delete` or `DeleteByKeyAndEnvironment`); `internal/flags/postgres.go` (Create/Update read/write `rollout_strategy`; Delete flag by id/key+env, CASCADE removes rules; add rule replace/CRUD for updateFlag with rules) |
| Service      | `internal/flags/service.go` (CreateFlag/UpdateFlag with rules + validation; EvaluateFlag by strategy; attribute evaluation) |
| GraphQL      | `graph/schema/flags.graphqls` (enum, EvaluationContextInput, extend FeatureFlag and inputs; evaluateFlag with context) |
| Resolvers    | `transport/graphql/flags.resolvers.go` (wire new inputs and context) |
| Generated    | `go run github.com/99designs/gqlgen generate` |

---

## 6. Remove / delete consistency across entities

So that developers can remove data in a predictable way, the API should support deletion (or equivalent) for the main entities. Current state and recommendations:

| Entity   | Remove today? | Recommendation |
|----------|----------------|----------------|
| **Users** | Yes | Already have `deleteUser(id: ID!): Boolean!`. Keep as is. |
| **Flags** | No | Add `deleteFlag(key: String!, environment: String!)` (or `deleteFlag(id: ID!)`) so a flag and all its rules can be removed. Store must delete rules (or rely on DB `ON DELETE CASCADE` for `flag_rules.flag_id`) and then delete the flag. |
| **Rules** | Via replace | No separate "delete rule" required in Phase 1: removing rules is done by updating the flag with a new `rules` list (empty to remove all, or a smaller list to remove some). Optional: add `deleteFlagRule(flagKey, ruleId)` later. |
| **Experiments** (Phase 2) | N/A | When experiments are added, provide a way to delete an experiment (and optionally its assignments/variants) so the API stays consistent. |

**Summary:** Users are already deletable. For Phase 1, add **delete flag** (and ensure rules are removed with it, e.g. via cascade or explicit delete). Rules are removed by **replacing the rules list** in updateFlag (e.g. `rules: []`). Later phases (e.g. experiments) should follow the same idea: each main entity has a clear way to be removed.

---

### 6.1 deleteFlag – detail

**GraphQL**

- Add to `Mutation`: `deleteFlag(key: String!, environment: String!): Boolean!`
- **Arguments:** `key` and `environment` identify the flag (same as create/update; client usually has key + env, not internal ID). Alternative: `deleteFlag(id: ID!)` if the client works with flag IDs; key+environment is consistent with the rest of the flags API.
- **Return:** `Boolean!` – `true` when the flag was found and deleted, so the client can distinguish success. On “flag not found”, return an error (and optionally `false` in a partial response; prefer a single source of truth: error = not found or failure).

**Behaviour**

1. Resolve the flag by `GetByKeyAndEnvironment(ctx, key, environment)`. If not found → return error (e.g. `ErrNotFound`), no delete.
2. Delete the flag row (by ID). **Database:** `flag_rules` already has `REFERENCES feature_flags(id) ON DELETE CASCADE`, so PostgreSQL will delete all rows in `flag_rules` for that `flag_id` when the flag is deleted. No extra application logic to delete rules.
3. Return `true` on success.

**Store layer**

- Add to `flags.Store`: `Delete(ctx context.Context, id string) error` (or `DeleteByKeyAndEnvironment(ctx, key, environment string) error`). Implementation: single `DELETE FROM feature_flags WHERE id = $1` (or `WHERE key = $1 AND environment = $2`). Rely on CASCADE for rules.
- Return `ErrNotFound` when no row was affected (`RowsAffected() == 0`).

**Errors**

- `ErrNotFound` when the flag does not exist (or invalid key/environment). Resolver returns this as a GraphQL error (message + optional `extensions.code`), same style as updateFlag.

**Authorization**

- Restrict to roles that may delete flags (e.g. `admin`, `developer`), same as createFlag/updateFlag. Use `auth.RequireRole(ctx, "admin", "developer")` in the resolver before calling the service.

---

### 6.2 Removing rules – detail

**Via UpdateFlag (replace list)**

- `UpdateFlagInput` is extended with optional `rules: [RuleInput!]`. Semantics: if `rules` is present, it **replaces** the entire set of rules for that flag (same as today’s “replace” semantics for the list).
  - `rules: []` or `rules: null` (if we treat null as “do not change rules”) – for “remove all” we need a defined behaviour: e.g. **empty array `[]` = replace with no rules** (delete all rules for the flag, set `rollout_strategy = 'none'`). If `rules` is omitted, do not change existing rules.
- Backend: load flag, validate new rules (strategy match, no mixed types). Delete all existing rules for the flag (e.g. `DELETE FROM flag_rules WHERE flag_id = $1`), then insert the new ones (if any). Update `feature_flags.rollout_strategy` (to `'none'` if no rules, or to the type of the new rules).
- **Example:** Flag has two percentage rules. Client sends `updateFlag(input: { key: "my-flag", enabled: true, rules: [{ type: PERCENTAGE, value: "50" }] })` → backend replaces with one rule (effectively “removed” one rule). Client sends `updateFlag(input: { key: "my-flag", enabled: true, rules: [] })` → backend deletes all rules, sets `rollout_strategy = 'none'`.

**Store layer for rules replace**

- Store needs a way to replace rules for a flag: e.g. `ReplaceRulesByFlagID(ctx, flagID string, rules []*Rule) error` that runs in a transaction: delete all rows in `flag_rules` for that `flag_id`, then insert the new rules (if any). Alternatively: `DeleteRulesByFlagID(ctx, flagID) error` + existing or new method to create rules; service calls both. The important point: one logical “replace” so the flag never has mixed rule types.

---

### 6.3 Reference: deleteUser (existing)

- `deleteUser(id: ID!): Boolean!` – returns `true` when the user was deleted, error when not found. No cascade from users to flags/experiments in the current schema; if later we add foreign keys to users (e.g. audit_logs.actor_id), we can add `ON DELETE SET NULL` or restrict delete. For Phase 1, deleteUser stays as is.

---

## 7. Out of scope (for this plan)

- Full attribute rule format and operators (separate small plan or follow-up).
- Audit logging (Phase 3).
- Changing percentage semantics (determinism is correct as-is).

---

## 8. Summary

- **Product:** One strategy per flag (percentage **or** attribute, never both).
- **Model:** `feature_flags.rollout_strategy` = `none` | `percentage` | `attribute`.
- **API:** (1) Extend flag type and create/update with `rolloutStrategy` and optional `rules`; (2) Replace `evaluateFlag(userId)` with `evaluateFlag(context: EvaluationContextInput!)`; (3) Validate that all rules for a flag match its strategy.
- **Evaluation:** Use only the rules for the flag’s strategy; no mixing.
- **Remove/delete:** Users already have delete; add deleteFlag (rules removed via DB cascade or explicitly); rules removed by sending new list in updateFlag (e.g. `rules: []` = remove all).
