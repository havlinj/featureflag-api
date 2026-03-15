//go:build integration

package flags

import (
	"context"
	"errors"
	"testing"

	"github.com/havlinj/featureflag-api/internal/db"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

func testDB(t *testing.T) (*db.DB, func()) {
	ctx := context.Background()
	ctr, err := postgres.Run(ctx, "postgres:16-alpine",
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("test"),
		postgres.WithPassword("test"),
		postgres.BasicWaitStrategies(),
	)
	if err != nil {
		t.Fatalf("start postgres: %v", err)
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

func truncateFlags(t *testing.T, database *db.DB) {
	t.Helper()
	ctx := context.Background()
	if _, err := database.Conn().ExecContext(ctx, "TRUNCATE flag_rules, feature_flags CASCADE"); err != nil {
		t.Fatalf("truncate: %v", err)
	}
}

func TestPostgresStore_Create_happy_path(t *testing.T) {
	database, cleanup := testDB(t)
	defer cleanup()
	truncateFlags(t, database)
	store := NewPostgresStore(database.Conn())
	ctx := context.Background()
	flag := &Flag{Key: "create-key", Description: strPtr("d"), Enabled: false, Environment: DeploymentStageDev, RolloutStrategy: RolloutStrategyNone}

	created, err := store.Create(ctx, flag)

	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if created.ID == "" {
		t.Error("expected ID to be set")
	}
	if created.CreatedAt.IsZero() {
		t.Error("expected CreatedAt to be set")
	}
	if created.Key != "create-key" || created.Environment != DeploymentStageDev || created.Enabled {
		t.Errorf("unexpected flag: %+v", created)
	}
}

func TestPostgresStore_Create_duplicate_key_returns_ErrDuplicateKey(t *testing.T) {
	database, cleanup := testDB(t)
	defer cleanup()
	truncateFlags(t, database)
	store := NewPostgresStore(database.Conn())
	ctx := context.Background()
	flag := &Flag{Key: "dup-key", Environment: DeploymentStageDev, Enabled: false, RolloutStrategy: RolloutStrategyNone}
	_, _ = store.Create(ctx, flag)

	_, err := store.Create(ctx, flag)

	if err == nil {
		t.Fatal("expected error on duplicate")
	}
	var e *DuplicateKeyError
	if !errors.As(err, &e) {
		t.Errorf("expected *DuplicateKeyError, got %v", err)
	}
	if e.Key != "dup-key" || e.Environment != "dev" {
		t.Errorf("expected Key=dup-key Environment=dev, got Key=%q Environment=%q", e.Key, e.Environment)
	}
}

func TestPostgresStore_GetByKeyAndEnvironment_happy_path(t *testing.T) {
	database, cleanup := testDB(t)
	defer cleanup()
	truncateFlags(t, database)
	store := NewPostgresStore(database.Conn())
	ctx := context.Background()
	flag := &Flag{Key: "get-key", Description: strPtr("desc"), Enabled: true, Environment: DeploymentStageStaging, RolloutStrategy: RolloutStrategyNone}
	created, err := store.Create(ctx, flag)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := store.GetByKeyAndEnvironment(ctx, "get-key", DeploymentStageStaging)

	if err != nil {
		t.Fatalf("GetByKeyAndEnvironment: %v", err)
	}
	if got == nil {
		t.Fatal("expected flag")
	}
	if got.ID != created.ID || got.Key != "get-key" || !got.Enabled {
		t.Errorf("unexpected: %+v", got)
	}
	if got.Description == nil || *got.Description != "desc" {
		t.Errorf("unexpected description: %v", got.Description)
	}
}

func TestPostgresStore_GetByKeyAndEnvironment_not_found_returns_nil_nil(t *testing.T) {
	database, cleanup := testDB(t)
	defer cleanup()
	truncateFlags(t, database)
	store := NewPostgresStore(database.Conn())
	ctx := context.Background()

	got, err := store.GetByKeyAndEnvironment(ctx, "nonexistent", DeploymentStageDev)

	if err != nil {
		t.Fatalf("GetByKeyAndEnvironment: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil, got %+v", got)
	}
}

func TestPostgresStore_Update_happy_path(t *testing.T) {
	database, cleanup := testDB(t)
	defer cleanup()
	truncateFlags(t, database)
	store := NewPostgresStore(database.Conn())
	ctx := context.Background()
	created, _ := store.Create(ctx, &Flag{Key: "up-key", Environment: DeploymentStageDev, Enabled: false, RolloutStrategy: RolloutStrategyNone})
	created.Enabled = true
	created.Description = strPtr("updated")

	err := store.Update(ctx, created)

	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	got, _ := store.GetByKeyAndEnvironment(ctx, "up-key", DeploymentStageDev)
	if got == nil || !got.Enabled {
		t.Errorf("expected enabled after update: %+v", got)
	}
	if got.Description == nil || *got.Description != "updated" {
		t.Errorf("unexpected description: %v", got.Description)
	}
}

func TestPostgresStore_Update_not_found_returns_ErrNotFound(t *testing.T) {
	database, cleanup := testDB(t)
	defer cleanup()
	truncateFlags(t, database)
	store := NewPostgresStore(database.Conn())
	ctx := context.Background()
	flag := &Flag{ID: "00000000-0000-0000-0000-000000000000", Key: "x", Environment: DeploymentStageDev, Enabled: true, RolloutStrategy: RolloutStrategyNone}

	err := store.Update(ctx, flag)

	if err == nil {
		t.Fatal("expected error")
	}
	var e *NotFoundError
	if !errors.As(err, &e) {
		t.Errorf("expected *NotFoundError, got %v", err)
	}
	if e.ID != "00000000-0000-0000-0000-000000000000" {
		t.Errorf("expected ID=00000000-0000-0000-0000-000000000000, got ID=%q", e.ID)
	}
}

func TestPostgresStore_GetRulesByFlagID_happy_path(t *testing.T) {
	database, cleanup := testDB(t)
	defer cleanup()
	truncateFlags(t, database)
	store := NewPostgresStore(database.Conn())
	ctx := context.Background()
	created, _ := store.Create(ctx, &Flag{Key: "rule-key", Environment: DeploymentStageDev, Enabled: false, RolloutStrategy: RolloutStrategyNone})
	_, err := database.Conn().ExecContext(ctx,
		"INSERT INTO flag_rules (flag_id, type, value) VALUES ($1, $2, $3)",
		created.ID, "percentage", "30",
	)
	if err != nil {
		t.Fatalf("insert rule: %v", err)
	}

	rules, err := store.GetRulesByFlagID(ctx, created.ID)

	if err != nil {
		t.Fatalf("GetRulesByFlagID: %v", err)
	}
	if len(rules) != 1 || rules[0].Type != RuleTypePercentage || rules[0].Value != "30" {
		t.Errorf("unexpected rules: %+v", rules)
	}
}

func TestPostgresStore_GetRulesByFlagID_no_rules_returns_empty_slice(t *testing.T) {
	database, cleanup := testDB(t)
	defer cleanup()
	truncateFlags(t, database)
	store := NewPostgresStore(database.Conn())
	ctx := context.Background()
	created, _ := store.Create(ctx, &Flag{Key: "norules", Environment: DeploymentStageDev, Enabled: false, RolloutStrategy: RolloutStrategyNone})

	rules, err := store.GetRulesByFlagID(ctx, created.ID)

	if err != nil {
		t.Fatalf("GetRulesByFlagID: %v", err)
	}
	if len(rules) != 0 {
		t.Errorf("expected no rules, got len=%d", len(rules))
	}
}

func TestPostgresStore_GetRulesByFlagID_invalid_uuid_returns_error(t *testing.T) {
	database, cleanup := testDB(t)
	defer cleanup()
	store := NewPostgresStore(database.Conn())
	ctx := context.Background()

	_, err := store.GetRulesByFlagID(ctx, "not-a-valid-uuid")

	if err == nil {
		t.Error("expected error for invalid UUID in flag_id")
	}
}

func strPtr(s string) *string { return &s }
