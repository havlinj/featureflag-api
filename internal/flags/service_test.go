package flags_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/havlinj/featureflag-api/graph/model"
	"github.com/havlinj/featureflag-api/internal/audit"
	"github.com/havlinj/featureflag-api/internal/auth"
	"github.com/havlinj/featureflag-api/internal/flags"
	"github.com/havlinj/featureflag-api/internal/flags/mock"
)

func stringPtr(s string) *string { return &s }

func evalCtx(userID string) model.EvaluationContextInput {
	return model.EvaluationContextInput{UserID: userID}
}

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
		Environment: flags.DeploymentStageDev,
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
	if got.ID != "id-1" || got.Key != "test-flag" || got.Enabled != false || got.Environment != string(flags.DeploymentStageDev) {
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
	existing := &flags.Flag{ID: "existing", Key: "test-flag", Environment: flags.DeploymentStageDev}
	store.GetByKeyAndEnvironmentReturns = []mock.GetByKeyResult{
		{Flag: existing, Err: nil},
	}
	svc := flags.NewService(store)
	input := model.CreateFlagInput{Key: "test-flag", Environment: "dev"}

	got, err := svc.CreateFlag(ctx, input)

	if got != nil {
		t.Errorf("expected nil result, got %+v", got)
	}
	var e *flags.DuplicateKeyError
	if !errors.As(err, &e) {
		t.Errorf("expected *DuplicateKeyError, got %v", err)
	}
	if e.Key != "test-flag" || e.Environment != "dev" {
		t.Errorf("expected Key=test-flag Environment=dev, got Key=%q Environment=%q", e.Key, e.Environment)
	}
	if len(store.CreateCalls) != 0 {
		t.Errorf("Create should not be called, got %d calls", len(store.CreateCalls))
	}
}

