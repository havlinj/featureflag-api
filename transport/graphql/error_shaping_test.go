package graphql

import (
	"context"
	"errors"
	"testing"

	"github.com/havlinj/featureflag-api/internal/auth"
	"github.com/havlinj/featureflag-api/internal/users"
)

func TestPresentError_unauthorized_is_sanitized(t *testing.T) {
	err := &auth.UnauthorizedError{Reason: "token expired"}

	presented := PresentError(context.Background(), err)

	if presented == nil {
		t.Fatal("expected presented error")
	}
	if presented.Message != msgUnauthorized {
		t.Fatalf("expected %q, got %q", msgUnauthorized, presented.Message)
	}
}

func TestPresentError_forbidden_is_sanitized(t *testing.T) {
	err := &auth.ForbiddenError{Role: "viewer", AllowedRoles: []string{"admin", "developer"}}

	presented := PresentError(context.Background(), err)

	if presented == nil {
		t.Fatal("expected presented error")
	}
	if presented.Message != msgForbidden {
		t.Fatalf("expected %q, got %q", msgForbidden, presented.Message)
	}
}

func TestPresentError_invalid_credentials_is_sanitized(t *testing.T) {
	err := &users.InvalidCredentialsError{Email: "user@example.com"}

	presented := PresentError(context.Background(), err)

	if presented == nil {
		t.Fatal("expected presented error")
	}
	if presented.Message != msgInvalidCredential {
		t.Fatalf("expected %q, got %q", msgInvalidCredential, presented.Message)
	}
}

func TestPresentError_internal_config_is_sanitized(t *testing.T) {
	err := errors.New("audit service not configured")

	presented := PresentError(context.Background(), err)

	if presented == nil {
		t.Fatal("expected presented error")
	}
	if presented.Message != msgInternalError {
		t.Fatalf("expected %q, got %q", msgInternalError, presented.Message)
	}
}
