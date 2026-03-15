package experiments

// Experiment is the domain entity for an A/B experiment (persistence layer).
type Experiment struct {
	ID          string
	Key         string
	Environment string
}

// Variant is one variant of an experiment (e.g. A, B, control).
type Variant struct {
	ID           string
	ExperimentID string
	Name         string
	Weight       int // 0..100, relative weight for assignment
}

// Assignment records which variant a user was assigned to for an experiment.
type Assignment struct {
	UserID       string
	ExperimentID string
	VariantID    string
}
