# PLAN: Authentication & Middleware for HTTPS/GraphQL

**Status**: NEEDS_APPROVAL  
**Date**: 2026-02-27  
**Scope**: JWT auth, middleware package, login, seed first admin (script required), RBAC at resolver level, explicit error responses.

**Location of this plan:** `docs/plan_auth_middleware.md`. Design and implementation decisions for this feature are taken from this document. Implementation must follow this plan unless we explicitly agree on a change.

---

## 1. Problem Definition

**What is being changed?**  
We currently expose GraphQL over HTTPS with no authentication. Any client can call all operations (createUser, createFlag, etc.). We need to identify the caller and enforce role-based access.

**Why is the change needed?**  
Phase 1 requires “Authentication middleware (JWT-based) and logging middleware” (architecture.mdc, progress.md). We need a single source of truth for “who is calling” and “what role they have” so that resolvers can allow or deny operations (e.g. only admin creates users).

---

## 2. Scope

**In scope:**

- New package `transport/graphql/middleware` with HTTP middleware only (no business logic).
- New package `internal/auth`: JWT parsing, validation, and extraction of user ID + role; no password hashing here (that stays in users or a dedicated place).
- User-facing **login**: one way to obtain a JWT (e.g. GraphQL mutation `login(email, password)` returning a token). Requires storing `password_hash` for users and a way to verify password.
- **Seed for first admin**: **required** – we provide a **script** `scripts/seed_admin.sql` (and a way to run it, e.g. wrapper script or documented steps). Run manually or from CI/CD after deploy. Idempotent (e.g. `ON CONFLICT DO NOTHING`). No automatic in-process seed at app startup; first admin is created only by running this script.
- **Auth middleware**: reads `Authorization: Bearer <token>`, validates JWT, puts user ID and role into request context; on missing/invalid token returns 401.
- **Logging middleware**: logs request (method, path, optional request ID) and response status; does not log bodies or tokens.
- **RBAC in resolvers**: resolvers that must be restricted (e.g. createUser, updateUser, deleteUser, createFlag, updateFlag) read caller from context and check role (e.g. admin only for user management); return 403 or equivalent when forbidden.
- **Explicit error responses over HTTPS**: when the user has no admin rights → 403 with a clear message (e.g. “user does not have admin rights”); when no admin exists in the system yet (seed not run) → distinct error/message (e.g. “admin has not been set up yet; run the seed script”) so the client can guide the operator.
- **Public operations**: login (and optionally health/read-only introspection) do not require a token; auth middleware skips or allows unauthenticated for those paths.
- Unit tests for auth (JWT parse/validate, context keys) and middleware (chain order, 401 on invalid token). Integration test: login → token → call protected mutation with token; call without token → 401.

**Out of scope (for this plan):**

- Refresh tokens, token revocation, or session store.
- Multiple issuers / JWKS; we use a single symmetric secret or single key pair for our own tokens.
- Audit logging (Phase 3); we only add auth and logging middleware here.
- Changing existing resolver signatures beyond adding context usage for auth.

---

## 3. Proposed Solution

### 3.1 High-level flow

1. **First admin via script**: Before or right after deploy, an operator runs a **seed script** (e.g. `scripts/seed_admin.sql`). The script inserts one admin user (email, role, password_hash) in an idempotent way (e.g. `INSERT ... ON CONFLICT (email) DO NOTHING`). How the password hash gets into the script is up to the script design: e.g. a wrapper `scripts/seed_admin.sh` that reads `FIRST_ADMIN_EMAIL` and `FIRST_ADMIN_PASSWORD` from env, computes the hash (e.g. with a small CLI or `htpasswd`), and passes it to `psql` / a parameterised SQL. We do **not** create the first admin inside the application at startup; only this script does it.
2. **Login**: Client calls `login(email, password)`. Server loads user by email, verifies password (bcrypt), builds JWT (e.g. `sub=user.ID`, `role=user.Role`, `exp`), signs with app secret/key, returns token string.
3. **Protected requests**: Client sends `Authorization: Bearer <token>`. Middleware validates token, extracts `sub` and `role`, puts them in `context.Context`. Resolver reads context; if role insufficient → return error (e.g. “forbidden”).
4. **Logging**: Middleware wraps handler, logs method, path, request ID (if any), status code, and duration; no auth token or body in logs.

