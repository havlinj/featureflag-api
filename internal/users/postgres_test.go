//go:build integration

package users

import (
	"context"
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

func truncateUsers(t *testing.T, database *db.DB) {
	t.Helper()
	ctx := context.Background()
	if _, err := database.Conn().ExecContext(ctx, "TRUNCATE users CASCADE"); err != nil {
		t.Fatalf("truncate: %v", err)
	}
}

func TestPostgresStore_Create_happy_path(t *testing.T) {
	database, cleanup := testDB(t)
	defer cleanup()
	truncateUsers(t, database)
	store := NewPostgresStore(database.Conn())
	ctx := context.Background()
	user := &User{Email: "admin@test.com", Role: RoleAdmin}

	created, err := store.Create(ctx, user)

	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if created.ID == "" {
		t.Error("expected ID to be set")
	}
	if created.Email != "admin@test.com" || created.Role != RoleAdmin {
		t.Errorf("unexpected user: %+v", created)
	}
}

func TestPostgresStore_Create_duplicate_email_returns_ErrDuplicateEmail(t *testing.T) {
	database, cleanup := testDB(t)
	defer cleanup()
	truncateUsers(t, database)
	store := NewPostgresStore(database.Conn())
	ctx := context.Background()
	user := &User{Email: "dup@test.com", Role: RoleViewer}
	_, err := store.Create(ctx, user)
	if err != nil {
		t.Fatalf("first Create: %v", err)
	}

	_, err = store.Create(ctx, user)

	if err == nil {
		t.Fatal("expected error on duplicate email")
	}
	var e *DuplicateEmailError
	if !errors.As(err, &e) {
		t.Errorf("expected *DuplicateEmailError, got %v", err)
	}
	if e.Email != "dup@test.com" {
		t.Errorf("expected Email=dup@test.com, got %q", e.Email)
	}
}

func TestPostgresStore_GetByID_happy_path(t *testing.T) {
	database, cleanup := testDB(t)
	defer cleanup()
	truncateUsers(t, database)
	store := NewPostgresStore(database.Conn())
	ctx := context.Background()
	created, err := store.Create(ctx, &User{Email: "get@test.com", Role: RoleDeveloper})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := store.GetByID(ctx, created.ID)

	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got == nil || got.ID != created.ID || got.Email != "get@test.com" {
		t.Errorf("unexpected: %+v", got)
	}
}

func TestPostgresStore_GetByID_not_found_returns_nil_nil(t *testing.T) {
	database, cleanup := testDB(t)
	defer cleanup()
	truncateUsers(t, database)
	store := NewPostgresStore(database.Conn())
	ctx := context.Background()

	got, err := store.GetByID(ctx, "00000000-0000-0000-0000-000000000000")

	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil, got %+v", got)
	}
}

func TestPostgresStore_GetByEmail_happy_path(t *testing.T) {
	database, cleanup := testDB(t)
	defer cleanup()
	truncateUsers(t, database)
	store := NewPostgresStore(database.Conn())
	ctx := context.Background()
	created, _ := store.Create(ctx, &User{Email: "email@test.com", Role: RoleAdmin})

	got, err := store.GetByEmail(ctx, "email@test.com")

	if err != nil {
		t.Fatalf("GetByEmail: %v", err)
	}
	if got == nil || got.ID != created.ID || got.Email != "email@test.com" {
		t.Errorf("unexpected: %+v", got)
	}
}

func TestPostgresStore_Update_happy_path(t *testing.T) {
	database, cleanup := testDB(t)
	defer cleanup()
	truncateUsers(t, database)
	store := NewPostgresStore(database.Conn())
	ctx := context.Background()
	created, _ := store.Create(ctx, &User{Email: "old@test.com", Role: RoleViewer})
	created.Email = "new@test.com"
	created.Role = RoleDeveloper

	err := store.Update(ctx, created)

	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	got, _ := store.GetByID(ctx, created.ID)
	if got.Email != "new@test.com" || got.Role != RoleDeveloper {
		t.Errorf("unexpected after update: %+v", got)
	}
}

func TestPostgresStore_Update_not_found_returns_ErrNotFound(t *testing.T) {
	database, cleanup := testDB(t)
	defer cleanup()
	store := NewPostgresStore(database.Conn())
	ctx := context.Background()
	user := &User{ID: "00000000-0000-0000-0000-000000000000", Email: "x@y.com", Role: RoleViewer}

	err := store.Update(ctx, user)

	if err == nil {
		t.Fatal("expected error")
	}
	var e *NotFoundError
	if !errors.As(err, &e) {
		t.Errorf("expected *NotFoundError, got %v", err)
	}
	if e.ID != "00000000-0000-0000-0000-000000000000" {
		t.Errorf("expected ID=00000000-0000-0000-0000-000000000000, got %q", e.ID)
	}
}

func TestPostgresStore_Delete_happy_path(t *testing.T) {
	database, cleanup := testDB(t)
	defer cleanup()
	truncateUsers(t, database)
	store := NewPostgresStore(database.Conn())
	ctx := context.Background()
	created, _ := store.Create(ctx, &User{Email: "del@test.com", Role: RoleViewer})

	err := store.Delete(ctx, created.ID)

	if err != nil {
		t.Fatalf("Delete: %v", err)
	}
	got, _ := store.GetByID(ctx, created.ID)
	if got != nil {
		t.Errorf("expected nil after delete, got %+v", got)
	}
}

func TestPostgresStore_Delete_not_found_returns_ErrNotFound(t *testing.T) {
	database, cleanup := testDB(t)
	defer cleanup()
	store := NewPostgresStore(database.Conn())
	ctx := context.Background()

	err := store.Delete(ctx, "00000000-0000-0000-0000-000000000000")

	if err == nil {
		t.Fatal("expected error")
	}
	var e *NotFoundError
	if !errors.As(err, &e) {
		t.Errorf("expected *NotFoundError, got %v", err)
	}
	if e.ID != "00000000-0000-0000-0000-000000000000" {
		t.Errorf("expected ID=00000000-0000-0000-0000-000000000000, got %q", e.ID)
	}
}
