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

// CreateFlag creates a new feature flag. Optional rules set rollout strategy; all rules must be same type.
func (s *Service) CreateFlag(ctx context.Context, input model.CreateFlagInput) (*model.FeatureFlag, error) {
	if err := s.ensureUniqueFlag(ctx, input.Key, input.Environment); err != nil {
		return nil, err
	}

	strategy := rolloutStrategyFromModel(input.RolloutStrategy)
	if len(input.Rules) > 0 {
		ruleType, err := validateRulesSameType(input.Rules)
		if err != nil {
			return nil, err
		}
		if strategy != RolloutStrategyNone && strategy != ruleTypeToStrategy(ruleType) {
			return nil, fmt.Errorf("%w: all rules must use the same strategy: percentage or attribute", ErrRulesStrategyMismatch)
		}
		strategy = ruleTypeToStrategy(ruleType)
	}

	flag := &Flag{
		Key:              input.Key,
		Description:      input.Description,
		Enabled:          false,
		Environment:      input.Environment,
		RolloutStrategy:  strategy,
	}

	created, err := s.Store.Create(ctx, flag)
	if err != nil {
		return nil, fmt.Errorf("create flag: %w", err)
	}

	if len(input.Rules) > 0 {
		rules := ruleInputsToRules(created.ID, input.Rules)
		if err := s.Store.ReplaceRulesByFlagID(ctx, created.ID, rules); err != nil {
			return nil, fmt.Errorf("create flag rules: %w", err)
		}
	}

	return flagToModel(created), nil
}

// UpdateFlag updates an existing feature flag. If Rules is present, replaces all rules and updates strategy.
func (s *Service) UpdateFlag(ctx context.Context, input model.UpdateFlagInput) (*model.FeatureFlag, error) {
	flag, err := s.Store.GetByKeyAndEnvironment(ctx, input.Key, defaultEnvironment)
	if err != nil {
		return nil, fmt.Errorf("get flag: %w", err)
	}
	if flag == nil {
		return nil, ErrNotFound
	}

	flag.Enabled = input.Enabled

	if input.Rules != nil {
		if len(input.Rules) == 0 {
			flag.RolloutStrategy = RolloutStrategyNone
			if err := s.Store.ReplaceRulesByFlagID(ctx, flag.ID, nil); err != nil {
				return nil, fmt.Errorf("update flag rules: %w", err)
			}
		} else {
			ruleType, err := validateRulesSameType(input.Rules)
			if err != nil {
				return nil, err
			}
			if flag.RolloutStrategy != RolloutStrategyNone && flag.RolloutStrategy != ruleTypeToStrategy(ruleType) {
				return nil, strategyMismatchError(flag.RolloutStrategy)
			}
			flag.RolloutStrategy = ruleTypeToStrategy(ruleType)
			rules := ruleInputsToRules(flag.ID, input.Rules)
			if err := s.Store.ReplaceRulesByFlagID(ctx, flag.ID, rules); err != nil {
				return nil, fmt.Errorf("update flag rules: %w", err)
			}
		}
	}

	if err := s.Store.Update(ctx, flag); err != nil {
		return nil, fmt.Errorf("update flag: %w", err)
	}

	return flagToModel(flag), nil
}

// EvaluateFlag returns whether the flag is enabled for the given evaluation context.
func (s *Service) EvaluateFlag(ctx context.Context, key string, evalCtx model.EvaluationContextInput) (bool, error) {
	if evalCtx.UserID == "" {
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

	return evaluateRulesByStrategy(flag.RolloutStrategy, evalCtx.UserID, evalCtx.Email, rules)
}

// DeleteFlag removes a flag by key and environment. Rules are removed by DB CASCADE.
func (s *Service) DeleteFlag(ctx context.Context, key, environment string) (bool, error) {
	flag, err := s.Store.GetByKeyAndEnvironment(ctx, key, environment)
	if err != nil {
		return false, fmt.Errorf("get flag: %w", err)
	}
	if flag == nil {
		return false, ErrNotFound
	}
	if err := s.Store.Delete(ctx, flag.ID); err != nil {
		return false, fmt.Errorf("delete flag: %w", err)
	}
	return true, nil
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
		ID:              flag.ID,
		Key:             flag.Key,
		Description:     flag.Description,
		Enabled:         flag.Enabled,
		Environment:     flag.Environment,
		RolloutStrategy: rolloutStrategyToModel(flag.RolloutStrategy),
	}
}

func rolloutStrategyFromModel(m *model.RolloutStrategy) RolloutStrategy {
	if m == nil {
		return RolloutStrategyNone
	}
	switch *m {
	case model.RolloutStrategyPercentage:
		return RolloutStrategyPercentage
	case model.RolloutStrategyAttribute:
		return RolloutStrategyAttribute
	default:
		return RolloutStrategyNone
	}
}

func rolloutStrategyToModel(s RolloutStrategy) model.RolloutStrategy {
	switch s {
	case RolloutStrategyPercentage:
		return model.RolloutStrategyPercentage
	case RolloutStrategyAttribute:
		return model.RolloutStrategyAttribute
	default:
		return model.RolloutStrategyNone
	}
}

func ruleTypeToStrategy(rt RuleType) RolloutStrategy {
	if rt == RuleTypePercentage {
		return RolloutStrategyPercentage
	}
	return RolloutStrategyAttribute
}

func validateRulesSameType(rules []*model.RuleInput) (RuleType, error) {
	if len(rules) == 0 {
		return "", nil
	}
	first := ruleInputToRuleType(rules[0])
	for _, r := range rules[1:] {
		if ruleInputToRuleType(r) != first {
			return "", fmt.Errorf("%w: all rules must use the same strategy: percentage or attribute", ErrRulesStrategyMismatch)
		}
	}
	return first, nil
}

func ruleInputToRuleType(r *model.RuleInput) RuleType {
	if r == nil {
		return ""
	}
	if r.Type == model.RolloutRuleTypePercentage {
		return RuleTypePercentage
	}
	return RuleTypeAttribute
}

func ruleInputsToRules(flagID string, inputs []*model.RuleInput) []*Rule {
	out := make([]*Rule, 0, len(inputs))
	for _, in := range inputs {
		if in == nil {
			continue
		}
		out = append(out, &Rule{
			FlagID: flagID,
			Type:   ruleInputToRuleType(in),
			Value:  in.Value,
		})
	}
	return out
}

const (
	msgStrategyMismatchPercentage = "flag uses percentage strategy; only percentage rules are allowed"
	msgStrategyMismatchAttribute  = "flag uses attribute strategy; only attribute rules are allowed"
)

func strategyMismatchError(current RolloutStrategy) error {
	msg := msgStrategyMismatchAttribute
	if current == RolloutStrategyPercentage {
		msg = msgStrategyMismatchPercentage
	}
	return fmt.Errorf("%w: %s", ErrRulesStrategyMismatch, msg)
}

func evaluateRulesByStrategy(strategy RolloutStrategy, userID string, email *string, rules []*Rule) (bool, error) {
	if strategy == RolloutStrategyPercentage {
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
	if strategy == RolloutStrategyAttribute {
		for _, rule := range rules {
			if rule.Type != RuleTypeAttribute {
				continue
			}
			enabled, err := evaluateAttributeRule(userID, email, rule.Value)
			if err != nil {
				return false, err
			}
			if enabled {
				return true, nil
			}
		}
		return false, nil
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
