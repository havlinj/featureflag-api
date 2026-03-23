package experiments

import "fmt"

// Operation identifiers for structured context propagation across service/repository layers.
const (
	opServiceCreateExperimentStoreCreateExperiment             = "experiments.service.create_experiment.store_create_experiment"
	opServiceEnsureUniqueExperimentStoreGetByKeyAndEnvironment = "experiments.service.ensure_unique_experiment.store_get_by_key_and_environment"
	opServiceCreateExperimentStoreCreateVariant                = "experiments.service.create_experiment.store_create_variant"
	opServiceGetExperimentStoreGetByKeyAndEnvironment          = "experiments.service.get_experiment.store_get_by_key_and_environment"
	opServiceGetAssignmentStoreGetVariantsByExperimentID       = "experiments.service.get_assignment.store_get_variants_by_experiment_id"
	opServiceGetAssignmentStoreGetAssignment                   = "experiments.service.get_assignment.store_get_assignment"
	opServiceGetAssignmentStoreUpsertAssignment                = "experiments.service.get_assignment.store_upsert_assignment"
	opServiceGetExperimentOrErrStoreGetByKeyAndEnvironment     = "experiments.service.get_experiment_or_err.store_get_by_key_and_environment"
	opRepoBeginTx                                              = "experiments.repo.begin_tx"
	opRepoCreateExperiment                                     = "experiments.repo.create_experiment"
	opRepoGetExperimentByKeyAndEnvironment                     = "experiments.repo.get_experiment_by_key_and_environment"
	opRepoGetExperimentByID                                    = "experiments.repo.get_experiment_by_id"
	opRepoCreateVariant                                        = "experiments.repo.create_variant"
	opRepoGetVariantsByExperimentIDQuery                       = "experiments.repo.get_variants_by_experiment_id.query"
	opRepoGetVariantsByExperimentIDScan                        = "experiments.repo.get_variants_by_experiment_id.scan"
	opRepoGetVariantsByExperimentIDIterate                     = "experiments.repo.get_variants_by_experiment_id.iterate"
	opRepoGetAssignment                                        = "experiments.repo.get_assignment"
	opRepoUpsertAssignment                                     = "experiments.repo.upsert_assignment"
)

// InvalidWeightsError is returned when variant weights are invalid (empty, negative, or sum != 100).
// Match with: var e *InvalidWeightsError; errors.As(err, &e)
type InvalidWeightsError struct {
	Sum     int   // actual sum
	Weights []int // weights that led to the error
	Reason  string
}

func (e *InvalidWeightsError) Error() string {
	if e.Reason != "" {
		return fmt.Sprintf("experiments: variant weights must sum to 100 (%s): sum=%d weights=%v", e.Reason, e.Sum, e.Weights)
	}
	return fmt.Sprintf("experiments: variant weights must sum to 100, got sum=%d weights=%v", e.Sum, e.Weights)
}

// ExperimentNotFoundError is returned when an experiment does not exist for the given key and environment.
type ExperimentNotFoundError struct {
	Key         string
	Environment string
}

func (e *ExperimentNotFoundError) Error() string {
	return fmt.Sprintf("experiments: experiment not found key=%q environment=%q", e.Key, e.Environment)
}

// DuplicateExperimentError is returned when creating an experiment that already exists (key + environment).
type DuplicateExperimentError struct {
	Key         string
	Environment string
}

func (e *DuplicateExperimentError) Error() string {
	return fmt.Sprintf("experiments: duplicate experiment key=%q environment=%q", e.Key, e.Environment)
}

// VariantNotFoundError is returned when no variants exist for an experiment or a variant ID is unknown.
type VariantNotFoundError struct {
	VariantID     string // set when a stored assignment references an unknown variant
	ExperimentKey string
	Environment   string
}

func (e *VariantNotFoundError) Error() string {
	if e.VariantID != "" {
		return fmt.Sprintf("experiments: variant not found variant_id=%q", e.VariantID)
	}
	return fmt.Sprintf("experiments: no variants for experiment key=%q environment=%q", e.ExperimentKey, e.Environment)
}

// InvalidUserIDError is returned when user ID is empty or invalid.
type InvalidUserIDError struct {
	UserID string
}

func (e *InvalidUserIDError) Error() string {
	return fmt.Sprintf("experiments: invalid user ID (got %q)", e.UserID)
}

// OperationError is returned when a store/service operation fails and should carry structured context.
type OperationError struct {
	Op            string
	Key           string
	Environment   string
	ExperimentID  string
	UserID        string
	VariantID     string
	VariantName   string
	VariantWeight int
	Cause         error
}

func (e *OperationError) Error() string {
	return fmt.Sprintf(
		"experiments: operation=%q key=%q environment=%q experiment_id=%q user_id=%q variant_id=%q variant_name=%q variant_weight=%d: %v",
		e.Op, e.Key, e.Environment, e.ExperimentID, e.UserID, e.VariantID, e.VariantName, e.VariantWeight, e.Cause,
	)
}

func (e *OperationError) Unwrap() error {
	return e.Cause
}
