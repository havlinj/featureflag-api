//go:build integration

package audit_test

import (
	"context"
	"errors"
	"testing"

	"github.com/havlinj/featureflag-api/internal/audit"
	"github.com/havlinj/featureflag-api/internal/testutil"
)

func TestPostgresStore_Create_and_GetByID(t *testing.T) {
	database, cleanup := testutil.PostgresForIntegration(t)
	defer cleanup()
	testutil.TruncateAll(t, database)

	ctx := context.Background()
	conn := database.Conn()
	store := audit.NewPostgresStore(conn)

	_, err := conn.ExecContext(ctx, "INSERT INTO users (id, email, role) VALUES ($1, $2, $3)", "11111111-1111-1111-1111-111111111111", "audit-admin@test.com", "admin")
	if err != nil {
		t.Fatalf("seed actor user: %v", err)
	}

	err = store.Create(ctx, &audit.Entry{
		Entity:   "feature_flag",
		EntityID: "flag-1",
		Action:   "create",
		ActorID:  "11111111-1111-1111-1111-111111111111",
	})
	if err != nil {
		t.Fatalf("create audit entry: %v", err)
	}

	var id string
	err = conn.QueryRowContext(ctx, "SELECT id FROM audit_logs LIMIT 1").Scan(&id)
	if err != nil {
		t.Fatalf("query audit id: %v", err)
	}

	got, err := store.GetByID(ctx, id)
	if err != nil {
		t.Fatalf("get by id: %v", err)
	}
	if got == nil {
		t.Fatal("expected audit entry, got nil")
	}
	if got.Entity != "feature_flag" || got.EntityID != "flag-1" || got.Action != "create" || got.ActorID != "11111111-1111-1111-1111-111111111111" {
		t.Fatalf("unexpected audit entry: %+v", got)
	}
}

func TestPostgresStore_List_filters_by_entity(t *testing.T) {
	database, cleanup := testutil.PostgresForIntegration(t)
	defer cleanup()
	testutil.TruncateAll(t, database)

	ctx := context.Background()
	conn := database.Conn()
	store := audit.NewPostgresStore(conn)

	_, err := conn.ExecContext(ctx, "INSERT INTO users (id, email, role) VALUES ($1, $2, $3)", "22222222-2222-2222-2222-222222222222", "audit-dev@test.com", "developer")
	if err != nil {
		t.Fatalf("seed actor user: %v", err)
	}

	err = store.Create(ctx, &audit.Entry{
		Entity:   "feature_flag",
		EntityID: "flag-1",
		Action:   "create",
		ActorID:  "22222222-2222-2222-2222-222222222222",
	})
	if err != nil {
		t.Fatalf("create audit feature flag: %v", err)
	}
	err = store.Create(ctx, &audit.Entry{
		Entity:   "user",
		EntityID: "user-1",
		Action:   "update",
		ActorID:  "22222222-2222-2222-2222-222222222222",
	})
	if err != nil {
		t.Fatalf("create audit user: %v", err)
	}

	entity := "feature_flag"
	list, err := store.List(ctx, audit.ListFilter{Entity: &entity}, 10, 0)
	if err != nil {
		t.Fatalf("list audit: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(list))
	}
	if list[0].Entity != "feature_flag" {
		t.Fatalf("unexpected entity: %s", list[0].Entity)
	}
}

func TestPostgresStore_Create_nil_entry_returns_error(t *testing.T) {
	store := &audit.PostgresStore{}
	err := store.Create(context.Background(), nil)
	var e *audit.NilEntryError
	if !errors.As(err, &e) {
		t.Fatalf("expected *audit.NilEntryError, got %T (%v)", err, err)
	}
}

func TestPostgresStore_List_negative_offset_returns_error(t *testing.T) {
	store := &audit.PostgresStore{}

	_, err := store.List(context.Background(), audit.ListFilter{}, audit.DefaultListLimit, -1)

	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, audit.ErrNegativeOffset) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPostgresStore_BeginTx_on_tx_scoped_store_returns_error(t *testing.T) {
	database, cleanup := testutil.PostgresForIntegration(t)
	defer cleanup()
	testutil.TruncateAll(t, database)

	ctx := context.Background()
	tx, err := database.Conn().BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("begin tx: %v", err)
	}
	defer func() { _ = tx.Rollback() }()

	root := audit.NewPostgresStore(database.Conn())
	txStore, ok := root.WithTx(tx).(*audit.PostgresStore)
	if !ok {
		t.Fatal("expected tx-scoped *audit.PostgresStore")
	}
	_, err = txStore.BeginTx(ctx)
	var e *audit.BeginTxUnsupportedError
	if !errors.As(err, &e) {
		t.Fatalf("expected *audit.BeginTxUnsupportedError, got %T (%v)", err, err)
	}
}
