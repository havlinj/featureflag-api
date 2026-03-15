package graphql

import (
	"context"
	"errors"
	"testing"

	"github.com/havlinj/featureflag-api/graph/model"
	"github.com/havlinj/featureflag-api/internal/auth"
	"github.com/havlinj/featureflag-api/internal/experiments"
)

// mockExperimentsService implements ExperimentsService for resolver tests.
type mockExperimentsService struct {
	CreateExperimentFunc func(ctx context.Context, input model.CreateExperimentInput) (*model.Experiment, error)
	GetExperimentFunc    func(ctx context.Context, key, environment string) (*model.Experiment, error)
	GetAssignmentFunc    func(ctx context.Context, userID, experimentKey, environment string) (*model.ExperimentVariant, error)
}

func (m *mockExperimentsService) CreateExperiment(ctx context.Context, input model.CreateExperimentInput) (*model.Experiment, error) {
	if m.CreateExperimentFunc != nil {
		return m.CreateExperimentFunc(ctx, input)
	}
	return nil, errors.New("mock CreateExperiment not set")
}

func (m *mockExperimentsService) GetExperiment(ctx context.Context, key, environment string) (*model.Experiment, error) {
	if m.GetExperimentFunc != nil {
		return m.GetExperimentFunc(ctx, key, environment)
	}
	return nil, errors.New("mock GetExperiment not set")
}

func (m *mockExperimentsService) GetAssignment(ctx context.Context, userID, experimentKey, environment string) (*model.ExperimentVariant, error) {
	if m.GetAssignmentFunc != nil {
		return m.GetAssignmentFunc(ctx, userID, experimentKey, environment)
	}
	return nil, errors.New("mock GetAssignment not set")
}

func TestExperiment_resolver_returns_nil_nil_when_not_found(t *testing.T) {
	mock := &mockExperimentsService{
		GetExperimentFunc: func(_ context.Context, key, env string) (*model.Experiment, error) {
			return nil, &experiments.ExperimentNotFoundError{Key: key, Environment: env}
		},
	}
	r := &Resolver{Experiments: mock}
	q := &queryResolver{r}

	exp, err := q.Experiment(auth.WithClaims(context.Background(), &auth.Claims{Sub: "u1", Role: "viewer"}), "missing", "prod")

	if err != nil {
		t.Errorf("expected nil error when experiment not found, got %v", err)
	}
	if exp != nil {
		t.Errorf("expected nil experiment when not found, got %+v", exp)
	}
}

func TestExperiment_resolver_returns_error_on_other_errors(t *testing.T) {
	wantErr := errors.New("db failure")
	mock := &mockExperimentsService{
		GetExperimentFunc: func(_ context.Context, _, _ string) (*model.Experiment, error) {
			return nil, wantErr
		},
	}
	r := &Resolver{Experiments: mock}
	q := &queryResolver{r}

	exp, err := q.Experiment(auth.WithClaims(context.Background(), &auth.Claims{Sub: "u1", Role: "viewer"}), "x", "dev")

	if exp != nil {
		t.Errorf("expected nil experiment, got %+v", exp)
	}
	if err != wantErr {
		t.Errorf("expected err %v, got %v", wantErr, err)
	}
}

func TestExperiment_resolver_returns_experiment_on_success(t *testing.T) {
	want := &model.Experiment{ID: "e1", Key: "ab-test", Environment: "dev"}
	mock := &mockExperimentsService{
		GetExperimentFunc: func(_ context.Context, _, _ string) (*model.Experiment, error) {
			return want, nil
		},
	}
	r := &Resolver{Experiments: mock}
	q := &queryResolver{r}

	got, err := q.Experiment(auth.WithClaims(context.Background(), &auth.Claims{Sub: "u1", Role: "viewer"}), "ab-test", "dev")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != want {
		t.Errorf("expected experiment %+v, got %+v", want, got)
	}
}

