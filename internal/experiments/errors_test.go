package experiments

import (
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
