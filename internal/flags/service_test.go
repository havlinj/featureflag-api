package flags_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jan-havlin-dev/featureflag-api/graph/model"
	"github.com/jan-havlin-dev/featureflag-api/internal/flags"
	"github.com/jan-havlin-dev/featureflag-api/internal/flags/mock"
)

func stringPtr(s string) *string { return &s }

// --- CreateFlag ---

func TestService_CreateFlag_happy_path(t *testing.T) {
	ctx := context.Background()
	store := &mock.Store{}
	store.GetByKeyAndEnvironmentReturns = []mock.GetByKeyResult{
		{Flag: nil, Err: nil},
	}
	created := &flags.Flag{
		ID:          "id-1",
		Key:         "test-flag",
		Description: stringPtr("desc"),
		Enabled:     false,
		Environment: "dev",
		CreatedAt:   time.Now(),
	}
	store.CreateReturns = []mock.CreateResult{
		{Flag: created, Err: nil},
	}
	svc := flags.NewService(store)
	input := model.CreateFlagInput{
		Key:         "test-flag",
		Description: stringPtr("desc"),
		Environment: "dev",
	}

	got, err := svc.CreateFlag(ctx, input)

	if err != nil {
		t.Fatalf("CreateFlag: %v", err)
	}
	if got == nil {
		t.Fatal("expected non-nil FeatureFlag")
	}
	if got.ID != "id-1" || got.Key != "test-flag" || got.Enabled != false || got.Environment != "dev" {
		t.Errorf("got %+v", got)
	}
	if len(store.GetByKeyAndEnvironmentCalls) != 1 {
		t.Errorf("GetByKeyAndEnvironment calls: want 1, got %d", len(store.GetByKeyAndEnvironmentCalls))
	}
	if store.GetByKeyAndEnvironmentCalls[0].Key != "test-flag" || store.GetByKeyAndEnvironmentCalls[0].Env != "dev" {
		t.Errorf("GetByKeyAndEnvironment called with wrong args: %+v", store.GetByKeyAndEnvironmentCalls[0])
	}
	if len(store.CreateCalls) != 1 {
		t.Errorf("Create calls: want 1, got %d", len(store.CreateCalls))
	}
	if store.CreateCalls[0].Flag.Key != "test-flag" || store.CreateCalls[0].Flag.Enabled != false {
		t.Errorf("Create called with wrong flag: %+v", store.CreateCalls[0].Flag)
	}
}

func TestService_CreateFlag_already_exists_returns_ErrDuplicateKey(t *testing.T) {
	ctx := context.Background()
	store := &mock.Store{}
	existing := &flags.Flag{ID: "existing", Key: "test-flag", Environment: "dev"}
	store.GetByKeyAndEnvironmentReturns = []mock.GetByKeyResult{
		{Flag: existing, Err: nil},
	}
	svc := flags.NewService(store)
	input := model.CreateFlagInput{Key: "test-flag", Environment: "dev"}

	got, err := svc.CreateFlag(ctx, input)

	if got != nil {
		t.Errorf("expected nil result, got %+v", got)
	}
	if !errors.Is(err, flags.ErrDuplicateKey) {
		t.Errorf("expected ErrDuplicateKey, got %v", err)
	}
	if len(store.CreateCalls) != 0 {
		t.Errorf("Create should not be called, got %d calls", len(store.CreateCalls))
	}
}

func TestService_CreateFlag_get_existing_error_returns_wrapped_error(t *testing.T) {
	ctx := context.Background()
	store := &mock.Store{}
	wantErr := errors.New("db error")
	store.GetByKeyAndEnvironmentReturns = []mock.GetByKeyResult{
		{Flag: nil, Err: wantErr},
	}
	svc := flags.NewService(store)
	input := model.CreateFlagInput{Key: "test-flag", Environment: "dev"}

	_, err := svc.CreateFlag(ctx, input)

	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, wantErr) {
		t.Errorf("expected wrapped %v, got %v", wantErr, err)
	}
	if len(store.CreateCalls) != 0 {
		t.Errorf("Create should not be called, got %d calls", len(store.CreateCalls))
	}
}

func TestService_CreateFlag_create_error_returns_wrapped_error(t *testing.T) {
	ctx := context.Background()
	store := &mock.Store{}
	store.GetByKeyAndEnvironmentReturns = []mock.GetByKeyResult{
		{Flag: nil, Err: nil},
	}
	wantErr := errors.New("insert failed")
	store.CreateReturns = []mock.CreateResult{
		{Flag: nil, Err: wantErr},
	}
	svc := flags.NewService(store)
	input := model.CreateFlagInput{Key: "test-flag", Environment: "dev"}

	_, err := svc.CreateFlag(ctx, input)

	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, wantErr) {
		t.Errorf("expected wrapped %v, got %v", wantErr, err)
	}
}

