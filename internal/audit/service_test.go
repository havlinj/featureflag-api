package audit_test

import (
	"context"
	"errors"
	"testing"

	"github.com/havlinj/featureflag-api/internal/audit"
)

type mockStore struct {
	getByIDFunc func(ctx context.Context, id string) (*audit.Entry, error)
	listFunc    func(ctx context.Context, filter audit.ListFilter, limit, offset int) ([]*audit.Entry, error)
}

func (m *mockStore) Create(ctx context.Context, entry *audit.Entry) error { return nil }
func (m *mockStore) GetByID(ctx context.Context, id string) (*audit.Entry, error) {
	return m.getByIDFunc(ctx, id)
}
func (m *mockStore) List(ctx context.Context, filter audit.ListFilter, limit, offset int) ([]*audit.Entry, error) {
	return m.listFunc(ctx, filter, limit, offset)
}

func TestService_GetByID_delegates_to_store(t *testing.T) {
	ctx := context.Background()
	store := &mockStore{
		getByIDFunc: func(ctx context.Context, id string) (*audit.Entry, error) {
			return &audit.Entry{ID: id, Entity: "feature_flag"}, nil
		},
		listFunc: func(ctx context.Context, filter audit.ListFilter, limit, offset int) ([]*audit.Entry, error) {
			return nil, nil
		},
	}
	svc := audit.NewService(store)

	got, err := svc.GetByID(ctx, "a1")

	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got == nil || got.ID != "a1" {
		t.Fatalf("unexpected result: %+v", got)
	}
}

func TestService_List_returns_error_from_store(t *testing.T) {
	ctx := context.Background()
	wantErr := errors.New("list failed")
	store := &mockStore{
		getByIDFunc: func(ctx context.Context, id string) (*audit.Entry, error) {
			return nil, nil
		},
		listFunc: func(ctx context.Context, filter audit.ListFilter, limit, offset int) ([]*audit.Entry, error) {
			return nil, wantErr
		},
	}
	svc := audit.NewService(store)

	_, err := svc.List(ctx, audit.ListFilter{}, 50, 0)

	if !errors.Is(err, wantErr) {
		t.Fatalf("expected %v, got %v", wantErr, err)
	}
}