func TestService_CreateFlag_get_existing_error_returns_wrapped_error(t *testing.T) {
	ctx := context.Background()
	store := &mock.Store{}
	wantErr := errors.New("GetByKeyAndEnvironment failed")
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
	wantErr := errors.New("Create failed")
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
	flag := &flags.Flag{ID: "f1", Key: "test-flag", Enabled: false, Environment: flags.DeploymentStageDev}
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
	if store.GetByKeyAndEnvironmentCalls[0].Env != flags.DeploymentStageDev {
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
	var e *flags.NotFoundError
	if !errors.As(err, &e) {
		t.Errorf("expected *NotFoundError, got %v", err)
	}
	if e.Key != "missing" || e.Environment != "dev" {
		t.Errorf("expected Key=missing Environment=dev, got Key=%q Environment=%q", e.Key, e.Environment)
	}
	if len(store.UpdateCalls) != 0 {
		t.Errorf("Update should not be called, got %d calls", len(store.UpdateCalls))
	}
}

func TestService_UpdateFlag_get_error_returns_wrapped_error(t *testing.T) {
	ctx := context.Background()
	store := &mock.Store{}
	wantErr := errors.New("GetByKeyAndEnvironment failed")
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
		{Flag: &flags.Flag{ID: "f1", Key: "test-flag", Environment: flags.DeploymentStageDev}, Err: nil},
	}
	wantErr := errors.New("Update failed")
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

	enabled, err := svc.EvaluateFlag(ctx, "any-key", evalCtx(""))

	if enabled {
		t.Error("expected false when userID is empty")
	}
	var e *flags.InvalidUserIDError
	if !errors.As(err, &e) {
		t.Errorf("expected *InvalidUserIDError, got %v", err)
	}
	if e.UserID != "" {
		t.Errorf("expected UserID empty, got %q", e.UserID)
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

	enabled, err := svc.EvaluateFlag(ctx, "missing", evalCtx("user-1"))

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
		{Flag: &flags.Flag{ID: "f1", Key: "off-flag", Enabled: false, Environment: flags.DeploymentStageDev, RolloutStrategy: flags.RolloutStrategyNone}, Err: nil},
	}
	svc := flags.NewService(store)

	enabled, err := svc.EvaluateFlag(ctx, "off-flag", evalCtx("user-1"))

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
		{Flag: &flags.Flag{ID: "f1", Key: "on-flag", Enabled: true, Environment: flags.DeploymentStageDev, RolloutStrategy: flags.RolloutStrategyNone}, Err: nil},
	}
	store.GetRulesByFlagIDReturns = []mock.GetRulesResult{
		{Rules: nil, Err: nil},
	}
	svc := flags.NewService(store)

	enabled, err := svc.EvaluateFlag(ctx, "on-flag", evalCtx("user-1"))

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
		{Flag: &flags.Flag{ID: "f1", Key: "pct", Enabled: true, Environment: flags.DeploymentStageDev, RolloutStrategy: flags.RolloutStrategyPercentage}, Err: nil},
	}
	store.GetRulesByFlagIDReturns = []mock.GetRulesResult{
		{Rules: []*flags.Rule{{Type: flags.RuleTypePercentage, Value: "0"}}, Err: nil},
	}
	svc := flags.NewService(store)

	enabled, err := svc.EvaluateFlag(ctx, "pct", evalCtx("any-user"))

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
		{Flag: &flags.Flag{ID: "f1", Key: "pct", Enabled: true, Environment: flags.DeploymentStageDev, RolloutStrategy: flags.RolloutStrategyPercentage}, Err: nil},
	}
	store.GetRulesByFlagIDReturns = []mock.GetRulesResult{
		{Rules: []*flags.Rule{{Type: flags.RuleTypePercentage, Value: "100"}}, Err: nil},
	}
	svc := flags.NewService(store)

	enabled, err := svc.EvaluateFlag(ctx, "pct", evalCtx("any-user"))

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
	flag := &flags.Flag{ID: "f1", Key: "pct", Enabled: true, Environment: flags.DeploymentStageDev, RolloutStrategy: flags.RolloutStrategyPercentage}
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

	got1, err1 := svc.EvaluateFlag(ctx, "pct", evalCtx(userID))
	got2, err2 := svc.EvaluateFlag(ctx, "pct", evalCtx(userID))

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
		{Flag: &flags.Flag{ID: "f1", Key: "pct", Enabled: true, Environment: flags.DeploymentStageDev, RolloutStrategy: flags.RolloutStrategyPercentage}, Err: nil},
	}
	store.GetRulesByFlagIDReturns = []mock.GetRulesResult{
		{Rules: []*flags.Rule{{Type: flags.RuleTypePercentage, Value: "x"}}, Err: nil},
	}
	svc := flags.NewService(store)

	_, err := svc.EvaluateFlag(ctx, "pct", evalCtx("user-1"))

	var e *flags.InvalidRuleError
	if !errors.As(err, &e) {
		t.Errorf("expected *InvalidRuleError, got %v", err)
	}
	if e.Value != "x" || e.Reason != "not a number" {
		t.Errorf("expected Value=x Reason=not a number, got Value=%q Reason=%q", e.Value, e.Reason)
	}
}

func TestService_EvaluateFlag_percentage_out_of_range_returns_ErrInvalidRule(t *testing.T) {
	ctx := context.Background()
	store := &mock.Store{}
	store.GetByKeyAndEnvironmentReturns = []mock.GetByKeyResult{
		{Flag: &flags.Flag{ID: "f1", Key: "pct", Enabled: true, Environment: flags.DeploymentStageDev, RolloutStrategy: flags.RolloutStrategyPercentage}, Err: nil},
	}
	store.GetRulesByFlagIDReturns = []mock.GetRulesResult{
		{Rules: []*flags.Rule{{Type: flags.RuleTypePercentage, Value: "150"}}, Err: nil},
	}
	svc := flags.NewService(store)

	_, err := svc.EvaluateFlag(ctx, "pct", evalCtx("user-1"))

	var e *flags.InvalidRuleError
	if !errors.As(err, &e) {
		t.Errorf("expected *InvalidRuleError, got %v", err)
	}
	if e.Value != "150" || e.Reason != "must be 0-100" {
		t.Errorf("expected Value=150 Reason=must be 0-100, got Value=%q Reason=%q", e.Value, e.Reason)
	}
}