// --- UpdateFlag ---

func TestService_UpdateFlag_happy_path(t *testing.T) {
	ctx := context.Background()
	store := &mock.Store{}
	flag := &flags.Flag{ID: "f1", Key: "test-flag", Enabled: false, Environment: "dev"}
	store.GetByKeyAndEnvironmentReturns = []mock.GetByKeyResult{
		{Flag: flag, Err: nil},
	}
	store.UpdateReturns = []error{nil}
	svc := flags.NewService(store)
	input := model.UpdateFlagInput{Key: "test-flag", Enabled: true}

	got, err := svc.UpdateFlag(ctx, input)

	if err != nil {
		t.Fatalf("UpdateFlag: %v", err)
	}
	if got == nil || !got.Enabled {
		t.Errorf("expected enabled flag, got %+v", got)
	}
	if flag.Enabled != true {
		t.Error("store flag should be updated to Enabled=true")
	}
	if len(store.GetByKeyAndEnvironmentCalls) != 1 {
		t.Errorf("GetByKeyAndEnvironment calls: want 1, got %d", len(store.GetByKeyAndEnvironmentCalls))
	}
	if store.GetByKeyAndEnvironmentCalls[0].Env != "dev" {
		t.Errorf("UpdateFlag should use defaultEnvironment dev, got %q", store.GetByKeyAndEnvironmentCalls[0].Env)
	}
	if len(store.UpdateCalls) != 1 {
		t.Errorf("Update calls: want 1, got %d", len(store.UpdateCalls))
	}
}

func TestService_UpdateFlag_not_found_returns_ErrNotFound(t *testing.T) {
	ctx := context.Background()
	store := &mock.Store{}
	store.GetByKeyAndEnvironmentReturns = []mock.GetByKeyResult{
		{Flag: nil, Err: nil},
	}
	svc := flags.NewService(store)
	input := model.UpdateFlagInput{Key: "missing", Enabled: true}

	got, err := svc.UpdateFlag(ctx, input)

	if got != nil {
		t.Errorf("expected nil result, got %+v", got)
	}
	if !errors.Is(err, flags.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
	if len(store.UpdateCalls) != 0 {
		t.Errorf("Update should not be called, got %d calls", len(store.UpdateCalls))
	}
}

func TestService_UpdateFlag_get_error_returns_wrapped_error(t *testing.T) {
	ctx := context.Background()
	store := &mock.Store{}
	wantErr := errors.New("db error")
	store.GetByKeyAndEnvironmentReturns = []mock.GetByKeyResult{
		{Flag: nil, Err: wantErr},
	}
	svc := flags.NewService(store)
	input := model.UpdateFlagInput{Key: "test-flag", Enabled: true}

	_, err := svc.UpdateFlag(ctx, input)

	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, wantErr) {
		t.Errorf("expected wrapped %v, got %v", wantErr, err)
	}
}

func TestService_UpdateFlag_update_error_returns_wrapped_error(t *testing.T) {
	ctx := context.Background()
	store := &mock.Store{}
	store.GetByKeyAndEnvironmentReturns = []mock.GetByKeyResult{
		{Flag: &flags.Flag{ID: "f1", Key: "test-flag", Environment: "dev"}, Err: nil},
	}
	wantErr := errors.New("update failed")
	store.UpdateReturns = []error{wantErr}
	svc := flags.NewService(store)
	input := model.UpdateFlagInput{Key: "test-flag", Enabled: true}

	_, err := svc.UpdateFlag(ctx, input)

	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, wantErr) {
		t.Errorf("expected wrapped %v, got %v", wantErr, err)
	}
}

// --- EvaluateFlag ---

func TestService_EvaluateFlag_empty_userID_returns_ErrInvalidUserID(t *testing.T) {
	ctx := context.Background()
	store := &mock.Store{}
	svc := flags.NewService(store)

	enabled, err := svc.EvaluateFlag(ctx, "any-key", "")

	if enabled {
		t.Error("expected false when userID is empty")
	}
	if !errors.Is(err, flags.ErrInvalidUserID) {
		t.Errorf("expected ErrInvalidUserID, got %v", err)
	}
	if len(store.GetByKeyAndEnvironmentCalls) != 0 {
		t.Error("store should not be called when userID is empty")
	}
}

func TestService_EvaluateFlag_flag_not_found_returns_false_nil(t *testing.T) {
	ctx := context.Background()
	store := &mock.Store{}
	store.GetByKeyAndEnvironmentReturns = []mock.GetByKeyResult{
		{Flag: nil, Err: nil},
	}
	svc := flags.NewService(store)

	enabled, err := svc.EvaluateFlag(ctx, "missing", "user-1")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if enabled {
		t.Error("expected false when flag not found")
	}
}

