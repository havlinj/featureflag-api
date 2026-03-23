package audit

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

// PostgresStore is the real persistence implementation for audit logs.
// It supports both *sql.DB and *sql.Tx through an internal executor interface.
type PostgresStore struct {
	exec  execQuerier
	begin beginTxer // optional; used only by non-tx instances
}

type execQuerier interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

type beginTxer interface {
	BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error)
}

// NewPostgresStore creates a Store backed by a PostgreSQL connection.
func NewPostgresStore(db *sql.DB) *PostgresStore {
	return &PostgresStore{exec: db, begin: db}
}

func newPostgresStoreWithTx(tx *sql.Tx) *PostgresStore {
	return &PostgresStore{exec: tx}
}

// BeginTx begins a transaction if this store was created from a *sql.DB.
// It is not part of the Store interface, but it is useful for domain-level atomicity.
func (p *PostgresStore) BeginTx(ctx context.Context) (*sql.Tx, error) {
	if p.begin == nil {
		return nil, errors.New("audit store: BeginTx not supported on tx-scoped store")
	}
	return p.begin.BeginTx(ctx, nil)
}

// WithTx returns a tx-scoped store instance.
func (p *PostgresStore) WithTx(tx *sql.Tx) Store {
	return newPostgresStoreWithTx(tx)
}

func (p *PostgresStore) Create(ctx context.Context, entry *Entry) error {
	if entry == nil {
		return errors.New("audit store: entry is nil")
	}
	_, err := p.exec.ExecContext(ctx,
		`INSERT INTO audit_logs (entity, entity_id, action, actor_id)
		 VALUES ($1, $2, $3, $4)`,
		entry.Entity, entry.EntityID, entry.Action, entry.ActorID,
	)
	return err
}

func (p *PostgresStore) GetByID(ctx context.Context, id string) (*Entry, error) {
	var out Entry
	var actorID sql.NullString
	var createdAt time.Time
	err := p.exec.QueryRowContext(ctx,
		`SELECT id, entity, entity_id, action, actor_id, created_at
		 FROM audit_logs WHERE id = $1`,
		id,
	).Scan(&out.ID, &out.Entity, &out.EntityID, &out.Action, &actorID, &createdAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	out.ActorID = actorID.String
	out.CreatedAt = createdAt
	return &out, nil
}

func (p *PostgresStore) List(ctx context.Context, filter ListFilter, limit, offset int) ([]*Entry, error) {
	if limit <= 0 {
		limit = DefaultListLimit
	}
	if limit > MaxListLimit {
		limit = MaxListLimit
	}
	if offset < 0 {
		return nil, fmt.Errorf("audit store: %w", ErrNegativeOffset)
	}

	conds := make([]string, 0, 3)
	args := make([]any, 0, 4)
	i := 1

	if filter.Entity != nil {
		conds = append(conds, fmt.Sprintf("entity = $%d", i))
		args = append(args, *filter.Entity)
		i++
	}
	if filter.Action != nil {
		conds = append(conds, fmt.Sprintf("action = $%d", i))
		args = append(args, *filter.Action)
		i++
	}
	if filter.ActorID != nil {
		conds = append(conds, fmt.Sprintf("actor_id = $%d", i))
		args = append(args, *filter.ActorID)
		i++
	}

	where := ""
	if len(conds) > 0 {
		where = " WHERE " + strings.Join(conds, " AND ")
	}

	// Stable order for deterministic tests.
	query := fmt.Sprintf(`
		SELECT id, entity, entity_id, action, actor_id, created_at
		FROM audit_logs%s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d`,
		where, i, i+1,
	)
	args = append(args, limit, offset)

	rows, err := p.exec.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*Entry
	for rows.Next() {
		var e Entry
		var actorID sql.NullString
		if err := rows.Scan(&e.ID, &e.Entity, &e.EntityID, &e.Action, &actorID, &e.CreatedAt); err != nil {
			return nil, err
		}
		e.ActorID = actorID.String
		out = append(out, &e)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

var _ Store = (*PostgresStore)(nil)
var _ TxAwareStore = (*PostgresStore)(nil)
