package graphql

import (
	"context"
	"errors"
	"testing"

	"github.com/havlinj/featureflag-api/graph/model"
	"github.com/havlinj/featureflag-api/internal/auth"
	"github.com/havlinj/featureflag-api/internal/experiments"
)

// mockExperimentsService implements ExperimentsService for the experiment query tests
// that need to assert resolver-specific behaviour (not-found → null, error pass-through).
type mockExperimentsService struct {
	GetExperimentFunc func(ctx context.Context, key, environment string) (*model.Experiment, error)
}

func newTestResolverWithExperiments(expSvc ExperimentsService) *Resolver {
	return NewResolver(nil, nil, expSvc, nil, nil, 0)
}

func (m *mockExperimentsService) CreateExperiment(ctx context.Context, input model.CreateExperimentInput) (*model.Experiment, error) {
	return nil, errors.New("mock not used")
}

func (m *mockExperimentsService) GetExperiment(ctx context.Context, key, environment string) (*model.Experiment, error) {
	if m.GetExperimentFunc != nil {
		return m.GetExperimentFunc(ctx, key, environment)
	}
	return nil, errors.New("mock GetExperiment not set")
}

func (m *mockExperimentsService) GetAssignment(ctx context.Context, userID, experimentKey, environment string) (*model.ExperimentVariant, error) {
	return nil, errors.New("mock not used")
}

// TestExperiment_resolver_returns_nil_nil_when_not_found asserts the resolver-specific behaviour:
// ExperimentNotFoundError from service is mapped to (nil, nil) so GraphQL returns null without an error.
func TestExperiment_resolver_returns_nil_nil_when_not_found(t *testing.T) {
	mock := &mockExperimentsService{
		GetExperimentFunc: func(_ context.Context, key, env string) (*model.Experiment, error) {
			return nil, &experiments.ExperimentNotFoundError{Key: key, Environment: env}
		},
	}
	r := newTestResolverWithExperiments(mock)
	q := &queryResolver{r}

	exp, err := q.Experiment(auth.WithClaims(context.Background(), &auth.Claims{Sub: "u1", Role: "viewer"}), "missing", "prod")

	if err != nil {
		t.Errorf("expected nil error when experiment not found, got %v", err)
	}
	if exp != nil {
		t.Errorf("expected nil experiment when not found, got %+v", exp)
	}
}

// TestExperiment_resolver_passes_through_error_when_not_not_found asserts that when the service
// returns an error other than ExperimentNotFoundError, the resolver returns (nil, err) unchanged.
func TestExperiment_resolver_passes_through_error_when_not_not_found(t *testing.T) {
	wantErr := errors.New("db failure")
	mock := &mockExperimentsService{
		GetExperimentFunc: func(_ context.Context, _, _ string) (*model.Experiment, error) {
			return nil, wantErr
		},
	}
	r := newTestResolverWithExperiments(mock)
	q := &queryResolver{r}

	exp, err := q.Experiment(auth.WithClaims(context.Background(), &auth.Claims{Sub: "u1", Role: "viewer"}), "x", "dev")

	if exp != nil {
		t.Errorf("expected nil experiment, got %+v", exp)
	}
	if err != wantErr {
		t.Errorf("expected err %v, got %v", wantErr, err)
	}
}

// TestCreateExperiment_resolver_requires_auth documents that createExperiment requires admin or developer.
func TestCreateExperiment_resolver_requires_auth(t *testing.T) {
	r := newTestResolverWithExperiments(nil)
	mut := &mutationResolver{r}

	input := model.CreateExperimentInput{
		Key:         "ab-test",
		Environment: "dev",
		Variants:    []*model.ExperimentVariantInput{{Name: "A", Weight: 50}, {Name: "B", Weight: 50}},
	}
	_, err := mut.CreateExperiment(context.Background(), input)

	if err == nil {
		t.Fatal("expected error when context has no claims")
	}
	var authErr *auth.UnauthorizedError
	if !errors.As(err, &authErr) {
		t.Errorf("expected UnauthorizedError, got %T: %v", err, err)
	}
}
