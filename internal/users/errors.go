package users

import "errors"

var (
	ErrDuplicateEmail    = errors.New("users: duplicate email")
	ErrNotFound          = errors.New("users: user not found")
	ErrInvalidCredentials = errors.New("invalid credentials")
)
