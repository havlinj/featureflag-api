package experiments_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/havlinj/featureflag-api/graph/model"
	"github.com/havlinj/featureflag-api/internal/audit"
	"github.com/havlinj/featureflag-api/internal/auth"
	"github.com/havlinj/featureflag-api/internal/experiments"
	"github.com/havlinj/featureflag-api/internal/experiments/mock"
	"github.com/havlinj/featureflag-api/internal/testutil/auditmock"
)

func variants50_50() []*model.ExperimentVariantInput {
	return []*model.ExperimentVariantInput{
		{Name: "A", Weight: 50},
		{Name: "B", Weight: 50},
	}
}

func variants90_10() []*model.ExperimentVariantInput {
	return []*model.ExperimentVariantInput{
		{Name: "control", Weight: 90},
		{Name: "treatment", Weight: 10},
	}
}

// --- CreateExperiment ---

func TestService_CreateExperiment_happy_path(t *testing.T) {
	ctx := context.Background()
	store := &mock.Store{}
	store.GetExperimentByKeyAndEnvironmentReturns = []mock.GetExperimentResult{
		{Exp: nil, Err: nil},
	}
	createdExp := &experiments.Experiment{ID: "exp-1", Key: "my-exp", Environment: "dev"}
	store.CreateExperimentReturns = []mock.CreateExperimentResult{
		{Exp: createdExp, Err: nil},
	}
	store.CreateVariantReturns = []mock.CreateVariantResult{
		{V: &experiments.Variant{ID: "v1", ExperimentID: "exp-1", Name: "A", Weight: 50}, Err: nil},
		{V: &experiments.Variant{ID: "v2", ExperimentID: "exp-1", Name: "B", Weight: 50}, Err: nil},
	}
	svc := experiments.NewService(store)
	input := model.CreateExperimentInput{Key: "my-exp", Environment: "dev", Variants: variants50_50()}

	got, err := svc.CreateExperiment(ctx, input)

	if err != nil {
		t.Fatalf("CreateExperiment: %v", err)
	}
	if got == nil || got.ID != "exp-1" || got.Key != "my-exp" || got.Environment != "dev" {
		t.Errorf("got %+v", got)
	}
	if len(store.CreateExperimentCalls) != 1 {
		t.Errorf("CreateExperiment calls: want 1, got %d", len(store.CreateExperimentCalls))
	}
	if len(store.CreateVariantCalls) != 2 {
		t.Errorf("CreateVariant calls: want 2, got %d", len(store.CreateVariantCalls))
	}
}

func TestService_CreateExperiment_weights_not_sum_100_returns_ErrInvalidWeights(t *testing.T) {
	ctx := context.Background()
	store := &mock.Store{}
	svc := experiments.NewService(store)
	input := model.CreateExperimentInput{Key: "x", Environment: "dev", Variants: []*model.ExperimentVariantInput{
		{Name: "A", Weight: 30},
		{Name: "B", Weight: 50},
	}}

	got, err := svc.CreateExperiment(ctx, input)

	if got != nil {
		t.Errorf("expected nil result, got %+v", got)
	}
	var e *experiments.InvalidWeightsError
	if !errors.As(err, &e) {
		t.Errorf("expected *InvalidWeightsError, got %v", err)
	}
	if e.Sum != 80 {
		t.Errorf("expected Sum=80, got %d", e.Sum)
	}
	if len(e.Weights) != 2 || e.Weights[0] != 30 || e.Weights[1] != 50 {
		t.Errorf("expected Weights=[30, 50], got %v", e.Weights)
	}
	if len(store.CreateExperimentCalls) != 0 {
		t.Errorf("CreateExperiment should not be called, got %d", len(store.CreateExperimentCalls))
	}
}

func TestService_CreateExperiment_empty_variants_returns_ErrInvalidWeights(t *testing.T) {
	ctx := context.Background()
	store := &mock.Store{}
	svc := experiments.NewService(store)
	input := model.CreateExperimentInput{Key: "x", Environment: "dev", Variants: nil}

	got, err := svc.CreateExperiment(ctx, input)

	if got != nil {
		t.Errorf("expected nil result, got %+v", got)
	}
	var e *experiments.InvalidWeightsError
	if !errors.As(err, &e) {
		t.Errorf("expected *InvalidWeightsError, got %v", err)
	}
	if e.Sum != 0 || e.Weights != nil || e.Reason != "no variants provided" {
		t.Errorf("expected Sum=0, Weights=nil, Reason=no variants provided; got Sum=%d Weights=%v Reason=%q", e.Sum, e.Weights, e.Reason)
	}
}