func TestService_EvaluateFlag_attribute_rule_match_returns_true(t *testing.T) {
	ctx := context.Background()
	store := &mock.Store{}
	store.GetByKeyAndEnvironmentReturns = []mock.GetByKeyResult{
		{Flag: &flags.Flag{ID: "f1", Key: "attr", Enabled: true, Environment: flags.DeploymentStageDev, RolloutStrategy: flags.RolloutStrategyAttribute}, Err: nil},
	}
	store.GetRulesByFlagIDReturns = []mock.GetRulesResult{
		{Rules: []*flags.Rule{{Type: flags.RuleTypeAttribute, Value: `{"attribute":"userId","op":"in","values":["user-1"]}`}}, Err: nil},
	}
	svc := flags.NewService(store)

	enabled, err := svc.EvaluateFlag(ctx, "attr", evalCtx("user-1"))

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !enabled {
		t.Error("expected true when attribute rule matches userId")
	}
}

func TestService_EvaluateFlag_get_flag_error_returns_error(t *testing.T) {
	ctx := context.Background()
	store := &mock.Store{}
	wantErr := errors.New("GetByKeyAndEnvironment failed")
	store.GetByKeyAndEnvironmentReturns = []mock.GetByKeyResult{
		{Flag: nil, Err: wantErr},
	}
	svc := flags.NewService(store)

	enabled, err := svc.EvaluateFlag(ctx, "key", evalCtx("user-1"))

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
		{Flag: &flags.Flag{ID: "f1", Key: "key", Enabled: true, Environment: flags.DeploymentStageDev, RolloutStrategy: flags.RolloutStrategyPercentage}, Err: nil},
	}
	wantErr := errors.New("GetRulesByFlagID failed")
	store.GetRulesByFlagIDReturns = []mock.GetRulesResult{
		{Rules: nil, Err: wantErr},
	}
	svc := flags.NewService(store)

	enabled, err := svc.EvaluateFlag(ctx, "key", evalCtx("user-1"))

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

	_, _ = svc.EvaluateFlag(ctx, "key", evalCtx("user-1"))

	if len(store.GetByKeyAndEnvironmentCalls) != 1 {
		t.Fatalf("expected 1 GetByKeyAndEnvironment call, got %d", len(store.GetByKeyAndEnvironmentCalls))
	}
	if store.GetByKeyAndEnvironmentCalls[0].Env != flags.DeploymentStageDev {
		t.Errorf("EvaluateFlag should use defaultEnvironment dev, got %q", store.GetByKeyAndEnvironmentCalls[0].Env)
	}
}

func TestService_CreateFlag_with_rules_sets_strategy_and_replaces_rules(t *testing.T) {
	ctx := context.Background()
	store := &mock.Store{}
	store.GetByKeyAndEnvironmentReturns = []mock.GetByKeyResult{
		{Flag: nil, Err: nil},
	}
	created := &flags.Flag{
		ID: "id-1", Key: "f", Environment: flags.DeploymentStageDev, RolloutStrategy: flags.RolloutStrategyPercentage,
		CreatedAt: time.Now(),
	}
	store.CreateReturns = []mock.CreateResult{
		{Flag: created, Err: nil},
	}
	store.ReplaceRulesByFlagIDReturns = []error{nil}
	svc := flags.NewService(store)
	input := model.CreateFlagInput{
		Key: "f", Environment: "dev",
		Rules: []*model.RuleInput{
			{Type: model.RolloutRuleTypePercentage, Value: "50"},
		},
	}

	got, err := svc.CreateFlag(ctx, input)

	if err != nil {
		t.Fatalf("CreateFlag: %v", err)
	}
	if got == nil || got.Key != "f" {
		t.Errorf("got %+v", got)
	}
	if len(store.ReplaceRulesByFlagIDCalls) != 1 || store.ReplaceRulesByFlagIDCalls[0].FlagID != "id-1" {
		t.Errorf("ReplaceRulesByFlagID calls: %+v", store.ReplaceRulesByFlagIDCalls)
	}
}

