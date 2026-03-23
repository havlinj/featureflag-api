//go:build integration

package experiments

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/havlinj/featureflag-api/internal/db"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

func testDB(t *testing.T) (*db.DB, func()) {
	t.Helper()
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

func truncateExperiments(t *testing.T, database *db.DB) {
	t.Helper()
	ctx := context.Background()
	if _, err := database.Conn().ExecContext(ctx,
		"TRUNCATE experiment_assignments, experiment_variants, experiments CASCADE"); err != nil {
		t.Fatalf("truncate: %v", err)
	}
}

func TestPostgresStore_CreateExperiment_happy_path(t *testing.T) {
	database, cleanup := testDB(t)
	defer cleanup()
	truncateExperiments(t, database)
	store := NewPostgresStore(database.Conn())
	ctx := context.Background()
	exp := &Experiment{Key: "exp-key", Environment: "dev"}

	created, err := store.CreateExperiment(ctx, exp)

	if err != nil {
		t.Fatalf("CreateExperiment: %v", err)
	}
	if created.ID == "" {
		t.Error("expected ID to be set")
	}
	if created.Key != "exp-key" || created.Environment != "dev" {
		t.Errorf("unexpected experiment: %+v", created)
	}
}

func TestPostgresStore_CreateExperiment_duplicate_returns_ErrDuplicateExperiment(t *testing.T) {
	database, cleanup := testDB(t)
	defer cleanup()
	truncateExperiments(t, database)
	store := NewPostgresStore(database.Conn())
	ctx := context.Background()
	exp := &Experiment{Key: "dup-key", Environment: "staging"}
	_, err := store.CreateExperiment(ctx, exp)
	if err != nil {
		t.Fatalf("first CreateExperiment: %v", err)
	}

	_, err = store.CreateExperiment(ctx, exp)

	if err == nil {
		t.Fatal("expected error on duplicate key and environment")
	}
	var e *DuplicateExperimentError
	if !errors.As(err, &e) {
		t.Errorf("expected *DuplicateExperimentError, got %v", err)
	}
	if e.Key != "dup-key" || e.Environment != "staging" {
		t.Errorf("expected Key=dup-key Environment=staging, got Key=%q Environment=%q", e.Key, e.Environment)
	}
}

func TestPostgresStore_GetExperimentByKeyAndEnvironment_happy_path(t *testing.T) {
	database, cleanup := testDB(t)
	defer cleanup()
	truncateExperiments(t, database)
	store := NewPostgresStore(database.Conn())
	ctx := context.Background()
	created, err := store.CreateExperiment(ctx, &Experiment{Key: "get-key", Environment: "prod"})
	if err != nil {
		t.Fatalf("CreateExperiment: %v", err)
	}

	got, err := store.GetExperimentByKeyAndEnvironment(ctx, "get-key", "prod")

	if err != nil {
		t.Fatalf("GetExperimentByKeyAndEnvironment: %v", err)
	}
	if got == nil {
		t.Fatal("expected experiment")
	}
	if got.ID != created.ID || got.Key != "get-key" || got.Environment != "prod" {
		t.Errorf("unexpected: %+v", got)
	}
}

func TestPostgresStore_GetExperimentByKeyAndEnvironment_not_found_returns_nil_nil(t *testing.T) {
	database, cleanup := testDB(t)
	defer cleanup()
	truncateExperiments(t, database)
	store := NewPostgresStore(database.Conn())
	ctx := context.Background()

	got, err := store.GetExperimentByKeyAndEnvironment(ctx, "nonexistent", "dev")

	if err != nil {
		t.Fatalf("GetExperimentByKeyAndEnvironment: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil, got %+v", got)
	}
}

func TestPostgresStore_GetExperimentByID_happy_path(t *testing.T) {
	database, cleanup := testDB(t)
	defer cleanup()
	truncateExperiments(t, database)
	store := NewPostgresStore(database.Conn())
	ctx := context.Background()
	created, err := store.CreateExperiment(ctx, &Experiment{Key: "id-key", Environment: "dev"})
	if err != nil {
		t.Fatalf("CreateExperiment: %v", err)
	}

	got, err := store.GetExperimentByID(ctx, created.ID)

	if err != nil {
		t.Fatalf("GetExperimentByID: %v", err)
	}
	if got == nil {
		t.Fatal("expected experiment")
	}
	if got.ID != created.ID || got.Key != "id-key" {
		t.Errorf("unexpected: %+v", got)
	}
}

func TestPostgresStore_GetExperimentByID_not_found_returns_nil_nil(t *testing.T) {
	database, cleanup := testDB(t)
	defer cleanup()
	store := NewPostgresStore(database.Conn())
	ctx := context.Background()

	got, err := store.GetExperimentByID(ctx, "00000000-0000-0000-0000-000000000000")

	if err != nil {
		t.Fatalf("GetExperimentByID: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil, got %+v", got)
	}
}

func TestPostgresStore_CreateVariant_happy_path(t *testing.T) {
	database, cleanup := testDB(t)
	defer cleanup()
	truncateExperiments(t, database)
	store := NewPostgresStore(database.Conn())
	ctx := context.Background()
	exp, err := store.CreateExperiment(ctx, &Experiment{Key: "var-exp", Environment: "dev"})
	if err != nil {
		t.Fatalf("CreateExperiment: %v", err)
	}
	v := &Variant{ExperimentID: exp.ID, Name: "control", Weight: 50}

	created, err := store.CreateVariant(ctx, v)

	if err != nil {
		t.Fatalf("CreateVariant: %v", err)
	}
	if created.ID == "" {
		t.Error("expected variant ID to be set")
	}
	if created.ExperimentID != exp.ID || created.Name != "control" || created.Weight != 50 {
		t.Errorf("unexpected variant: %+v", created)
	}
}

func TestPostgresStore_CreateVariant_nonexistent_experiment_id_returns_error(t *testing.T) {
	database, cleanup := testDB(t)
	defer cleanup()
	truncateExperiments(t, database)
	store := NewPostgresStore(database.Conn())
	ctx := context.Background()
	v := &Variant{ExperimentID: "00000000-0000-0000-0000-000000000000", Name: "A", Weight: 50}

	_, err := store.CreateVariant(ctx, v)

	if err == nil {
		t.Fatal("expected error when experiment_id does not exist (FK violation)")
	}
}

func TestPostgresStore_GetVariantsByExperimentID_happy_path(t *testing.T) {
	database, cleanup := testDB(t)
	defer cleanup()
	truncateExperiments(t, database)
	store := NewPostgresStore(database.Conn())
	ctx := context.Background()
	exp, _ := store.CreateExperiment(ctx, &Experiment{Key: "multi-var", Environment: "dev"})
	_, _ = store.CreateVariant(ctx, &Variant{ExperimentID: exp.ID, Name: "A", Weight: 50})
	_, _ = store.CreateVariant(ctx, &Variant{ExperimentID: exp.ID, Name: "B", Weight: 50})

	list, err := store.GetVariantsByExperimentID(ctx, exp.ID)

	if err != nil {
		t.Fatalf("GetVariantsByExperimentID: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 variants, got %d", len(list))
	}
	names := make(map[string]int)
	for _, v := range list {
		if v.ExperimentID != exp.ID {
			t.Errorf("variant has wrong experiment_id: %s", v.ExperimentID)
		}
		names[v.Name] = v.Weight
	}
	if names["A"] != 50 || names["B"] != 50 {
		t.Errorf("expected A=50 and B=50, got %v", names)
	}
}

func TestPostgresStore_GetVariantsByExperimentID_no_variants_returns_empty_slice(t *testing.T) {
	database, cleanup := testDB(t)
	defer cleanup()
	truncateExperiments(t, database)
	store := NewPostgresStore(database.Conn())
	ctx := context.Background()
	exp, _ := store.CreateExperiment(ctx, &Experiment{Key: "empty-var", Environment: "dev"})

	list, err := store.GetVariantsByExperimentID(ctx, exp.ID)

	if err != nil {
		t.Fatalf("GetVariantsByExperimentID: %v", err)
	}
	if len(list) != 0 {
		t.Errorf("expected no variants, got len=%d", len(list))
	}
}

func TestPostgresStore_GetVariantsByExperimentID_invalid_uuid_returns_error(t *testing.T) {
	database, cleanup := testDB(t)
	defer cleanup()
	store := NewPostgresStore(database.Conn())
	ctx := context.Background()

	_, err := store.GetVariantsByExperimentID(ctx, "not-a-valid-uuid")

	if err == nil {
		t.Error("expected error for invalid UUID in experiment_id")
	}
}

func TestPostgresStore_GetAssignment_happy_path(t *testing.T) {
	database, cleanup := testDB(t)
	defer cleanup()
	truncateExperiments(t, database)
	store := NewPostgresStore(database.Conn())
	ctx := context.Background()
	userID := seedUserForAssignment(t, database)
	exp, _ := store.CreateExperiment(ctx, &Experiment{Key: "assign-exp", Environment: "dev"})
	v, _ := store.CreateVariant(ctx, &Variant{ExperimentID: exp.ID, Name: "B", Weight: 50})
	err := store.UpsertAssignment(ctx, &Assignment{UserID: userID, ExperimentID: exp.ID, VariantID: v.ID})
	if err != nil {
		t.Fatalf("UpsertAssignment setup: %v", err)
	}

	got, err := store.GetAssignment(ctx, userID, exp.ID)

	if err != nil {
		t.Fatalf("GetAssignment: %v", err)
	}
	if got == nil {
		t.Fatal("expected assignment")
	}
	if got.UserID != userID || got.ExperimentID != exp.ID || got.VariantID != v.ID {
		t.Errorf("unexpected assignment: %+v", got)
	}
}

func TestPostgresStore_GetAssignment_not_found_returns_nil_nil(t *testing.T) {
	database, cleanup := testDB(t)
	defer cleanup()
	truncateExperiments(t, database)
	store := NewPostgresStore(database.Conn())
	ctx := context.Background()
	userID := seedUserForAssignment(t, database)
	exp, _ := store.CreateExperiment(ctx, &Experiment{Key: "no-assign", Environment: "dev"})

	got, err := store.GetAssignment(ctx, userID, exp.ID)

	if err != nil {
		t.Fatalf("GetAssignment: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil, got %+v", got)
	}
}

func TestPostgresStore_UpsertAssignment_insert_new(t *testing.T) {
	database, cleanup := testDB(t)
	defer cleanup()
	truncateExperiments(t, database)
	store := NewPostgresStore(database.Conn())
	ctx := context.Background()
	userID := seedUserForAssignment(t, database)
	exp, _ := store.CreateExperiment(ctx, &Experiment{Key: "upsert-exp", Environment: "dev"})
	v, _ := store.CreateVariant(ctx, &Variant{ExperimentID: exp.ID, Name: "A", Weight: 100})
	a := &Assignment{UserID: userID, ExperimentID: exp.ID, VariantID: v.ID}

	err := store.UpsertAssignment(ctx, a)

	if err != nil {
		t.Fatalf("UpsertAssignment: %v", err)
	}
	got, _ := store.GetAssignment(ctx, userID, exp.ID)
	if got == nil || got.VariantID != v.ID {
		t.Errorf("expected assignment to be stored, got %+v", got)
	}
}

func TestPostgresStore_UpsertAssignment_update_existing(t *testing.T) {
	database, cleanup := testDB(t)
	defer cleanup()
	truncateExperiments(t, database)
	store := NewPostgresStore(database.Conn())
	ctx := context.Background()
	userID := seedUserForAssignment(t, database)
	exp, _ := store.CreateExperiment(ctx, &Experiment{Key: "upsert-update", Environment: "dev"})
	vA, _ := store.CreateVariant(ctx, &Variant{ExperimentID: exp.ID, Name: "A", Weight: 50})
	vB, _ := store.CreateVariant(ctx, &Variant{ExperimentID: exp.ID, Name: "B", Weight: 50})
	_ = store.UpsertAssignment(ctx, &Assignment{UserID: userID, ExperimentID: exp.ID, VariantID: vA.ID})

	err := store.UpsertAssignment(ctx, &Assignment{UserID: userID, ExperimentID: exp.ID, VariantID: vB.ID})

	if err != nil {
		t.Fatalf("UpsertAssignment: %v", err)
	}
	got, _ := store.GetAssignment(ctx, userID, exp.ID)
	if got == nil || got.VariantID != vB.ID {
		t.Errorf("expected assignment updated to variant B, got %+v", got)
	}
}

func TestPostgresStore_UpsertAssignment_nonexistent_user_id_returns_error(t *testing.T) {
	database, cleanup := testDB(t)
	defer cleanup()
	truncateExperiments(t, database)
	store := NewPostgresStore(database.Conn())
	ctx := context.Background()
	exp, _ := store.CreateExperiment(ctx, &Experiment{Key: "bad-user", Environment: "dev"})
	v, _ := store.CreateVariant(ctx, &Variant{ExperimentID: exp.ID, Name: "A", Weight: 100})
	a := &Assignment{UserID: "00000000-0000-0000-0000-000000000000", ExperimentID: exp.ID, VariantID: v.ID}

	err := store.UpsertAssignment(ctx, a)

	if err == nil {
		t.Fatal("expected error when user_id does not exist (FK violation)")
	}
}

func TestPostgresStore_UpsertAssignment_nonexistent_variant_id_returns_error(t *testing.T) {
	database, cleanup := testDB(t)
	defer cleanup()
	truncateExperiments(t, database)
	store := NewPostgresStore(database.Conn())
	ctx := context.Background()
	userID := seedUserForAssignment(t, database)
	exp, _ := store.CreateExperiment(ctx, &Experiment{Key: "bad-variant", Environment: "dev"})
	a := &Assignment{UserID: userID, ExperimentID: exp.ID, VariantID: "00000000-0000-0000-0000-000000000000"}

	err := store.UpsertAssignment(ctx, a)

	if err == nil {
		t.Fatal("expected error when variant_id does not exist (FK violation)")
	}
}

func TestPostgresStore_BeginTx_happy_path(t *testing.T) {
	database, cleanup := testDB(t)
	defer cleanup()
	store := NewPostgresStore(database.Conn())
	ctx := context.Background()

	tx, err := store.BeginTx(ctx)
	if err != nil {
		t.Fatalf("BeginTx: %v", err)
	}
	if err := tx.Rollback(); err != nil {
		t.Fatalf("rollback: %v", err)
	}
}

func TestPostgresStore_BeginTx_not_supported_on_tx_scoped_store(t *testing.T) {
	database, cleanup := testDB(t)
	defer cleanup()
	ctx := context.Background()
	root := NewPostgresStore(database.Conn())

	tx, err := root.BeginTx(ctx)
	if err != nil {
		t.Fatalf("BeginTx: %v", err)
	}
	defer func() { _ = tx.Rollback() }()

	scoped := root.WithTx(tx)
	ps, ok := scoped.(*PostgresStore)
	if !ok {
		t.Fatalf("expected *PostgresStore, got %T", scoped)
	}

	_, err = ps.BeginTx(ctx)
	if err == nil {
		t.Fatal("expected error")
	}
	want := "experiments store: BeginTx not supported on tx-scoped store"
	if err.Error() != want {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPostgresStore_BeginTx_begin_failure_returns_wrapped_operation_error(t *testing.T) {
	ctx := context.Background()
	wantErr := errors.New("db begin failed")
	badBegin := &stubBeginTxer{err: wantErr}
	store := &PostgresStore{exec: stubExecQuerier{}, begin: badBegin}

	_, err := store.BeginTx(ctx)
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected wrapped %v, got %v", wantErr, err)
	}
	var opErr *OperationError
	if !errors.As(err, &opErr) {
		t.Fatalf("expected *OperationError, got %T", err)
	}
	if opErr.Op != opRepoBeginTx {
		t.Fatalf("unexpected op %q", opErr.Op)
	}
}

// stubBeginTxer implements beginTxer for BeginTx error-path tests without a real DB.
type stubBeginTxer struct {
	err error
}

func (s *stubBeginTxer) BeginTx(_ context.Context, _ *sql.TxOptions) (*sql.Tx, error) {
	return nil, s.err
}

// stubExecQuerier satisfies execQuerier with no-ops (unused by BeginTx tests).
type stubExecQuerier struct{}

func (stubExecQuerier) ExecContext(context.Context, string, ...any) (sql.Result, error) {
	return nil, errors.New("stub: ExecContext not implemented")
}

func (stubExecQuerier) QueryContext(context.Context, string, ...any) (*sql.Rows, error) {
	return nil, errors.New("stub: QueryContext not implemented")
}

func (stubExecQuerier) QueryRowContext(context.Context, string, ...any) *sql.Row {
	return nil
}

func seedUserForAssignment(t *testing.T, database *db.DB) string {
	t.Helper()
	ctx := context.Background()
	var id string
	err := database.Conn().QueryRowContext(ctx,
		`INSERT INTO users (email, role) VALUES ($1, 'viewer') RETURNING id`,
		"assign-user@test.com",
	).Scan(&id)
	if err != nil {
		t.Fatalf("seed user: %v", err)
	}
	return id
}
