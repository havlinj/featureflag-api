package users

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
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
		`INSERT INTO users (email, role, password_hash) VALUES ($1, $2, $3) RETURNING id, created_at`,
		user.Email, user.Role, nullString(user.PasswordHash),
	).Scan(&id, &createdAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, fmt.Errorf("users: duplicate email=%q: %w", user.Email, ErrDuplicateEmail)
		}
		return nil, err
	}
	out := *user
	out.ID = id
	out.CreatedAt = createdAt
	return &out, nil
}

func nullString(s *string) interface{} {
	if s == nil {
		return nil
	}
	return *s
}

func scanNullString(s *sql.NullString) *string {
	if s != nil && s.Valid {
		return &s.String
	}
	return nil
}

// GetByID returns the user by ID, or (nil, nil) if not found.
func (p *PostgresStore) GetByID(ctx context.Context, id string) (*User, error) {
	var u User
	var ph sql.NullString
	err := p.conn.QueryRowContext(ctx,
		`SELECT id, email, role, password_hash, created_at FROM users WHERE id = $1`,
		id,
	).Scan(&u.ID, &u.Email, &u.Role, &ph, &u.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	u.PasswordHash = scanNullString(&ph)
	return &u, nil
}

// GetByEmail returns the user by email, or (nil, nil) if not found.
func (p *PostgresStore) GetByEmail(ctx context.Context, email string) (*User, error) {
	var u User
	var ph sql.NullString
	err := p.conn.QueryRowContext(ctx,
		`SELECT id, email, role, password_hash, created_at FROM users WHERE email = $1`,
		email,
	).Scan(&u.ID, &u.Email, &u.Role, &ph, &u.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	u.PasswordHash = scanNullString(&ph)
	return &u, nil
}

// Update updates an existing user by ID. Returns ErrNotFound if no row was updated.
func (p *PostgresStore) Update(ctx context.Context, user *User) error {
	res, err := p.conn.ExecContext(ctx,
		`UPDATE users SET email = $1, role = $2, password_hash = $4 WHERE id = $3`,
		user.Email, user.Role, user.ID, nullString(user.PasswordHash),
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("users: user not found id=%q: %w", user.ID, ErrNotFound)
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
		return fmt.Errorf("users: user not found id=%q: %w", id, ErrNotFound)
	}
	return nil
}

var _ Store = (*PostgresStore)(nil)
