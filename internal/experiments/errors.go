package experiments

import "fmt"

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
