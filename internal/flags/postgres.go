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
	conn *sql.DB
}

// NewPostgresStore returns a Store that uses the given *sql.DB (e.g. from db.DB.Conn()).
func NewPostgresStore(conn *sql.DB) *PostgresStore {
	return &PostgresStore{conn: conn}
}

// Create creates a new flag in the database. Key, Description, Enabled, Environment,
// and RolloutStrategy must be set; ID and CreatedAt are set by the DB.
// Returns ErrDuplicateKey if (key, environment) already exists.
func (p *PostgresStore) Create(ctx context.Context, flag *Flag) (*Flag, error) {
	var id string
	var createdAt time.Time
	err := p.conn.QueryRowContext(ctx,
		`INSERT INTO feature_flags (key, description, enabled, environment, rollout_strategy)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id, created_at`,
		flag.Key, flag.Description, flag.Enabled, flag.Environment, flag.RolloutStrategy,
	).Scan(&id, &createdAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, ErrDuplicateKey
		}
		return nil, err
	}
	out := *flag
	out.ID = id
	out.CreatedAt = createdAt
	return &out, nil
}

// GetByKeyAndEnvironment returns the flag for the given key and environment,
// or (nil, nil) if not found. Returns an error only on DB failure.
func (p *PostgresStore) GetByKeyAndEnvironment(ctx context.Context, key, environment string) (*Flag, error) {
	var f Flag
	var desc sql.NullString
	var strategy string
	err := p.conn.QueryRowContext(ctx,
		`SELECT id, key, description, enabled, environment, rollout_strategy, created_at
		 FROM feature_flags WHERE key = $1 AND environment = $2`,
		key, environment,
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

// Update updates an existing flag by ID. Returns ErrNotFound if no row was updated.
func (p *PostgresStore) Update(ctx context.Context, flag *Flag) error {
	res, err := p.conn.ExecContext(ctx,
		`UPDATE feature_flags SET key = $1, description = $2, enabled = $3, environment = $4, rollout_strategy = $5 WHERE id = $6`,
		flag.Key, flag.Description, flag.Enabled, flag.Environment, flag.RolloutStrategy, flag.ID,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// GetRulesByFlagID returns all rules for the given flag, or nil if none (no error).
func (p *PostgresStore) GetRulesByFlagID(ctx context.Context, flagID string) ([]*Rule, error) {
	rows, err := p.conn.QueryContext(ctx,
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
// Returns ErrNotFound if no row was deleted.
func (p *PostgresStore) Delete(ctx context.Context, id string) error {
	res, err := p.conn.ExecContext(ctx, `DELETE FROM feature_flags WHERE id = $1`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// ReplaceRulesByFlagID replaces all rules for the flag: deletes existing rules, then inserts new ones.
func (p *PostgresStore) ReplaceRulesByFlagID(ctx context.Context, flagID string, rules []*Rule) error {
	tx, err := p.conn.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	_, err = tx.ExecContext(ctx, `DELETE FROM flag_rules WHERE flag_id = $1`, flagID)
	if err != nil {
		return err
	}
	for _, r := range rules {
		_, err = tx.ExecContext(ctx,
			`INSERT INTO flag_rules (flag_id, type, value) VALUES ($1, $2, $3)`,
			flagID, r.Type, r.Value,
		)
		if err != nil {
			return err
		}
	}
	return tx.Commit()
}

// Ensure PostgresStore implements Store at compile time.
var _ Store = (*PostgresStore)(nil)