func TestService_EvaluateFlag_flag_disabled_returns_false_nil(t *testing.T) {
	ctx := context.Background()
	store := &mock.Store{}
	store.GetByKeyAndEnvironmentReturns = []mock.GetByKeyResult{
		{Flag: &flags.Flag{ID: "f1", Key: "off-flag", Enabled: false, Environment: "dev"}, Err: nil},
	}
	svc := flags.NewService(store)

	enabled, err := svc.EvaluateFlag(ctx, "off-flag", "user-1")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if enabled {
		t.Error("expected false when flag is disabled")
	}
	if len(store.GetRulesByFlagIDCalls) != 0 {
		t.Error("GetRulesByFlagID should not be called when flag is disabled")
	}
}

func TestService_EvaluateFlag_enabled_no_rules_returns_true_nil(t *testing.T) {
	ctx := context.Background()
	store := &mock.Store{}
	store.GetByKeyAndEnvironmentReturns = []mock.GetByKeyResult{
		{Flag: &flags.Flag{ID: "f1", Key: "on-flag", Enabled: true, Environment: "dev"}, Err: nil},
	}
	store.GetRulesByFlagIDReturns = []mock.GetRulesResult{
		{Rules: nil, Err: nil},
	}
	svc := flags.NewService(store)

	enabled, err := svc.EvaluateFlag(ctx, "on-flag", "user-1")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !enabled {
		t.Error("expected true when flag enabled and no rules")
	}
}

func TestService_EvaluateFlag_percentage_0_returns_false(t *testing.T) {
	ctx := context.Background()
	store := &mock.Store{}
	store.GetByKeyAndEnvironmentReturns = []mock.GetByKeyResult{
		{Flag: &flags.Flag{ID: "f1", Key: "pct", Enabled: true, Environment: "dev"}, Err: nil},
	}
	store.GetRulesByFlagIDReturns = []mock.GetRulesResult{
		{Rules: []*flags.Rule{{Type: flags.RuleTypePercentage, Value: "0"}}, Err: nil},
	}
	svc := flags.NewService(store)

	enabled, err := svc.EvaluateFlag(ctx, "pct", "any-user")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if enabled {
		t.Error("expected false for 0% rollout")
	}
}

func TestService_EvaluateFlag_percentage_100_returns_true(t *testing.T) {
	ctx := context.Background()
	store := &mock.Store{}
	store.GetByKeyAndEnvironmentReturns = []mock.GetByKeyResult{
		{Flag: &flags.Flag{ID: "f1", Key: "pct", Enabled: true, Environment: "dev"}, Err: nil},
	}
	store.GetRulesByFlagIDReturns = []mock.GetRulesResult{
		{Rules: []*flags.Rule{{Type: flags.RuleTypePercentage, Value: "100"}}, Err: nil},
	}
	svc := flags.NewService(store)

	enabled, err := svc.EvaluateFlag(ctx, "pct", "any-user")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !enabled {
		t.Error("expected true for 100% rollout")
	}
}

func TestService_EvaluateFlag_percentage_deterministic_same_user_same_result(t *testing.T) {
	ctx := context.Background()
	store := &mock.Store{}
	flag := &flags.Flag{ID: "f1", Key: "pct", Enabled: true, Environment: "dev"}
	rules := []*flags.Rule{{Type: flags.RuleTypePercentage, Value: "50"}}
	store.GetByKeyAndEnvironmentReturns = []mock.GetByKeyResult{
		{Flag: flag, Err: nil},
		{Flag: flag, Err: nil},
	}
	store.GetRulesByFlagIDReturns = []mock.GetRulesResult{
		{Rules: rules, Err: nil},
		{Rules: rules, Err: nil},
	}
	svc := flags.NewService(store)
	userID := "deterministic-user"

	got1, err1 := svc.EvaluateFlag(ctx, "pct", userID)
	got2, err2 := svc.EvaluateFlag(ctx, "pct", userID)

	if err1 != nil {
		t.Fatalf("first call: %v", err1)
	}
	if err2 != nil {
		t.Fatalf("second call: %v", err2)
	}
	if got1 != got2 {
		t.Errorf("same userID must get same result: got %v and %v", got1, got2)
	}
}

