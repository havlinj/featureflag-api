package experiments

import "errors"

var (
	ErrExperimentNotFound   = errors.New("experiments: experiment not found")
	ErrDuplicateExperiment  = errors.New("experiments: duplicate experiment key and environment")
	ErrVariantNotFound      = errors.New("experiments: variant not found")
	ErrInvalidWeights       = errors.New("experiments: variant weights must sum to 100")
	ErrInvalidUserID        = errors.New("experiments: invalid user ID")
)
