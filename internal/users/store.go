package users

import (
	"context"
	"database/sql"
)

// Store is the persistence interface for the users domain. Implemented by
// PostgresStore and by a mock for unit tests.
type Store interface {
	Create(ctx context.Context, user *User) (*User, error)
	GetByID(ctx context.Context, id string) (*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
	Update(ctx context.Context, user *User) error
	Delete(ctx context.Context, id string) error
}

// TxAwareStore is an optional interface for stores that can execute operations inside a provided sql.Tx.
type TxAwareStore interface {
	Store
	WithTx(tx *sql.Tx) Store
}
