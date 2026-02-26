package flags

import "errors"

var (
	ErrDuplicateKey   = errors.New("flags: duplicate key and environment")
	ErrNotFound       = errors.New("flags: flag not found")
	ErrInvalidUserID  = errors.New("flags: invalid user ID")
	ErrInvalidRule    = errors.New("flags: invalid rule configuration")
)
