package flags

import (
	"testing"
)

func TestDuplicateKeyError_Error_full_message(t *testing.T) {
	e := &DuplicateKeyError{Key: "my-flag", Environment: "prod"}
	got := e.Error()
	want := `flags: duplicate key="my-flag" environment="prod"`
	if got != want {
		t.Errorf("Error() = %q; want %q", got, want)
	}
}

func TestNotFoundError_Error_by_id_full_message(t *testing.T) {
	e := &NotFoundError{ID: "id-123"}
	got := e.Error()
	want := `flags: flag not found id="id-123"`
	if got != want {
		t.Errorf("Error() = %q; want %q", got, want)
	}
}

func TestNotFoundError_Error_by_key_env_full_message(t *testing.T) {
	e := &NotFoundError{Key: "x", Environment: "staging"}
	got := e.Error()
	want := `flags: flag not found key="x" environment="staging"`
	if got != want {
		t.Errorf("Error() = %q; want %q", got, want)
	}
}

func TestInvalidUserIDError_Error_full_message(t *testing.T) {
	e := &InvalidUserIDError{UserID: ""}
	got := e.Error()
	want := `flags: invalid user ID (got "")`
	if got != want {
		t.Errorf("Error() = %q; want %q", got, want)
	}
}

func TestInvalidRuleError_Error_with_reason_full_message(t *testing.T) {
	e := &InvalidRuleError{Value: "bad", Reason: "not a number"}
	got := e.Error()
	want := `flags: invalid rule (not a number) value="bad"`
	if got != want {
		t.Errorf("Error() = %q; want %q", got, want)
	}
}

func TestInvalidRuleError_Error_with_op_full_message(t *testing.T) {
	e := &InvalidRuleError{Value: "v", Op: "unknown"}
	got := e.Error()
	want := `flags: invalid rule op="unknown" value="v"`
	if got != want {
		t.Errorf("Error() = %q; want %q", got, want)
	}
}

func TestInvalidRuleError_Error_value_only_full_message(t *testing.T) {
	e := &InvalidRuleError{Value: "x"}
	got := e.Error()
	want := `flags: invalid rule value="x"`
	if got != want {
		t.Errorf("Error() = %q; want %q", got, want)
	}
}

func TestRulesStrategyMismatchError_Error_with_message_full_message(t *testing.T) {
	e := &RulesStrategyMismatchError{CurrentStrategy: "percentage", Message: "only percentage rules allowed"}
	got := e.Error()
	want := `flags: rules strategy mismatch (current="percentage"): only percentage rules allowed`
	if got != want {
		t.Errorf("Error() = %q; want %q", got, want)
	}
}

func TestRulesStrategyMismatchError_Error_with_rule_types_full_message(t *testing.T) {
	e := &RulesStrategyMismatchError{RuleTypes: []string{"percentage", "attribute"}}
	got := e.Error()
	want := "flags: rules do not match (mixed types [percentage attribute])"
	if got != want {
		t.Errorf("Error() = %q; want %q", got, want)
	}
}

func TestRulesStrategyMismatchError_Error_current_only_full_message(t *testing.T) {
	e := &RulesStrategyMismatchError{CurrentStrategy: "attribute"}
	got := e.Error()
	want := `flags: rules strategy mismatch (current="attribute")`
	if got != want {
		t.Errorf("Error() = %q; want %q", got, want)
	}
}
