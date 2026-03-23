package users

import "fmt"

// Operation identifiers for structured context propagation across service/repository layers.
const (
	opServiceCreateUserStoreCreate            = "users.service.create_user.store_create"
	opServiceGetUserStoreGetByID              = "users.service.get_user.store_get_by_id"
	opServiceGetUserByEmailStoreGetByEmail    = "users.service.get_user_by_email.store_get_by_email"
	opServiceLoginStoreGetByEmail             = "users.service.login.store_get_by_email"
	opServiceUpdateUserStoreGetByID           = "users.service.update_user.store_get_by_id"
	opServiceUpdateUserStoreUpdate            = "users.service.update_user.store_update"
	opServiceDeleteUserStoreDelete            = "users.service.delete_user.store_delete"
	opServiceEnsureUniqueEmailStoreGetByEmail = "users.service.ensure_unique_email.store_get_by_email"
	opRepoCreate                              = "users.repo.create"
	opRepoGetByID                             = "users.repo.get_by_id"
	opRepoGetByEmail                          = "users.repo.get_by_email"
	opRepoUpdate                              = "users.repo.update"
	opRepoDelete                              = "users.repo.delete"
)

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
	return "users: invalid credentials"
}

// OperationError is returned when a store/service operation fails and should carry structured context.
type OperationError struct {
	Op    string
	ID    string
	Email string
	Role  string
	Cause error
}

func (e *OperationError) Error() string {
	return fmt.Sprintf("users: operation=%q id=%q email=%q role=%q: %v", e.Op, e.ID, e.Email, e.Role, e.Cause)
}

func (e *OperationError) Unwrap() error {
	return e.Cause
}
