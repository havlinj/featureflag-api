package db

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

func TestDB_EnsureSchema_and_naked_write_read(t *testing.T) {
	database := arrangeDBWithSchema(t)
	truncateTables(t, database)

	flagID := insertFeatureFlagRow(t, database.Conn(), "naked-key", "naked desc", false, "dev")
	insertFlagRuleRow(t, database.Conn(), flagID, "percentage", "50")

	assertFeatureFlagReadBack(t, database.Conn(), "naked-key", "dev", "naked desc", false)
	assertFlagRuleReadBack(t, database.Conn(), flagID, "percentage", "50")
}

func TestDB_EnsureSchema_experiment_assignment_variant_belongs_to_same_experiment(t *testing.T) {
	database := arrangeDBWithSchema(t)
	truncateTables(t, database)
	ctx := context.Background()
	conn := database.Conn()
	userID := insertUserRow(t, conn, "a@test.com")
	exp1 := insertExperimentRow(t, conn, "exp-a", "dev")
	exp2 := insertExperimentRow(t, conn, "exp-b", "dev")
	variantFromExp2 := insertVariantRow(t, conn, exp2, "B", 100)

	_, err := conn.ExecContext(ctx,
		`INSERT INTO experiment_assignments (user_id, experiment_id, variant_id) VALUES ($1, $2, $3)`,
		userID, exp1, variantFromExp2,
	)

	if err == nil {
		t.Fatal("expected FK violation for mismatched experiment_id/variant_id, got nil")
	}
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		t.Fatalf("expected pg error, got %v", err)
	}
	if pgErr.Code != "23503" {
		t.Fatalf("expected FK violation code 23503, got %s", pgErr.Code)
	}
}

// arrangeDBWithSchema starts Postgres, opens DB, runs EnsureSchema. Caller must not close DB (cleanup registered).
func arrangeDBWithSchema(t *testing.T) *DB {
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
	t.Cleanup(func() { _ = ctr.Terminate(ctx) })

	dsn, err := ctr.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("connection string: %v", err)
	}
	database, err := Open(ctx, dsn)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })

	if err := database.EnsureSchema(ctx); err != nil {
		t.Fatalf("ensure schema: %v", err)
	}
	return database
}

func truncateTables(t *testing.T, database *DB) {
	t.Helper()
	ctx := context.Background()
	if _, err := database.Conn().ExecContext(ctx, "TRUNCATE audit_logs, experiment_assignments, experiment_variants, experiments, flag_rules, feature_flags, users CASCADE"); err != nil {
		t.Fatalf("truncate: %v", err)
	}
}

func insertUserRow(t *testing.T, conn *sql.DB, email string) string {
	t.Helper()
	ctx := context.Background()
	var id string
	err := conn.QueryRowContext(ctx,
		`INSERT INTO users (email, role) VALUES ($1, 'admin') RETURNING id`,
		email,
	).Scan(&id)
	if err != nil {
		t.Fatalf("insert users: %v", err)
	}
	return id
}

func insertExperimentRow(t *testing.T, conn *sql.DB, key, environment string) string {
	t.Helper()
	ctx := context.Background()
	var id string
	err := conn.QueryRowContext(ctx,
		`INSERT INTO experiments (key, environment) VALUES ($1, $2) RETURNING id`,
		key, environment,
	).Scan(&id)
	if err != nil {
		t.Fatalf("insert experiments: %v", err)
	}
	return id
}

func insertVariantRow(t *testing.T, conn *sql.DB, experimentID, name string, weight int) string {
	t.Helper()
	ctx := context.Background()
	var id string
	err := conn.QueryRowContext(ctx,
		`INSERT INTO experiment_variants (experiment_id, name, weight) VALUES ($1, $2, $3) RETURNING id`,
		experimentID, name, weight,
	).Scan(&id)
	if err != nil {
		t.Fatalf("insert experiment_variants: %v", err)
	}
	return id
}

func insertFeatureFlagRow(t *testing.T, conn *sql.DB, key, description string, enabled bool, environment string) string {
	t.Helper()
	ctx := context.Background()
	var id string
	err := conn.QueryRowContext(ctx,
		`INSERT INTO feature_flags (key, description, enabled, environment) VALUES ($1, $2, $3, $4) RETURNING id`,
		key, description, enabled, environment,
	).Scan(&id)
	if err != nil {
		t.Fatalf("insert feature_flags: %v", err)
	}
	return id
}

func insertFlagRuleRow(t *testing.T, conn *sql.DB, flagID, ruleType, value string) {
	t.Helper()
	ctx := context.Background()
	_, err := conn.ExecContext(ctx,
		`INSERT INTO flag_rules (flag_id, type, value) VALUES ($1, $2, $3)`,
		flagID, ruleType, value,
	)
	if err != nil {
		t.Fatalf("insert flag_rules: %v", err)
	}
}

func assertFeatureFlagReadBack(t *testing.T, conn *sql.DB, key, environment, wantDescription string, wantEnabled bool) {
	t.Helper()
	ctx := context.Background()
	var id, k, env string
	var desc sql.NullString
	var enabled bool
	var createdAt time.Time
	row := conn.QueryRowContext(ctx,
		`SELECT id, key, description, enabled, environment, created_at FROM feature_flags WHERE key = $1 AND environment = $2`,
		key, environment,
	)
	if err := row.Scan(&id, &k, &desc, &enabled, &env, &createdAt); err != nil {
		t.Fatalf("select feature_flags: %v", err)
	}
	if id == "" {
		t.Error("feature_flags id should be set")
	}
	if k != key || env != environment {
		t.Errorf("feature_flags key/env: got %q/%q, want %q/%q", k, env, key, environment)
	}
	if wantDescription != "" && (!desc.Valid || desc.String != wantDescription) {
		t.Errorf("feature_flags description: got %v, want %q", desc, wantDescription)
	}
	if enabled != wantEnabled {
		t.Errorf("feature_flags enabled: got %v, want %v", enabled, wantEnabled)
	}
	if createdAt.IsZero() {
		t.Error("feature_flags created_at should be set")
	}
}

func assertFlagRuleReadBack(t *testing.T, conn *sql.DB, flagID, wantType, wantValue string) {
	t.Helper()
	ctx := context.Background()
	var ruleID, ruleFlagID, ruleType, ruleValue string
	row := conn.QueryRowContext(ctx,
		`SELECT id, flag_id, type, value FROM flag_rules WHERE flag_id = $1`, flagID,
	)
	if err := row.Scan(&ruleID, &ruleFlagID, &ruleType, &ruleValue); err != nil {
		t.Fatalf("select flag_rules: %v", err)
	}
	if ruleID == "" {
		t.Error("flag_rules id should be set")
	}
	if ruleFlagID != flagID || ruleType != wantType || ruleValue != wantValue {
		t.Errorf("flag_rules: got flag_id=%q type=%q value=%q, want flag_id=%q type=%q value=%q",
			ruleFlagID, ruleType, ruleValue, flagID, wantType, wantValue)
	}
}
