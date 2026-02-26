package flags

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jan-havlin-dev/featureflag-api/graph/model"
)

// mockStore implements Store for service tests without importing testutil (avoids import cycle).
type mockStore struct {
	createCalls                   []struct{ ctx context.Context; flag *Flag }
	createReturns                 []struct{ flag *Flag; err error }
	getByKeyCalls                 []struct{ ctx context.Context; key, env string }
	getByKeyReturns               []struct{ flag *Flag; err error }
	updateCalls                   []struct{ ctx context.Context; flag *Flag }
	updateReturns                 []error
	getRulesByFlagIDCalls         []struct{ ctx context.Context; flagID string }
	getRulesByFlagIDReturns       []struct{ rules []*Rule; err error }
}

func (m *mockStore) Create(ctx context.Context, flag *Flag) (*Flag, error) {
	m.createCalls = append(m.createCalls, struct{ ctx context.Context; flag *Flag }{ctx, flag})
	if len(m.createReturns) == 0 {
		return nil, errors.New("no mock return")
	}
	r := m.createReturns[0]
	m.createReturns = m.createReturns[1:]
	return r.flag, r.err
}

func (m *mockStore) GetByKeyAndEnvironment(ctx context.Context, key, environment string) (*Flag, error) {
	m.getByKeyCalls = append(m.getByKeyCalls, struct{ ctx context.Context; key, env string }{ctx, key, environment})
	if len(m.getByKeyReturns) == 0 {
		return nil, errors.New("no mock return")
	}
	r := m.getByKeyReturns[0]
	m.getByKeyReturns = m.getByKeyReturns[1:]
	return r.flag, r.err
}

func (m *mockStore) Update(ctx context.Context, flag *Flag) error {
	m.updateCalls = append(m.updateCalls, struct{ ctx context.Context; flag *Flag }{ctx, flag})
	if len(m.updateReturns) == 0 {
		return errors.New("no mock return")
	}
	r := m.updateReturns[0]
	m.updateReturns = m.updateReturns[1:]
	return r
}

func (m *mockStore) GetRulesByFlagID(ctx context.Context, flagID string) ([]*Rule, error) {
	m.getRulesByFlagIDCalls = append(m.getRulesByFlagIDCalls, struct{ ctx context.Context; flagID string }{ctx, flagID})
	if len(m.getRulesByFlagIDReturns) == 0 {
		return nil, errors.New("no mock return")
	}
	r := m.getRulesByFlagIDReturns[0]
	m.getRulesByFlagIDReturns = m.getRulesByFlagIDReturns[1:]
	return r.rules, r.err
}

var _ Store = (*mockStore)(nil)

func stringPtr(s string) *string { return &s }

func TestService_CreateFlag_happy_path(t *testing.T) {
	ctx := context.Background()
	store := &mockStore{}
	store.getByKeyReturns = []struct{ flag *Flag; err error }{
		{flag: nil, err: nil},
	}
	created := &Flag{
		ID:          "id-1",
		Key:         "test-flag",
		Description: stringPtr("desc"),
		Enabled:     false,
		Environment: "dev",
		CreatedAt:   time.Now(),
	}
	store.createReturns = []struct{ flag *Flag; err error }{
		{flag: created, err: nil},
	}
	svc := NewService(store)
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
	if len(store.getByKeyCalls) != 1 {
		t.Errorf("GetByKeyAndEnvironment calls: want 1, got %d", len(store.getByKeyCalls))
	}
	if store.getByKeyCalls[0].key != "test-flag" || store.getByKeyCalls[0].env != "dev" {
		t.Errorf("GetByKeyAndEnvironment called with wrong args: %+v", store.getByKeyCalls[0])
	}
	if len(store.createCalls) != 1 {
		t.Errorf("Create calls: want 1, got %d", len(store.createCalls))
	}
	if store.createCalls[0].flag.Key != "test-flag" || store.createCalls[0].flag.Enabled != false {
		t.Errorf("Create called with wrong flag: %+v", store.createCalls[0].flag)
	}
}