### 3.2 Package layout

- **`internal/auth`**
  - `auth.go`: types (e.g. `Claims`: Sub, Role, Exp), context keys, `FromContext(ctx) (userID, role, ok)`.
  - `jwt.go`: `ParseAndValidate(tokenString string, secret []byte) (*Claims, error)`; use a single library (e.g. `github.com/golang-jwt/jwt/v5`). Symmetric HMAC for now.
  - No HTTP, no DB access here; only JWT and context. Password hashing can live in `internal/users` or a small `internal/auth/password` (bcrypt).

- **`transport/graphql/middleware`**
  - `chain.go`: `Chain(inner http.Handler, mws ...func(http.Handler) http.Handler) http.Handler` (or similar) to compose middleware.
  - `auth.go`: HTTP middleware that reads `Authorization` header, calls `auth.ParseAndValidate`, sets claims in context; if token missing/invalid and path is protected → write 401 and do not call inner handler. Paths like `"/"` (GraphQL) can be considered protected by default; we allow unauthenticated only for login (handled inside GraphQL: login mutation does not require auth).
  - `logging.go`: HTTP middleware that logs request start, then calls inner; in a defer or wrapper logs status and duration. Optional: add request ID (header or generate) and put in context for logging.
  - `recovery.go` (optional): panic recovery, log and return 500.

- **GraphQL and login**
  - Login is a GraphQL mutation (e.g. `login(email: String!, password: String!): LoginPayload` with `token: String!`). Resolver uses `users.Store` (or a small “auth service” that uses Store) to get user by email and verify password, then builds JWT and returns it. No auth middleware required for the “login” operation: either we allow unauthenticated for all GraphQL and enforce “must be authenticated” inside resolvers for protected ops, or we skip auth for a specific operation; the simplest is “auth middleware runs for all requests, but if token is missing we still pass request to GraphQL” and each resolver that needs auth checks context and returns error if not authenticated. Alternatively: auth middleware returns 401 only when token is present but invalid; when token is missing, we still call GraphQL and resolvers return “unauthorized” for protected ops. Decision: **auth middleware** returns 401 if token is missing or invalid for the whole GraphQL endpoint; **login** is exposed via a separate HTTP path (e.g. `POST /login`) that does not go through the auth middleware. That way we don’t need “optional auth” in middleware. So: two options:
  - **A)** Single GraphQL endpoint; auth middleware sets context when token is valid; when token is missing, context has no user. Resolvers for protected ops check context; if no user → return GraphQL error “unauthorized”. Login mutation does not require auth.
  - **B)** Login on separate path (e.g. `POST /login`); only the GraphQL handler is wrapped with auth middleware that requires valid token (401 if missing/invalid). So unauthenticated clients can only hit `/login`.

  We choose **A)** so that the client always talks to one GraphQL endpoint; login mutation is public, other operations require valid JWT and the right role. Middleware: if `Authorization` present and invalid → 401. If missing → do not set user in context; resolvers that need auth check context and return error. If present and valid → set context. So middleware never returns 401 for “missing” token; it only returns 401 for “invalid” token. Resolvers return “unauthorized” when token is missing and operation is protected. Simpler: **middleware**: missing token → no user in context; invalid token → 401. **Resolvers**: protected op and no user in context → return error (e.g. “unauthorized”).

- **App wiring**
  - In `internal/app/app.go`: build handler chain: `logging → auth → graphqlHandler`. Pass JWT secret and possibly user store (for login) into the chain. GraphQL handler is gqlgen’s handler. Resolver gets `users.Service` and a way to issue JWT (e.g. a small `auth.Issuer` interface or function that takes user id + role and returns signed token). Login mutation resolver: get user by email, verify password, call issuer, return token.

### 3.3 Data flow (example)

- **Login**: `Client → POST /graphql { mutation { login(email, password) { token } } }` → no auth middleware requirement (token missing) → GraphQL → Login resolver → users.Store.GetByEmail + verify password → auth.IssueToken(userID, role) → return token.
- **CreateUser**: `Client → POST /graphql { mutation { createUser(...) } }` with `Authorization: Bearer <token>` → auth middleware parses token, sets user in context → GraphQL → CreateUser resolver → from context: must be admin → users.Service.CreateUser → return.
- **CreateUser without token**: same, but context has no user → CreateUser resolver returns “unauthorized” (or “forbidden”).

