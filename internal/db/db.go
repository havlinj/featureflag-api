package db

import (
	"context"
	"database/sql"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// DB holds the database connection and is responsible for initialization (schema).
// It provides Conn() so that domain stores (e.g. flags.PostgresStore) can be built
// with the same connection. Application flow: Open → EnsureSchema → create stores
// via Conn() → run app. If the database already exists, EnsureSchema is idempotent
// (CREATE TABLE IF NOT EXISTS).
type DB struct {
	conn *sql.DB
}

// Open connects to PostgreSQL using the given DSN. It does not create schema;
// call EnsureSchema(ctx) after Open to create tables if they do not exist.
func Open(ctx context.Context, dsn string) (*DB, error) {
	conn, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}
	if err := conn.PingContext(ctx); err != nil {
		_ = conn.Close()
		return nil, err
	}
	return &DB{conn: conn}, nil
}

// Conn returns the underlying *sql.DB for constructing domain stores (e.g. flags.NewPostgresStore(db.Conn())).
// Do not close it; use DB.Close() instead.
func (d *DB) Conn() *sql.DB {
	return d.conn
}

// EnsureSchema creates the feature_flags and flag_rules tables if they do not exist.
// Safe to call on an already-initialized database (idempotent).
func (d *DB) EnsureSchema(ctx context.Context) error {
	for _, q := range schemaSQL {
		if _, err := d.conn.ExecContext(ctx, q); err != nil {
			return err
		}
	}
	return nil
}

// Close closes the database connection.
func (d *DB) Close() error {
	return d.conn.Close()
}
