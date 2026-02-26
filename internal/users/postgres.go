package users

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

// NewPostgresStore returns a Store that uses the given *sql.DB.
func NewPostgresStore(conn *sql.DB) *PostgresStore {
	return &PostgresStore{conn: conn}
}

// Create persists a new user. Email and Role must be set; ID and CreatedAt are set by the DB.
// Returns ErrDuplicateEmail if email already exists.
func (p *PostgresStore) Create(ctx context.Context, user *User) (*User, error) {
	var id string
	var createdAt time.Time
	err := p.conn.QueryRowContext(ctx,
		`INSERT INTO users (email, role) VALUES ($1, $2) RETURNING id, created_at`,
		user.Email, user.Role,
	).Scan(&id, &createdAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, ErrDuplicateEmail
		}
		return nil, err
	}
	out := *user
	out.ID = id
	out.CreatedAt = createdAt
	return &out, nil
}

// GetByID returns the user by ID, or (nil, nil) if not found.
func (p *PostgresStore) GetByID(ctx context.Context, id string) (*User, error) {
	var u User
	err := p.conn.QueryRowContext(ctx,
		`SELECT id, email, role, created_at FROM users WHERE id = $1`,
		id,
	).Scan(&u.ID, &u.Email, &u.Role, &u.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

// GetByEmail returns the user by email, or (nil, nil) if not found.
func (p *PostgresStore) GetByEmail(ctx context.Context, email string) (*User, error) {
	var u User
	err := p.conn.QueryRowContext(ctx,
		`SELECT id, email, role, created_at FROM users WHERE email = $1`,
		email,
	).Scan(&u.ID, &u.Email, &u.Role, &u.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

// Update updates an existing user by ID. Returns ErrNotFound if no row was updated.
func (p *PostgresStore) Update(ctx context.Context, user *User) error {
	res, err := p.conn.ExecContext(ctx,
		`UPDATE users SET email = $1, role = $2 WHERE id = $3`,
		user.Email, user.Role, user.ID,
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

// Delete removes a user by ID. Returns ErrNotFound if no row was deleted.
func (p *PostgresStore) Delete(ctx context.Context, id string) error {
	res, err := p.conn.ExecContext(ctx, `DELETE FROM users WHERE id = $1`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

var _ Store = (*PostgresStore)(nil)
