# Session 2 – Summary

**Date**: 2026-02-26

## 1. User management – domain & persistence

- **internal/db/schema.go**: Added `users` table (id UUID PK, email TEXT UNIQUE, role TEXT CHECK admin/developer/viewer, created_at TIMESTAMPTZ). `truncateTables` in db_test truncates `flag_rules`, `feature_flags`, `users` CASCADE.
- **internal/users/entity.go**: `User` (ID, Email, Role, CreatedAt), `Role` constants (admin, developer, viewer).
- **internal/users/errors.go**: `ErrDuplicateEmail`, `ErrNotFound`.
- **internal/users/store.go**: Interface `Store` – `Create`, `GetByID`, `GetByEmail`, `Update`, `Delete`.
- **internal/users/postgres.go**: `PostgresStore` with `NewPostgresStore(conn *sql.DB)`; all methods implemented (Create with RETURNING, GetByID/GetByEmail returning (nil,nil) when not found, Update, Delete with ErrNotFound when 0 rows).
- **internal/users/postgres_test.go**: Build tag `integration`; `testDB(t)` and `truncateUsers(t, database)`; tests for Create (happy + duplicate email), GetByID, GetByEmail, Update, Delete.

## 2. User management – service layer

- **internal/users/service.go**: `Service` with `Store`; `NewService(store)`. `CreateUser` (GetByEmail check, role parse, Create), `GetUser`, `GetUserByEmail`, `UpdateUser` (GetByID then Update), `DeleteUser` (Delete; returns true/false for found/not found).

## 3. GraphQL – users schema & resolvers

- **graph/schema/users.graphqls**: Type `User` (id, email, role, createdAt), `CreateUserInput`, `UpdateUserInput`, `extend type Query { user(id: ID!) }`, `extend type Mutation { createUser, updateUser, deleteUser(id: ID!): Boolean! }`.
- **gqlgen**: Ran `go run github.com/99designs/gqlgen generate`; **graph/generated.go**, **graph/model/models_gen.go** and **transport/graphql/users.resolvers.go** generated. Resolvers call `users.Service` (CreateUser, GetUser, GetUserByEmail, UpdateUser, DeleteUser).
- **transport/graphql/resolver.go**: Added `Users *users.Service`; **internal/app/app.go**: `NewApp(tlsConfig, flagsStore, usersStore users.Store)` builds both Flags and Users services and passes them to the resolver.

## 4. Single mock per domain (no duplicate test stubs)

- **internal/flags/mock/mock.go**: Queue-based `Store` implementing `flags.Store`; result types `CreateResult`, `GetByKeyResult`, `GetRulesResult`; `ErrNoMoreReturns` when queue empty. Single source of flags mock for unit tests.
- **internal/users/mock/mock.go**: Same pattern for users – queue-based `Store` with `CreateResult`, `GetByIDResult`, `GetByEmailResult`, `UpdateReturns`, `DeleteReturns`.
- **Removed**: **internal/testutil/flags_mock.go** – duplicate of flags store mock; replaced by `internal/flags/mock`.
- **Unit tests**: `internal/flags/service_test.go` uses package `flags_test` and imports `internal/flags/mock`; `internal/users/service_test.go` uses package `users_test` and imports `internal/users/mock`. No local mock structs in test files.

## 5. Integration tests with real PostgreSQL

- **internal/testutil/postgres.go**: `PostgresForIntegration(t)` – starts testcontainers Postgres (postgres:16-alpine), opens DB, `EnsureSchema`, returns `*db.DB` and cleanup. `TruncateAll(t, database)` – truncates `flag_rules`, `feature_flags`, `users` CASCADE.
- **test/integration/integration_api_test.go**: No mocks. Each test: `PostgresForIntegration(t)`, `TruncateAll(t, database)`, `flags.NewPostgresStore(database.Conn())`, `users.NewPostgresStore(database.Conn())`, `app.NewApp(tlsConfig, flagsStore, usersStore)`.
- **TestFlagsAPI_GraphQLOverHTTPS**: createFlag → updateFlag → evaluateFlag; asserts on real data and that `evaluateFlag` returns true (flag enabled, no rules).
- **TestUsersAPI_GraphQLOverHTTPS**: createUser → user(id) → updateUser → user(id) → deleteUser → user(id) is null; uses real UUID from createUser response.
- **Removed**: In-memory users store and all mock enqueue setup from integration tests.

## 6. Git & docs

- **Commit**: Amended last commit to include previously unadded files (go.mod, go.sum, graph schema/generated, internal/flags/mock, internal/users/*, internal/testutil/postgres.go, deletion of testutil/flags_mock.go, transport/graphql/users.resolvers.go). Commit message updated to describe user management, single mock per domain, and integration tests with real Postgres.
- **docs/progress.md**: Phase 1 progress set to ~60%; status and milestones updated (Flags + Users core, integration tests with real Postgres); checklist and metrics updated; changelog entry added for session 2 (user management, mocks consolidation, real-DB integration tests).

## Files touched (overview)

- **New**: graph/schema/users.graphqls, internal/flags/mock/mock.go, internal/testutil/postgres.go, internal/users/entity.go, errors.go, store.go, postgres.go, postgres_test.go, service.go, service_test.go, internal/users/mock/mock.go, transport/graphql/users.resolvers.go.
- **Modified**: go.mod, go.sum, graph/generated.go, graph/model/models_gen.go, internal/app/app.go, internal/db/schema.go, internal/db/db_test.go (truncateTables includes users), internal/flags/service_test.go (package flags_test, use flags/mock), test/integration/integration_api_test.go (real Postgres, no mocks), transport/graphql/resolver.go, docs/progress.md.
- **Removed**: internal/testutil/flags_mock.go.

## Dependencies

- No new direct dependencies; testcontainers and pgx already in use. gqlgen regenerate only.
