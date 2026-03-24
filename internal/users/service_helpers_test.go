package users

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/havlinj/featureflag-api/internal/audit"
	"github.com/havlinj/featureflag-api/internal/auth"
)

type minimalUsersTxAwareStore struct {
	gotTx *sql.Tx
}

func (s *minimalUsersTxAwareStore) Create(ctx context.Context, user *User) (*User, error) {
	return nil, nil
}
func (s *minimalUsersTxAwareStore) GetByID(ctx context.Context, id string) (*User, error) {
	return nil, nil
}
func (s *minimalUsersTxAwareStore) GetByEmail(ctx context.Context, email string) (*User, error) {
	return nil, nil
}
func (s *minimalUsersTxAwareStore) Update(ctx context.Context, user *User) error { return nil }
func (s *minimalUsersTxAwareStore) Delete(ctx context.Context, id string) error  { return nil }
func (s *minimalUsersTxAwareStore) WithTx(tx *sql.Tx) Store {
	s.gotTx = tx
	return s
}

type minimalUsersAuditStore struct {
	gotTx *sql.Tx
}

func (s *minimalUsersAuditStore) Create(ctx context.Context, entry *audit.Entry) error { return nil }
func (s *minimalUsersAuditStore) GetByID(ctx context.Context, id string) (*audit.Entry, error) {
	return nil, nil
}
func (s *minimalUsersAuditStore) List(ctx context.Context, filter audit.ListFilter, limit, offset int) ([]*audit.Entry, error) {
	return nil, nil
}
func (s *minimalUsersAuditStore) BeginTx(ctx context.Context) (*sql.Tx, error) { return &sql.Tx{}, nil }
func (s *minimalUsersAuditStore) WithTx(tx *sql.Tx) audit.Store {
	s.gotTx = tx
	return s
}

func TestPrepareAuditTx_Success_ConfiguresTxScopedStores(t *testing.T) {
	userStore := &minimalUsersTxAwareStore{}
	auditStore := &minimalUsersAuditStore{}
	svc := NewServiceWithAudit(userStore, auditStore)
	ctx := auth.WithActorID(context.Background(), "actor-1")

	out, err := svc.prepareAuditTx(ctx)

	if err != nil {
		t.Fatalf("prepareAuditTx: %v", err)
	}
	if out == nil || out.actorID != "actor-1" || out.tx == nil {
		t.Fatalf("unexpected audit tx context: %+v", out)
	}
	if userStore.gotTx == nil || auditStore.gotTx == nil {
		t.Fatal("expected both stores to receive tx via WithTx")
	}
}

func TestUserToModel_NilAndNonNil(t *testing.T) {
	if got := userToModel(nil); got != nil {
		t.Fatalf("expected nil output for nil user, got %+v", got)
	}

	createdAt := time.Date(2026, 3, 24, 10, 0, 0, 0, time.UTC)
	u := &User{ID: "u1", Email: "x@y.com", Role: RoleDeveloper, CreatedAt: createdAt}
	got := userToModel(u)
	if got == nil || got.ID != "u1" || got.Email != "x@y.com" || got.Role != "developer" {
		t.Fatalf("unexpected mapped user: %+v", got)
	}
	if got.CreatedAt != "2026-03-24T10:00:00Z" {
		t.Fatalf("unexpected CreatedAt: %q", got.CreatedAt)
	}
}
