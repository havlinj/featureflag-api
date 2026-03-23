package flags

import "fmt"

// Operation identifiers for structured context propagation across service/repository layers.
const (
	opServiceCreateFlagStoreCreate                   = "flags.service.create_flag.store_create"
	opServiceCreateFlagReplaceRulesByFlagID          = "flags.service.create_flag.replace_rules_by_flag_id"
	opServiceUpdateFlagStoreUpdate                   = "flags.service.update_flag.store_update"
	opServiceGetFlagOrErrStoreGetByKeyAndEnvironment = "flags.service.get_flag_or_err.store_get_by_key_and_environment"
	opServiceUpdateFlagReplaceRulesByFlagIDClear     = "flags.service.update_flag.replace_rules_by_flag_id.clear"
	opServiceUpdateFlagReplaceRulesByFlagID          = "flags.service.update_flag.replace_rules_by_flag_id"
	opServiceEvaluateFlagStoreGetByKeyAndEnvironment = "flags.service.evaluate_flag.store_get_by_key_and_environment"
	opServiceEvaluateFlagStoreGetRulesByFlagID       = "flags.service.evaluate_flag.store_get_rules_by_flag_id"
	opServiceDeleteFlagStoreDelete                   = "flags.service.delete_flag.store_delete"
	opServiceEnsureUniqueFlagStoreGetByKeyAndEnv     = "flags.service.ensure_unique_flag.store_get_by_key_and_environment"
	opRepoCreate                                     = "flags.repo.create"
	opRepoGetByKeyAndEnvironment                     = "flags.repo.get_by_key_and_environment"
	opRepoUpdate                                     = "flags.repo.update"
	opRepoGetRulesByFlagIDQuery                      = "flags.repo.get_rules_by_flag_id.query"
	opRepoGetRulesByFlagIDScan                       = "flags.repo.get_rules_by_flag_id.scan"
	opRepoGetRulesByFlagIDIterate                    = "flags.repo.get_rules_by_flag_id.iterate"
	opRepoDelete                                     = "flags.repo.delete"
	opRepoReplaceRulesByFlagIDDelete                 = "flags.repo.replace_rules_by_flag_id.delete"
	opRepoReplaceRulesByFlagIDInsert                 = "flags.repo.replace_rules_by_flag_id.insert"
	opRepoReplaceRulesByFlagIDBeginTx                = "flags.repo.replace_rules_by_flag_id.begin_tx"
	opRepoReplaceRulesByFlagIDCommit                 = "flags.repo.replace_rules_by_flag_id.commit"
)

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

// OperationError is returned when a store/service operation fails and should carry structured context.
type OperationError struct {
	Op          string
	Key         string
	Environment string
	FlagID      string
	Cause       error
}

func (e *OperationError) Error() string {
	return fmt.Sprintf("flags: operation=%q key=%q environment=%q flag_id=%q: %v", e.Op, e.Key, e.Environment, e.FlagID, e.Cause)
}

func (e *OperationError) Unwrap() error {
	return e.Cause
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
