package testutil

import (
	"context"
	"errors"
	"sync"

	"github.com/jan-havlin-dev/featureflag-api/internal/flags"
)

// ErrNoMoreFlagsReturns is returned when a mock method is called but its return queue is empty.
// Enqueue enough results for each expected call.
var ErrNoMoreFlagsReturns = errors.New("testutil/flags_mock: no more return values enqueued")

// MockFlagsStore is a dumb stub implementing flags.Store. It does not store data; it records
// arguments and returns the next enqueued result per method. Each method has a
// queue of results (either value or error); repeated calls consume the queue.
// When the queue is empty, the method returns (nil, ErrNoMoreFlagsReturns) or ErrNoMoreFlagsReturns.
type MockFlagsStore struct {
	mu sync.Mutex

	CreateCalls []struct {
		Ctx  context.Context
		Flag *flags.Flag
	}
	CreateReturns []FlagsCreateResult

	GetByKeyAndEnvironmentCalls []struct {
		Ctx         context.Context
		Key         string
		Environment string
	}
	GetByKeyAndEnvironmentReturns []FlagsGetByKeyResult

	UpdateCalls []struct {
		Ctx  context.Context
		Flag *flags.Flag
	}
	UpdateReturns []error

	GetRulesByFlagIDCalls []struct {
		Ctx    context.Context
		FlagID string
	}
	GetRulesByFlagIDReturns []FlagsGetRulesResult
}

// Create records the call and returns the next CreateResult from the queue.
func (m *MockFlagsStore) Create(ctx context.Context, flag *flags.Flag) (*flags.Flag, error) {
	m.mu.Lock()
	m.CreateCalls = append(m.CreateCalls, struct {
		Ctx  context.Context
		Flag *flags.Flag
	}{Ctx: ctx, Flag: flag})
	var out *flags.Flag
	var err error
	if len(m.CreateReturns) > 0 {
		r := m.CreateReturns[0]
		m.CreateReturns = m.CreateReturns[1:]
		out, err = r.Flag, r.Err
	} else {
		err = ErrNoMoreFlagsReturns
	}
	m.mu.Unlock()
	return out, err
}

// GetByKeyAndEnvironment records the call and returns the next GetByKeyResult from the queue.
func (m *MockFlagsStore) GetByKeyAndEnvironment(ctx context.Context, key, environment string) (*flags.Flag, error) {
	m.mu.Lock()
	m.GetByKeyAndEnvironmentCalls = append(m.GetByKeyAndEnvironmentCalls, struct {
		Ctx         context.Context
		Key         string
		Environment string
	}{Ctx: ctx, Key: key, Environment: environment})
	var out *flags.Flag
	var err error
	if len(m.GetByKeyAndEnvironmentReturns) > 0 {
		r := m.GetByKeyAndEnvironmentReturns[0]
		m.GetByKeyAndEnvironmentReturns = m.GetByKeyAndEnvironmentReturns[1:]
		out, err = r.Flag, r.Err
	} else {
		err = ErrNoMoreFlagsReturns
	}
	m.mu.Unlock()
	return out, err
}

// Update records the call and returns the next error from UpdateReturns (nil = success).
func (m *MockFlagsStore) Update(ctx context.Context, flag *flags.Flag) error {
	m.mu.Lock()
	m.UpdateCalls = append(m.UpdateCalls, struct {
		Ctx  context.Context
		Flag *flags.Flag
	}{Ctx: ctx, Flag: flag})
	var err error
	if len(m.UpdateReturns) > 0 {
		err = m.UpdateReturns[0]
		m.UpdateReturns = m.UpdateReturns[1:]
	} else {
		err = ErrNoMoreFlagsReturns
	}
	m.mu.Unlock()
	return err
}

// GetRulesByFlagID records the call and returns the next GetRulesResult from the queue.
func (m *MockFlagsStore) GetRulesByFlagID(ctx context.Context, flagID string) ([]*flags.Rule, error) {
	m.mu.Lock()
	m.GetRulesByFlagIDCalls = append(m.GetRulesByFlagIDCalls, struct {
		Ctx    context.Context
		FlagID string
	}{Ctx: ctx, FlagID: flagID})
	var out []*flags.Rule
	var err error
	if len(m.GetRulesByFlagIDReturns) > 0 {
		r := m.GetRulesByFlagIDReturns[0]
		m.GetRulesByFlagIDReturns = m.GetRulesByFlagIDReturns[1:]
		out, err = r.Rules, r.Err
	} else {
		err = ErrNoMoreFlagsReturns
	}
	m.mu.Unlock()
	return out, err
}

var _ flags.Store = (*MockFlagsStore)(nil)

// FlagsCreateResult represents a single return for Create: either a flag (Ok) or an error (Err).
type FlagsCreateResult struct {
	Flag *flags.Flag
	Err  error
}

// FlagsGetByKeyResult represents a single return for GetByKeyAndEnvironment.
type FlagsGetByKeyResult struct {
	Flag *flags.Flag
	Err  error
}

// FlagsGetRulesResult represents a single return for GetRulesByFlagID.
type FlagsGetRulesResult struct {
	Rules []*flags.Rule
	Err   error
}
