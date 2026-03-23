package audit_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/havlinj/featureflag-api/internal/audit"
	"github.com/havlinj/featureflag-api/internal/auth"
)

type txStarterOnlyMock struct {
	beginErr error
}

func (m *txStarterOnlyMock) Create(ctx context.Context, entry *audit.Entry) error {
	return nil
}

func (m *txStarterOnlyMock) GetByID(ctx context.Context, id string) (*audit.Entry, error) {
	return nil, nil
}

func (m *txStarterOnlyMock) List(ctx context.Context, filter audit.ListFilter, limit, offset int) ([]*audit.Entry, error) {
	return nil, nil
}

func (m *txStarterOnlyMock) BeginTx(ctx context.Context) (*sql.Tx, error) {
	return nil, m.beginErr
}

type txAwareOnlyMock struct{}

func (m *txAwareOnlyMock) Create(ctx context.Context, entry *audit.Entry) error {
	return nil
}

func (m *txAwareOnlyMock) GetByID(ctx context.Context, id string) (*audit.Entry, error) {
	return nil, nil
}

func (m *txAwareOnlyMock) List(ctx context.Context, filter audit.ListFilter, limit, offset int) ([]*audit.Entry, error) {
	return nil, nil
}

func (m *txAwareOnlyMock) WithTx(tx *sql.Tx) audit.Store {
	return m
}

func TestPrepareWriteTx_missing_actor_returns_error(t *testing.T) {
	_, _, _, err := audit.PrepareWriteTx(context.Background(), &txAwareOnlyMock{})

	if err == nil || err.Error() != "audit: missing actor id in context" {
		t.Fatalf("expected missing actor error, got %v", err)
	}
}

func TestPrepareWriteTx_missing_tx_starter_returns_error(t *testing.T) {
	ctx := auth.WithActorID(context.Background(), "u1")

	_, _, _, err := audit.PrepareWriteTx(ctx, &txAwareOnlyMock{})

	if err == nil || err.Error() != "audit: audit store cannot start transactions" {
		t.Fatalf("expected tx starter error, got %v", err)
	}
}

func TestPrepareWriteTx_missing_tx_aware_returns_error(t *testing.T) {
	ctx := auth.WithActorID(context.Background(), "u1")

	_, _, _, err := audit.PrepareWriteTx(ctx, &txStarterOnlyMock{})

	if err == nil || err.Error() != "audit: audit store is not tx-aware" {
		t.Fatalf("expected tx-aware error, got %v", err)
	}
}

func TestPrepareWriteTx_begin_tx_error_is_returned(t *testing.T) {
	ctx := auth.WithActorID(context.Background(), "u1")
	wantErr := errors.New("begin tx failed")
	store := &auditTxAwareStarterMock{txStarterOnlyMock: txStarterOnlyMock{beginErr: wantErr}}

	_, _, _, err := audit.PrepareWriteTx(ctx, store)

	if !errors.Is(err, wantErr) {
		t.Fatalf("expected %v, got %v", wantErr, err)
	}
}

type auditTxAwareStarterMock struct {
	txStarterOnlyMock
}

func (m *auditTxAwareStarterMock) WithTx(tx *sql.Tx) audit.Store {
	return m
}