func TestService_CreateFlag_already_exists_returns_ErrDuplicateKey(t *testing.T) {
	ctx := context.Background()
	store := &mockStore{}
	existing := &Flag{ID: "existing", Key: "test-flag", Environment: "dev"}
	store.getByKeyReturns = []struct{ flag *Flag; err error }{
		{flag: existing, err: nil},
	}
	svc := NewService(store)
	input := model.CreateFlagInput{Key: "test-flag", Environment: "dev"}

	got, err := svc.CreateFlag(ctx, input)

	if got != nil {
		t.Errorf("expected nil result, got %+v", got)
	}
	if !errors.Is(err, ErrDuplicateKey) {
		t.Errorf("expected ErrDuplicateKey, got %v", err)
	}
	if len(store.createCalls) != 0 {
		t.Errorf("Create should not be called, got %d calls", len(store.createCalls))
	}
}

func TestService_CreateFlag_get_existing_error_returns_wrapped_error(t *testing.T) {
	ctx := context.Background()
	store := &mockStore{}
	wantErr := errors.New("db error")
	store.getByKeyReturns = []struct{ flag *Flag; err error }{
		{flag: nil, err: wantErr},
	}
	svc := NewService(store)
	input := model.CreateFlagInput{Key: "test-flag", Environment: "dev"}

	_, err := svc.CreateFlag(ctx, input)

	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, wantErr) {
		t.Errorf("expected wrapped %v, got %v", wantErr, err)
	}
	if len(store.createCalls) != 0 {
		t.Errorf("Create should not be called, got %d calls", len(store.createCalls))
	}
}

func TestService_CreateFlag_create_error_returns_wrapped_error(t *testing.T) {
	ctx := context.Background()
	store := &mockStore{}
	store.getByKeyReturns = []struct{ flag *Flag; err error }{
		{flag: nil, err: nil},
	}
	wantErr := errors.New("insert failed")
	store.createReturns = []struct{ flag *Flag; err error }{
		{flag: nil, err: wantErr},
	}
	svc := NewService(store)
	input := model.CreateFlagInput{Key: "test-flag", Environment: "dev"}

	_, err := svc.CreateFlag(ctx, input)

	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, wantErr) {
		t.Errorf("expected wrapped %v, got %v", wantErr, err)
	}
}

func TestService_UpdateFlag_happy_path(t *testing.T) {
	ctx := context.Background()
	store := &mockStore{}
	flag := &Flag{ID: "f1", Key: "test-flag", Enabled: false, Environment: "dev"}
	store.getByKeyReturns = []struct{ flag *Flag; err error }{
		{flag: flag, err: nil},
	}
	store.updateReturns = []error{nil}
	svc := NewService(store)
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
	if len(store.getByKeyCalls) != 1 {
		t.Errorf("GetByKeyAndEnvironment calls: want 1, got %d", len(store.getByKeyCalls))
	}
	if store.getByKeyCalls[0].env != "dev" {
		t.Errorf("UpdateFlag should use defaultEnvironment dev, got %q", store.getByKeyCalls[0].env)
	}
	if len(store.updateCalls) != 1 {
		t.Errorf("Update calls: want 1, got %d", len(store.updateCalls))
	}
}

func TestService_UpdateFlag_not_found_returns_ErrNotFound(t *testing.T) {
	ctx := context.Background()
	store := &mockStore{}
	store.getByKeyReturns = []struct{ flag *Flag; err error }{
		{flag: nil, err: nil},
	}
	svc := NewService(store)
	input := model.UpdateFlagInput{Key: "missing", Enabled: true}

	got, err := svc.UpdateFlag(ctx, input)

	if got != nil {
		t.Errorf("expected nil result, got %+v", got)
	}
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
	if len(store.updateCalls) != 0 {
		t.Errorf("Update should not be called, got %d calls", len(store.updateCalls))
	}
}

