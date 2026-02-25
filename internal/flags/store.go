package flags

import "context"

// Store is the persistence interface for the flags domain. It is implemented by
// a real DB client (e.g. Postgres) and by a mock for unit tests.
//
// Intended usage from flags.Service:
//
//   - CreateFlag: Check uniqueness via GetByKeyAndEnvironment; if not found, Create.
//     Then optionally create default rule or leave without rules.
//   - UpdateFlag: Resolve flag via GetByKeyAndEnvironment (environment from context
//     or API when added); then Update(flag) to set Enabled (and other fields).
//   - EvaluateFlag: GetByKeyAndEnvironment to load flag; if disabled or not found
//     return false. GetRulesByFlagID to load rules; service applies percentage/
//     attribute logic and returns bool.
type Store interface {
	// Create persists a new flag. Key, Description, Enabled, Environment must be set;
	// ID and CreatedAt are typically set by the implementation.
	Create(ctx context.Context, flag *Flag) (*Flag, error)

	// GetByKeyAndEnvironment returns the flag for the given key and environment,
	// or nil and no error if not found. Error only for DB failures.
	GetByKeyAndEnvironment(ctx context.Context, key, environment string) (*Flag, error)

	// Update updates an existing flag by ID (e.g. Enabled, Description).
	Update(ctx context.Context, flag *Flag) error

	// GetRulesByFlagID returns all rules for the given flag (for rollout evaluation).
	GetRulesByFlagID(ctx context.Context, flagID string) ([]*Rule, error)
}
