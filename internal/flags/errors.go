package flags

import "errors"

var (
	ErrDuplicateKey = errors.New("flags: duplicate key and environment")
	ErrNotFound     = errors.New("flags: flag not found")
)
