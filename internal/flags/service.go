package flags

import (
	"context"

	"github.com/jan-havlin-dev/featureflag-api/graph/model"
)

// Service holds core business logic for feature flags. It depends on Store for
// persistence so that the latter can be mocked in unit tests.
//
// Intended logic (to implement later):
//   - CreateFlag: GetByKeyAndEnvironment to check uniqueness; Create; map to model.FeatureFlag.
//   - UpdateFlag: Resolve flag (by key; environment from context/API when added); Update(flag); return model.
//   - EvaluateFlag: GetByKeyAndEnvironment; if missing or !enabled return false; GetRulesByFlagID;
//     apply percentage/attribute rules in service; return bool.
type Service struct {
	Store Store
}

// NewService returns a flags service that uses the given store.
func NewService(store Store) *Service {
	return &Service{Store: store}
}

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