func TestService_EvaluateFlag_percentage_invalid_value_returns_ErrInvalidRule(t *testing.T) {
	ctx := context.Background()
	store := &mock.Store{}
	store.GetByKeyAndEnvironmentReturns = []mock.GetByKeyResult{
		{Flag: &flags.Flag{ID: "f1", Key: "pct", Enabled: true, Environment: "dev"}, Err: nil},
	}
	store.GetRulesByFlagIDReturns = []mock.GetRulesResult{
		{Rules: []*flags.Rule{{Type: flags.RuleTypePercentage, Value: "x"}}, Err: nil},
	}
	svc := flags.NewService(store)

	_, err := svc.EvaluateFlag(ctx, "pct", "user-1")

	if !errors.Is(err, flags.ErrInvalidRule) {
		t.Errorf("expected ErrInvalidRule, got %v", err)
	}
}

func TestService_EvaluateFlag_percentage_out_of_range_returns_ErrInvalidRule(t *testing.T) {
	ctx := context.Background()
	store := &mock.Store{}
	store.GetByKeyAndEnvironmentReturns = []mock.GetByKeyResult{
		{Flag: &flags.Flag{ID: "f1", Key: "pct", Enabled: true, Environment: "dev"}, Err: nil},
	}
	store.GetRulesByFlagIDReturns = []mock.GetRulesResult{
		{Rules: []*flags.Rule{{Type: flags.RuleTypePercentage, Value: "150"}}, Err: nil},
	}
	svc := flags.NewService(store)

	_, err := svc.EvaluateFlag(ctx, "pct", "user-1")

	if !errors.Is(err, flags.ErrInvalidRule) {
		t.Errorf("expected ErrInvalidRule, got %v", err)
	}
}

func TestService_EvaluateFlag_attribute_only_fallback_returns_true(t *testing.T) {
	ctx := context.Background()
	store := &mock.Store{}
	store.GetByKeyAndEnvironmentReturns = []mock.GetByKeyResult{
		{Flag: &flags.Flag{ID: "f1", Key: "attr", Enabled: true, Environment: "dev"}, Err: nil},
	}
	store.GetRulesByFlagIDReturns = []mock.GetRulesResult{
		{Rules: []*flags.Rule{{Type: flags.RuleTypeAttribute, Value: `{"email":"@x.com"}`}}, Err: nil},
	}
	svc := flags.NewService(store)

	enabled, err := svc.EvaluateFlag(ctx, "attr", "user-1")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !enabled {
		t.Error("expected true when only attribute rules (fallback to enabled)")
	}
}

func TestService_EvaluateFlag_get_flag_error_returns_error(t *testing.T) {
	ctx := context.Background()
	store := &mock.Store{}
	wantErr := errors.New("db error")
	store.GetByKeyAndEnvironmentReturns = []mock.GetByKeyResult{
		{Flag: nil, Err: wantErr},
	}
	svc := flags.NewService(store)

	enabled, err := svc.EvaluateFlag(ctx, "key", "user-1")

	if enabled {
		t.Error("expected false on error")
	}
	if err == nil || !errors.Is(err, wantErr) {
		t.Errorf("expected wrapped %v, got %v", wantErr, err)
	}
}

func TestService_EvaluateFlag_get_rules_error_returns_error(t *testing.T) {
	ctx := context.Background()
	store := &mock.Store{}
	store.GetByKeyAndEnvironmentReturns = []mock.GetByKeyResult{
		{Flag: &flags.Flag{ID: "f1", Key: "key", Enabled: true, Environment: "dev"}, Err: nil},
	}
	wantErr := errors.New("rules db error")
	store.GetRulesByFlagIDReturns = []mock.GetRulesResult{
		{Rules: nil, Err: wantErr},
	}
	svc := flags.NewService(store)

	enabled, err := svc.EvaluateFlag(ctx, "key", "user-1")

	if enabled {
		t.Error("expected false on error")
	}
	if err == nil || !errors.Is(err, wantErr) {
		t.Errorf("expected wrapped %v, got %v", wantErr, err)
	}
}

func TestService_EvaluateFlag_uses_default_environment_dev(t *testing.T) {
	ctx := context.Background()
	store := &mock.Store{}
	store.GetByKeyAndEnvironmentReturns = []mock.GetByKeyResult{
		{Flag: nil, Err: nil},
	}
	svc := flags.NewService(store)

	_, _ = svc.EvaluateFlag(ctx, "key", "user-1")

	if len(store.GetByKeyAndEnvironmentCalls) != 1 {
		t.Fatalf("expected 1 GetByKeyAndEnvironment call, got %d", len(store.GetByKeyAndEnvironmentCalls))
	}
	if store.GetByKeyAndEnvironmentCalls[0].Env != "dev" {
		t.Errorf("EvaluateFlag should use defaultEnvironment dev, got %q", store.GetByKeyAndEnvironmentCalls[0].Env)
	}
}