func TestService_CreateExperiment_negative_weight_returns_ErrInvalidWeights(t *testing.T) {
	ctx := context.Background()
	store := &mock.Store{}
	svc := experiments.NewService(store)
	input := model.CreateExperimentInput{Key: "x", Environment: "dev", Variants: []*model.ExperimentVariantInput{
		{Name: "A", Weight: 60},
		{Name: "B", Weight: -10},
		{Name: "C", Weight: 50},
	}}

	got, err := svc.CreateExperiment(ctx, input)

	if got != nil {
		t.Errorf("expected nil result, got %+v", got)
	}
	var e *experiments.InvalidWeightsError
	if !errors.As(err, &e) {
		t.Errorf("expected *InvalidWeightsError, got %v", err)
	}
	if e.Reason != "negative weight" {
		t.Errorf("expected Reason=negative weight, got %q", e.Reason)
	}
	if len(e.Weights) != 2 || e.Weights[1] != -10 {
		t.Errorf("expected Weights to contain -10, got %v", e.Weights)
	}
}

func TestService_CreateExperiment_duplicate_returns_ErrDuplicateExperiment(t *testing.T) {
	ctx := context.Background()
	store := &mock.Store{}
	existing := &experiments.Experiment{ID: "e1", Key: "dup", Environment: "dev"}
	store.GetExperimentByKeyAndEnvironmentReturns = []mock.GetExperimentResult{
		{Exp: existing, Err: nil},
	}
	svc := experiments.NewService(store)
	input := model.CreateExperimentInput{Key: "dup", Environment: "dev", Variants: variants50_50()}

	got, err := svc.CreateExperiment(ctx, input)

	if got != nil {
		t.Errorf("expected nil result, got %+v", got)
	}
	var dupErr *experiments.DuplicateExperimentError
	if !errors.As(err, &dupErr) {
		t.Errorf("expected *DuplicateExperimentError, got %v", err)
	}
	if dupErr.Key != "dup" || dupErr.Environment != "dev" {
		t.Errorf("expected Key=dup Environment=dev, got Key=%q Environment=%q", dupErr.Key, dupErr.Environment)
	}
	if len(store.CreateExperimentCalls) != 0 {
		t.Errorf("CreateExperiment should not be called, got %d", len(store.CreateExperimentCalls))
	}
}

func TestService_CreateExperiment_store_create_error_returns_wrapped(t *testing.T) {
	ctx := context.Background()
	store := &mock.Store{}
	store.GetExperimentByKeyAndEnvironmentReturns = []mock.GetExperimentResult{{Exp: nil, Err: nil}}
	wantErr := errors.New("CreateExperiment failed")
	store.CreateExperimentReturns = []mock.CreateExperimentResult{{Exp: nil, Err: wantErr}}
	svc := experiments.NewService(store)
	input := model.CreateExperimentInput{Key: "x", Environment: "dev", Variants: variants50_50()}

	_, err := svc.CreateExperiment(ctx, input)

	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, wantErr) {
		t.Errorf("expected wrapped %v, got %v", wantErr, err)
	}
	var opErr *experiments.OperationError
	if !errors.As(err, &opErr) {
		t.Fatalf("expected *experiments.OperationError, got %T", err)
	}
	if opErr.Op != "experiments.service.create_experiment.store_create_experiment" {
		t.Fatalf("unexpected op %q", opErr.Op)
	}
	if opErr.Key != "x" || opErr.Environment != "dev" {
		t.Fatalf("unexpected context fields: %+v", opErr)
	}
}

