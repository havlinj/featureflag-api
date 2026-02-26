package db

// schemaSQL defines the initial schema. Order matters: users first (no deps),
// then feature_flags (referenced by flag_rules.flag_id).
var schemaSQL = []string{
	`CREATE TABLE IF NOT EXISTS users (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		email TEXT NOT NULL UNIQUE,
		role TEXT NOT NULL CHECK (role IN ('admin', 'developer', 'viewer')),
		created_at TIMESTAMPTZ NOT NULL DEFAULT now()
	)`,
	`CREATE TABLE IF NOT EXISTS feature_flags (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		key TEXT NOT NULL,
		description TEXT,
		enabled BOOLEAN NOT NULL DEFAULT false,
		environment TEXT NOT NULL,
		created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
		UNIQUE(key, environment)
	)`,
	`CREATE TABLE IF NOT EXISTS flag_rules (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		flag_id UUID NOT NULL REFERENCES feature_flags(id) ON DELETE CASCADE,
		type TEXT NOT NULL CHECK (type IN ('percentage', 'attribute')),
		value TEXT NOT NULL
	)`,
}
