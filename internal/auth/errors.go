package auth

import "fmt"

// UnauthorizedError is returned when the context has no or invalid claims.
type UnauthorizedError struct {
	Reason string
}

func (e *UnauthorizedError) Error() string {
	if e.Reason != "" {
		return fmt.Sprintf("auth: unauthorized: %s", e.Reason)
	}
	return "auth: unauthorized (no or invalid claims)"
}

// ForbiddenError is returned when the user's role is not in the allowed set.
type ForbiddenError struct {
	Role         string
	AllowedRoles []string
}

func (e *ForbiddenError) Error() string {
	return fmt.Sprintf("auth: forbidden (role %q not in allowed %v)", e.Role, e.AllowedRoles)
}
