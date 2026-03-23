package audit

import "fmt"

// Operation identifiers for structured context propagation across repository layer.
const (
	opRepoCreate      = "audit.repo.create"
	opRepoGetByID     = "audit.repo.get_by_id"
	opRepoListQuery   = "audit.repo.list.query"
	opRepoListScan    = "audit.repo.list.scan"
	opRepoListIterate = "audit.repo.list.iterate"
)

// OperationError is returned when an audit store/service operation fails with structured context.
type OperationError struct {
	Op       string
	ID       string
	Entity   string
	EntityID string
	Action   string
	ActorID  string
	Limit    int
	Offset   int
	Cause    error
}

func (e *OperationError) Error() string {
	return fmt.Sprintf(
		"audit: operation=%q id=%q entity=%q entity_id=%q action=%q actor_id=%q limit=%d offset=%d: %v",
		e.Op, e.ID, e.Entity, e.EntityID, e.Action, e.ActorID, e.Limit, e.Offset, e.Cause,
	)
}

func (e *OperationError) Unwrap() error {
	return e.Cause
}

// MissingActorIDError is returned when audit write is requested without actor in context.
type MissingActorIDError struct{}

func (e *MissingActorIDError) Error() string {
	return "audit: missing actor id in context"
}

// TxStarterRequiredError is returned when audit store cannot start transactions.
type TxStarterRequiredError struct{}

func (e *TxStarterRequiredError) Error() string {
	return "audit: audit store cannot start transactions"
}

// TxAwareRequiredError is returned when audit store cannot create tx-scoped instances.
type TxAwareRequiredError struct{}

func (e *TxAwareRequiredError) Error() string {
	return "audit: audit store is not tx-aware"
}

// NilEntryError is returned when Create receives a nil audit entry.
type NilEntryError struct{}

func (e *NilEntryError) Error() string {
	return "audit store: entry is nil"
}

// BeginTxUnsupportedError is returned when BeginTx is called on tx-scoped store.
type BeginTxUnsupportedError struct{}

func (e *BeginTxUnsupportedError) Error() string {
	return "audit store: BeginTx not supported on tx-scoped store"
}
