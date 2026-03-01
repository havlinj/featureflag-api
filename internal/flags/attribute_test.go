package flags

import (
	"errors"
	"testing"
)

func TestAttributeValue_userId_returns_userID(t *testing.T) {
	got := attributeValue("userId", "u1", nil)
	if got != "u1" {
		t.Errorf("attributeValue(userId): got %q, want u1", got)
	}
}

func TestAttributeValue_user_id_returns_userID(t *testing.T) {
	got := attributeValue("user_id", "u2", nil)
	if got != "u2" {
		t.Errorf("attributeValue(user_id): got %q, want u2", got)
	}
}

func TestAttributeValue_email_nil_returns_empty(t *testing.T) {
	got := attributeValue("email", "u1", nil)
	if got != "" {
		t.Errorf("attributeValue(email, nil): got %q, want empty", got)
	}
}

func TestAttributeValue_email_set_returns_email(t *testing.T) {
	email := "a@b.com"
	got := attributeValue("email", "u1", &email)
	if got != "a@b.com" {
		t.Errorf("attributeValue(email, set): got %q, want a@b.com", got)
	}
}

func TestAttributeValue_unknown_returns_empty(t *testing.T) {
	got := attributeValue("other", "u1", nil)
	if got != "" {
		t.Errorf("attributeValue(other): got %q, want empty", got)
	}
}

func TestEvaluateAttributeRule_suffix_match_returns_true(t *testing.T) {
	email := "user@company.com"
	enabled, err := evaluateAttributeRule("u1", &email, `{"attribute":"email","op":"suffix","value":"@company.com"}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !enabled {
		t.Error("expected true for suffix match")
	}
}

func TestEvaluateAttributeRule_suffix_no_match_returns_false(t *testing.T) {
	email := "user@other.com"
	enabled, err := evaluateAttributeRule("u1", &email, `{"attribute":"email","op":"suffix","value":"@company.com"}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if enabled {
		t.Error("expected false for suffix no match")
	}
}

func TestEvaluateAttributeRule_suffix_empty_value_returns_ErrInvalidRule(t *testing.T) {
	_, err := evaluateAttributeRule("u1", nil, `{"attribute":"email","op":"suffix","value":""}`)
	if !errors.Is(err, ErrInvalidRule) {
		t.Errorf("expected ErrInvalidRule, got %v", err)
	}
}

func TestEvaluateAttributeRule_in_match_returns_true(t *testing.T) {
	enabled, err := evaluateAttributeRule("u1", nil, `{"attribute":"userId","op":"in","values":["u0","u1","u2"]}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !enabled {
		t.Error("expected true for in match")
	}
}

func TestEvaluateAttributeRule_in_no_match_returns_false(t *testing.T) {
	enabled, err := evaluateAttributeRule("u1", nil, `{"attribute":"userId","op":"in","values":["u0","u2"]}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if enabled {
		t.Error("expected false for in no match")
	}
}

func TestEvaluateAttributeRule_in_empty_values_returns_ErrInvalidRule(t *testing.T) {
	_, err := evaluateAttributeRule("u1", nil, `{"attribute":"userId","op":"in","values":[]}`)
	if !errors.Is(err, ErrInvalidRule) {
		t.Errorf("expected ErrInvalidRule, got %v", err)
	}
}

func TestEvaluateAttributeRule_eq_match_returns_true(t *testing.T) {
	enabled, err := evaluateAttributeRule("u1", nil, `{"attribute":"userId","op":"eq","value":"u1"}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !enabled {
		t.Error("expected true for eq match")
	}
}

func TestEvaluateAttributeRule_eq_no_match_returns_false(t *testing.T) {
	enabled, err := evaluateAttributeRule("u1", nil, `{"attribute":"userId","op":"eq","value":"u2"}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if enabled {
		t.Error("expected false for eq no match")
	}
}

func TestEvaluateAttributeRule_invalid_json_returns_ErrInvalidRule(t *testing.T) {
	_, err := evaluateAttributeRule("u1", nil, `{invalid`)
	if !errors.Is(err, ErrInvalidRule) {
		t.Errorf("expected ErrInvalidRule, got %v", err)
	}
}

func TestEvaluateAttributeRule_unknown_op_returns_ErrInvalidRule(t *testing.T) {
	_, err := evaluateAttributeRule("u1", nil, `{"attribute":"userId","op":"unknown","value":"u1"}`)
	if !errors.Is(err, ErrInvalidRule) {
		t.Errorf("expected ErrInvalidRule, got %v", err)
	}
}
