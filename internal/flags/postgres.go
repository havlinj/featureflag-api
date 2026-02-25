package flags

import "context"

// PostgresStore is the real persistence implementation using PostgreSQL.
// Not implemented yet; methods panic.
type PostgresStore struct {
	// TODO: *sql.DB or *pgxpool.Pool, connection config, etc.
}

// Create creates a new flag in the database.
func (p *PostgresStore) Create(ctx context.Context, flag *Flag) (*Flag, error) {
	panic("unimplemented")
}

// GetByKeyAndEnvironment loads a flag by key and environment.
func (p *PostgresStore) GetByKeyAndEnvironment(ctx context.Context, key, environment string) (*Flag, error) {
	panic("unimplemented")
}

// Update updates an existing flag by ID.
func (p *PostgresStore) Update(ctx context.Context, flag *Flag) error {
	panic("unimplemented")
}

// GetRulesByFlagID returns all rules for the given flag.
func (p *PostgresStore) GetRulesByFlagID(ctx context.Context, flagID string) ([]*Rule, error) {
	panic("unimplemented")
}

// Ensure PostgresStore implements Store at compile time.
var _ Store = (*PostgresStore)(nil)
