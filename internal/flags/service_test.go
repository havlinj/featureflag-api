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
	"github.com/havlinj/featureflag-api/internal/testutil/auditmock"
)

func stringPtr(s string) *string { return &s }

func evalCtx(userID string) model.EvaluationContextInput {
	return model.EvaluationContextInput{UserID: userID}
}

func evalCtxWithEmail(userID, email string) model.EvaluationContextInput {
	return model.EvaluationContextInput{UserID: userID, Email: &email}
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
	if got.RolloutStrategy != model.RolloutStrategyNone {
		t.Errorf("expected RolloutStrategy NONE on create without rules, got %v", got.RolloutStrategy)
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
	var opErr *flags.OperationError
	if !errors.As(err, &opErr) {
		t.Fatalf("expected *flags.OperationError, got %T", err)
	}
	if opErr.Op != "flags.service.ensure_unique_flag.store_get_by_key_and_environment" {
		t.Fatalf("unexpected op %q", opErr.Op)
	}
	if opErr.Key != "test-flag" || opErr.Environment != "dev" {
		t.Fatalf("unexpected context fields: %+v", opErr)
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
	var opErr *flags.OperationError
	if !errors.As(err, &opErr) {
		t.Fatalf("expected *flags.OperationError, got %T", err)
	}
	if opErr.Op != "flags.service.create_flag.store_create" {
		t.Fatalf("unexpected op %q", opErr.Op)
	}
	if opErr.Key != "test-flag" || opErr.Environment != "dev" {
		t.Fatalf("unexpected context fields: %+v", opErr)
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

func TestService_UpdateFlag_uses_input_environment_when_provided(t *testing.T) {
	ctx := context.Background()
	store := &mock.Store{}
	flag := &flags.Flag{ID: "f1", Key: "test-flag", Enabled: false, Environment: flags.DeploymentStage("staging")}
	store.GetByKeyAndEnvironmentReturns = []mock.GetByKeyResult{
		{Flag: flag, Err: nil},
	}
	store.UpdateReturns = []error{nil}
	svc := flags.NewService(store)
	env := "staging"
	input := model.UpdateFlagInput{Key: "test-flag", Environment: &env, Enabled: true}

	got, err := svc.UpdateFlag(ctx, input)

	if err != nil {
		t.Fatalf("UpdateFlag: %v", err)
	}
	if got == nil || !got.Enabled {
		t.Errorf("expected enabled flag, got %+v", got)
	}
	if len(store.GetByKeyAndEnvironmentCalls) != 1 {
		t.Fatalf("GetByKeyAndEnvironment calls: want 1, got %d", len(store.GetByKeyAndEnvironmentCalls))
	}
	if store.GetByKeyAndEnvironmentCalls[0].Env != flags.DeploymentStage("staging") {
		t.Errorf("expected environment staging, got %q", store.GetByKeyAndEnvironmentCalls[0].Env)
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
	var opErr *flags.OperationError
	if !errors.As(err, &opErr) {
		t.Fatalf("expected *flags.OperationError, got %T", err)
	}
	if opErr.Op != "flags.service.get_flag_or_err.store_get_by_key_and_environment" {
		t.Fatalf("unexpected op %q", opErr.Op)
	}
	if opErr.Key != "test-flag" || opErr.Environment != "dev" {
		t.Fatalf("unexpected context fields: %+v", opErr)
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
	var opErr *flags.OperationError
	if !errors.As(err, &opErr) {
		t.Fatalf("expected *flags.OperationError, got %T", err)
	}
	if opErr.Op != "flags.service.update_flag.store_update" {
		t.Fatalf("unexpected op %q", opErr.Op)
	}
	if opErr.Key != "test-flag" || opErr.Environment != "dev" || opErr.FlagID != "f1" {
		t.Fatalf("unexpected context fields: %+v", opErr)
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
	var opErr *flags.OperationError
	if !errors.As(err, &opErr) {
		t.Fatalf("expected *flags.OperationError, got %T", err)
	}
	if opErr.Op != "flags.service.evaluate_flag.store_get_by_key_and_environment" {
		t.Fatalf("unexpected op %q", opErr.Op)
	}
	if opErr.Key != "key" || opErr.Environment != "dev" {
		t.Fatalf("unexpected context fields: %+v", opErr)
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

func TestService_EvaluateFlagInEnvironment_uses_provided_environment(t *testing.T) {
	ctx := context.Background()
	store := &mock.Store{}
	store.GetByKeyAndEnvironmentReturns = []mock.GetByKeyResult{
		{Flag: nil, Err: nil},
	}
	svc := flags.NewService(store)

	_, _ = svc.EvaluateFlagInEnvironment(ctx, "key", flags.DeploymentStage("staging"), evalCtx("user-1"))

	if len(store.GetByKeyAndEnvironmentCalls) != 1 {
		t.Fatalf("expected 1 GetByKeyAndEnvironment call, got %d", len(store.GetByKeyAndEnvironmentCalls))
	}
	if store.GetByKeyAndEnvironmentCalls[0].Env != flags.DeploymentStage("staging") {
		t.Errorf("expected staging env, got %q", store.GetByKeyAndEnvironmentCalls[0].Env)
	}
}

func TestService_EvaluateFlagInEnvironment_evaluates_rules_in_that_environment(t *testing.T) {
	ctx := context.Background()
	store := &mock.Store{}
	store.GetByKeyAndEnvironmentReturns = []mock.GetByKeyResult{
		{Flag: &flags.Flag{ID: "f1", Key: "rollout", Enabled: true, Environment: flags.DeploymentStage("staging"), RolloutStrategy: flags.RolloutStrategyPercentage}, Err: nil},
	}
	store.GetRulesByFlagIDReturns = []mock.GetRulesResult{
		{Rules: []*flags.Rule{{Type: flags.RuleTypePercentage, Value: "100"}}, Err: nil},
	}
	svc := flags.NewService(store)

	enabled, err := svc.EvaluateFlagInEnvironment(ctx, "rollout", flags.DeploymentStage("staging"), evalCtx("any-user"))

	if err != nil {
		t.Fatalf("EvaluateFlagInEnvironment: %v", err)
	}
	if !enabled {
		t.Fatal("expected true for 100% in staging")
	}
	if len(store.GetByKeyAndEnvironmentCalls) != 1 || store.GetByKeyAndEnvironmentCalls[0].Env != flags.DeploymentStage("staging") {
		t.Fatalf("unexpected store calls: %+v", store.GetByKeyAndEnvironmentCalls)
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
	if got.RolloutStrategy != model.RolloutStrategyPercentage {
		t.Errorf("expected RolloutStrategy PERCENTAGE, got %v", got.RolloutStrategy)
	}
}

func TestService_CreateFlag_attribute_rules_maps_rollout_strategy_attribute(t *testing.T) {
	ctx := context.Background()
	store := &mock.Store{}
	store.GetByKeyAndEnvironmentReturns = []mock.GetByKeyResult{
		{Flag: nil, Err: nil},
	}
	created := &flags.Flag{
		ID: "id-attr", Key: "attr-key", Environment: flags.DeploymentStageDev, RolloutStrategy: flags.RolloutStrategyAttribute,
		CreatedAt: time.Now(),
	}
	store.CreateReturns = []mock.CreateResult{{Flag: created, Err: nil}}
	store.ReplaceRulesByFlagIDReturns = []error{nil}
	svc := flags.NewService(store)
	input := model.CreateFlagInput{
		Key: "attr-key", Environment: "dev",
		Rules: []*model.RuleInput{
			{Type: model.RolloutRuleTypeAttribute, Value: `{"attribute":"userId","op":"in","values":["u1"]}`},
		},
	}

	got, err := svc.CreateFlag(ctx, input)

	if err != nil {
		t.Fatalf("CreateFlag: %v", err)
	}
	if got.RolloutStrategy != model.RolloutStrategyAttribute {
		t.Errorf("expected RolloutStrategy ATTRIBUTE, got %v", got.RolloutStrategy)
	}
}

func TestService_CreateFlag_explicit_rollout_without_rules_uses_input_strategy(t *testing.T) {
	ctx := context.Background()
	store := &mock.Store{}
	store.GetByKeyAndEnvironmentReturns = []mock.GetByKeyResult{{Flag: nil, Err: nil}}
	rs := model.RolloutStrategyPercentage
	created := &flags.Flag{
		ID: "id-pct", Key: "pct-only", Environment: flags.DeploymentStageStaging, RolloutStrategy: flags.RolloutStrategyPercentage,
		CreatedAt: time.Now(),
	}
	store.CreateReturns = []mock.CreateResult{{Flag: created, Err: nil}}
	svc := flags.NewService(store)
	input := model.CreateFlagInput{
		Key: "pct-only", Environment: "staging", RolloutStrategy: &rs,
	}

	got, err := svc.CreateFlag(ctx, input)

	if err != nil {
		t.Fatalf("CreateFlag: %v", err)
	}
	if got.RolloutStrategy != model.RolloutStrategyPercentage || got.Environment != "staging" {
		t.Errorf("got %+v", got)
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

func TestService_EvaluateFlag_attribute_email_suffix_match_returns_true(t *testing.T) {
	ctx := context.Background()
	store := &mock.Store{}
	store.GetByKeyAndEnvironmentReturns = []mock.GetByKeyResult{
		{Flag: &flags.Flag{ID: "f1", Key: "corp", Enabled: true, Environment: flags.DeploymentStageDev, RolloutStrategy: flags.RolloutStrategyAttribute}, Err: nil},
	}
	store.GetRulesByFlagIDReturns = []mock.GetRulesResult{
		{Rules: []*flags.Rule{{Type: flags.RuleTypeAttribute, Value: `{"attribute":"email","op":"suffix","value":"@company.test"}`}}, Err: nil},
	}
	svc := flags.NewService(store)

	enabled, err := svc.EvaluateFlag(ctx, "corp", evalCtxWithEmail("user-1", "alice@company.test"))

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !enabled {
		t.Error("expected true when email suffix rule matches")
	}
}

func TestService_EvaluateFlag_attribute_eq_userId_returns_true(t *testing.T) {
	ctx := context.Background()
	store := &mock.Store{}
	store.GetByKeyAndEnvironmentReturns = []mock.GetByKeyResult{
		{Flag: &flags.Flag{ID: "f1", Key: "eq-flag", Enabled: true, Environment: flags.DeploymentStageDev, RolloutStrategy: flags.RolloutStrategyAttribute}, Err: nil},
	}
	store.GetRulesByFlagIDReturns = []mock.GetRulesResult{
		{Rules: []*flags.Rule{{Type: flags.RuleTypeAttribute, Value: `{"attribute":"userId","op":"eq","value":"exact-user"}`}}, Err: nil},
	}
	svc := flags.NewService(store)

	enabled, err := svc.EvaluateFlag(ctx, "eq-flag", evalCtx("exact-user"))

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !enabled {
		t.Error("expected true when userId equals rule value")
	}
}

func TestService_EvaluateFlag_attribute_second_rule_matches_when_first_does_not(t *testing.T) {
	ctx := context.Background()
	store := &mock.Store{}
	store.GetByKeyAndEnvironmentReturns = []mock.GetByKeyResult{
		{Flag: &flags.Flag{ID: "f1", Key: "multi-attr", Enabled: true, Environment: flags.DeploymentStageDev, RolloutStrategy: flags.RolloutStrategyAttribute}, Err: nil},
	}
	store.GetRulesByFlagIDReturns = []mock.GetRulesResult{
		{Rules: []*flags.Rule{
			{Type: flags.RuleTypeAttribute, Value: `{"attribute":"userId","op":"in","values":["other"]}`},
			{Type: flags.RuleTypeAttribute, Value: `{"attribute":"userId","op":"in","values":["winner"]}`},
		}, Err: nil},
	}
	svc := flags.NewService(store)

	enabled, err := svc.EvaluateFlag(ctx, "multi-attr", evalCtx("winner"))

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !enabled {
		t.Error("expected true when a later attribute rule matches")
	}
}

func TestService_EvaluateFlag_attribute_rule_invalid_json_returns_InvalidRuleError(t *testing.T) {
	ctx := context.Background()
	store := &mock.Store{}
	store.GetByKeyAndEnvironmentReturns = []mock.GetByKeyResult{
		{Flag: &flags.Flag{ID: "f1", Key: "bad-json", Enabled: true, Environment: flags.DeploymentStageDev, RolloutStrategy: flags.RolloutStrategyAttribute}, Err: nil},
	}
	store.GetRulesByFlagIDReturns = []mock.GetRulesResult{
		{Rules: []*flags.Rule{{Type: flags.RuleTypeAttribute, Value: `{not-valid-json`}}, Err: nil},
	}
	svc := flags.NewService(store)

	_, err := svc.EvaluateFlag(ctx, "bad-json", evalCtx("user-1"))

	var e *flags.InvalidRuleError
	if !errors.As(err, &e) {
		t.Fatalf("expected *InvalidRuleError, got %v", err)
	}
	if e.Reason != "JSON parse failed" {
		t.Errorf("expected Reason=JSON parse failed, got %q", e.Reason)
	}
}

func TestService_EvaluateFlag_rollout_none_ignores_rules_returns_true(t *testing.T) {
	ctx := context.Background()
	store := &mock.Store{}
	store.GetByKeyAndEnvironmentReturns = []mock.GetByKeyResult{
		{Flag: &flags.Flag{ID: "f1", Key: "none-strat", Enabled: true, Environment: flags.DeploymentStageDev, RolloutStrategy: flags.RolloutStrategyNone}, Err: nil},
	}
	store.GetRulesByFlagIDReturns = []mock.GetRulesResult{
		{Rules: []*flags.Rule{{Type: flags.RuleTypePercentage, Value: "0"}}, Err: nil},
	}
	svc := flags.NewService(store)

	enabled, err := svc.EvaluateFlag(ctx, "none-strat", evalCtx("user-1"))

	if err != nil {
		t.Fatalf("EvaluateFlag: %v", err)
	}
	if !enabled {
		t.Fatal("with rollout_strategy none, evaluation is always true when flag is enabled")
	}
}

func TestService_EvaluateFlag_percentage_skips_non_percentage_rules(t *testing.T) {
	ctx := context.Background()
	store := &mock.Store{}
	store.GetByKeyAndEnvironmentReturns = []mock.GetByKeyResult{
		{Flag: &flags.Flag{ID: "f1", Key: "mixed-pct", Enabled: true, Environment: flags.DeploymentStageDev, RolloutStrategy: flags.RolloutStrategyPercentage}, Err: nil},
	}
	store.GetRulesByFlagIDReturns = []mock.GetRulesResult{
		{Rules: []*flags.Rule{
			{Type: flags.RuleTypeAttribute, Value: `{"attribute":"userId","op":"in","values":["other"]}`},
			{Type: flags.RuleTypePercentage, Value: "100"},
		}, Err: nil},
	}
	svc := flags.NewService(store)

	enabled, err := svc.EvaluateFlag(ctx, "mixed-pct", evalCtx("user-1"))

	if err != nil {
		t.Fatalf("EvaluateFlag: %v", err)
	}
	if !enabled {
		t.Fatal("expected first applicable percentage rule (100%) to win")
	}
}

func TestService_EvaluateFlag_attribute_skips_percentage_rules(t *testing.T) {
	ctx := context.Background()
	store := &mock.Store{}
	store.GetByKeyAndEnvironmentReturns = []mock.GetByKeyResult{
		{Flag: &flags.Flag{ID: "f1", Key: "mixed-attr", Enabled: true, Environment: flags.DeploymentStageDev, RolloutStrategy: flags.RolloutStrategyAttribute}, Err: nil},
	}
	store.GetRulesByFlagIDReturns = []mock.GetRulesResult{
		{Rules: []*flags.Rule{
			{Type: flags.RuleTypePercentage, Value: "0"},
			{Type: flags.RuleTypeAttribute, Value: `{"attribute":"userId","op":"in","values":["user-1"]}`},
		}, Err: nil},
	}
	svc := flags.NewService(store)

	enabled, err := svc.EvaluateFlag(ctx, "mixed-attr", evalCtx("user-1"))

	if err != nil {
		t.Fatalf("EvaluateFlag: %v", err)
	}
	if !enabled {
		t.Fatal("expected attribute rule to apply after skipping percentage rows")
	}
}

func TestService_EvaluateFlag_percentage_negative_returns_InvalidRuleError(t *testing.T) {
	ctx := context.Background()
	store := &mock.Store{}
	store.GetByKeyAndEnvironmentReturns = []mock.GetByKeyResult{
		{Flag: &flags.Flag{ID: "f1", Key: "neg", Enabled: true, Environment: flags.DeploymentStageDev, RolloutStrategy: flags.RolloutStrategyPercentage}, Err: nil},
	}
	store.GetRulesByFlagIDReturns = []mock.GetRulesResult{
		{Rules: []*flags.Rule{{Type: flags.RuleTypePercentage, Value: "-1"}}, Err: nil},
	}
	svc := flags.NewService(store)

	_, err := svc.EvaluateFlag(ctx, "neg", evalCtx("user-1"))

	var e *flags.InvalidRuleError
	if !errors.As(err, &e) {
		t.Fatalf("expected *InvalidRuleError, got %v", err)
	}
	if e.Reason != "must be 0-100" {
		t.Errorf("expected Reason=must be 0-100, got %q", e.Reason)
	}
}

func TestService_EvaluateFlag_percentage_not_a_number_returns_InvalidRuleError(t *testing.T) {
	ctx := context.Background()
	store := &mock.Store{}
	store.GetByKeyAndEnvironmentReturns = []mock.GetByKeyResult{
		{Flag: &flags.Flag{ID: "f1", Key: "nan", Enabled: true, Environment: flags.DeploymentStageDev, RolloutStrategy: flags.RolloutStrategyPercentage}, Err: nil},
	}
	store.GetRulesByFlagIDReturns = []mock.GetRulesResult{
		{Rules: []*flags.Rule{{Type: flags.RuleTypePercentage, Value: "forty-two"}}, Err: nil},
	}
	svc := flags.NewService(store)

	_, err := svc.EvaluateFlag(ctx, "nan", evalCtx("user-1"))

	var e *flags.InvalidRuleError
	if !errors.As(err, &e) {
		t.Fatalf("expected *InvalidRuleError, got %v", err)
	}
	if e.Reason != "not a number" || e.Value != "forty-two" {
		t.Errorf("expected Value=forty-two Reason=not a number, got Value=%q Reason=%q", e.Value, e.Reason)
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
	var opErr *flags.OperationError
	if !errors.As(err, &opErr) {
		t.Fatalf("expected *flags.OperationError, got %T", err)
	}
	if opErr.Op != "flags.service.create_flag.replace_rules_by_flag_id" {
		t.Fatalf("unexpected op %q", opErr.Op)
	}
	if opErr.FlagID != "id-1" {
		t.Fatalf("unexpected context fields: %+v", opErr)
	}
}

func TestService_EvaluateFlag_get_rules_error_includes_context(t *testing.T) {
	ctx := context.Background()
	store := &mock.Store{}
	store.GetByKeyAndEnvironmentReturns = []mock.GetByKeyResult{
		{Flag: &flags.Flag{ID: "f1", Key: "ctx-flag", Enabled: true, Environment: flags.DeploymentStageDev}, Err: nil},
	}
	wantErr := errors.New("GetRulesByFlagID failed")
	store.GetRulesByFlagIDReturns = []mock.GetRulesResult{
		{Rules: nil, Err: wantErr},
	}
	svc := flags.NewService(store)

	_, err := svc.EvaluateFlag(ctx, "ctx-flag", evalCtx("user-1"))

	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected wrapped %v, got %v", wantErr, err)
	}
	var opErr *flags.OperationError
	if !errors.As(err, &opErr) {
		t.Fatalf("expected *flags.OperationError, got %T", err)
	}
	if opErr.Op != "flags.service.evaluate_flag.store_get_rules_by_flag_id" {
		t.Fatalf("unexpected op %q", opErr.Op)
	}
	if opErr.FlagID != "f1" || opErr.Key != "ctx-flag" || opErr.Environment != "dev" {
		t.Fatalf("unexpected context fields: %+v", opErr)
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
	var opErr *flags.OperationError
	if !errors.As(err, &opErr) {
		t.Fatalf("expected *flags.OperationError, got %T", err)
	}
	if opErr.Op != "flags.service.update_flag.replace_rules_by_flag_id.clear" {
		t.Fatalf("unexpected op %q", opErr.Op)
	}
	if opErr.FlagID != "f1" || opErr.Key != "f" || opErr.Environment != "dev" {
		t.Fatalf("unexpected context fields: %+v", opErr)
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

func TestService_UpdateFlag_existing_attribute_rollout_percentage_rules_returns_strategy_mismatch(t *testing.T) {
	ctx := context.Background()
	store := &mock.Store{}
	store.GetByKeyAndEnvironmentReturns = []mock.GetByKeyResult{
		{Flag: &flags.Flag{ID: "f1", Key: "f", Enabled: true, Environment: flags.DeploymentStageDev, RolloutStrategy: flags.RolloutStrategyAttribute}, Err: nil},
	}
	svc := flags.NewService(store)
	input := model.UpdateFlagInput{
		Key:     "f",
		Enabled: true,
		Rules:   []*model.RuleInput{{Type: model.RolloutRuleTypePercentage, Value: "50"}},
	}

	got, err := svc.UpdateFlag(ctx, input)

	if got != nil {
		t.Errorf("expected nil, got %+v", got)
	}
	var e *flags.RulesStrategyMismatchError
	if !errors.As(err, &e) {
		t.Errorf("expected *RulesStrategyMismatchError, got %v", err)
	}
	if e.CurrentStrategy != "attribute" {
		t.Errorf("expected CurrentStrategy=attribute, got %q", e.CurrentStrategy)
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
	var opErr *flags.OperationError
	if !errors.As(err, &opErr) {
		t.Fatalf("expected *flags.OperationError, got %T", err)
	}
	if opErr.Op != "flags.service.update_flag.replace_rules_by_flag_id" {
		t.Fatalf("unexpected op %q", opErr.Op)
	}
	if opErr.FlagID != "f1" || opErr.Key != "f" || opErr.Environment != "dev" {
		t.Fatalf("unexpected context fields: %+v", opErr)
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
	var opErr *flags.OperationError
	if !errors.As(err, &opErr) {
		t.Fatalf("expected *flags.OperationError, got %T", err)
	}
	if opErr.Op != "flags.service.delete_flag.store_delete" {
		t.Fatalf("unexpected op %q", opErr.Op)
	}
	if opErr.FlagID != "f1" || opErr.Key != "x" || opErr.Environment != "dev" {
		t.Fatalf("unexpected context fields: %+v", opErr)
	}
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
	svc := flags.NewServiceWithAudit(store, &auditmock.TxAware{})
	input := model.CreateFlagInput{Key: "a", Environment: "dev"}

	_, err := svc.CreateFlag(context.Background(), input)

	var e *audit.MissingActorIDError
	if !errors.As(err, &e) {
		t.Fatalf("expected *audit.MissingActorIDError, got %T (%v)", err, err)
	}
}

func TestService_CreateFlag_withAudit_notTxAwareAuditStore_returns_error(t *testing.T) {
	store := &txAwareFlagsStoreMock{inner: &mock.Store{}}
	svc := flags.NewServiceWithAudit(store, &auditmock.TxStarter{})
	input := model.CreateFlagInput{Key: "a", Environment: "dev"}
	ctx := auth.WithActorID(context.Background(), "u1")

	_, err := svc.CreateFlag(ctx, input)

	var e *audit.TxAwareRequiredError
	if !errors.As(err, &e) {
		t.Fatalf("expected *audit.TxAwareRequiredError, got %T (%v)", err, err)
	}
}

func TestService_CreateFlag_withAudit_beginTx_error_is_returned(t *testing.T) {
	store := &txAwareFlagsStoreMock{inner: &mock.Store{}}
	wantErr := errors.New("begin tx failed")
	svc := flags.NewServiceWithAudit(store, &auditmock.TxAware{
		TxStarter: auditmock.TxStarter{BeginErr: wantErr},
	})
	input := model.CreateFlagInput{Key: "a", Environment: "dev"}
	ctx := auth.WithActorID(context.Background(), "u1")

	_, err := svc.CreateFlag(ctx, input)

	if !errors.Is(err, wantErr) {
		t.Fatalf("expected %v, got %v", wantErr, err)
	}
}

// UpdateFlag with Rules == nil must not clear rollout or touch rule persistence (distinct from Rules == []).
func TestService_UpdateFlag_nil_rules_preserves_rollout_and_skips_rule_replace(t *testing.T) {
	ctx := context.Background()
	store := &mock.Store{}
	flag := &flags.Flag{
		ID:              "f1",
		Key:             "rollout-flag",
		Enabled:         false,
		Environment:     flags.DeploymentStageDev,
		RolloutStrategy: flags.RolloutStrategyPercentage,
	}
	store.GetByKeyAndEnvironmentReturns = []mock.GetByKeyResult{{Flag: flag, Err: nil}}
	store.UpdateReturns = []error{nil}
	svc := flags.NewService(store)
	input := model.UpdateFlagInput{Key: "rollout-flag", Enabled: true, Rules: nil}

	got, err := svc.UpdateFlag(ctx, input)

	if err != nil {
		t.Fatalf("UpdateFlag: %v", err)
	}
	if got == nil {
		t.Fatal("expected non-nil updated flag")
	}
	if flag.RolloutStrategy != flags.RolloutStrategyPercentage {
		t.Fatalf("expected rollout strategy unchanged, got %q", flag.RolloutStrategy)
	}
	if len(store.ReplaceRulesByFlagIDCalls) != 0 {
		t.Fatalf("ReplaceRulesByFlagID must not run when rules are omitted (nil), got %d calls", len(store.ReplaceRulesByFlagIDCalls))
	}
}

// UpdateFlag with Rules == [] must clear rollout strategy and persist empty rule set.
func TestService_UpdateFlag_empty_rules_clears_rollout_and_replaces_rules(t *testing.T) {
	ctx := context.Background()
	store := &mock.Store{}
	flag := &flags.Flag{
		ID:              "f1",
		Key:             "rollout-flag",
		Enabled:         true,
		Environment:     flags.DeploymentStageDev,
		RolloutStrategy: flags.RolloutStrategyPercentage,
	}
	store.GetByKeyAndEnvironmentReturns = []mock.GetByKeyResult{{Flag: flag, Err: nil}}
	store.ReplaceRulesByFlagIDReturns = []error{nil}
	store.UpdateReturns = []error{nil}
	svc := flags.NewService(store)
	input := model.UpdateFlagInput{Key: "rollout-flag", Enabled: true, Rules: []*model.RuleInput{}}

	got, err := svc.UpdateFlag(ctx, input)

	if err != nil {
		t.Fatalf("UpdateFlag: %v", err)
	}
	if got == nil {
		t.Fatal("expected non-nil updated flag")
	}
	if flag.RolloutStrategy != flags.RolloutStrategyNone {
		t.Fatalf("expected rollout strategy to be cleared to none, got %q", flag.RolloutStrategy)
	}
	if len(store.ReplaceRulesByFlagIDCalls) != 1 {
		t.Fatalf("expected 1 ReplaceRulesByFlagID call, got %d", len(store.ReplaceRulesByFlagIDCalls))
	}
	if store.ReplaceRulesByFlagIDCalls[0].FlagID != "f1" {
		t.Fatalf("unexpected ReplaceRulesByFlagID args: %+v", store.ReplaceRulesByFlagIDCalls[0])
	}
	if store.ReplaceRulesByFlagIDCalls[0].Rules != nil {
		t.Fatal("expected ReplaceRulesByFlagID to receive nil rules when clearing")
	}
}
