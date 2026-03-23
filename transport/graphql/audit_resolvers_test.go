package graphql

import (
	"context"
	"errors"
	"testing"

	"github.com/havlinj/featureflag-api/internal/audit"
	"github.com/havlinj/featureflag-api/internal/auth"
)

type auditResolverMockStore struct {
	getByIDFunc func(ctx context.Context, id string) (*audit.Entry, error)
	listFunc    func(ctx context.Context, filter audit.ListFilter, limit, offset int) ([]*audit.Entry, error)
}

func (m *auditResolverMockStore) Create(ctx context.Context, entry *audit.Entry) error { return nil }
func (m *auditResolverMockStore) GetByID(ctx context.Context, id string) (*audit.Entry, error) {
	return m.getByIDFunc(ctx, id)
}
func (m *auditResolverMockStore) List(ctx context.Context, filter audit.ListFilter, limit, offset int) ([]*audit.Entry, error) {
	return m.listFunc(ctx, filter, limit, offset)
}

func TestAuditLog_resolver_requires_auth(t *testing.T) {
	r := &Resolver{Audit: nil}
	q := &queryResolver{r}

	_, err := q.AuditLog(context.Background(), "id-1")

	if err == nil {
		t.Fatal("expected auth error")
	}
	var authErr *auth.UnauthorizedError
	if !errors.As(err, &authErr) {
		t.Fatalf("expected UnauthorizedError, got %T %v", err, err)
	}
}

func TestAuditLog_resolver_returns_nil_when_not_found(t *testing.T) {
	mock := &auditResolverMockStore{
		getByIDFunc: func(ctx context.Context, id string) (*audit.Entry, error) { return nil, nil },
		listFunc:    func(ctx context.Context, filter audit.ListFilter, limit, offset int) ([]*audit.Entry, error) { return nil, nil },
	}
	r := &Resolver{Audit: audit.NewService(mock)}
	q := &queryResolver{r}
	ctx := auth.WithClaims(context.Background(), &auth.Claims{Sub: "u1", Role: "admin"})

	got, err := q.AuditLog(ctx, "missing")

	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if got != nil {
		t.Fatalf("expected nil, got %+v", got)
	}
}

func TestAuditLogs_resolver_rejects_negative_offset(t *testing.T) {
	mock := &auditResolverMockStore{
		getByIDFunc: func(ctx context.Context, id string) (*audit.Entry, error) { return nil, nil },
		listFunc:    func(ctx context.Context, filter audit.ListFilter, limit, offset int) ([]*audit.Entry, error) { return nil, nil },
	}
	r := &Resolver{Audit: audit.NewService(mock)}
	q := &queryResolver{r}
	ctx := auth.WithClaims(context.Background(), &auth.Claims{Sub: "u1", Role: "admin"})
	offset := -1

	_, err := q.AuditLogs(ctx, nil, nil, &offset)

	if err == nil {
		t.Fatal("expected error")
	}
	if err.Error() != "offset must be >= 0" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAuditLogs_resolver_uses_defaults_and_maps_entries(t *testing.T) {
	called := false
	mock := &auditResolverMockStore{
		getByIDFunc: func(ctx context.Context, id string) (*audit.Entry, error) { return nil, nil },
		listFunc: func(ctx context.Context, filter audit.ListFilter, limit, offset int) ([]*audit.Entry, error) {
			called = true
			if limit != 50 || offset != 0 {
				t.Fatalf("expected defaults limit=50 offset=0, got limit=%d offset=%d", limit, offset)
			}
			return []*audit.Entry{{ID: "a1", Entity: "feature_flag", EntityID: "f1", Action: "create", ActorID: "u1"}}, nil
		},
	}
	r := &Resolver{Audit: audit.NewService(mock)}
	q := &queryResolver{r}
	ctx := auth.WithClaims(context.Background(), &auth.Claims{Sub: "u1", Role: "admin"})

	got, err := q.AuditLogs(ctx, nil, nil, nil)

	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if !called {
		t.Fatal("expected list to be called")
	}
	if len(got) != 1 || got[0].ID != "a1" || got[0].Entity != "feature_flag" || got[0].EntityID != "f1" {
		t.Fatalf("unexpected response: %+v", got)
	}
}

