package testutil

import (
	"context"
	"testing"

	"github.com/havlinj/featureflag-api/internal/auth"
	"github.com/havlinj/featureflag-api/internal/db"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

// PostgresForIntegration starts a Postgres container, opens the DB, runs EnsureSchema,
// and returns the DB plus a cleanup function. Use in integration tests that need a real DB.
// Call TruncateAll before each test if you need an empty schema.
func PostgresForIntegration(t *testing.T) (*db.DB, func()) {
	t.Helper()
	ctx := context.Background()
	ctr, err := postgres.Run(ctx, "postgres:16-alpine",
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("test"),
		postgres.WithPassword("test"),
		postgres.BasicWaitStrategies(),
	)
	if err != nil {
		t.Fatalf("start postgres container: %v", err)
	}
	cleanup := func() { _ = ctr.Terminate(ctx) }

	dsn, err := ctr.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		cleanup()
		t.Fatalf("connection string: %v", err)
	}
	database, err := db.Open(ctx, dsn)
	if err != nil {
		cleanup()
		t.Fatalf("open db: %v", err)
	}
	cleanupDB := func() { _ = database.Close() }
	cleanup = func() { cleanupDB(); _ = ctr.Terminate(ctx) }

	if err := database.EnsureSchema(ctx); err != nil {
		cleanup()
		t.Fatalf("ensure schema: %v", err)
	}
	return database, cleanup
}

// TruncateAll truncates all application tables for a clean state.
// Order: child tables first (experiment_assignments, experiment_variants, experiments, flag_rules, feature_flags, users).
func TruncateAll(t *testing.T, database *db.DB) {
	t.Helper()
	ctx := context.Background()
	if _, err := database.Conn().ExecContext(ctx,
		"TRUNCATE experiment_assignments, experiment_variants, experiments, flag_rules, feature_flags, users CASCADE"); err != nil {
		t.Fatalf("truncate: %v", err)
	}
}

// SeedAdminAndLogin inserts an admin user with the given email and password into the DB,
// then calls the login mutation via the client and returns the JWT token.
// Use this in integration tests to obtain a token for protected operations.
func SeedAdminAndLogin(t *testing.T, database *db.DB, client *GraphQLClient, email, password string) string {
	t.Helper()
	ctx := context.Background()
	hash, err := auth.HashPassword(password)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	_, err = database.Conn().ExecContext(ctx,
		"INSERT INTO users (email, role, password_hash) VALUES ($1, 'admin', $2)",
		email, hash,
	)
	if err != nil {
		t.Fatalf("seed admin user: %v", err)
	}
	resp, err := client.DoRequest(`
		mutation Login($input: LoginInput!) {
			login(input: $input) { token }
		}
	`, map[string]interface{}{
		"input": map[string]interface{}{"email": email, "password": password},
	})
	if err != nil {
		t.Fatalf("login request: %v", err)
	}
	if resp.Data == nil || resp.Errors != nil && len(resp.Errors) > 0 {
		t.Fatalf("login: expected data, got data=%v errors=%v", resp.Data, resp.Errors)
	}
	loginData, _ := resp.Data["login"].(map[string]interface{})
	token, _ := loginData["token"].(string)
	if token == "" {
		t.Fatal("login: expected non-empty token")
	}
	return token
}