func TestService_CreateFlag_mixed_rule_types_returns_ErrRulesStrategyMismatch(t *testing.T) {
	ctx := context.Background()
	store := &mock.Store{}
	store.GetByKeyAndEnvironmentReturns = []mock.GetByKeyResult{
		{Flag: nil, Err: nil},
	}
	svc := flags.NewService(store)
	input := model.CreateFlagInput{
		Key: "f", Environment: "dev",
		Rules: []*model.RuleInput{
			{Type: model.RolloutRuleTypePercentage, Value: "30"},
			{Type: model.RolloutRuleTypeAttribute, Value: `{"attribute":"userId","op":"in","values":["x"]}`},
		},
	}

	got, err := svc.CreateFlag(ctx, input)

	if got != nil {
		t.Errorf("expected nil, got %+v", got)
	}
	var e *flags.RulesStrategyMismatchError
	if !errors.As(err, &e) {
		t.Errorf("expected *RulesStrategyMismatchError, got %v", err)
	}
	if len(e.RuleTypes) != 2 {
		t.Errorf("expected RuleTypes len 2, got %v", e.RuleTypes)
	}
	if len(store.CreateCalls) != 0 {
		t.Error("Create should not be called when rules are mixed")
	}
}

func TestService_DeleteFlag_found_returns_true(t *testing.T) {
	ctx := context.Background()
	store := &mock.Store{}
	store.GetByKeyAndEnvironmentReturns = []mock.GetByKeyResult{
		{Flag: &flags.Flag{ID: "f1", Key: "x", Environment: flags.DeploymentStageDev}, Err: nil},
	}
	store.DeleteReturns = []error{nil}
	svc := flags.NewService(store)

	result, err := svc.DeleteFlag(ctx, "x", "dev")

	if err != nil {
		t.Fatalf("DeleteFlag: %v", err)
	}
	if !result {
		t.Error("expected true when flag deleted")
	}
	if len(store.DeleteCalls) != 1 || store.DeleteCalls[0].ID != "f1" {
		t.Errorf("Delete calls: %+v", store.DeleteCalls)
	}
}

func TestService_DeleteFlag_not_found_returns_false_and_ErrNotFound(t *testing.T) {
	ctx := context.Background()
	store := &mock.Store{}
	store.GetByKeyAndEnvironmentReturns = []mock.GetByKeyResult{
		{Flag: nil, Err: nil},
	}
	svc := flags.NewService(store)

	result, err := svc.DeleteFlag(ctx, "missing", "dev")

	if result {
		t.Error("expected false when flag not found")
	}
	var e *flags.NotFoundError
	if !errors.As(err, &e) {
		t.Errorf("expected *NotFoundError, got %v", err)
	}
	if e.Key != "missing" || e.Environment != "dev" {
		t.Errorf("expected Key=missing Environment=dev, got Key=%q Environment=%q", e.Key, e.Environment)
	}
	if len(store.DeleteCalls) != 0 {
		t.Error("Delete should not be called when flag not found")
	}
}

