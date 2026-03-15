package flags

import (
	"context"
	"fmt"
	"hash/fnv"
	"strconv"

	"github.com/havlinj/featureflag-api/graph/model"
)

const defaultEnvironment = DeploymentStageDev

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
	env := DeploymentStage(input.Environment)
	if err := s.ensureUniqueFlag(ctx, input.Key, env); err != nil {
		return nil, err
	}
	strategy, err := resolveStrategyForCreate(input)
	if err != nil {
		return nil, err
	}
	flag := &Flag{
		Key:             input.Key,
		Description:     input.Description,
		Enabled:         false,
		Environment:     env,
		RolloutStrategy: strategy,
	}
	created, err := s.Store.Create(ctx, flag)
	if err != nil {
		return nil, fmt.Errorf("create flag: %w", err)
	}
	if err := persistRulesForNewFlag(ctx, s.Store, created.ID, input.Rules); err != nil {
		return nil, err
	}
	return flagToModel(created), nil
}

func resolveStrategyForCreate(input model.CreateFlagInput) (RolloutStrategy, error) {
	strategy := rolloutStrategyFromModel(input.RolloutStrategy)
	if len(input.Rules) == 0 {
		return strategy, nil
	}
	ruleType, err := validateRulesSameType(input.Rules)
	if err != nil {
		return "", err
	}
	if strategy != RolloutStrategyNone && strategy != ruleTypeToStrategy(ruleType) {
		return "", fmt.Errorf("%w: all rules must use the same strategy: percentage or attribute", ErrRulesStrategyMismatch)
	}
	return ruleTypeToStrategy(ruleType), nil
}

func persistRulesForNewFlag(ctx context.Context, store Store, flagID string, rules []*model.RuleInput) error {
	if len(rules) == 0 {
		return nil
	}
	if err := store.ReplaceRulesByFlagID(ctx, flagID, ruleInputsToRules(flagID, rules)); err != nil {
		return fmt.Errorf("create flag rules: %w", err)
	}
	return nil
}

// UpdateFlag updates an existing feature flag. If Rules is present, replaces all rules and updates strategy.
func (s *Service) UpdateFlag(ctx context.Context, input model.UpdateFlagInput) (*model.FeatureFlag, error) {
	flag, err := s.getFlagOrErr(ctx, input.Key, defaultEnvironment)
	if err != nil {
		return nil, err
	}
	flag.Enabled = input.Enabled
	if err := s.applyRulesUpdate(ctx, flag, input.Rules); err != nil {
		return nil, err
	}
	if err := s.Store.Update(ctx, flag); err != nil {
		return nil, fmt.Errorf("update flag: %w", err)
	}
	return flagToModel(flag), nil
}

func (s *Service) getFlagOrErr(ctx context.Context, key string, env DeploymentStage) (*Flag, error) {
	flag, err := s.Store.GetByKeyAndEnvironment(ctx, key, env)
	if err != nil {
		return nil, fmt.Errorf("get flag: %w", err)
	}
	if flag == nil {
		return nil, fmt.Errorf("flags: flag not found key=%q environment=%q: %w", key, env, ErrNotFound)
	}
	return flag, nil
}

func (s *Service) applyRulesUpdate(ctx context.Context, flag *Flag, rules []*model.RuleInput) error {
	if rules == nil {
		return nil
	}
	if len(rules) == 0 {
		flag.RolloutStrategy = RolloutStrategyNone
		if err := s.Store.ReplaceRulesByFlagID(ctx, flag.ID, nil); err != nil {
			return fmt.Errorf("update flag rules: %w", err)
		}
		return nil
	}
	ruleType, err := validateRulesSameType(rules)
	if err != nil {
		return err
	}
	if flag.RolloutStrategy != RolloutStrategyNone && flag.RolloutStrategy != ruleTypeToStrategy(ruleType) {
		return strategyMismatchError(flag.RolloutStrategy)
	}
	flag.RolloutStrategy = ruleTypeToStrategy(ruleType)
	if err := s.Store.ReplaceRulesByFlagID(ctx, flag.ID, ruleInputsToRules(flag.ID, rules)); err != nil {
		return fmt.Errorf("update flag rules: %w", err)
	}
	return nil
}

// EvaluateFlag returns whether the flag is enabled for the given evaluation context.
func (s *Service) EvaluateFlag(ctx context.Context, key string, evalCtx model.EvaluationContextInput) (bool, error) {
	if evalCtx.UserID == "" {
		return false, fmt.Errorf("flags: invalid user ID (empty): %w", ErrInvalidUserID)
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

// DeleteFlag removes a flag by key and deployment stage. Rules are removed by DB CASCADE.
func (s *Service) DeleteFlag(ctx context.Context, key string, env DeploymentStage) (bool, error) {
	flag, err := s.getFlagOrErr(ctx, key, env)
	if err != nil {
		return false, err
	}
	if err := s.Store.Delete(ctx, flag.ID); err != nil {
		return false, fmt.Errorf("delete flag: %w", err)
	}
	return true, nil
}

func (s *Service) ensureUniqueFlag(ctx context.Context, key string, env DeploymentStage) error {
	existing, err := s.Store.GetByKeyAndEnvironment(ctx, key, env)
	if err != nil {
		return fmt.Errorf("check existing flag: %w", err)
	}
	if existing != nil {
		return fmt.Errorf("flags: duplicate key=%q environment=%q: %w", key, env, ErrDuplicateKey)
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
		Environment:     string(flag.Environment),
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
	ruleTypes := make([]string, 0, len(rules))
	for _, r := range rules {
		ruleTypes = append(ruleTypes, string(ruleInputToRuleType(r)))
	}
	for _, r := range rules[1:] {
		if ruleInputToRuleType(r) != first {
			return "", fmt.Errorf("flags: rules do not match (mixed types %v): %w", ruleTypes, ErrRulesStrategyMismatch)
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
	return fmt.Errorf("flags: strategy mismatch (current=%q): %s: %w", current, msg, ErrRulesStrategyMismatch)
}

func evaluateRulesByStrategy(strategy RolloutStrategy, userID string, email *string, rules []*Rule) (bool, error) {
	if strategy == RolloutStrategyPercentage {
		return evaluatePercentageRules(userID, rules)
	}
	if strategy == RolloutStrategyAttribute {
		return evaluateAttributeRules(userID, email, rules)
	}
	return true, nil
}

func evaluatePercentageRules(userID string, rules []*Rule) (bool, error) {
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

func evaluateAttributeRules(userID string, email *string, rules []*Rule) (bool, error) {
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

func evaluatePercentageRule(userID, value string) (bool, error) {
	percentage, err := strconv.Atoi(value)
	if err != nil {
		return false, fmt.Errorf("flags: invalid percentage rule value=%q (not a number): %w", value, ErrInvalidRule)
	}
	if percentage < 0 || percentage > 100 {
		return false, fmt.Errorf("flags: invalid percentage rule value=%q (must be 0-100): %w", value, ErrInvalidRule)
	}

	bucket := hashToBucket(userID)
	return bucket < percentage, nil
}

func hashToBucket(userID string) int {
	h := fnv.New32a()
	_, _ = h.Write([]byte(userID))
	return int(h.Sum32() % 100)
}
