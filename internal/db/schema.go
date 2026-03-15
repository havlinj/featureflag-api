package db

// schemaSQL defines the initial schema. Order matters: users first (no deps),
// then feature_flags (referenced by flag_rules.flag_id).
var schemaSQL = []string{
	`CREATE TABLE IF NOT EXISTS users (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		email TEXT NOT NULL UNIQUE,
		role TEXT NOT NULL CHECK (role IN ('admin', 'developer', 'viewer')),
		password_hash TEXT,
		created_at TIMESTAMPTZ NOT NULL DEFAULT now()
	)`,
	`CREATE TABLE IF NOT EXISTS feature_flags (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		key TEXT NOT NULL,
		description TEXT,
		enabled BOOLEAN NOT NULL DEFAULT false,
		environment TEXT NOT NULL,
		rollout_strategy TEXT NOT NULL DEFAULT 'none' CHECK (rollout_strategy IN ('none', 'percentage', 'attribute')),
		created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
		UNIQUE(key, environment)
	)`,
	`CREATE TABLE IF NOT EXISTS flag_rules (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		flag_id UUID NOT NULL REFERENCES feature_flags(id) ON DELETE CASCADE,
		type TEXT NOT NULL CHECK (type IN ('percentage', 'attribute')),
		value TEXT NOT NULL
	)`,
	`CREATE TABLE IF NOT EXISTS experiments (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		key TEXT NOT NULL,
		environment TEXT NOT NULL,
		UNIQUE(key, environment)
	)`,
	`CREATE TABLE IF NOT EXISTS experiment_variants (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		experiment_id UUID NOT NULL REFERENCES experiments(id) ON DELETE CASCADE,
		name TEXT NOT NULL,
		weight INTEGER NOT NULL CHECK (weight >= 0)
	)`,
	`CREATE TABLE IF NOT EXISTS experiment_assignments (
		user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		experiment_id UUID NOT NULL REFERENCES experiments(id) ON DELETE CASCADE,
		variant_id UUID NOT NULL REFERENCES experiment_variants(id) ON DELETE CASCADE,
		PRIMARY KEY (user_id, experiment_id)
	)`,
}
