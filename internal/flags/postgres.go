package flags

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
)

// PostgresStore is the real persistence implementation using PostgreSQL.
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

// NewPostgresStore returns a Store that uses the given *sql.DB (e.g. from db.DB.Conn()).
func NewPostgresStore(conn *sql.DB) *PostgresStore {
	return &PostgresStore{exec: conn, begin: conn}
}

func newPostgresStoreWithTx(tx *sql.Tx) *PostgresStore {
	return &PostgresStore{exec: tx}
}

// WithTx returns a tx-scoped Store.
func (p *PostgresStore) WithTx(tx *sql.Tx) Store {
	return newPostgresStoreWithTx(tx)
}

// Create creates a new flag in the database. Key, Description, Enabled, Environment,
// and RolloutStrategy must be set; ID and CreatedAt are set by the DB.
// Returns *DuplicateKeyError if (key, environment) already exists.
func (p *PostgresStore) Create(ctx context.Context, flag *Flag) (*Flag, error) {
	var id string
	var createdAt time.Time
	err := p.exec.QueryRowContext(ctx,
		`INSERT INTO feature_flags (key, description, enabled, environment, rollout_strategy)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id, created_at`,
		flag.Key, flag.Description, flag.Enabled, flag.Environment, flag.RolloutStrategy,
	).Scan(&id, &createdAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, &DuplicateKeyError{Key: flag.Key, Environment: string(flag.Environment)}
		}
		return nil, err
	}
	out := *flag
	out.ID = id
	out.CreatedAt = createdAt
	return &out, nil
}

// GetByKeyAndEnvironment returns the flag for the given key and deployment stage,
// or (nil, nil) if not found. Returns an error only on DB failure.
func (p *PostgresStore) GetByKeyAndEnvironment(ctx context.Context, key string, env DeploymentStage) (*Flag, error) {
	var f Flag
	var desc sql.NullString
	var strategy string
	err := p.exec.QueryRowContext(ctx,
		`SELECT id, key, description, enabled, environment, rollout_strategy, created_at
		 FROM feature_flags WHERE key = $1 AND environment = $2`,
		key, env,
	).Scan(&f.ID, &f.Key, &desc, &f.Enabled, &f.Environment, &strategy, &f.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if desc.Valid {
		f.Description = &desc.String
	}
	f.RolloutStrategy = RolloutStrategy(strategy)
	return &f, nil
}

// Update updates an existing flag by ID. Returns *NotFoundError if no row was updated.
func (p *PostgresStore) Update(ctx context.Context, flag *Flag) error {
	res, err := p.exec.ExecContext(ctx,
		`UPDATE feature_flags SET key = $1, description = $2, enabled = $3, environment = $4, rollout_strategy = $5 WHERE id = $6`,
		flag.Key, flag.Description, flag.Enabled, flag.Environment, flag.RolloutStrategy, flag.ID,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return &NotFoundError{ID: flag.ID}
	}
	return nil
}

// GetRulesByFlagID returns all rules for the given flag, or nil if none (no error).
func (p *PostgresStore) GetRulesByFlagID(ctx context.Context, flagID string) ([]*Rule, error) {
	rows, err := p.exec.QueryContext(ctx,
		`SELECT id, flag_id, type, value FROM flag_rules WHERE flag_id = $1 ORDER BY id`,
		flagID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var rules []*Rule
	for rows.Next() {
		var r Rule
		if err := rows.Scan(&r.ID, &r.FlagID, &r.Type, &r.Value); err != nil {
			return nil, err
		}
		rules = append(rules, &r)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return rules, nil
}

// Delete removes a flag by ID. Rules are removed by DB ON DELETE CASCADE.
// Returns *NotFoundError if no row was deleted.
func (p *PostgresStore) Delete(ctx context.Context, id string) error {
	res, err := p.exec.ExecContext(ctx, `DELETE FROM feature_flags WHERE id = $1`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return &NotFoundError{ID: id}
	}
	return nil
}

// ReplaceRulesByFlagID replaces all rules for the flag: deletes existing rules, then inserts new ones.
func (p *PostgresStore) ReplaceRulesByFlagID(ctx context.Context, flagID string, rules []*Rule) error {
	// If this store is tx-scoped (begin == nil), we must not start a nested transaction.
	if p.begin == nil {
		if _, err := p.exec.ExecContext(ctx, `DELETE FROM flag_rules WHERE flag_id = $1`, flagID); err != nil {
			return err
		}
		for _, r := range rules {
			if _, err := p.exec.ExecContext(ctx,
				`INSERT INTO flag_rules (flag_id, type, value) VALUES ($1, $2, $3)`,
				flagID, r.Type, r.Value,
			); err != nil {
				return err
			}
		}
		return nil
	}

	// Otherwise, start a transaction (standalone usage).
	tx, err := p.begin.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	txStore := newPostgresStoreWithTx(tx)
	if err := txStore.ReplaceRulesByFlagID(ctx, flagID, rules); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	committed = true
	return nil
}

// Ensure PostgresStore implements Store at compile time.
var _ Store = (*PostgresStore)(nil)