func TestService_CreateExperiment_create_variant_error_returns_wrapped(t *testing.T) {
	ctx := context.Background()
	store := &mock.Store{}
	store.GetExperimentByKeyAndEnvironmentReturns = []mock.GetExperimentResult{{Exp: nil, Err: nil}}
	store.CreateExperimentReturns = []mock.CreateExperimentResult{
		{Exp: &experiments.Experiment{ID: "exp-1", Key: "x", Environment: "dev"}, Err: nil},
	}
	wantErr := errors.New("CreateVariant failed")
	store.CreateVariantReturns = []mock.CreateVariantResult{{V: nil, Err: wantErr}}
	svc := experiments.NewService(store)
	input := model.CreateExperimentInput{Key: "x", Environment: "dev", Variants: variants50_50()}

	_, err := svc.CreateExperiment(ctx, input)

	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, wantErr) {
		t.Errorf("expected wrapped %v, got %v", wantErr, err)
	}
	var opErr *experiments.OperationError
	if !errors.As(err, &opErr) {
		t.Fatalf("expected *experiments.OperationError, got %T", err)
	}
	if opErr.Op != "experiments.service.create_experiment.store_create_variant" {
		t.Fatalf("unexpected op %q", opErr.Op)
	}
	if opErr.ExperimentID != "exp-1" || opErr.VariantName != "A" || opErr.VariantWeight != 50 {
		t.Fatalf("unexpected context fields: %+v", opErr)
	}
}

// --- GetExperiment ---

func TestService_GetExperiment_happy_path(t *testing.T) {
	ctx := context.Background()
	store := &mock.Store{}
	exp := &experiments.Experiment{ID: "e1", Key: "get-exp", Environment: "prod"}
	store.GetExperimentByKeyAndEnvironmentReturns = []mock.GetExperimentResult{{Exp: exp, Err: nil}}
	svc := experiments.NewService(store)

	got, err := svc.GetExperiment(ctx, "get-exp", "prod")

	if err != nil {
		t.Fatalf("GetExperiment: %v", err)
	}
	if got == nil || got.ID != "e1" || got.Key != "get-exp" || got.Environment != "prod" {
		t.Errorf("got %+v", got)
	}
}

func TestService_GetExperiment_store_error_returns_wrapped(t *testing.T) {
	ctx := context.Background()
	store := &mock.Store{}
	wantErr := errors.New("GetExperimentByKeyAndEnvironment failed")
	store.GetExperimentByKeyAndEnvironmentReturns = []mock.GetExperimentResult{{Exp: nil, Err: wantErr}}
	svc := experiments.NewService(store)

	_, err := svc.GetExperiment(ctx, "x", "dev")

	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, wantErr) {
		t.Errorf("expected wrapped %v, got %v", wantErr, err)
	}
	var opErr *experiments.OperationError
	if !errors.As(err, &opErr) {
		t.Fatalf("expected *experiments.OperationError, got %T", err)
	}
	if opErr.Op != "experiments.service.get_experiment.store_get_by_key_and_environment" {
		t.Fatalf("unexpected op %q", opErr.Op)
	}
	if opErr.Key != "x" || opErr.Environment != "dev" {
		t.Fatalf("unexpected context fields: %+v", opErr)
	}
}

func TestService_GetExperiment_not_found_returns_ErrExperimentNotFound(t *testing.T) {
	ctx := context.Background()
	store := &mock.Store{}
	store.GetExperimentByKeyAndEnvironmentReturns = []mock.GetExperimentResult{{Exp: nil, Err: nil}}
	svc := experiments.NewService(store)

	got, err := svc.GetExperiment(ctx, "nonexistent", "dev")

	if got != nil {
		t.Errorf("expected nil result, got %+v", got)
	}
	var e *experiments.ExperimentNotFoundError
	if !errors.As(err, &e) {
		t.Errorf("expected *ExperimentNotFoundError, got %v", err)
	}
	if e.Key != "nonexistent" || e.Environment != "dev" {
		t.Errorf("expected Key=nonexistent Environment=dev, got Key=%q Environment=%q", e.Key, e.Environment)
	}
}

// --- GetAssignment ---

func TestService_GetAssignment_empty_userID_returns_ErrInvalidUserID(t *testing.T) {
	ctx := context.Background()
	store := &mock.Store{}
	svc := experiments.NewService(store)

	got, err := svc.GetAssignment(ctx, "", "exp-key", "dev")

	if got != nil {
		t.Errorf("expected nil result, got %+v", got)
	}
	var e *experiments.InvalidUserIDError
	if !errors.As(err, &e) {
		t.Errorf("expected *InvalidUserIDError, got %v", err)
	}
	if e.UserID != "" {
		t.Errorf("expected UserID=\"\", got %q", e.UserID)
	}
	if len(store.GetAssignmentCalls) != 0 {
		t.Errorf("GetAssignment should not be called, got %d", len(store.GetAssignmentCalls))
	}
}

