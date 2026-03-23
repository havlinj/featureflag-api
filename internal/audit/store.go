package audit

import (
	"context"
	"database/sql"
	"errors"
)

const DefaultListLimit = 50
const MaxListLimit = 200

var ErrNegativeOffset = errors.New("offset must be >= 0")

// ListFilter allows optional narrowing of audit logs.
type ListFilter struct {
	Entity  *string
	Action  *string
	ActorID *string
}

// Store is the persistence interface for the audit module.
// It is intentionally small: services call Create for writes, resolvers call Get/List for reads.
type Store interface {
	Create(ctx context.Context, entry *Entry) error
	GetByID(ctx context.Context, id string) (*Entry, error)
	List(ctx context.Context, filter ListFilter, limit, offset int) ([]*Entry, error)
}

// TxAwareStore is an optional interface implemented by stores that can operate inside a provided sql.Tx.
// Domain services will use it for atomic business+audit operations.
type TxAwareStore interface {
	Store
	WithTx(tx *sql.Tx) Store
}

// TxStarter is an optional interface implemented by stores that can begin transactions.
// It is used by domain services to coordinate atomic business+audit writes.
type TxStarter interface {
	BeginTx(ctx context.Context) (*sql.Tx, error)
}
