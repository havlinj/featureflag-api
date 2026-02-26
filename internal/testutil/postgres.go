package testutil

import (
	"context"
	"testing"

	"github.com/jan-havlin-dev/featureflag-api/internal/db"
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

// TruncateAll truncates all application tables (users, feature_flags, flag_rules) for a clean state.
func TruncateAll(t *testing.T, database *db.DB) {
	t.Helper()
	ctx := context.Background()
	if _, err := database.Conn().ExecContext(ctx, "TRUNCATE flag_rules, feature_flags, users CASCADE"); err != nil {
		t.Fatalf("truncate: %v", err)
	}
}