### 3.4 Database change

- Add to `users` table: `password_hash TEXT` (nullable for existing users; for login we require it). The seed script sets it for the first admin when the operator runs it. Migration in `internal/db/schema.go`: add column (e.g. `ALTER TABLE` or new schemaSQL entry; if we use only CREATE TABLE IF NOT EXISTS we may need a separate migration step for adding column to existing DBs—for simplicity we can add `password_hash TEXT` to the initial users table definition and document that new deployments get it; existing DBs get a one-off migration or manual column add).

  For minimal scope: add `password_hash TEXT` to the `users` table in schema (new deployments). Existing integration tests that create users without password can set password_hash to NULL; login only works for users with non-null password_hash.

### 3.5 RBAC matrix and how it is implemented

**What is the RBAC matrix?**  
It is a small “who may do what” table we use as documentation and as the source of truth for resolver checks. It is **not** a separate table in the database. We implement it by **checks inside each protected resolver**: the resolver reads the caller from `auth.FromContext(ctx)` (user ID, role); if the caller is missing or their role is not allowed for that operation, it returns a GraphQL error (e.g. “unauthorized” or “forbidden”). No central “RBAC engine”; each resolver enforces its own rule.

**Implementation in code:**  
- In each protected resolver (e.g. `CreateUser`, `UpdateUser`, `DeleteUser`, `CreateFlag`, `UpdateFlag`): at the start of the resolver, call e.g. `userID, role, ok := auth.FromContext(ctx)`; if `!ok` or role not allowed, return `nil, errUnauthorized` or `nil, errForbidden` (typed errors that the transport can map to HTTP 401/403 if desired).  
- Optionally: a small helper in `internal/auth` or `transport/graphql`, e.g. `RequireRole(ctx, allowedRoles ...Role) (userID string, err error)`, so resolvers do not repeat the same if/switch. The matrix below is what that helper (or the inline checks) will enforce.

**Matrix (operation → allowed roles):**

| Operation | Allowed roles | Note |
|-----------|----------------|------|
| createUser, updateUser, deleteUser | admin | User management only for admin. |
| createFlag, updateFlag | admin, developer | To be decided; often developers can manage flags. |
| evaluateFlag, getFlag, user(id), userByEmail | any authenticated | Require valid token; role not restricted. |
| login | — | No auth; public. |

### 3.6 Explicit error responses over HTTPS

The application **must** return clear, distinguishable responses so clients (and operators) know what went wrong:

- **User has no admin rights**: When the caller is authenticated but their role is not allowed for the operation (e.g. developer calls `createUser`), the API responds with **403 Forbidden** (or equivalent in GraphQL error payload) and an explicit message, e.g. *"user does not have admin rights"* or *"forbidden: admin role required"*. The client can show this directly to the user.
- **Admin not yet configured**: When an operation requires that at least one admin exists (e.g. user management or a check “is the system initialised?”), and there is **no admin in the database** (seed script has not been run), the API responds with a distinct error, e.g. **503 Service Unavailable** or a dedicated code/message such as *"admin has not been set up yet; run the seed script"*. This allows the client or docs to tell the operator to run `scripts/seed_admin.sql` (or the wrapper) first.

Implementation: resolvers (or a small auth helper) return **typed errors** (e.g. `ErrForbidden`, `ErrAdminNotConfigured`); the transport layer maps them to the appropriate HTTP status and body/message so that HTTPS responses are explicit and consistent.

---

## 4. Affected Files

**New files:**

- `internal/auth/auth.go` – context keys, Claims type, FromContext.
- `internal/auth/jwt.go` – ParseAndValidate, IssueToken (or Sign).
- `internal/auth/password.go` (or under users) – HashPassword, PasswordMatches (bcrypt).
- `transport/graphql/middleware/chain.go` – handler chain.
- `transport/graphql/middleware/auth.go` – JWT extraction and context injection; 401 on invalid token.
- `transport/graphql/middleware/logging.go` – request logging.
- **Required:** `scripts/seed_admin.sql` – idempotent SQL that inserts the first admin (e.g. `INSERT ... ON CONFLICT (email) DO NOTHING`). **Required:** a way to run it with real data – e.g. `scripts/seed_admin.sh` that reads env (`FIRST_ADMIN_EMAIL`, `FIRST_ADMIN_PASSWORD`), computes password hash, and invokes the SQL (via `psql` or a small Go CLI). Document in `scripts/README.md` how to run the seed (once per environment).
- GraphQL: `graph/schema/auth.graphqls` (or extend existing) – `LoginInput`, `LoginPayload { token }`, `mutation { login(...) }`.
- Resolver: `transport/graphql/auth.resolvers.go` (or similar) – login mutation; and updates to users.resolvers.go / flags.resolvers.go to check context and role.

