package experiments

import (
	"context"
	"database/sql"
)

// Store is the persistence interface for the experiments domain.
// Implemented by PostgresStore and by a mock for unit tests.
type Store interface {
	// CreateExperiment persists a new experiment. Key and Environment must be set.
	CreateExperiment(ctx context.Context, exp *Experiment) (*Experiment, error)

	// GetExperimentByKeyAndEnvironment returns the experiment for the given key and environment,
	// or (nil, nil) if not found.
	GetExperimentByKeyAndEnvironment(ctx context.Context, key, environment string) (*Experiment, error)

	// GetExperimentByID returns the experiment by ID, or (nil, nil) if not found.
	GetExperimentByID(ctx context.Context, id string) (*Experiment, error)

	// CreateVariant persists a variant for an experiment.
	CreateVariant(ctx context.Context, v *Variant) (*Variant, error)

	// GetVariantsByExperimentID returns all variants for an experiment, ordered by id.
	GetVariantsByExperimentID(ctx context.Context, experimentID string) ([]*Variant, error)

	// GetAssignment returns the user's assigned variant for an experiment, or (nil, nil) if not assigned.
	GetAssignment(ctx context.Context, userID, experimentID string) (*Assignment, error)

	// UpsertAssignment sets or updates the user's variant assignment for an experiment.
	UpsertAssignment(ctx context.Context, a *Assignment) error
}

// TxAwareStore is an optional interface for stores that can execute operations inside a provided sql.Tx.
type TxAwareStore interface {
	Store
	WithTx(tx *sql.Tx) Store
}
