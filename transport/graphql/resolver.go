package graphql

// This file will not be regenerated automatically.
//
// It serves as dependency injection for your app, add any dependencies you require
// here.

import (
	"github.com/jan-havlin-dev/featureflag-api/internal/flags"
	"github.com/jan-havlin-dev/featureflag-api/internal/users"
)

// Resolver wires GraphQL resolvers to the service layers.
type Resolver struct {
	Flags *flags.Service
	Users *users.Service
}
