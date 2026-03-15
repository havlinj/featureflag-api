package users

import "fmt"

// DuplicateEmailError is returned when creating a user with an email that already exists.
type DuplicateEmailError struct {
	Email string
}

func (e *DuplicateEmailError) Error() string {
	return fmt.Sprintf("users: duplicate email=%q", e.Email)
}

// NotFoundError is returned when a user is not found (by ID or by email).
type NotFoundError struct {
	ID    string
	Email string
}

func (e *NotFoundError) Error() string {
	if e.ID != "" {
		return fmt.Sprintf("users: user not found id=%q", e.ID)
	}
	return fmt.Sprintf("users: user not found email=%q", e.Email)
}

// InvalidCredentialsError is returned when login fails due to wrong password or missing hash.
type InvalidCredentialsError struct {
	Email string
}

func (e *InvalidCredentialsError) Error() string {
	return fmt.Sprintf("users: invalid credentials email=%q", e.Email)
}
