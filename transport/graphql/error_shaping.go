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
)

// PresentError maps internal errors to safe external GraphQL messages.
func PresentError(ctx context.Context, err error) *gqlerror.Error {
	if isUnauthorizedError(err) {
		return graphql.DefaultErrorPresenter(ctx, errors.New(msgUnauthorized))
	}
	if isForbiddenError(err) {
		return graphql.DefaultErrorPresenter(ctx, errors.New(msgForbidden))
	}
	if isInvalidCredentialsError(err) {
		return graphql.DefaultErrorPresenter(ctx, errors.New(msgInvalidCredential))
	}
	if isInternalConfigError(err) {
		return graphql.DefaultErrorPresenter(ctx, errors.New(msgInternalError))
	}
	return graphql.DefaultErrorPresenter(ctx, err)
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
