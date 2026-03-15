package auth

import (
	"testing"
)

func TestUnauthorizedError_Error_with_reason_full_message(t *testing.T) {
	e := &UnauthorizedError{Reason: "no or invalid claims"}
	got := e.Error()
	want := "auth: unauthorized: no or invalid claims"
	if got != want {
		t.Errorf("Error() = %q; want %q", got, want)
	}
}

func TestUnauthorizedError_Error_empty_reason_full_message(t *testing.T) {
	e := &UnauthorizedError{}
	got := e.Error()
	want := "auth: unauthorized (no or invalid claims)"
	if got != want {
		t.Errorf("Error() = %q; want %q", got, want)
	}
}

func TestForbiddenError_Error_full_message(t *testing.T) {
	e := &ForbiddenError{Role: "viewer", AllowedRoles: []string{"admin", "developer"}}
	got := e.Error()
	want := `auth: forbidden (role "viewer" not in allowed [admin developer])`
	if got != want {
		t.Errorf("Error() = %q; want %q", got, want)
	}
}