func TestService_GetAssignment_experiment_not_found_returns_ErrExperimentNotFound(t *testing.T) {
	ctx := context.Background()
	store := &mock.Store{}
	store.GetExperimentByKeyAndEnvironmentReturns = []mock.GetExperimentResult{{Exp: nil, Err: nil}}
	svc := experiments.NewService(store)

	got, err := svc.GetAssignment(ctx, "user-1", "no-exp", "dev")

	if got != nil {
		t.Errorf("expected nil result, got %+v", got)
	}
	var e *experiments.ExperimentNotFoundError
	if !errors.As(err, &e) {
		t.Errorf("expected *ExperimentNotFoundError, got %v", err)
	}
	if e.Key != "no-exp" || e.Environment != "dev" {
		t.Errorf("expected Key=no-exp Environment=dev, got Key=%q Environment=%q", e.Key, e.Environment)
	}
}

func TestService_GetAssignment_stored_assignment_unknown_variant_returns_ErrVariantNotFound(t *testing.T) {
	ctx := context.Background()
	store := &mock.Store{}
	exp := &experiments.Experiment{ID: "exp-1", Key: "orphan", Environment: "dev"}
	store.GetExperimentByKeyAndEnvironmentReturns = []mock.GetExperimentResult{{Exp: exp, Err: nil}}
	store.GetVariantsByExperimentIDReturns = []mock.GetVariantsResult{
		{Variants: []*experiments.Variant{{ID: "vA", ExperimentID: "exp-1", Name: "A", Weight: 100}}, Err: nil},
	}
	store.GetAssignmentReturns = []mock.GetAssignmentResult{
		{A: &experiments.Assignment{UserID: "u1", ExperimentID: "exp-1", VariantID: "deleted-variant-id"}, Err: nil},
	}
	svc := experiments.NewService(store)

	got, err := svc.GetAssignment(ctx, "u1", "orphan", "dev")

	if got != nil {
		t.Errorf("expected nil result, got %+v", got)
	}
	var e *experiments.VariantNotFoundError
	if !errors.As(err, &e) {
		t.Errorf("expected *VariantNotFoundError, got %v", err)
	}
	if e.VariantID != "deleted-variant-id" {
		t.Errorf("expected VariantID=deleted-variant-id, got %q", e.VariantID)
	}
}

func TestService_GetAssignment_existing_assignment_returns_stored_variant(t *testing.T) {
	ctx := context.Background()
	store := &mock.Store{}
	exp := &experiments.Experiment{ID: "exp-1", Key: "ab", Environment: "dev"}
	store.GetExperimentByKeyAndEnvironmentReturns = []mock.GetExperimentResult{{Exp: exp, Err: nil}}
	variantB := &experiments.Variant{ID: "vB", ExperimentID: "exp-1", Name: "B", Weight: 50}
	store.GetVariantsByExperimentIDReturns = []mock.GetVariantsResult{
		{Variants: []*experiments.Variant{
			{ID: "vA", ExperimentID: "exp-1", Name: "A", Weight: 50},
			variantB,
		}, Err: nil},
	}
	store.GetAssignmentReturns = []mock.GetAssignmentResult{
		{A: &experiments.Assignment{UserID: "user-1", ExperimentID: "exp-1", VariantID: "vB"}, Err: nil},
	}
	svc := experiments.NewService(store)

	got, err := svc.GetAssignment(ctx, "user-1", "ab", "dev")

	if err != nil {
		t.Fatalf("GetAssignment: %v", err)
	}
	if got == nil || got.ID != "vB" || got.Name != "B" {
		t.Errorf("got %+v", got)
	}
	if len(store.UpsertAssignmentCalls) != 0 {
		t.Errorf("UpsertAssignment should not be called when assignment exists, got %d", len(store.UpsertAssignmentCalls))
	}
}

