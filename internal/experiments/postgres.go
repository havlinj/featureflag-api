package experiments

import (
	"context"
	"database/sql"
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
)

// PostgresStore implements Store using PostgreSQL.
type PostgresStore struct {
	conn *sql.DB
}

// NewPostgresStore returns a Store that uses the given *sql.DB.
func NewPostgresStore(conn *sql.DB) *PostgresStore {
	return &PostgresStore{conn: conn}
}

// CreateExperiment inserts a new experiment. Returns *DuplicateExperimentError if (key, environment) exists.
func (p *PostgresStore) CreateExperiment(ctx context.Context, exp *Experiment) (*Experiment, error) {
	var id string
	err := p.conn.QueryRowContext(ctx,
		`INSERT INTO experiments (key, environment) VALUES ($1, $2) RETURNING id`,
		exp.Key, exp.Environment,
	).Scan(&id)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, &DuplicateExperimentError{Key: exp.Key, Environment: exp.Environment}
		}
		return nil, err
	}
	out := *exp
	out.ID = id
	return &out, nil
}

// GetExperimentByKeyAndEnvironment returns the experiment or (nil, nil) if not found.
func (p *PostgresStore) GetExperimentByKeyAndEnvironment(ctx context.Context, key, environment string) (*Experiment, error) {
	var exp Experiment
	err := p.conn.QueryRowContext(ctx,
		`SELECT id, key, environment FROM experiments WHERE key = $1 AND environment = $2`,
		key, environment,
	).Scan(&exp.ID, &exp.Key, &exp.Environment)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &exp, nil
}

// GetExperimentByID returns the experiment by ID or (nil, nil) if not found.
func (p *PostgresStore) GetExperimentByID(ctx context.Context, id string) (*Experiment, error) {
	var exp Experiment
	err := p.conn.QueryRowContext(ctx,
		`SELECT id, key, environment FROM experiments WHERE id = $1`,
		id,
	).Scan(&exp.ID, &exp.Key, &exp.Environment)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &exp, nil
}

// CreateVariant inserts a variant for an experiment.
func (p *PostgresStore) CreateVariant(ctx context.Context, v *Variant) (*Variant, error) {
	var id string
	err := p.conn.QueryRowContext(ctx,
		`INSERT INTO experiment_variants (experiment_id, name, weight) VALUES ($1, $2, $3) RETURNING id`,
		v.ExperimentID, v.Name, v.Weight,
	).Scan(&id)
	if err != nil {
		return nil, err
	}
	out := *v
	out.ID = id
	return &out, nil
}

// GetVariantsByExperimentID returns all variants for an experiment.
func (p *PostgresStore) GetVariantsByExperimentID(ctx context.Context, experimentID string) ([]*Variant, error) {
	rows, err := p.conn.QueryContext(ctx,
		`SELECT id, experiment_id, name, weight FROM experiment_variants WHERE experiment_id = $1 ORDER BY id`,
		experimentID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*Variant
	for rows.Next() {
		var v Variant
		if err := rows.Scan(&v.ID, &v.ExperimentID, &v.Name, &v.Weight); err != nil {
			return nil, err
		}
		list = append(list, &v)
	}
	return list, rows.Err()
}

// GetAssignment returns the user's assignment for an experiment or (nil, nil) if not assigned.
func (p *PostgresStore) GetAssignment(ctx context.Context, userID, experimentID string) (*Assignment, error) {
	var a Assignment
	err := p.conn.QueryRowContext(ctx,
		`SELECT user_id, experiment_id, variant_id FROM experiment_assignments WHERE user_id = $1 AND experiment_id = $2`,
		userID, experimentID,
	).Scan(&a.UserID, &a.ExperimentID, &a.VariantID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &a, nil
}

// UpsertAssignment inserts or updates the assignment (user_id, experiment_id) -> variant_id.
func (p *PostgresStore) UpsertAssignment(ctx context.Context, a *Assignment) error {
	_, err := p.conn.ExecContext(ctx,
		`INSERT INTO experiment_assignments (user_id, experiment_id, variant_id)
		 VALUES ($1, $2, $3)
		 ON CONFLICT (user_id, experiment_id) DO UPDATE SET variant_id = $3`,
		a.UserID, a.ExperimentID, a.VariantID,
	)
	return err
}
