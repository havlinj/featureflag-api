package flags

import (
	"context"
	"errors"
	"sync"
)

// ErrNoMoreReturns is returned when a mock method is called but its return queue is empty.
// Enqueue enough results for each expected call.
var ErrNoMoreReturns = errors.New("flags/mock: no more return values enqueued")

// MockStore is a dumb stub implementing Store. It does not store data; it records
// arguments and returns the next enqueued result per method. Each method has a
// queue of results (either value or error); repeated calls consume the queue.
// When the queue is empty, the method returns (nil, ErrNoMoreReturns) or ErrNoMoreReturns.
type MockStore struct {
	mu sync.Mutex

	CreateCalls []struct {
		Ctx  context.Context
		Flag *Flag
	}
	CreateReturns []CreateResult

	GetByKeyAndEnvironmentCalls []struct {
		Ctx         context.Context
		Key         string
		Environment string
	}
	GetByKeyAndEnvironmentReturns []GetByKeyResult

	UpdateCalls []struct {
		Ctx  context.Context
		Flag *Flag
	}
	UpdateReturns []error

	GetRulesByFlagIDCalls []struct {
		Ctx    context.Context
		FlagID string
	}
	GetRulesByFlagIDReturns []GetRulesResult
}

// Create records the call and returns the next CreateResult from the queue.
func (m *MockStore) Create(ctx context.Context, flag *Flag) (*Flag, error) {
	m.mu.Lock()
	m.CreateCalls = append(m.CreateCalls, struct {
		Ctx  context.Context
		Flag *Flag
	}{Ctx: ctx, Flag: flag})
	var out *Flag
	var err error
	if len(m.CreateReturns) > 0 {
		r := m.CreateReturns[0]
		m.CreateReturns = m.CreateReturns[1:]
		out, err = r.Flag, r.Err
	} else {
		err = ErrNoMoreReturns
	}
	m.mu.Unlock()
	return out, err
}

// GetByKeyAndEnvironment records the call and returns the next GetByKeyResult from the queue.
func (m *MockStore) GetByKeyAndEnvironment(ctx context.Context, key, environment string) (*Flag, error) {
	m.mu.Lock()
	m.GetByKeyAndEnvironmentCalls = append(m.GetByKeyAndEnvironmentCalls, struct {
		Ctx         context.Context
		Key         string
		Environment string
	}{Ctx: ctx, Key: key, Environment: environment})
	var out *Flag
	var err error
	if len(m.GetByKeyAndEnvironmentReturns) > 0 {
		r := m.GetByKeyAndEnvironmentReturns[0]
		m.GetByKeyAndEnvironmentReturns = m.GetByKeyAndEnvironmentReturns[1:]
		out, err = r.Flag, r.Err
	} else {
		err = ErrNoMoreReturns
	}
	m.mu.Unlock()
	return out, err
}

// Update records the call and returns the next error from UpdateReturns (nil = success).
func (m *MockStore) Update(ctx context.Context, flag *Flag) error {
	m.mu.Lock()
	m.UpdateCalls = append(m.UpdateCalls, struct {
		Ctx  context.Context
		Flag *Flag
	}{Ctx: ctx, Flag: flag})
	var err error
	if len(m.UpdateReturns) > 0 {
		err = m.UpdateReturns[0]
		m.UpdateReturns = m.UpdateReturns[1:]
	} else {
		err = ErrNoMoreReturns
	}
	m.mu.Unlock()
	return err
}

// GetRulesByFlagID records the call and returns the next GetRulesResult from the queue.
func (m *MockStore) GetRulesByFlagID(ctx context.Context, flagID string) ([]*Rule, error) {
	m.mu.Lock()
	m.GetRulesByFlagIDCalls = append(m.GetRulesByFlagIDCalls, struct {
		Ctx    context.Context
		FlagID string
	}{Ctx: ctx, FlagID: flagID})
	var out []*Rule
	var err error
	if len(m.GetRulesByFlagIDReturns) > 0 {
		r := m.GetRulesByFlagIDReturns[0]
		m.GetRulesByFlagIDReturns = m.GetRulesByFlagIDReturns[1:]
		out, err = r.Rules, r.Err
	} else {
		err = ErrNoMoreReturns
	}
	m.mu.Unlock()
	return out, err
}

var _ Store = (*MockStore)(nil)

// CreateResult represents a single return for Create: either a flag (Ok) or an error (Err).
// Only one of Flag or Err should be set per element (Rust: Result<Flag, error>).
type CreateResult struct {
	Flag *Flag
	Err  error
}

// GetByKeyResult represents a single return for GetByKeyAndEnvironment.
type GetByKeyResult struct {
	Flag *Flag
	Err  error
}

// GetRulesResult represents a single return for GetRulesByFlagID.
type GetRulesResult struct {
	Rules []*Rule
	Err   error
}