func TestService_GetAssignment_no_variants_returns_ErrVariantNotFound(t *testing.T) {
	ctx := context.Background()
	store := &mock.Store{}
	exp := &experiments.Experiment{ID: "exp-1", Key: "empty", Environment: "dev"}
	store.GetExperimentByKeyAndEnvironmentReturns = []mock.GetExperimentResult{{Exp: exp, Err: nil}}
	store.GetVariantsByExperimentIDReturns = []mock.GetVariantsResult{{Variants: nil, Err: nil}}
	store.GetAssignmentReturns = []mock.GetAssignmentResult{{A: nil, Err: nil}}
	svc := experiments.NewService(store)

	got, err := svc.GetAssignment(ctx, "user-1", "empty", "dev")

	if got != nil {
		t.Errorf("expected nil result, got %+v", got)
	}
	var e *experiments.VariantNotFoundError
	if !errors.As(err, &e) {
		t.Errorf("expected *VariantNotFoundError, got %v", err)
	}
	if e.ExperimentKey != "empty" || e.Environment != "dev" || e.VariantID != "" {
		t.Errorf("expected ExperimentKey=empty Environment=dev VariantID=empty; got ExperimentKey=%q Environment=%q VariantID=%q", e.ExperimentKey, e.Environment, e.VariantID)
	}
}

func TestService_GetAssignment_new_assignment_computed_and_persisted(t *testing.T) {
	ctx := context.Background()
	store := &mock.Store{}
	exp := &experiments.Experiment{ID: "exp-1", Key: "new", Environment: "dev"}
	store.GetExperimentByKeyAndEnvironmentReturns = []mock.GetExperimentResult{{Exp: exp, Err: nil}}
	variants := []*experiments.Variant{
		{ID: "vA", ExperimentID: "exp-1", Name: "A", Weight: 50},
		{ID: "vB", ExperimentID: "exp-1", Name: "B", Weight: 50},
	}
	store.GetVariantsByExperimentIDReturns = []mock.GetVariantsResult{{Variants: variants, Err: nil}}
	store.GetAssignmentReturns = []mock.GetAssignmentResult{{A: nil, Err: nil}}
	store.UpsertAssignmentReturns = []error{nil}
	svc := experiments.NewService(store)

	got, err := svc.GetAssignment(ctx, "user-deterministic", "new", "dev")

	if err != nil {
		t.Fatalf("GetAssignment: %v", err)
	}
	if got == nil {
		t.Fatal("expected variant")
	}
	if got.Name != "A" && got.Name != "B" {
		t.Errorf("expected variant A or B, got %s", got.Name)
	}
	if len(store.UpsertAssignmentCalls) != 1 {
		t.Errorf("UpsertAssignment calls: want 1, got %d", len(store.UpsertAssignmentCalls))
	}
	if store.UpsertAssignmentCalls[0].A.UserID != "user-deterministic" || store.UpsertAssignmentCalls[0].A.ExperimentID != "exp-1" {
		t.Errorf("UpsertAssignment called with wrong args: %+v", store.UpsertAssignmentCalls[0])
	}
}

func TestService_GetAssignment_upsert_error_returns_wrapped(t *testing.T) {
	ctx := context.Background()
	store := &mock.Store{}
	exp := &experiments.Experiment{ID: "exp-1", Key: "upsert-err", Environment: "dev"}
	store.GetExperimentByKeyAndEnvironmentReturns = []mock.GetExperimentResult{{Exp: exp, Err: nil}}
	variants := []*experiments.Variant{
		{ID: "vA", ExperimentID: "exp-1", Name: "A", Weight: 100},
	}
	store.GetVariantsByExperimentIDReturns = []mock.GetVariantsResult{{Variants: variants, Err: nil}}
	store.GetAssignmentReturns = []mock.GetAssignmentResult{{A: nil, Err: nil}}
	wantErr := errors.New("UpsertAssignment failed")
	store.UpsertAssignmentReturns = []error{wantErr}
	svc := experiments.NewService(store)

	got, err := svc.GetAssignment(ctx, "user-1", "upsert-err", "dev")

	if got != nil {
		t.Errorf("expected nil result, got %+v", got)
	}
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, wantErr) {
		t.Errorf("expected wrapped %v, got %v", wantErr, err)
	}
	var opErr *experiments.OperationError
	if !errors.As(err, &opErr) {
		t.Fatalf("expected *experiments.OperationError, got %T", err)
	}
	if opErr.Op != "experiments.service.get_assignment.store_upsert_assignment" {
		t.Fatalf("unexpected op %q", opErr.Op)
	}
	if opErr.UserID != "user-1" || opErr.ExperimentID != "exp-1" || opErr.VariantID != "vA" || opErr.Key != "upsert-err" || opErr.Environment != "dev" {
		t.Fatalf("unexpected context fields: %+v", opErr)
	}
}

