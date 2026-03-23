package experiments

import (
	"errors"
	"testing"
)

func TestInvalidWeightsError_Error_full_message(t *testing.T) {
	e := &InvalidWeightsError{Sum: 80, Weights: []int{40, 40}, Reason: "sum != 100"}
	got := e.Error()
	want := "experiments: variant weights must sum to 100 (sum != 100): sum=80 weights=[40 40]"
	if got != want {
		t.Errorf("Error() = %q; want %q", got, want)
	}
}

func TestInvalidWeightsError_Error_without_reason_full_message(t *testing.T) {
	e := &InvalidWeightsError{Sum: 50, Weights: []int{25, 25}}
	got := e.Error()
	want := "experiments: variant weights must sum to 100, got sum=50 weights=[25 25]"
	if got != want {
		t.Errorf("Error() = %q; want %q", got, want)
	}
}

func TestExperimentNotFoundError_Error_full_message(t *testing.T) {
	e := &ExperimentNotFoundError{Key: "my-exp", Environment: "prod"}
	got := e.Error()
	want := `experiments: experiment not found key="my-exp" environment="prod"`
	if got != want {
		t.Errorf("Error() = %q; want %q", got, want)
	}
}

func TestDuplicateExperimentError_Error_full_message(t *testing.T) {
	e := &DuplicateExperimentError{Key: "dup", Environment: "staging"}
	got := e.Error()
	want := `experiments: duplicate experiment key="dup" environment="staging"`
	if got != want {
		t.Errorf("Error() = %q; want %q", got, want)
	}
}

func TestVariantNotFoundError_Error_by_variant_id_full_message(t *testing.T) {
	e := &VariantNotFoundError{VariantID: "v-unknown", ExperimentKey: "e1", Environment: "dev"}
	got := e.Error()
	want := `experiments: variant not found variant_id="v-unknown"`
	if got != want {
		t.Errorf("Error() = %q; want %q", got, want)
	}
}

func TestVariantNotFoundError_Error_no_variants_full_message(t *testing.T) {
	e := &VariantNotFoundError{ExperimentKey: "e1", Environment: "prod"}
	got := e.Error()
	want := `experiments: no variants for experiment key="e1" environment="prod"`
	if got != want {
		t.Errorf("Error() = %q; want %q", got, want)
	}
}

func TestInvalidUserIDError_Error_full_message(t *testing.T) {
	e := &InvalidUserIDError{UserID: ""}
	got := e.Error()
	want := `experiments: invalid user ID (got "")`
	if got != want {
		t.Errorf("Error() = %q; want %q", got, want)
	}
}

func TestOperationError_Error_and_unwrap_are_deterministic(t *testing.T) {
	causeA := errors.New("db failure A")
	errA := &OperationError{
		Op:            opServiceGetAssignmentStoreUpsertAssignment,
		Key:           "ab-test",
		Environment:   "prod",
		ExperimentID:  "e1",
		UserID:        "u1",
		VariantID:     "v1",
		VariantName:   "A",
		VariantWeight: 50,
		Cause:         causeA,
	}

	if got, want := errA.Error(), `experiments: operation="experiments.service.get_assignment.store_upsert_assignment" key="ab-test" environment="prod" experiment_id="e1" user_id="u1" variant_id="v1" variant_name="A" variant_weight=50: db failure A`; got != want {
		t.Errorf("Error() = %q; want %q", got, want)
	}
	if !errors.Is(errA, causeA) {
		t.Errorf("errors.Is(errA, causeA) = false; want true")
	}

	causeB := errors.New("db failure B")
	errB := &OperationError{
		Op:           opRepoGetVariantsByExperimentIDScan,
		ExperimentID: "e2",
		Cause:        causeB,
	}

	if got, want := errB.Error(), `experiments: operation="experiments.repo.get_variants_by_experiment_id.scan" key="" environment="" experiment_id="e2" user_id="" variant_id="" variant_name="" variant_weight=0: db failure B`; got != want {
		t.Errorf("Error() = %q; want %q", got, want)
	}
	if !errors.Is(errB, causeB) {
		t.Errorf("errors.Is(errB, causeB) = false; want true")
	}
}
