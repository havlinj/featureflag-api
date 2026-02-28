package auth

import (
	"context"
)

type contextKey string

const claimsKey contextKey = "auth_claims"

// Claims holds JWT payload data (sub = user ID, role, exp).
type Claims struct {
	Sub  string
	Role string
}

// WithClaims returns a context with the given claims attached.
func WithClaims(ctx context.Context, claims *Claims) context.Context {
	return context.WithValue(ctx, claimsKey, claims)
}

// FromContext returns the claims from context and true if present.
// If no claims are set, returns (nil, false).
func FromContext(ctx context.Context) (*Claims, bool) {
	claims, ok := ctx.Value(claimsKey).(*Claims)
	return claims, ok
}

// RequireRole checks that the context has valid claims and that the role
// is one of allowedRoles. Returns the subject (user ID) and nil on success.
// Returns ErrUnauthorized when no claims, ErrForbidden when role not allowed.
func RequireRole(ctx context.Context, allowedRoles ...string) (userID string, err error) {
	claims, ok := FromContext(ctx)
	if !ok || claims == nil {
		return "", ErrUnauthorized
	}
	for _, r := range allowedRoles {
		if claims.Role == r {
			return claims.Sub, nil
		}
	}
	return "", ErrForbidden
}
