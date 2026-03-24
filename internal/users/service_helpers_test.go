package users

import (
	"context"
	"database/sql"
	"errors"
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

type minimalUsersNonTxStore struct{}

func (s *minimalUsersNonTxStore) Create(ctx context.Context, user *User) (*User, error) {
	return nil, nil
}
func (s *minimalUsersNonTxStore) GetByID(ctx context.Context, id string) (*User, error) {
	return nil, nil
}
func (s *minimalUsersNonTxStore) GetByEmail(ctx context.Context, email string) (*User, error) {
	return nil, nil
}
func (s *minimalUsersNonTxStore) Update(ctx context.Context, user *User) error { return nil }
func (s *minimalUsersNonTxStore) Delete(ctx context.Context, id string) error  { return nil }

type minimalUsersAuditStore struct {
	gotTx    *sql.Tx
	beginErr error
}

func (s *minimalUsersAuditStore) Create(ctx context.Context, entry *audit.Entry) error { return nil }
func (s *minimalUsersAuditStore) GetByID(ctx context.Context, id string) (*audit.Entry, error) {
	return nil, nil
}
func (s *minimalUsersAuditStore) List(ctx context.Context, filter audit.ListFilter, limit, offset int) ([]*audit.Entry, error) {
	return nil, nil
}
func (s *minimalUsersAuditStore) BeginTx(ctx context.Context) (*sql.Tx, error) {
	if s.beginErr != nil {
		return nil, s.beginErr
	}
	return &sql.Tx{}, nil
}
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

func TestPrepareAuditTx_StoreNotTxAware_ReturnsError(t *testing.T) {
	svc := NewServiceWithAudit(&minimalUsersNonTxStore{}, &minimalUsersAuditStore{})

	out, err := svc.prepareAuditTx(context.Background())

	if err == nil {
		t.Fatal("expected error for non tx-aware store")
	}
	if out != nil {
		t.Fatalf("expected nil audit tx context, got %+v", out)
	}
}

func TestPrepareAuditTx_BeginTxError_Propagates(t *testing.T) {
	expected := errors.New("begin tx failed")
	userStore := &minimalUsersTxAwareStore{}
	auditStore := &minimalUsersAuditStore{beginErr: expected}
	svc := NewServiceWithAudit(userStore, auditStore)
	ctx := auth.WithActorID(context.Background(), "actor-1")

	out, err := svc.prepareAuditTx(ctx)

	if out != nil {
		t.Fatalf("expected nil context, got %+v", out)
	}
	if !errors.Is(err, expected) {
		t.Fatalf("expected begin error, got %v", err)
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