func TestService_UpdateFlag_get_error_returns_wrapped_error(t *testing.T) {
	ctx := context.Background()
	store := &mockStore{}
	wantErr := errors.New("db error")
	store.getByKeyReturns = []struct{ flag *Flag; err error }{
		{flag: nil, err: wantErr},
	}
	svc := NewService(store)
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
	store := &mockStore{}
	store.getByKeyReturns = []struct{ flag *Flag; err error }{
		{flag: &Flag{ID: "f1", Key: "test-flag", Environment: "dev"}, err: nil},
	}
	wantErr := errors.New("update failed")
	store.updateReturns = []error{wantErr}
	svc := NewService(store)
	input := model.UpdateFlagInput{Key: "test-flag", Enabled: true}

	_, err := svc.UpdateFlag(ctx, input)

	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, wantErr) {
		t.Errorf("expected wrapped %v, got %v", wantErr, err)
	}
}

func TestService_EvaluateFlag_empty_userID_returns_ErrInvalidUserID(t *testing.T) {
	ctx := context.Background()
	store := &mockStore{}
	svc := NewService(store)

	enabled, err := svc.EvaluateFlag(ctx, "any-key", "")

	if enabled {
		t.Error("expected false when userID is empty")
	}
	if !errors.Is(err, ErrInvalidUserID) {
		t.Errorf("expected ErrInvalidUserID, got %v", err)
	}
	if len(store.getByKeyCalls) != 0 {
		t.Error("store should not be called when userID is empty")
	}
}

func TestService_EvaluateFlag_flag_not_found_returns_false_nil(t *testing.T) {
	ctx := context.Background()
	store := &mockStore{}
	store.getByKeyReturns = []struct{ flag *Flag; err error }{
		{flag: nil, err: nil},
	}
	svc := NewService(store)

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
	store := &mockStore{}
	store.getByKeyReturns = []struct{ flag *Flag; err error }{
		{flag: &Flag{ID: "f1", Key: "off-flag", Enabled: false, Environment: "dev"}, err: nil},
	}
	svc := NewService(store)

	enabled, err := svc.EvaluateFlag(ctx, "off-flag", "user-1")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if enabled {
		t.Error("expected false when flag is disabled")
	}
	if len(store.getRulesByFlagIDCalls) != 0 {
		t.Error("GetRulesByFlagID should not be called when flag is disabled")
	}
}

func TestService_EvaluateFlag_enabled_no_rules_returns_true_nil(t *testing.T) {
	ctx := context.Background()
	store := &mockStore{}
	store.getByKeyReturns = []struct{ flag *Flag; err error }{
		{flag: &Flag{ID: "f1", Key: "on-flag", Enabled: true, Environment: "dev"}, err: nil},
	}
	store.getRulesByFlagIDReturns = []struct{ rules []*Rule; err error }{
		{rules: nil, err: nil},
	}
	svc := NewService(store)

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
	store := &mockStore{}
	store.getByKeyReturns = []struct{ flag *Flag; err error }{
		{flag: &Flag{ID: "f1", Key: "pct", Enabled: true, Environment: "dev"}, err: nil},
	}
	store.getRulesByFlagIDReturns = []struct{ rules []*Rule; err error }{
		{rules: []*Rule{{Type: RuleTypePercentage, Value: "0"}}, err: nil},
	}
	svc := NewService(store)

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
	store := &mockStore{}
	store.getByKeyReturns = []struct{ flag *Flag; err error }{
		{flag: &Flag{ID: "f1", Key: "pct", Enabled: true, Environment: "dev"}, err: nil},
	}
	store.getRulesByFlagIDReturns = []struct{ rules []*Rule; err error }{
		{rules: []*Rule{{Type: RuleTypePercentage, Value: "100"}}, err: nil},
	}
	svc := NewService(store)

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
	store := &mockStore{}
	flag := &Flag{ID: "f1", Key: "pct", Enabled: true, Environment: "dev"}
	rules := []*Rule{{Type: RuleTypePercentage, Value: "50"}}
	store.getByKeyReturns = []struct{ flag *Flag; err error }{
		{flag: flag, err: nil},
		{flag: flag, err: nil},
	}
	store.getRulesByFlagIDReturns = []struct{ rules []*Rule; err error }{
		{rules: rules, err: nil},
		{rules: rules, err: nil},
	}
	svc := NewService(store)
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
	store := &mockStore{}
	store.getByKeyReturns = []struct{ flag *Flag; err error }{
		{flag: &Flag{ID: "f1", Key: "pct", Enabled: true, Environment: "dev"}, err: nil},
	}
	store.getRulesByFlagIDReturns = []struct{ rules []*Rule; err error }{
		{rules: []*Rule{{Type: RuleTypePercentage, Value: "x"}}, err: nil},
	}
	svc := NewService(store)

	_, err := svc.EvaluateFlag(ctx, "pct", "user-1")

	if !errors.Is(err, ErrInvalidRule) {
		t.Errorf("expected ErrInvalidRule, got %v", err)
	}
}

