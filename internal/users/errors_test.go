package users

import (
	"testing"
)

func TestDuplicateEmailError_Error_full_message(t *testing.T) {
	e := &DuplicateEmailError{Email: "a@b.com"}
	got := e.Error()
	want := `users: duplicate email="a@b.com"`
	if got != want {
		t.Errorf("Error() = %q; want %q", got, want)
	}
}

func TestNotFoundError_Error_by_id_full_message(t *testing.T) {
	e := &NotFoundError{ID: "user-123"}
	got := e.Error()
	want := `users: user not found id="user-123"`
	if got != want {
		t.Errorf("Error() = %q; want %q", got, want)
	}
}

func TestNotFoundError_Error_by_email_full_message(t *testing.T) {
	e := &NotFoundError{Email: "missing@test.com"}
	got := e.Error()
	want := `users: user not found email="missing@test.com"`
	if got != want {
		t.Errorf("Error() = %q; want %q", got, want)
	}
}

func TestInvalidCredentialsError_Error_full_message(t *testing.T) {
	e := &InvalidCredentialsError{Email: "u@x.com"}
	got := e.Error()
	want := `users: invalid credentials email="u@x.com"`
	if got != want {
		t.Errorf("Error() = %q; want %q", got, want)
	}
}
