package audit

import (
	"context"
	"database/sql"
	"errors"

	"github.com/havlinj/featureflag-api/internal/auth"
)

// PrepareWriteTx validates actor/audit capabilities and opens a write transaction.
func PrepareWriteTx(ctx context.Context, store Store) (string, *sql.Tx, Store, error) {
	actorID, ok := auth.ActorIDFromContext(ctx)
	if !ok {
		return "", nil, nil, errors.New("audit: missing actor id in context")
	}

	txStarter, ok := store.(TxStarter)
	if !ok {
		return "", nil, nil, errors.New("audit: audit store cannot start transactions")
	}

	txAware, ok := store.(TxAwareStore)
	if !ok {
		return "", nil, nil, errors.New("audit: audit store is not tx-aware")
	}

	tx, err := txStarter.BeginTx(ctx)
	if err != nil {
		return "", nil, nil, err
	}

	return actorID, tx, txAware.WithTx(tx), nil
}
