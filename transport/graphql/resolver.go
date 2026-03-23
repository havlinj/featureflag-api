package graphql

// This file will not be regenerated automatically.
//
// It serves as dependency injection for your app, add any dependencies you require
// here.

import (
	"context"
	"time"

	"github.com/havlinj/featureflag-api/graph/model"
	"github.com/havlinj/featureflag-api/internal/audit"
	"github.com/havlinj/featureflag-api/internal/experiments"
	"github.com/havlinj/featureflag-api/internal/flags"
	"github.com/havlinj/featureflag-api/internal/users"
)

// ExperimentsService is the interface used by experiments resolvers (for wiring and testing).
type ExperimentsService interface {
	CreateExperiment(ctx context.Context, input model.CreateExperimentInput) (*model.Experiment, error)
	GetExperiment(ctx context.Context, key, environment string) (*model.Experiment, error)
	GetAssignment(ctx context.Context, userID, experimentKey, environment string) (*model.ExperimentVariant, error)
}

// Resolver wires GraphQL resolvers to the service layers.
type Resolver struct {
	flags       *flags.Service
	users       *users.Service
	experiments ExperimentsService
	audit       *audit.Service
	jwtSecret   []byte
	jwtExpiry   time.Duration
}

func NewResolver(
	flagsService *flags.Service,
	usersService *users.Service,
	experimentsService ExperimentsService,
	auditService *audit.Service,
	jwtSecret []byte,
	jwtExpiry time.Duration,
) *Resolver {
	return &Resolver{
		flags:       flagsService,
		users:       usersService,
		experiments: experimentsService,
		audit:       auditService,
		jwtSecret:   jwtSecret,
		jwtExpiry:   jwtExpiry,
	}
}

// Ensure *experiments.Service implements ExperimentsService at compile time.
var _ ExperimentsService = (*experiments.Service)(nil)
