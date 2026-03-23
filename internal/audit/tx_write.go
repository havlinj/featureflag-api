package audit

import (
	"context"
	"database/sql"

	"github.com/havlinj/featureflag-api/internal/auth"
)

// PrepareWriteTx validates actor/audit capabilities and opens a write transaction.
func PrepareWriteTx(ctx context.Context, store Store) (string, *sql.Tx, Store, error) {
	actorID, ok := auth.ActorIDFromContext(ctx)
	if !ok {
		return "", nil, nil, &MissingActorIDError{}
	}

	txStarter, ok := store.(TxStarter)
	if !ok {
		return "", nil, nil, &TxStarterRequiredError{}
	}

	txAware, ok := store.(TxAwareStore)
	if !ok {
		return "", nil, nil, &TxAwareRequiredError{}
	}

	tx, err := txStarter.BeginTx(ctx)
	if err != nil {
		return "", nil, nil, err
	}

	return actorID, tx, txAware.WithTx(tx), nil
}