func TestService_GetAssignment_upsert_error_includes_context(t *testing.T) {
	ctx := context.Background()
	store := &mock.Store{}
	exp := &experiments.Experiment{ID: "exp-1", Key: "upsert-ctx", Environment: "dev"}
	store.GetExperimentByKeyAndEnvironmentReturns = []mock.GetExperimentResult{{Exp: exp, Err: nil}}
	store.GetVariantsByExperimentIDReturns = []mock.GetVariantsResult{
		{Variants: []*experiments.Variant{{ID: "vA", ExperimentID: "exp-1", Name: "A", Weight: 100}}, Err: nil},
	}
	store.GetAssignmentReturns = []mock.GetAssignmentResult{{A: nil, Err: nil}}
	wantErr := errors.New("UpsertAssignment failed")
	store.UpsertAssignmentReturns = []error{wantErr}
	svc := experiments.NewService(store)

	_, err := svc.GetAssignment(ctx, "user-1", "upsert-ctx", "dev")

	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected wrapped %v, got %v", wantErr, err)
	}
	var opErr *experiments.OperationError
	if !errors.As(err, &opErr) {
		t.Fatalf("expected *experiments.OperationError, got %T", err)
	}
	if opErr.Op != "experiments.service.get_assignment.store_upsert_assignment" {
		t.Fatalf("unexpected op %q", opErr.Op)
	}
	if opErr.UserID != "user-1" || opErr.ExperimentID != "exp-1" || opErr.VariantID != "vA" || opErr.Key != "upsert-ctx" || opErr.Environment != "dev" {
		t.Fatalf("unexpected context fields: %+v", opErr)
	}
}

func TestService_GetAssignment_deterministic_same_user_same_variant(t *testing.T) {
	ctx := context.Background()
	store := &mock.Store{}
	exp := &experiments.Experiment{ID: "exp-1", Key: "det", Environment: "dev"}
	store.GetExperimentByKeyAndEnvironmentReturns = []mock.GetExperimentResult{{Exp: exp, Err: nil}, {Exp: exp, Err: nil}}
	variants := []*experiments.Variant{
		{ID: "vA", ExperimentID: "exp-1", Name: "A", Weight: 50},
		{ID: "vB", ExperimentID: "exp-1", Name: "B", Weight: 50},
	}
	store.GetVariantsByExperimentIDReturns = []mock.GetVariantsResult{{Variants: variants, Err: nil}, {Variants: variants, Err: nil}}
	store.GetAssignmentReturns = []mock.GetAssignmentResult{{A: nil, Err: nil}, {A: nil, Err: nil}}
	store.UpsertAssignmentReturns = []error{nil, nil}
	svc := experiments.NewService(store)

	got1, err1 := svc.GetAssignment(ctx, "same-user", "det", "dev")
	got2, err2 := svc.GetAssignment(ctx, "same-user", "det", "dev")

	if err1 != nil || err2 != nil {
		t.Fatalf("GetAssignment: err1=%v err2=%v", err1, err2)
	}
	if got1.Name != got2.Name {
		t.Errorf("deterministic: same user should get same variant, got %q and %q", got1.Name, got2.Name)
	}
}

type txAwareExperimentsStoreMock struct {
	inner experiments.Store
}

func (s *txAwareExperimentsStoreMock) CreateExperiment(ctx context.Context, exp *experiments.Experiment) (*experiments.Experiment, error) {
	return s.inner.CreateExperiment(ctx, exp)
}

func (s *txAwareExperimentsStoreMock) GetExperimentByKeyAndEnvironment(ctx context.Context, key, environment string) (*experiments.Experiment, error) {
	return s.inner.GetExperimentByKeyAndEnvironment(ctx, key, environment)
}

