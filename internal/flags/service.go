package flags

import (
	"context"
	"fmt"
	"hash/fnv"
	"strconv"

	"github.com/jan-havlin-dev/featureflag-api/graph/model"
)

const defaultEnvironment = "dev"

// Service holds core business logic for feature flags. It depends on Store for
// persistence so that the latter can be mocked in unit tests.
type Service struct {
	Store Store
}

// NewService returns a flags service that uses the given store.
func NewService(store Store) *Service {
	return &Service{Store: store}
}

// CreateFlag creates a new feature flag.
func (s *Service) CreateFlag(ctx context.Context, input model.CreateFlagInput) (*model.FeatureFlag, error) {
	if err := s.ensureUniqueFlag(ctx, input.Key, input.Environment); err != nil {
		return nil, err
	}

	flag := &Flag{
		Key:         input.Key,
		Description: input.Description,
		Enabled:     false,
		Environment: input.Environment,
	}

	created, err := s.Store.Create(ctx, flag)
	if err != nil {
		return nil, fmt.Errorf("create flag: %w", err)
	}

	return flagToModel(created), nil
}

// UpdateFlag updates an existing feature flag (e.g. enabled state).
func (s *Service) UpdateFlag(ctx context.Context, input model.UpdateFlagInput) (*model.FeatureFlag, error) {
	flag, err := s.Store.GetByKeyAndEnvironment(ctx, input.Key, defaultEnvironment)
	if err != nil {
		return nil, fmt.Errorf("get flag: %w", err)
	}
	if flag == nil {
		return nil, ErrNotFound
	}

	flag.Enabled = input.Enabled

	if err := s.Store.Update(ctx, flag); err != nil {
		return nil, fmt.Errorf("update flag: %w", err)
	}

	return flagToModel(flag), nil
}

// EvaluateFlag returns whether the flag is enabled for the given user.
func (s *Service) EvaluateFlag(ctx context.Context, key, userID string) (bool, error) {
	if userID == "" {
		return false, ErrInvalidUserID
	}

	flag, err := s.Store.GetByKeyAndEnvironment(ctx, key, defaultEnvironment)
	if err != nil {
		return false, fmt.Errorf("get flag: %w", err)
	}
	if flag == nil || !flag.Enabled {
		return false, nil
	}

	rules, err := s.Store.GetRulesByFlagID(ctx, flag.ID)
	if err != nil {
		return false, fmt.Errorf("get rules: %w", err)
	}
	if len(rules) == 0 {
		return true, nil
	}

	return evaluateRulesForUser(userID, rules)
}

func (s *Service) ensureUniqueFlag(ctx context.Context, key, environment string) error {
	existing, err := s.Store.GetByKeyAndEnvironment(ctx, key, environment)
	if err != nil {
		return fmt.Errorf("check existing flag: %w", err)
	}
	if existing != nil {
		return ErrDuplicateKey
	}
	return nil
}

func flagToModel(flag *Flag) *model.FeatureFlag {
	if flag == nil {
		return nil
	}

	return &model.FeatureFlag{
		ID:          flag.ID,
		Key:         flag.Key,
		Description: flag.Description,
		Enabled:     flag.Enabled,
		Environment: flag.Environment,
	}
}

func evaluateRulesForUser(userID string, rules []*Rule) (bool, error) {
	for _, rule := range rules {
		if rule.Type != RuleTypePercentage {
			continue
		}

		enabled, err := evaluatePercentageRule(userID, rule.Value)
		if err != nil {
			return false, err
		}
		return enabled, nil
	}

	return true, nil
}

func evaluatePercentageRule(userID, value string) (bool, error) {
	percentage, err := strconv.Atoi(value)
	if err != nil {
		return false, ErrInvalidRule
	}
	if percentage < 0 || percentage > 100 {
		return false, ErrInvalidRule
	}

	bucket := hashToBucket(userID)
	return bucket < percentage, nil
}

func hashToBucket(userID string) int {
	h := fnv.New32a()
	_, _ = h.Write([]byte(userID))
	return int(h.Sum32() % 100)
}