**Modified files:**

- `internal/db/schema.go` – add `password_hash` to users table (or migration).
- `internal/users/entity.go` – add PasswordHash to User if we store it in entity (optional; can stay DB-only).
- `internal/users/postgres.go` – Create/Update/GetByEmail handle password_hash.
- `internal/app/app.go` – build middleware chain (logging, auth), wrap GraphQL handler; pass secret and possibly issuer into middleware and resolver.
- `transport/graphql/resolver.go` – add auth issuer or secret for login resolver; resolvers that need RBAC get context and check role.
- `internal/users/service.go` – optional: GetUserByEmailForLogin that returns user with password hash for verification (or keep in store layer).
- `docs/progress.md` – update Phase 1 checklist and changelog.
- Integration test: extend to login, then call protected mutation with token; add test for 401 on invalid token.

---

## 5. Edge Cases & Risks

- **Clock skew**: JWT `exp` validation; use library default (allow small skew). Low risk.
- **Secret management**: JWT secret in env (e.g. `JWT_SECRET`); never commit. Document in README.
- **Password in seed script**: If using a wrapper script, `FIRST_ADMIN_PASSWORD` in env; hash before insert; avoid logging. Never commit plaintext passwords.
- **Existing users without password**: login returns “invalid credentials” for users with NULL password_hash; admin can set password via future “set password” or we leave seed as the only way to set first admin password. No backdoor.
- **GraphQL and 401**: When middleware returns 401, body can be empty or JSON `{"error":"unauthorized"}`. Client must handle 401 and re-login.
- **Optional auth**: We decided resolvers that require auth check context; if no user, return GraphQL error. So no 401 from middleware for “no token”—only for “invalid token”. Consistent and simple.

---

## 6. Testing Strategy

- **Unit**
  - `internal/auth`: ParseAndValidate with valid/invalid/expired token; IssueToken and then parse; FromContext.
  - `internal/auth/password`: Hash and compare.
  - `transport/graphql/middleware`: Auth middleware with mock next handler—valid token → context set; invalid token → 401, next not called; no token → next called, context empty. Logging middleware: next called, status and duration logged (capture log output or use a test logger).
- **Integration**
  - Seed script: run `scripts/seed_admin.sql` (or wrapper) with env, check DB for one admin with non-null password_hash; run again, idempotent (no duplicate or error).
  - Login: call login mutation with correct email/password → get token; wrong password → error; no such user → error.
  - Protected op: with token from login → createUser (as admin) succeeds; with token as developer → createUser returns forbidden; without token → createUser returns unauthorized; invalid token in header → 401 from middleware (if we do 401 for invalid token).
  - Logging: one integration test that triggers a request and asserts log line contains path and status (or skip if too brittle).

---

## 7. Summary

| Item | Decision |
|------|----------|
| Middleware package | `transport/graphql/middleware` (chain, auth, logging) |
| Auth package | `internal/auth` (JWT parse/validate, issue, context, password hash/compare) |
| Login | GraphQL mutation `login(email, password)` returning `{ token }` |
| First admin | **Required** script `scripts/seed_admin.sql` + way to run it (e.g. `scripts/seed_admin.sh` + `scripts/README.md`). Idempotent; run manually or CI. No in-app bootstrap. |
| Auth middleware | Valid token → set user/role in context; invalid token → 401; missing token → no user in context |
| RBAC | Resolvers read context; admin for user CRUD; admin/developer for flag mutations (configurable). Explicit errors: 403 “no admin rights”, distinct “admin not configured” when no admin in DB. |
| DB | Add `password_hash` to users; seed sets it for first admin |

---

*End of plan. Ready for APPROVAL before implementation.*
