package flags

import (
	"context"

	"github.com/jan-havlin-dev/featureflag-api/graph/model"
)

// Service holds core business logic for feature flags (not implemented yet).
type Service struct{}

// CreateFlag creates a new feature flag.
func (s *Service) CreateFlag(ctx context.Context, input model.CreateFlagInput) (*model.FeatureFlag, error) {
	panic("unimplemented")
}

// UpdateFlag updates an existing feature flag (e.g. enabled state).
func (s *Service) UpdateFlag(ctx context.Context, input model.UpdateFlagInput) (*model.FeatureFlag, error) {
	panic("unimplemented")
}

// EvaluateFlag returns whether the flag is enabled for the given user.
func (s *Service) EvaluateFlag(ctx context.Context, key, userID string) (bool, error) {
	panic("unimplemented")
}
