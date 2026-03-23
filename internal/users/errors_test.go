package users

import (
	"errors"
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
	want := "users: invalid credentials"
	if got != want {
		t.Errorf("Error() = %q; want %q", got, want)
	}
}

func TestOperationError_Error_and_unwrap_are_deterministic(t *testing.T) {
	causeA := errors.New("db failure A")
	errA := &OperationError{
		Op:    opServiceUpdateUserStoreUpdate,
		ID:    "u1",
		Email: "a@b.com",
		Role:  "admin",
		Cause: causeA,
	}

	if got, want := errA.Error(), `users: operation="users.service.update_user.store_update" id="u1" email="a@b.com" role="admin": db failure A`; got != want {
		t.Errorf("Error() = %q; want %q", got, want)
	}
	if !errors.Is(errA, causeA) {
		t.Errorf("errors.Is(errA, causeA) = false; want true")
	}

	causeB := errors.New("db failure B")
	errB := &OperationError{
		Op:    opRepoGetByEmail,
		ID:    "",
		Email: "x@y.com",
		Role:  "",
		Cause: causeB,
	}

	if got, want := errB.Error(), `users: operation="users.repo.get_by_email" id="" email="x@y.com" role="": db failure B`; got != want {
		t.Errorf("Error() = %q; want %q", got, want)
	}
	if !errors.Is(errB, causeB) {
		t.Errorf("errors.Is(errB, causeB) = false; want true")
	}
}
