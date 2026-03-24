package flags

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/havlinj/featureflag-api/graph/model"
	"github.com/havlinj/featureflag-api/internal/audit"
	"github.com/havlinj/featureflag-api/internal/auth"
)

func helperStrPtr(v string) *string {
	return &v
}

func TestFlagToModel_TableDriven(t *testing.T) {
	cases := []struct {
		name string
		in   *Flag
	}{
		{
			name: "nil input returns nil",
			in:   nil,
		},
		{
			name: "none strategy maps to NONE",
			in: &Flag{
				ID:              "f-none",
				Key:             "none-flag",
				Description:     helperStrPtr("desc-none"),
				Enabled:         true,
				Environment:     DeploymentStageDev,
				RolloutStrategy: RolloutStrategyNone,
			},
		},
		{
			name: "percentage strategy maps to PERCENTAGE",
			in: &Flag{
				ID:              "f-pct",
				Key:             "pct-flag",
				Description:     helperStrPtr("desc-pct"),
				Enabled:         true,
				Environment:     DeploymentStageStaging,
				RolloutStrategy: RolloutStrategyPercentage,
			},
		},
		{
			name: "attribute strategy maps to ATTRIBUTE",
			in: &Flag{
				ID:              "f-attr",
				Key:             "attr-flag",
				Description:     helperStrPtr("desc-attr"),
				Enabled:         false,
				Environment:     DeploymentStageProd,
				RolloutStrategy: RolloutStrategyAttribute,
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := flagToModel(tc.in)

			if tc.in == nil {
				if got != nil {
					t.Fatalf("expected nil output for nil input, got %+v", got)
				}
				return
			}

			if got == nil {
				t.Fatal("expected non-nil model")
			}
			if got.ID != tc.in.ID || got.Key != tc.in.Key || got.Enabled != tc.in.Enabled || got.Environment != string(tc.in.Environment) {
				t.Fatalf("unexpected mapped fields: got %+v input %+v", got, tc.in)
			}
			if got.Description != tc.in.Description {
				t.Fatalf("unexpected description mapping: got=%v want=%v", got.Description, tc.in.Description)
			}
			if got.RolloutStrategy != rolloutStrategyToModel(tc.in.RolloutStrategy) {
				t.Fatalf("unexpected rollout strategy mapping: got=%q want=%q", got.RolloutStrategy, rolloutStrategyToModel(tc.in.RolloutStrategy))
			}
		})
	}
}

func TestRuleInputToRuleType_TableDriven(t *testing.T) {
	cases := []struct {
		name string
		in   *model.RuleInput
		want RuleType
	}{
		{
			name: "nil input returns empty type",
			in:   nil,
			want: "",
		},
		{
			name: "percentage input maps to percentage type",
			in: &model.RuleInput{
				Type:  model.RolloutRuleTypePercentage,
				Value: "50",
			},
			want: RuleTypePercentage,
		},
		{
			name: "attribute input maps to attribute type",
			in: &model.RuleInput{
				Type:  model.RolloutRuleTypeAttribute,
				Value: `{"attribute":"userId","op":"in","values":["u1"]}`,
			},
			want: RuleTypeAttribute,
		},
		{
			name: "unknown type falls back to attribute type",
			in: &model.RuleInput{
				Type:  model.RolloutRuleType("UNSPECIFIED"),
				Value: "x",
			},
			want: RuleTypeAttribute,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ruleInputToRuleType(tc.in)
			if got != tc.want {
				t.Fatalf("want %q, got %q", tc.want, got)
			}
		})
	}
}