func TestService_EvaluateFlag_percentage_out_of_range_returns_ErrInvalidRule(t *testing.T) {
	ctx := context.Background()
	store := &mockStore{}
	store.getByKeyReturns = []struct{ flag *Flag; err error }{
		{flag: &Flag{ID: "f1", Key: "pct", Enabled: true, Environment: "dev"}, err: nil},
	}
	store.getRulesByFlagIDReturns = []struct{ rules []*Rule; err error }{
		{rules: []*Rule{{Type: RuleTypePercentage, Value: "150"}}, err: nil},
	}
	svc := NewService(store)

	_, err := svc.EvaluateFlag(ctx, "pct", "user-1")

	if !errors.Is(err, ErrInvalidRule) {
		t.Errorf("expected ErrInvalidRule, got %v", err)
	}
}

func TestService_EvaluateFlag_attribute_only_fallback_returns_true(t *testing.T) {
	ctx := context.Background()
	store := &mockStore{}
	store.getByKeyReturns = []struct{ flag *Flag; err error }{
		{flag: &Flag{ID: "f1", Key: "attr", Enabled: true, Environment: "dev"}, err: nil},
	}
	store.getRulesByFlagIDReturns = []struct{ rules []*Rule; err error }{
		{rules: []*Rule{{Type: RuleTypeAttribute, Value: `{"email":"@x.com"}`}}, err: nil},
	}
	svc := NewService(store)

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
	store := &mockStore{}
	wantErr := errors.New("db error")
	store.getByKeyReturns = []struct{ flag *Flag; err error }{
		{flag: nil, err: wantErr},
	}
	svc := NewService(store)

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
	store := &mockStore{}
	store.getByKeyReturns = []struct{ flag *Flag; err error }{
		{flag: &Flag{ID: "f1", Key: "key", Enabled: true, Environment: "dev"}, err: nil},
	}
	wantErr := errors.New("rules db error")
	store.getRulesByFlagIDReturns = []struct{ rules []*Rule; err error }{
		{rules: nil, err: wantErr},
	}
	svc := NewService(store)

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
	store := &mockStore{}
	store.getByKeyReturns = []struct{ flag *Flag; err error }{
		{flag: nil, err: nil},
	}
	svc := NewService(store)

	_, _ = svc.EvaluateFlag(ctx, "key", "user-1")

	if len(store.getByKeyCalls) != 1 {
		t.Fatalf("expected 1 GetByKeyAndEnvironment call, got %d", len(store.getByKeyCalls))
	}
	if store.getByKeyCalls[0].env != "dev" {
		t.Errorf("EvaluateFlag should use defaultEnvironment dev, got %q", store.getByKeyCalls[0].env)
	}
}