func (s *txAwareExperimentsStoreMock) GetExperimentByID(ctx context.Context, id string) (*experiments.Experiment, error) {
	return s.inner.GetExperimentByID(ctx, id)
}

func (s *txAwareExperimentsStoreMock) CreateVariant(ctx context.Context, v *experiments.Variant) (*experiments.Variant, error) {
	return s.inner.CreateVariant(ctx, v)
}

func (s *txAwareExperimentsStoreMock) GetVariantsByExperimentID(ctx context.Context, experimentID string) ([]*experiments.Variant, error) {
	return s.inner.GetVariantsByExperimentID(ctx, experimentID)
}

func (s *txAwareExperimentsStoreMock) GetAssignment(ctx context.Context, userID, experimentID string) (*experiments.Assignment, error) {
	return s.inner.GetAssignment(ctx, userID, experimentID)
}

func (s *txAwareExperimentsStoreMock) UpsertAssignment(ctx context.Context, a *experiments.Assignment) error {
	return s.inner.UpsertAssignment(ctx, a)
}

func (s *txAwareExperimentsStoreMock) WithTx(tx *sql.Tx) experiments.Store {
	return s
}

type txCapableExperimentsStoreMock struct {
	txAwareExperimentsStoreMock
	beginErr error
}

func (s *txCapableExperimentsStoreMock) BeginTx(ctx context.Context) (*sql.Tx, error) {
	return nil, s.beginErr
}

func TestService_CreateExperiment_withAudit_missingActor_returns_error(t *testing.T) {
	store := &txAwareExperimentsStoreMock{inner: &mock.Store{}}
	svc := experiments.NewServiceWithAudit(store, &auditmock.TxAware{})
	input := model.CreateExperimentInput{
		Key:         "ab-test",
		Environment: "dev",
		Variants:    variants50_50(),
	}

	_, err := svc.CreateExperiment(context.Background(), input)

	var e *audit.MissingActorIDError
	if !errors.As(err, &e) {
		t.Fatalf("expected *audit.MissingActorIDError, got %T (%v)", err, err)
	}
}

func TestService_CreateExperiment_withAudit_notTxAwareAuditStore_returns_error(t *testing.T) {
	store := &txAwareExperimentsStoreMock{inner: &mock.Store{}}
	svc := experiments.NewServiceWithAudit(store, &auditmock.TxStarter{})
	input := model.CreateExperimentInput{
		Key:         "ab-test",
		Environment: "dev",
		Variants:    variants50_50(),
	}
	ctx := auth.WithActorID(context.Background(), "u1")

	_, err := svc.CreateExperiment(ctx, input)

	var e *audit.TxAwareRequiredError
	if !errors.As(err, &e) {
		t.Fatalf("expected *audit.TxAwareRequiredError, got %T (%v)", err, err)
	}
}

func TestService_CreateExperiment_withAudit_beginTx_error_is_returned(t *testing.T) {
	store := &txAwareExperimentsStoreMock{inner: &mock.Store{}}
	wantErr := errors.New("begin tx failed")
	svc := experiments.NewServiceWithAudit(store, &auditmock.TxAware{
		TxStarter: auditmock.TxStarter{BeginErr: wantErr},
	})
	input := model.CreateExperimentInput{
		Key:         "ab-test",
		Environment: "dev",
		Variants:    variants50_50(),
	}
	ctx := auth.WithActorID(context.Background(), "u1")

	_, err := svc.CreateExperiment(ctx, input)

	if !errors.Is(err, wantErr) {
		t.Fatalf("expected %v, got %v", wantErr, err)
	}
}

func TestService_CreateExperiment_withoutAudit_storeBeginTx_error_is_returned(t *testing.T) {
	store := &txCapableExperimentsStoreMock{
		txAwareExperimentsStoreMock: txAwareExperimentsStoreMock{inner: &mock.Store{}},
		beginErr:                    errors.New("begin tx failed"),
	}
	svc := experiments.NewService(store)
	input := model.CreateExperimentInput{
		Key:         "ab-test",
		Environment: "dev",
		Variants:    variants50_50(),
	}

	_, err := svc.CreateExperiment(context.Background(), input)

	if !errors.Is(err, store.beginErr) {
		t.Fatalf("expected %v, got %v", store.beginErr, err)
	}
}