func TestRuleInputsToRules_SkipsNilAndMapsTypes(t *testing.T) {
	in := []*model.RuleInput{
		nil,
		{Type: model.RolloutRuleTypePercentage, Value: "20"},
		{Type: model.RolloutRuleTypeAttribute, Value: `{"attribute":"email","op":"suffix","value":"@company.test"}`},
	}

	out := ruleInputsToRules("flag-1", in)

	if len(out) != 2 {
		t.Fatalf("expected 2 mapped rules (nil skipped), got %d", len(out))
	}
	if out[0].FlagID != "flag-1" || out[0].Type != RuleTypePercentage || out[0].Value != "20" {
		t.Fatalf("unexpected first mapped rule: %+v", out[0])
	}
	if out[1].FlagID != "flag-1" || out[1].Type != RuleTypeAttribute {
		t.Fatalf("unexpected second mapped rule: %+v", out[1])
	}
}

func TestValidateRulesSameType_TableDriven(t *testing.T) {
	cases := []struct {
		name    string
		in      []*model.RuleInput
		want    RuleType
		wantErr bool
	}{
		{
			name: "empty list returns empty type and no error",
			in:   nil,
			want: "",
		},
		{
			name: "single percentage rule",
			in: []*model.RuleInput{
				{Type: model.RolloutRuleTypePercentage, Value: "25"},
			},
			want: RuleTypePercentage,
		},
		{
			name: "single attribute rule",
			in: []*model.RuleInput{
				{Type: model.RolloutRuleTypeAttribute, Value: `{"attribute":"userId","op":"eq","value":"u1"}`},
			},
			want: RuleTypeAttribute,
		},
		{
			name: "mixed rule types returns mismatch error",
			in: []*model.RuleInput{
				{Type: model.RolloutRuleTypePercentage, Value: "25"},
				{Type: model.RolloutRuleTypeAttribute, Value: `{"attribute":"userId","op":"eq","value":"u1"}`},
			},
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := validateRulesSameType(tc.in)

			if tc.wantErr {
				if err == nil {
					t.Fatal("expected mismatch error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Fatalf("want %q, got %q", tc.want, got)
			}
		})
	}
}

func TestEvaluatePercentageRule_TableDriven(t *testing.T) {
	cases := []struct {
		name      string
		userID    string
		value     string
		wantError bool
	}{
		{
			name:   "0 percent always false",
			userID: "user-a",
			value:  "0",
		},
		{
			name:   "100 percent always true",
			userID: "user-a",
			value:  "100",
		},
		{
			name:      "invalid number returns error",
			userID:    "user-a",
			value:     "abc",
			wantError: true,
		},
		{
			name:      "negative percentage returns error",
			userID:    "user-a",
			value:     "-1",
			wantError: true,
		},
		{
			name:      "percentage over 100 returns error",
			userID:    "user-a",
			value:     "101",
			wantError: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := evaluatePercentageRule(tc.userID, tc.value)

			if tc.wantError {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tc.value == "0" && got {
				t.Fatal("expected false for 0%")
			}
			if tc.value == "100" && !got {
				t.Fatal("expected true for 100%")
			}
		})
	}
}

type minimalFlagsTxAwareStore struct {
	deleteErr error
	gotTx     *sql.Tx
}

func (s *minimalFlagsTxAwareStore) Create(ctx context.Context, flag *Flag) (*Flag, error) {
	return nil, nil
}
func (s *minimalFlagsTxAwareStore) GetByKeyAndEnvironment(ctx context.Context, key string, env DeploymentStage) (*Flag, error) {
	return &Flag{ID: "flag-1", Key: key, Environment: env}, nil
}
func (s *minimalFlagsTxAwareStore) Update(ctx context.Context, flag *Flag) error { return nil }
func (s *minimalFlagsTxAwareStore) Delete(ctx context.Context, id string) error  { return s.deleteErr }
func (s *minimalFlagsTxAwareStore) GetRulesByFlagID(ctx context.Context, flagID string) ([]*Rule, error) {
	return nil, nil
}
func (s *minimalFlagsTxAwareStore) ReplaceRulesByFlagID(ctx context.Context, flagID string, rules []*Rule) error {
	return nil
}
func (s *minimalFlagsTxAwareStore) WithTx(tx *sql.Tx) Store {
	s.gotTx = tx
	return s
}

type minimalAuditTxStore struct {
	beginErr error
	gotTx    *sql.Tx
}

func (s *minimalAuditTxStore) Create(ctx context.Context, entry *audit.Entry) error { return nil }
func (s *minimalAuditTxStore) GetByID(ctx context.Context, id string) (*audit.Entry, error) {
	return nil, nil
}
func (s *minimalAuditTxStore) List(ctx context.Context, filter audit.ListFilter, limit, offset int) ([]*audit.Entry, error) {
	return nil, nil
}
func (s *minimalAuditTxStore) BeginTx(ctx context.Context) (*sql.Tx, error) {
	if s.beginErr != nil {
		return nil, s.beginErr
	}
	return &sql.Tx{}, nil
}
func (s *minimalAuditTxStore) WithTx(tx *sql.Tx) audit.Store {
	s.gotTx = tx
	return s
}

func TestPrepareAuditTx_Success_ConfiguresTxScopedStores(t *testing.T) {
	flagsStore := &minimalFlagsTxAwareStore{}
	auditStore := &minimalAuditTxStore{}
	svc := NewServiceWithAudit(flagsStore, auditStore)
	ctx := auth.WithActorID(context.Background(), "actor-1")

	out, err := svc.prepareAuditTx(ctx)

	if err != nil {
		t.Fatalf("prepareAuditTx: %v", err)
	}
	if out == nil || out.actorID != "actor-1" || out.tx == nil {
		t.Fatalf("unexpected audit tx context: %+v", out)
	}
	if flagsStore.gotTx == nil {
		t.Fatal("expected flags store WithTx to be called")
	}
	if auditStore.gotTx == nil {
		t.Fatal("expected audit store WithTx to be called")
	}
}

func TestDeleteFlagWithStoreAndID_TableDriven(t *testing.T) {
	svc := NewService(&minimalFlagsTxAwareStore{})
	ctx := context.Background()

	t.Run("delete success", func(t *testing.T) {
		store := &minimalFlagsTxAwareStore{}
		ok, id, err := svc.deleteFlagWithStoreAndID(ctx, store, "k1", DeploymentStageDev)
		if err != nil || !ok || id == "" {
			t.Fatalf("unexpected result: ok=%v id=%q err=%v", ok, id, err)
		}
	})

	t.Run("delete returns operation error", func(t *testing.T) {
		store := &minimalFlagsTxAwareStore{deleteErr: errors.New("delete failed")}
		ok, id, err := svc.deleteFlagWithStoreAndID(ctx, store, "k1", DeploymentStageDev)
		if ok || id != "" || err == nil {
			t.Fatalf("unexpected result: ok=%v id=%q err=%v", ok, id, err)
		}
		var opErr *OperationError
		if !errors.As(err, &opErr) {
			t.Fatalf("expected *OperationError, got %T", err)
		}
	})
}

func TestDeleteFlag_NoAudit_Path(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		svc := NewService(&minimalFlagsTxAwareStore{})
		ok, err := svc.DeleteFlag(ctx, "k1", DeploymentStageDev)
		if err != nil {
			t.Fatalf("DeleteFlag: %v", err)
		}
		if !ok {
			t.Fatal("expected flag to be deleted")
		}
	})

	t.Run("delete store error", func(t *testing.T) {
		svc := NewService(&minimalFlagsTxAwareStore{deleteErr: errors.New("boom")})
		ok, err := svc.DeleteFlag(ctx, "k1", DeploymentStageDev)
		if ok {
			t.Fatal("expected delete to fail")
		}
		if err == nil {
			t.Fatal("expected operation error")
		}
		var opErr *OperationError
		if !errors.As(err, &opErr) {
			t.Fatalf("expected *OperationError, got %T", err)
		}
	})
}
