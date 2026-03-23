package graphql

import (
	"context"
	"errors"
	"strings"

	"github.com/99designs/gqlgen/graphql"
	"github.com/vektah/gqlparser/v2/gqlerror"

	"github.com/havlinj/featureflag-api/internal/auth"
	"github.com/havlinj/featureflag-api/internal/users"
)

const (
	msgUnauthorized      = "unauthorized"
	msgForbidden         = "forbidden"
	msgInvalidCredential = "invalid credentials"
	msgInternalError     = "internal error"

	codeUnauthorized      = "UNAUTHORIZED"
	codeForbidden         = "FORBIDDEN"
	codeInvalidCredential = "INVALID_CREDENTIALS"
	codeInternal          = "INTERNAL"
)

// PresentError maps internal errors to safe external GraphQL messages.
func PresentError(ctx context.Context, err error) *gqlerror.Error {
	if isUnauthorizedError(err) {
		return presentWithCode(ctx, msgUnauthorized, codeUnauthorized)
	}
	if isForbiddenError(err) {
		return presentWithCode(ctx, msgForbidden, codeForbidden)
	}
	if isInvalidCredentialsError(err) {
		return presentWithCode(ctx, msgInvalidCredential, codeInvalidCredential)
	}
	if isInternalConfigError(err) {
		return presentWithCode(ctx, msgInternalError, codeInternal)
	}
	return graphql.DefaultErrorPresenter(ctx, err)
}

func presentWithCode(ctx context.Context, message, code string) *gqlerror.Error {
	ge := graphql.DefaultErrorPresenter(ctx, errors.New(message))
	if ge.Extensions == nil {
		ge.Extensions = make(map[string]any, 1)
	}
	ge.Extensions["code"] = code
	return ge
}

func isUnauthorizedError(err error) bool {
	var target *auth.UnauthorizedError
	return errors.As(err, &target)
}

func isForbiddenError(err error) bool {
	var target *auth.ForbiddenError
	return errors.As(err, &target)
}

func isInvalidCredentialsError(err error) bool {
	var target *users.InvalidCredentialsError
	return errors.As(err, &target)
}

func isInternalConfigError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "service not configured")
}
