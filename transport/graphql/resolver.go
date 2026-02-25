package graphql

// This file will not be regenerated automatically.
//
// It serves as dependency injection for your app, add any dependencies you require
// here.

import "github.com/jan-havlin-dev/featureflag-api/internal/flags"

// Resolver wires GraphQL resolvers to the flags service layer.
type Resolver struct {
	Flags *flags.Service
}
