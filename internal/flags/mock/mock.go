// Package mock provides a queue-based Store implementation for testing (unit and integration).
package mock

import (
	"context"
	"errors"
	"sync"

	"github.com/jan-havlin-dev/featureflag-api/internal/flags"
)

// ErrNoMoreReturns is returned when a method is called but its return queue is empty.
var ErrNoMoreReturns = errors.New("flags/mock: no more return values enqueued")

// Store implements flags.Store. It records calls and returns the next enqueued result per method.
// Enqueue enough results for each expected call in your test.
type Store struct {
	mu sync.Mutex

	CreateCalls []struct {
		Ctx  context.Context
		Flag *flags.Flag
	}
	CreateReturns               []CreateResult
	GetByKeyAndEnvironmentCalls []struct {
		Ctx      context.Context
		Key, Env string
	}
	GetByKeyAndEnvironmentReturns []GetByKeyResult
	UpdateCalls                   []struct {
		Ctx  context.Context
		Flag *flags.Flag
	}
	UpdateReturns []error
	DeleteCalls   []struct {
		Ctx context.Context
		ID  string
	}
	DeleteReturns         []error
	GetRulesByFlagIDCalls []struct {
		Ctx    context.Context
		FlagID string
	}
	GetRulesByFlagIDReturns   []GetRulesResult
	ReplaceRulesByFlagIDCalls []struct {
		Ctx    context.Context
		FlagID string
		Rules  []*flags.Rule
	}
	ReplaceRulesByFlagIDReturns []error
}

// CreateResult is a single return for Create.
type CreateResult struct {
	Flag *flags.Flag
	Err  error
}

// GetByKeyResult is a single return for GetByKeyAndEnvironment.
type GetByKeyResult struct {
	Flag *flags.Flag
	Err  error
}

// GetRulesResult is a single return for GetRulesByFlagID.
type GetRulesResult struct {
	Rules []*flags.Rule
	Err   error
}

func (m *Store) Create(ctx context.Context, flag *flags.Flag) (*flags.Flag, error) {
	m.mu.Lock()
	m.CreateCalls = append(m.CreateCalls, struct {
		Ctx  context.Context
		Flag *flags.Flag
	}{ctx, flag})
	var out *flags.Flag
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

func (m *Store) GetByKeyAndEnvironment(ctx context.Context, key, environment string) (*flags.Flag, error) {
	m.mu.Lock()
	m.GetByKeyAndEnvironmentCalls = append(m.GetByKeyAndEnvironmentCalls, struct {
		Ctx      context.Context
		Key, Env string
	}{ctx, key, environment})
	var out *flags.Flag
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

func (m *Store) Update(ctx context.Context, flag *flags.Flag) error {
	m.mu.Lock()
	m.UpdateCalls = append(m.UpdateCalls, struct {
		Ctx  context.Context
		Flag *flags.Flag
	}{ctx, flag})
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

func (m *Store) Delete(ctx context.Context, id string) error {
	m.mu.Lock()
	m.DeleteCalls = append(m.DeleteCalls, struct {
		Ctx context.Context
		ID  string
	}{ctx, id})
	var err error
	if len(m.DeleteReturns) > 0 {
		err = m.DeleteReturns[0]
		m.DeleteReturns = m.DeleteReturns[1:]
	} else {
		err = ErrNoMoreReturns
	}
	m.mu.Unlock()
	return err
}

func (m *Store) GetRulesByFlagID(ctx context.Context, flagID string) ([]*flags.Rule, error) {
	m.mu.Lock()
	m.GetRulesByFlagIDCalls = append(m.GetRulesByFlagIDCalls, struct {
		Ctx    context.Context
		FlagID string
	}{ctx, flagID})
	var out []*flags.Rule
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

func (m *Store) ReplaceRulesByFlagID(ctx context.Context, flagID string, rules []*flags.Rule) error {
	m.mu.Lock()
	m.ReplaceRulesByFlagIDCalls = append(m.ReplaceRulesByFlagIDCalls, struct {
		Ctx    context.Context
		FlagID string
		Rules  []*flags.Rule
	}{ctx, flagID, rules})
	var err error
	if len(m.ReplaceRulesByFlagIDReturns) > 0 {
		err = m.ReplaceRulesByFlagIDReturns[0]
		m.ReplaceRulesByFlagIDReturns = m.ReplaceRulesByFlagIDReturns[1:]
	} else {
		err = ErrNoMoreReturns
	}
	m.mu.Unlock()
	return err
}

var _ flags.Store = (*Store)(nil)
