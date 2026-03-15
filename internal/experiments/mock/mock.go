// Package mock provides a queue-based Store implementation for testing (unit tests).
package mock

import (
	"context"
	"errors"
	"sync"

	"github.com/havlinj/featureflag-api/internal/experiments"
)

// ErrNoMoreReturns is returned when a method is called but its return queue is empty.
var ErrNoMoreReturns = errors.New("experiments/mock: no more return values enqueued")

// Store implements experiments.Store. It records calls and returns the next enqueued result per method.
// Enqueue enough results for each expected call in your test.
type Store struct {
	mu sync.Mutex

	CreateExperimentCalls []struct {
		Ctx context.Context
		Exp *experiments.Experiment
	}
	CreateExperimentReturns []CreateExperimentResult

	GetExperimentByKeyAndEnvironmentCalls []struct {
		Ctx         context.Context
		Key         string
		Environment string
	}
	GetExperimentByKeyAndEnvironmentReturns []GetExperimentResult

	GetExperimentByIDCalls []struct {
		Ctx context.Context
		ID  string
	}
	GetExperimentByIDReturns []GetExperimentResult

	CreateVariantCalls []struct {
		Ctx context.Context
		V   *experiments.Variant
	}
	CreateVariantReturns []CreateVariantResult

	GetVariantsByExperimentIDCalls []struct {
		Ctx          context.Context
		ExperimentID string
	}
	GetVariantsByExperimentIDReturns []GetVariantsResult

	GetAssignmentCalls []struct {
		Ctx          context.Context
		UserID       string
		ExperimentID string
	}
	GetAssignmentReturns []GetAssignmentResult

	UpsertAssignmentCalls []struct {
		Ctx context.Context
		A   *experiments.Assignment
	}
	UpsertAssignmentReturns []error
}

// CreateExperimentResult is a single return for CreateExperiment.
type CreateExperimentResult struct {
	Exp *experiments.Experiment
	Err error
}

// GetExperimentResult is a single return for GetExperimentByKeyAndEnvironment and GetExperimentByID.
type GetExperimentResult struct {
	Exp *experiments.Experiment
	Err error
}

// CreateVariantResult is a single return for CreateVariant.
type CreateVariantResult struct {
	V   *experiments.Variant
	Err error
}

// GetVariantsResult is a single return for GetVariantsByExperimentID.
type GetVariantsResult struct {
	Variants []*experiments.Variant
	Err      error
}

// GetAssignmentResult is a single return for GetAssignment.
type GetAssignmentResult struct {
	A   *experiments.Assignment
	Err error
}

func (m *Store) CreateExperiment(ctx context.Context, exp *experiments.Experiment) (*experiments.Experiment, error) {
	m.mu.Lock()
	m.CreateExperimentCalls = append(m.CreateExperimentCalls, struct {
		Ctx context.Context
		Exp *experiments.Experiment
	}{ctx, exp})
	var out *experiments.Experiment
	var err error
	if len(m.CreateExperimentReturns) > 0 {
		r := m.CreateExperimentReturns[0]
		m.CreateExperimentReturns = m.CreateExperimentReturns[1:]
		out, err = r.Exp, r.Err
	} else {
		err = ErrNoMoreReturns
	}
	m.mu.Unlock()
	return out, err
}

func (m *Store) GetExperimentByKeyAndEnvironment(ctx context.Context, key, environment string) (*experiments.Experiment, error) {
	m.mu.Lock()
	m.GetExperimentByKeyAndEnvironmentCalls = append(m.GetExperimentByKeyAndEnvironmentCalls, struct {
		Ctx         context.Context
		Key         string
		Environment string
	}{ctx, key, environment})
	var out *experiments.Experiment
	var err error
	if len(m.GetExperimentByKeyAndEnvironmentReturns) > 0 {
		r := m.GetExperimentByKeyAndEnvironmentReturns[0]
		m.GetExperimentByKeyAndEnvironmentReturns = m.GetExperimentByKeyAndEnvironmentReturns[1:]
		out, err = r.Exp, r.Err
	} else {
		err = ErrNoMoreReturns
	}
	m.mu.Unlock()
	return out, err
}

func (m *Store) GetExperimentByID(ctx context.Context, id string) (*experiments.Experiment, error) {
	m.mu.Lock()
	m.GetExperimentByIDCalls = append(m.GetExperimentByIDCalls, struct {
		Ctx context.Context
		ID  string
	}{ctx, id})
	var out *experiments.Experiment
	var err error
	if len(m.GetExperimentByIDReturns) > 0 {
		r := m.GetExperimentByIDReturns[0]
		m.GetExperimentByIDReturns = m.GetExperimentByIDReturns[1:]
		out, err = r.Exp, r.Err
	} else {
		err = ErrNoMoreReturns
	}
	m.mu.Unlock()
	return out, err
}

func (m *Store) CreateVariant(ctx context.Context, v *experiments.Variant) (*experiments.Variant, error) {
	m.mu.Lock()
	m.CreateVariantCalls = append(m.CreateVariantCalls, struct {
		Ctx context.Context
		V   *experiments.Variant
	}{ctx, v})
	var out *experiments.Variant
	var err error
	if len(m.CreateVariantReturns) > 0 {
		r := m.CreateVariantReturns[0]
		m.CreateVariantReturns = m.CreateVariantReturns[1:]
		out, err = r.V, r.Err
	} else {
		err = ErrNoMoreReturns
	}
	m.mu.Unlock()
	return out, err
}

func (m *Store) GetVariantsByExperimentID(ctx context.Context, experimentID string) ([]*experiments.Variant, error) {
	m.mu.Lock()
	m.GetVariantsByExperimentIDCalls = append(m.GetVariantsByExperimentIDCalls, struct {
		Ctx          context.Context
		ExperimentID string
	}{ctx, experimentID})
	var out []*experiments.Variant
	var err error
	if len(m.GetVariantsByExperimentIDReturns) > 0 {
		r := m.GetVariantsByExperimentIDReturns[0]
		m.GetVariantsByExperimentIDReturns = m.GetVariantsByExperimentIDReturns[1:]
		out, err = r.Variants, r.Err
	} else {
		err = ErrNoMoreReturns
	}
	m.mu.Unlock()
	return out, err
}

func (m *Store) GetAssignment(ctx context.Context, userID, experimentID string) (*experiments.Assignment, error) {
	m.mu.Lock()
	m.GetAssignmentCalls = append(m.GetAssignmentCalls, struct {
		Ctx          context.Context
		UserID       string
		ExperimentID string
	}{ctx, userID, experimentID})
	var out *experiments.Assignment
	var err error
	if len(m.GetAssignmentReturns) > 0 {
		r := m.GetAssignmentReturns[0]
		m.GetAssignmentReturns = m.GetAssignmentReturns[1:]
		out, err = r.A, r.Err
	} else {
		err = ErrNoMoreReturns
	}
	m.mu.Unlock()
	return out, err
}

func (m *Store) UpsertAssignment(ctx context.Context, a *experiments.Assignment) error {
	m.mu.Lock()
	m.UpsertAssignmentCalls = append(m.UpsertAssignmentCalls, struct {
		Ctx context.Context
		A   *experiments.Assignment
	}{ctx, a})
	var err error
	if len(m.UpsertAssignmentReturns) > 0 {
		err = m.UpsertAssignmentReturns[0]
		m.UpsertAssignmentReturns = m.UpsertAssignmentReturns[1:]
	} else {
		err = ErrNoMoreReturns
	}
	m.mu.Unlock()
	return err
}

var _ experiments.Store = (*Store)(nil)
