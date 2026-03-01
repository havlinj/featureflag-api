package auth

import (
	"context"
	"testing"
)

func TestWithClaims_FromContext_roundtrip(t *testing.T) {
	ctx := context.Background()
	claims := &Claims{Sub: "user-1", Role: "admin"}

	ctxWith := WithClaims(ctx, claims)
	out, ok := FromContext(ctxWith)
	if !ok || out == nil {
		t.Fatal("expected claims from context")
	}
	if out.Sub != claims.Sub || out.Role != claims.Role {
		t.Errorf("expected %+v, got %+v", claims, out)
	}
}

func TestFromContext_emptyContextReturnsFalse(t *testing.T) {
	_, ok := FromContext(context.Background())
	if ok {
		t.Error("expected false when no claims in context")
	}
}

func TestRequireRole_allowedRoleReturnsUserID(t *testing.T) {
	ctx := WithClaims(context.Background(), &Claims{Sub: "u1", Role: "developer"})

	userID, err := RequireRole(ctx, "admin", "developer")

	if err != nil {
		t.Fatalf("RequireRole: %v", err)
	}
	if userID != "u1" {
		t.Errorf("expected userID u1, got %q", userID)
	}
}

func TestRequireRole_disallowedRoleReturnsErrForbidden(t *testing.T) {
	ctx := WithClaims(context.Background(), &Claims{Sub: "u1", Role: "viewer"})

	_, err := RequireRole(ctx, "admin", "developer")

	if err == nil {
		t.Fatal("expected error for disallowed role")
	}
	if err != ErrForbidden {
		t.Errorf("expected ErrForbidden, got %v", err)
	}
}

func TestRequireRole_noClaimsReturnsErrUnauthorized(t *testing.T) {
	_, err := RequireRole(context.Background(), "admin")

	if err == nil {
		t.Fatal("expected error when no claims")
	}
	if err != ErrUnauthorized {
		t.Errorf("expected ErrUnauthorized, got %v", err)
	}
}

func TestRequireRole_nilClaimsReturnsErrUnauthorized(t *testing.T) {
	ctx := WithClaims(context.Background(), nil)

	_, err := RequireRole(ctx, "admin")

	if err == nil {
		t.Fatal("expected error when claims nil")
	}
	if err != ErrUnauthorized {
		t.Errorf("expected ErrUnauthorized, got %v", err)
	}
}