func TestExperiment_resolver_requires_auth(t *testing.T) {
	mock := &mockExperimentsService{
		GetExperimentFunc: func(_ context.Context, _, _ string) (*model.Experiment, error) {
			return &model.Experiment{}, nil
		},
	}
	r := &Resolver{Experiments: mock}
	q := &queryResolver{r}

	_, err := q.Experiment(context.Background(), "x", "dev")

	if err == nil {
		t.Fatal("expected error when context has no claims")
	}
	var authErr *auth.UnauthorizedError
	if !errors.As(err, &authErr) {
		t.Errorf("expected UnauthorizedError, got %T: %v", err, err)
	}
}

func TestCreateExperiment_resolver_requires_auth(t *testing.T) {
	mock := &mockExperimentsService{
		CreateExperimentFunc: func(_ context.Context, _ model.CreateExperimentInput) (*model.Experiment, error) {
			return &model.Experiment{}, nil
		},
	}
	r := &Resolver{Experiments: mock}
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

func TestCreateExperiment_resolver_returns_error_when_experiments_nil(t *testing.T) {
	r := &Resolver{Experiments: nil}
	mut := &mutationResolver{r}

	input := model.CreateExperimentInput{
		Key:         "ab-test",
		Environment: "dev",
		Variants:    []*model.ExperimentVariantInput{{Name: "A", Weight: 50}, {Name: "B", Weight: 50}},
	}
	_, err := mut.CreateExperiment(auth.WithClaims(context.Background(), &auth.Claims{Sub: "u1", Role: "admin"}), input)

	if err == nil {
		t.Fatal("expected error when Experiments service is nil")
	}
	if err.Error() != "experiments service not configured" {
		t.Errorf("expected 'experiments service not configured', got %q", err.Error())
	}
}

func TestGetAssignment_resolver_returns_error_when_experiments_nil(t *testing.T) {
	r := &Resolver{Experiments: nil}
	q := &queryResolver{r}

	_, err := q.GetAssignment(auth.WithClaims(context.Background(), &auth.Claims{Sub: "u1", Role: "viewer"}), "user-1", "ab-test", "dev")

	if err == nil {
		t.Fatal("expected error when Experiments service is nil")
	}
	if err.Error() != "experiments service not configured" {
		t.Errorf("expected 'experiments service not configured', got %q", err.Error())
	}
}

func TestCreateExperiment_resolver_delegates_to_service(t *testing.T) {
	want := &model.Experiment{ID: "e1", Key: "ab-test", Environment: "dev"}
	mock := &mockExperimentsService{
		CreateExperimentFunc: func(_ context.Context, input model.CreateExperimentInput) (*model.Experiment, error) {
			if input.Key != "ab-test" || input.Environment != "dev" {
				return nil, errors.New("unexpected input")
			}
			return want, nil
		},
	}
	r := &Resolver{Experiments: mock}
	mut := &mutationResolver{r}

	input := model.CreateExperimentInput{
		Key:         "ab-test",
		Environment: "dev",
		Variants:    []*model.ExperimentVariantInput{{Name: "control", Weight: 50}, {Name: "treatment", Weight: 50}},
	}
	got, err := mut.CreateExperiment(auth.WithClaims(context.Background(), &auth.Claims{Sub: "u1", Role: "admin"}), input)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != want {
		t.Errorf("expected %+v, got %+v", want, got)
	}
}

func TestCreateExperiment_resolver_forbids_viewer_role(t *testing.T) {
	mock := &mockExperimentsService{
		CreateExperimentFunc: func(_ context.Context, _ model.CreateExperimentInput) (*model.Experiment, error) {
			return &model.Experiment{}, nil
		},
	}
	r := &Resolver{Experiments: mock}
	mut := &mutationResolver{r}

	input := model.CreateExperimentInput{
		Key:         "ab-test",
		Environment: "dev",
		Variants:    []*model.ExperimentVariantInput{{Name: "A", Weight: 50}, {Name: "B", Weight: 50}},
	}
	_, err := mut.CreateExperiment(auth.WithClaims(context.Background(), &auth.Claims{Sub: "u1", Role: "viewer"}), input)

	if err == nil {
		t.Fatal("expected error when viewer calls createExperiment")
	}
	var forbid *auth.ForbiddenError
	if !errors.As(err, &forbid) {
		t.Errorf("expected ForbiddenError, got %T: %v", err, err)
	}
}

func TestCreateExperiment_resolver_returns_service_error(t *testing.T) {
	mock := &mockExperimentsService{
		CreateExperimentFunc: func(_ context.Context, _ model.CreateExperimentInput) (*model.Experiment, error) {
			return nil, &experiments.DuplicateExperimentError{Key: "ab-test", Environment: "dev"}
		},
	}
	r := &Resolver{Experiments: mock}
	mut := &mutationResolver{r}

	input := model.CreateExperimentInput{
		Key:         "ab-test",
		Environment: "dev",
		Variants:    []*model.ExperimentVariantInput{{Name: "A", Weight: 50}, {Name: "B", Weight: 50}},
	}
	got, err := mut.CreateExperiment(auth.WithClaims(context.Background(), &auth.Claims{Sub: "u1", Role: "admin"}), input)

	if got != nil {
		t.Errorf("expected nil experiment on error, got %+v", got)
	}
	var duplicateErr *experiments.DuplicateExperimentError
	if !errors.As(err, &duplicateErr) {
		t.Errorf("expected DuplicateExperimentError, got %v", err)
	}
}

func TestExperiment_resolver_returns_error_when_experiments_nil(t *testing.T) {
	r := &Resolver{Experiments: nil}
	q := &queryResolver{r}

	exp, err := q.Experiment(auth.WithClaims(context.Background(), &auth.Claims{Sub: "u1", Role: "viewer"}), "x", "dev")

	if exp != nil {
		t.Errorf("expected nil experiment, got %+v", exp)
	}
	if err == nil {
		t.Fatal("expected error when Experiments service is nil")
	}
	if err.Error() != "experiments service not configured" {
		t.Errorf("expected 'experiments service not configured', got %q", err.Error())
	}
}

func TestGetAssignment_resolver_requires_auth(t *testing.T) {
	mock := &mockExperimentsService{
		GetAssignmentFunc: func(_ context.Context, _, _, _ string) (*model.ExperimentVariant, error) {
			return &model.ExperimentVariant{}, nil
		},
	}
	r := &Resolver{Experiments: mock}
	q := &queryResolver{r}

	_, err := q.GetAssignment(context.Background(), "user-1", "ab-test", "dev")

	if err == nil {
		t.Fatal("expected error when context has no claims")
	}
	var authErr *auth.UnauthorizedError
	if !errors.As(err, &authErr) {
		t.Errorf("expected UnauthorizedError, got %T: %v", err, err)
	}
}

func TestGetAssignment_resolver_delegates_to_service(t *testing.T) {
	want := &model.ExperimentVariant{ID: "v1", ExperimentID: "e1", Name: "control", Weight: 50}
	mock := &mockExperimentsService{
		GetAssignmentFunc: func(_ context.Context, userID, expKey, env string) (*model.ExperimentVariant, error) {
			if userID != "user-1" || expKey != "ab-test" || env != "dev" {
				return nil, errors.New("unexpected args")
			}
			return want, nil
		},
	}
	r := &Resolver{Experiments: mock}
	q := &queryResolver{r}

	got, err := q.GetAssignment(auth.WithClaims(context.Background(), &auth.Claims{Sub: "u1", Role: "viewer"}), "user-1", "ab-test", "dev")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != want {
		t.Errorf("expected %+v, got %+v", want, got)
	}
}

func TestGetAssignment_resolver_returns_service_error(t *testing.T) {
	mock := &mockExperimentsService{
		GetAssignmentFunc: func(_ context.Context, _, _, _ string) (*model.ExperimentVariant, error) {
			return nil, &experiments.ExperimentNotFoundError{Key: "missing", Environment: "prod"}
		},
	}
	r := &Resolver{Experiments: mock}
	q := &queryResolver{r}

	got, err := q.GetAssignment(auth.WithClaims(context.Background(), &auth.Claims{Sub: "u1", Role: "viewer"}), "user-1", "missing", "prod")

	if got != nil {
		t.Errorf("expected nil variant on error, got %+v", got)
	}
	var nfErr *experiments.ExperimentNotFoundError
	if !errors.As(err, &nfErr) {
		t.Errorf("expected ExperimentNotFoundError, got %v", err)
	}
}
