package flags

import "fmt"

// DuplicateKeyError is returned when creating a flag that already exists (key + environment).
// Match with: var e *DuplicateKeyError; errors.As(err, &e)
type DuplicateKeyError struct {
	Key         string
	Environment string
}

func (e *DuplicateKeyError) Error() string {
	return fmt.Sprintf("flags: duplicate key=%q environment=%q", e.Key, e.Environment)
}

// NotFoundError is returned when a flag is not found (by key+environment or by id).
type NotFoundError struct {
	Key         string
	Environment string
	ID          string
}

func (e *NotFoundError) Error() string {
	if e.ID != "" {
		return fmt.Sprintf("flags: flag not found id=%q", e.ID)
	}
	return fmt.Sprintf("flags: flag not found key=%q environment=%q", e.Key, e.Environment)
}

// InvalidUserIDError is returned when user ID is empty or invalid in evaluation context.
type InvalidUserIDError struct {
	UserID string
}

func (e *InvalidUserIDError) Error() string {
	return fmt.Sprintf("flags: invalid user ID (got %q)", e.UserID)
}

// InvalidRuleError is returned when a rollout rule is invalid (value, op, or JSON).
type InvalidRuleError struct {
	Value  string
	Op     string
	Reason string
}

func (e *InvalidRuleError) Error() string {
	if e.Reason != "" {
		return fmt.Sprintf("flags: invalid rule (%s) value=%q", e.Reason, e.Value)
	}
	if e.Op != "" {
		return fmt.Sprintf("flags: invalid rule op=%q value=%q", e.Op, e.Value)
	}
	return fmt.Sprintf("flags: invalid rule value=%q", e.Value)
}

// RulesStrategyMismatchError is returned when rules do not match the flag's rollout strategy.
type RulesStrategyMismatchError struct {
	CurrentStrategy string
	RuleTypes       []string
	Message         string
}

func (e *RulesStrategyMismatchError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("flags: rules strategy mismatch (current=%q): %s", e.CurrentStrategy, e.Message)
	}
	if len(e.RuleTypes) > 0 {
		return fmt.Sprintf("flags: rules do not match (mixed types %v)", e.RuleTypes)
	}
	return fmt.Sprintf("flags: rules strategy mismatch (current=%q)", e.CurrentStrategy)
}
