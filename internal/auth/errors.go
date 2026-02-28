package auth

import "errors"

var (
	ErrUnauthorized       = errors.New("unauthorized")
	ErrForbidden          = errors.New("user does not have required rights")
	ErrAdminNotConfigured = errors.New("admin has not been set up yet; run the seed script")
)