func TestService_EvaluateFlag_attribute_no_match_returns_false(t *testing.T) {
	ctx := context.Background()
	store := &mock.Store{}
	store.GetByKeyAndEnvironmentReturns = []mock.GetByKeyResult{
		{Flag: &flags.Flag{ID: "f1", Key: "attr", Enabled: true, Environment: flags.DeploymentStageDev, RolloutStrategy: flags.RolloutStrategyAttribute}, Err: nil},
	}
	store.GetRulesByFlagIDReturns = []mock.GetRulesResult{
		{Rules: []*flags.Rule{{Type: flags.RuleTypeAttribute, Value: `{"attribute":"userId","op":"in","values":["other-user"]}`}}, Err: nil},
	}
	svc := flags.NewService(store)

	enabled, err := svc.EvaluateFlag(ctx, "attr", evalCtx("user-1"))

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if enabled {
		t.Error("expected false when attribute rule does not match")
	}
}

func TestService_CreateFlag_rolloutStrategy_mismatch_with_rules_returns_ErrRulesStrategyMismatch(t *testing.T) {
	ctx := context.Background()
	store := &mock.Store{}
	store.GetByKeyAndEnvironmentReturns = []mock.GetByKeyResult{
		{Flag: nil, Err: nil},
	}
	svc := flags.NewService(store)
	strategy := model.RolloutStrategyAttribute
	input := model.CreateFlagInput{
		Key:             "f",
		Environment:     "dev",
		RolloutStrategy: &strategy,
		Rules:           []*model.RuleInput{{Type: model.RolloutRuleTypePercentage, Value: "50"}},
	}

	got, err := svc.CreateFlag(ctx, input)

	if got != nil {
		t.Errorf("expected nil, got %+v", got)
	}
	var e *flags.RulesStrategyMismatchError
	if !errors.As(err, &e) {
		t.Errorf("expected *RulesStrategyMismatchError, got %v", err)
	}
	if e.Message == "" {
		t.Error("expected Message to be set")
	}
	if len(store.CreateCalls) != 0 {
		t.Error("Create should not be called when strategy and rules type mismatch")
	}
}

func TestService_CreateFlag_ReplaceRulesByFlagID_error_returns_wrapped_error(t *testing.T) {
	ctx := context.Background()
	store := &mock.Store{}
	store.GetByKeyAndEnvironmentReturns = []mock.GetByKeyResult{
		{Flag: nil, Err: nil},
	}
	created := &flags.Flag{ID: "id-1", Key: "f", Environment: flags.DeploymentStageDev, RolloutStrategy: flags.RolloutStrategyPercentage, CreatedAt: time.Now()}
	store.CreateReturns = []mock.CreateResult{
		{Flag: created, Err: nil},
	}
	wantErr := errors.New("ReplaceRulesByFlagID failed on CreateFlag")
	store.ReplaceRulesByFlagIDReturns = []error{wantErr}
	svc := flags.NewService(store)
	input := model.CreateFlagInput{
		Key:         "f",
		Environment: "dev",
		Rules:       []*model.RuleInput{{Type: model.RolloutRuleTypePercentage, Value: "50"}},
	}

	got, err := svc.CreateFlag(ctx, input)

	if got != nil {
		t.Errorf("expected nil, got %+v", got)
	}
	if !errors.Is(err, wantErr) {
		t.Errorf("expected wrapped %v, got %v", wantErr, err)
	}
}

func TestService_UpdateFlag_rules_empty_ReplaceRulesByFlagID_error_returns_wrapped_error(t *testing.T) {
	ctx := context.Background()
	store := &mock.Store{}
	store.GetByKeyAndEnvironmentReturns = []mock.GetByKeyResult{
		{Flag: &flags.Flag{ID: "f1", Key: "f", Enabled: true, Environment: flags.DeploymentStageDev, RolloutStrategy: flags.RolloutStrategyPercentage}, Err: nil},
	}
	wantErr := errors.New("ReplaceRulesByFlagID failed when clearing rules")
	store.ReplaceRulesByFlagIDReturns = []error{wantErr}
	store.UpdateReturns = []error{nil}
	svc := flags.NewService(store)
	input := model.UpdateFlagInput{Key: "f", Enabled: true, Rules: []*model.RuleInput{}}

	got, err := svc.UpdateFlag(ctx, input)

	if got != nil {
		t.Errorf("expected nil, got %+v", got)
	}
	if !errors.Is(err, wantErr) {
		t.Errorf("expected wrapped %v, got %v", wantErr, err)
	}
}

