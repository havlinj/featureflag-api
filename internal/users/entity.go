package users

import "time"

// User is the domain entity for a user (persistence layer).
// PasswordHash is used only for login; never exposed via GraphQL.
type User struct {
	ID           string
	Email        string
	Role         Role
	PasswordHash *string
	CreatedAt    time.Time
}

// Role is the user's role for RBAC (admin, developer, viewer).
type Role string

const (
	RoleAdmin     Role = "admin"
	RoleDeveloper Role = "developer"
	RoleViewer    Role = "viewer"
)
