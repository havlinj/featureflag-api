package auditmock

import (
	"context"
	"database/sql"

	"github.com/havlinj/featureflag-api/internal/audit"
)

type TxStarter struct {
	BeginErr error
}

func (m *TxStarter) Create(ctx context.Context, entry *audit.Entry) error {
	return nil
}

func (m *TxStarter) GetByID(ctx context.Context, id string) (*audit.Entry, error) {
	return nil, nil
}

func (m *TxStarter) List(ctx context.Context, filter audit.ListFilter, limit, offset int) ([]*audit.Entry, error) {
	return nil, nil
}

func (m *TxStarter) BeginTx(ctx context.Context) (*sql.Tx, error) {
	return nil, m.BeginErr
}

type TxAware struct {
	TxStarter
}

func (m *TxAware) WithTx(tx *sql.Tx) audit.Store {
	return m
}