func TestService_UpdateFlag_rules_strategy_mismatch_returns_ErrRulesStrategyMismatch(t *testing.T) {
	ctx := context.Background()
	store := &mock.Store{}
	store.GetByKeyAndEnvironmentReturns = []mock.GetByKeyResult{
		{Flag: &flags.Flag{ID: "f1", Key: "f", Enabled: true, Environment: flags.DeploymentStageDev, RolloutStrategy: flags.RolloutStrategyPercentage}, Err: nil},
	}
	store.UpdateReturns = []error{nil}
	svc := flags.NewService(store)
	input := model.UpdateFlagInput{
		Key:     "f",
		Enabled: true,
		Rules:   []*model.RuleInput{{Type: model.RolloutRuleTypeAttribute, Value: `{"attribute":"userId","op":"in","values":["x"]}`}},
	}

	got, err := svc.UpdateFlag(ctx, input)

	if got != nil {
		t.Errorf("expected nil, got %+v", got)
	}
	var e *flags.RulesStrategyMismatchError
	if !errors.As(err, &e) {
		t.Errorf("expected *RulesStrategyMismatchError, got %v", err)
	}
	if e.CurrentStrategy != "percentage" {
		t.Errorf("expected CurrentStrategy=percentage, got %q", e.CurrentStrategy)
	}
	if len(store.ReplaceRulesByFlagIDCalls) != 0 {
		t.Error("ReplaceRulesByFlagID should not be called when strategy mismatch")
	}
}

func TestService_UpdateFlag_ReplaceRulesByFlagID_error_returns_wrapped_error(t *testing.T) {
	ctx := context.Background()
	store := &mock.Store{}
	store.GetByKeyAndEnvironmentReturns = []mock.GetByKeyResult{
		{Flag: &flags.Flag{ID: "f1", Key: "f", Enabled: true, Environment: flags.DeploymentStageDev, RolloutStrategy: flags.RolloutStrategyPercentage}, Err: nil},
	}
	wantErr := errors.New("ReplaceRulesByFlagID failed on UpdateFlag")
	store.ReplaceRulesByFlagIDReturns = []error{wantErr}
	store.UpdateReturns = []error{nil}
	svc := flags.NewService(store)
	input := model.UpdateFlagInput{
		Key:     "f",
		Enabled: true,
		Rules:   []*model.RuleInput{{Type: model.RolloutRuleTypePercentage, Value: "25"}},
	}

	got, err := svc.UpdateFlag(ctx, input)

	if got != nil {
		t.Errorf("expected nil, got %+v", got)
	}
	if !errors.Is(err, wantErr) {
		t.Errorf("expected wrapped %v, got %v", wantErr, err)
	}
}

func TestService_DeleteFlag_StoreDelete_error_returns_wrapped_error(t *testing.T) {
	ctx := context.Background()
	store := &mock.Store{}
	store.GetByKeyAndEnvironmentReturns = []mock.GetByKeyResult{
		{Flag: &flags.Flag{ID: "f1", Key: "x", Environment: flags.DeploymentStageDev}, Err: nil},
	}
	wantErr := errors.New("Store.Delete failed")
	store.DeleteReturns = []error{wantErr}
	svc := flags.NewService(store)

	result, err := svc.DeleteFlag(ctx, "x", "dev")

	if result {
		t.Error("expected false when Delete fails")
	}
	if !errors.Is(err, wantErr) {
		t.Errorf("expected wrapped %v, got %v", wantErr, err)
	}
}

type auditTxStarterFlagsMock struct {
	beginErr error
}

func (s *auditTxStarterFlagsMock) Create(ctx context.Context, entry *audit.Entry) error {
	return nil
}

