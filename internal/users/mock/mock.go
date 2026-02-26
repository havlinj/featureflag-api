// Package mock provides a queue-based Store implementation for testing (unit and integration).
package mock

import (
	"context"
	"errors"
	"sync"

	"github.com/jan-havlin-dev/featureflag-api/internal/users"
)

// ErrNoMoreReturns is returned when a method is called but its return queue is empty.
var ErrNoMoreReturns = errors.New("users/mock: no more return values enqueued")

// Store implements users.Store. It records calls and returns the next enqueued result per method.
// Enqueue enough results for each expected call in your test.
type Store struct {
	mu sync.Mutex

	CreateCalls            []struct{ Ctx context.Context; User *users.User }
	CreateReturns          []CreateResult
	GetByIDCalls           []struct{ Ctx context.Context; ID string }
	GetByIDReturns         []GetByIDResult
	GetByEmailCalls        []struct{ Ctx context.Context; Email string }
	GetByEmailReturns      []GetByEmailResult
	UpdateCalls            []struct{ Ctx context.Context; User *users.User }
	UpdateReturns          []error
	DeleteCalls            []struct{ Ctx context.Context; ID string }
	DeleteReturns          []error
}

// CreateResult is a single return for Create.
type CreateResult struct {
	User *users.User
	Err  error
}

// GetByIDResult is a single return for GetByID.
type GetByIDResult struct {
	User *users.User
	Err  error
}

// GetByEmailResult is a single return for GetByEmail.
type GetByEmailResult struct {
	User *users.User
	Err  error
}

func (m *Store) Create(ctx context.Context, user *users.User) (*users.User, error) {
	m.mu.Lock()
	m.CreateCalls = append(m.CreateCalls, struct{ Ctx context.Context; User *users.User }{ctx, user})
	var out *users.User
	var err error
	if len(m.CreateReturns) > 0 {
		r := m.CreateReturns[0]
		m.CreateReturns = m.CreateReturns[1:]
		out, err = r.User, r.Err
	} else {
		err = ErrNoMoreReturns
	}
	m.mu.Unlock()
	return out, err
}

func (m *Store) GetByID(ctx context.Context, id string) (*users.User, error) {
	m.mu.Lock()
	m.GetByIDCalls = append(m.GetByIDCalls, struct{ Ctx context.Context; ID string }{ctx, id})
	var out *users.User
	var err error
	if len(m.GetByIDReturns) > 0 {
		r := m.GetByIDReturns[0]
		m.GetByIDReturns = m.GetByIDReturns[1:]
		out, err = r.User, r.Err
	} else {
		err = ErrNoMoreReturns
	}
	m.mu.Unlock()
	return out, err
}

func (m *Store) GetByEmail(ctx context.Context, email string) (*users.User, error) {
	m.mu.Lock()
	m.GetByEmailCalls = append(m.GetByEmailCalls, struct{ Ctx context.Context; Email string }{ctx, email})
	var out *users.User
	var err error
	if len(m.GetByEmailReturns) > 0 {
		r := m.GetByEmailReturns[0]
		m.GetByEmailReturns = m.GetByEmailReturns[1:]
		out, err = r.User, r.Err
	} else {
		err = ErrNoMoreReturns
	}
	m.mu.Unlock()
	return out, err
}

func (m *Store) Update(ctx context.Context, user *users.User) error {
	m.mu.Lock()
	m.UpdateCalls = append(m.UpdateCalls, struct{ Ctx context.Context; User *users.User }{ctx, user})
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
	m.DeleteCalls = append(m.DeleteCalls, struct{ Ctx context.Context; ID string }{ctx, id})
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

var _ users.Store = (*Store)(nil)
