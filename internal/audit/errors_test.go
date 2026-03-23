package audit

import (
	"errors"
	"testing"
)

func TestAuditGuardErrors_Error_messages(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{
			name: "missing actor id",
			err:  &MissingActorIDError{},
			want: "audit: missing actor id in context",
		},
		{
			name: "tx starter required",
			err:  &TxStarterRequiredError{},
			want: "audit: audit store cannot start transactions",
		},
		{
			name: "tx aware required",
			err:  &TxAwareRequiredError{},
			want: "audit: audit store is not tx-aware",
		},
		{
			name: "nil entry",
			err:  &NilEntryError{},
			want: "audit store: entry is nil",
		},
		{
			name: "begin tx unsupported",
			err:  &BeginTxUnsupportedError{},
			want: "audit store: BeginTx not supported on tx-scoped store",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.want {
				t.Fatalf("Error() = %q; want %q", got, tt.want)
			}
		})
	}
}

func TestOperationError_Error_and_unwrap_are_deterministic(t *testing.T) {
	causeA := errors.New("db failure A")
	errA := &OperationError{
		Op:       opRepoCreate,
		ID:       "a1",
		Entity:   "flag",
		EntityID: "f1",
		Action:   "create",
		ActorID:  "u1",
		Limit:    10,
		Offset:   5,
		Cause:    causeA,
	}

	if got, want := errA.Error(), `audit: operation="audit.repo.create" id="a1" entity="flag" entity_id="f1" action="create" actor_id="u1" limit=10 offset=5: db failure A`; got != want {
		t.Errorf("Error() = %q; want %q", got, want)
	}
	if !errors.Is(errA, causeA) {
		t.Errorf("errors.Is(errA, causeA) = false; want true")
	}

	causeB := errors.New("db failure B")
	errB := &OperationError{
		Op:    opRepoListIterate,
		Limit: 100,
		Cause: causeB,
	}

	if got, want := errB.Error(), `audit: operation="audit.repo.list.iterate" id="" entity="" entity_id="" action="" actor_id="" limit=100 offset=0: db failure B`; got != want {
		t.Errorf("Error() = %q; want %q", got, want)
	}
	if !errors.Is(errB, causeB) {
		t.Errorf("errors.Is(errB, causeB) = false; want true")
	}
}