func (s *auditTxStarterFlagsMock) GetByID(ctx context.Context, id string) (*audit.Entry, error) {
	return nil, nil
}

func (s *auditTxStarterFlagsMock) List(ctx context.Context, filter audit.ListFilter, limit, offset int) ([]*audit.Entry, error) {
	return nil, nil
}

func (s *auditTxStarterFlagsMock) BeginTx(ctx context.Context) (*sql.Tx, error) {
	return nil, s.beginErr
}

type auditTxAwareFlagsMock struct {
	auditTxStarterFlagsMock
}

func (s *auditTxAwareFlagsMock) WithTx(tx *sql.Tx) audit.Store {
	return s
}

type txAwareFlagsStoreMock struct {
	inner flags.Store
}

func (s *txAwareFlagsStoreMock) Create(ctx context.Context, flag *flags.Flag) (*flags.Flag, error) {
	return s.inner.Create(ctx, flag)
}

func (s *txAwareFlagsStoreMock) GetByKeyAndEnvironment(ctx context.Context, key string, env flags.DeploymentStage) (*flags.Flag, error) {
	return s.inner.GetByKeyAndEnvironment(ctx, key, env)
}

func (s *txAwareFlagsStoreMock) Update(ctx context.Context, flag *flags.Flag) error {
	return s.inner.Update(ctx, flag)
}

func (s *txAwareFlagsStoreMock) Delete(ctx context.Context, id string) error {
	return s.inner.Delete(ctx, id)
}

func (s *txAwareFlagsStoreMock) GetRulesByFlagID(ctx context.Context, flagID string) ([]*flags.Rule, error) {
	return s.inner.GetRulesByFlagID(ctx, flagID)
}

func (s *txAwareFlagsStoreMock) ReplaceRulesByFlagID(ctx context.Context, flagID string, rules []*flags.Rule) error {
	return s.inner.ReplaceRulesByFlagID(ctx, flagID, rules)
}

func (s *txAwareFlagsStoreMock) WithTx(tx *sql.Tx) flags.Store {
	return s
}

func TestService_CreateFlag_withAudit_missingActor_returns_error(t *testing.T) {
	store := &txAwareFlagsStoreMock{inner: &mock.Store{}}
	svc := flags.NewServiceWithAudit(store, &auditTxAwareFlagsMock{})
	input := model.CreateFlagInput{Key: "a", Environment: "dev"}

	_, err := svc.CreateFlag(context.Background(), input)

	if err == nil || err.Error() != "audit: missing actor id in context" {
		t.Fatalf("expected missing actor error, got %v", err)
	}
}

func TestService_CreateFlag_withAudit_notTxAwareAuditStore_returns_error(t *testing.T) {
	store := &txAwareFlagsStoreMock{inner: &mock.Store{}}
	svc := flags.NewServiceWithAudit(store, &auditTxStarterFlagsMock{})
	input := model.CreateFlagInput{Key: "a", Environment: "dev"}
	ctx := auth.WithActorID(context.Background(), "u1")

	_, err := svc.CreateFlag(ctx, input)

	if err == nil || err.Error() != "audit: audit store is not tx-aware" {
		t.Fatalf("expected tx-aware audit store error, got %v", err)
	}
}

func TestService_CreateFlag_withAudit_beginTx_error_is_returned(t *testing.T) {
	store := &txAwareFlagsStoreMock{inner: &mock.Store{}}
	wantErr := errors.New("begin tx failed")
	svc := flags.NewServiceWithAudit(store, &auditTxAwareFlagsMock{
		auditTxStarterFlagsMock: auditTxStarterFlagsMock{beginErr: wantErr},
	})
	input := model.CreateFlagInput{Key: "a", Environment: "dev"}
	ctx := auth.WithActorID(context.Background(), "u1")

	_, err := svc.CreateFlag(ctx, input)

	if !errors.Is(err, wantErr) {
		t.Fatalf("expected %v, got %v", wantErr, err)
	}
}
