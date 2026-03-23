package flags

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"hash/fnv"
	"strconv"

	"github.com/havlinj/featureflag-api/graph/model"
	"github.com/havlinj/featureflag-api/internal/audit"
)

const defaultEnvironment = DeploymentStageDev

// Service holds core business logic for feature flags. It depends on Store for
// persistence so that the latter can be mocked in unit tests.
type Service struct {
	store Store
	audit audit.Store
}

type auditTxContext struct {
	actorID string
	tx      *sql.Tx
	store   Store
	audit   audit.Store
}

// NewService returns a flags service that uses the given store.
func NewService(store Store) *Service {
	return &Service{store: store}
}

// NewServiceWithAudit returns a flags service that writes audit logs for critical mutations.
func NewServiceWithAudit(store Store, auditStore audit.Store) *Service {
	return &Service{store: store, audit: auditStore}
}

// CreateFlag creates a new feature flag. Optional rules set rollout strategy; all rules must be same type.
func (s *Service) CreateFlag(ctx context.Context, input model.CreateFlagInput) (*model.FeatureFlag, error) {
	if s.audit == nil {
		created, err := s.createFlagWithStore(ctx, s.store, input)
		if err != nil {
			return nil, err
		}
		return flagToModel(created), nil
	}

	auditCtx, err := s.prepareAuditTx(ctx)
	if err != nil {
		return nil, err
	}
	committed := false
	defer func() {
		if !committed {
			_ = auditCtx.tx.Rollback()
		}
	}()

	created, err := s.createFlagWithStore(ctx, auditCtx.store, input)
	if err != nil {
		return nil, err
	}
	if err := auditCtx.audit.Create(ctx, &audit.Entry{
		Entity:   audit.EntityFeatureFlag,
		EntityID: created.ID,
		Action:   audit.ActionCreate,
		ActorID:  auditCtx.actorID,
	}); err != nil {
		return nil, fmt.Errorf("audit create entry: %w", err)
	}
	if err := auditCtx.tx.Commit(); err != nil {
		return nil, err
	}
	committed = true
	return flagToModel(created), nil
}

func (s *Service) createFlagWithStore(ctx context.Context, store Store, input model.CreateFlagInput) (*Flag, error) {
	env := DeploymentStage(input.Environment)
	if err := s.ensureUniqueFlagWithStore(ctx, store, input.Key, env); err != nil {
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
	created, err := store.Create(ctx, flag)
	if err != nil {
		return nil, &OperationError{Op: opServiceCreateFlagStoreCreate, Key: input.Key, Environment: string(env), Cause: err}
	}
	if err := persistRulesForNewFlag(ctx, store, created.ID, input.Rules); err != nil {
		return nil, err
	}
	return created, nil
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
		return "", &RulesStrategyMismatchError{Message: "all rules must use the same strategy: percentage or attribute"}
	}
	return ruleTypeToStrategy(ruleType), nil
}

func persistRulesForNewFlag(ctx context.Context, store Store, flagID string, rules []*model.RuleInput) error {
	if len(rules) == 0 {
		return nil
	}
	if err := store.ReplaceRulesByFlagID(ctx, flagID, ruleInputsToRules(flagID, rules)); err != nil {
		return &OperationError{Op: opServiceCreateFlagReplaceRulesByFlagID, FlagID: flagID, Cause: err}
	}
	return nil
}

// UpdateFlag updates an existing feature flag. If Rules is present, replaces all rules and updates strategy.
func (s *Service) UpdateFlag(ctx context.Context, input model.UpdateFlagInput) (*model.FeatureFlag, error) {
	if s.audit == nil {
		updated, err := s.updateFlagWithStore(ctx, s.store, input)
		if err != nil {
			return nil, err
		}
		return flagToModel(updated), nil
	}

	auditCtx, err := s.prepareAuditTx(ctx)
	if err != nil {
		return nil, err
	}
	committed := false
	defer func() {
		if !committed {
			_ = auditCtx.tx.Rollback()
		}
	}()

	updated, err := s.updateFlagWithStore(ctx, auditCtx.store, input)
	if err != nil {
		return nil, err
	}
	if err := auditCtx.audit.Create(ctx, &audit.Entry{
		Entity:   audit.EntityFeatureFlag,
		EntityID: updated.ID,
		Action:   audit.ActionUpdate,
		ActorID:  auditCtx.actorID,
	}); err != nil {
		return nil, fmt.Errorf("audit create entry: %w", err)
	}
	if err := auditCtx.tx.Commit(); err != nil {
		return nil, err
	}
	committed = true
	return flagToModel(updated), nil
}

func (s *Service) updateFlagWithStore(ctx context.Context, store Store, input model.UpdateFlagInput) (*Flag, error) {
	env := defaultEnvironment
	if input.Environment != nil && *input.Environment != "" {
		env = DeploymentStage(*input.Environment)
	}

	flag, err := s.getFlagOrErrWithStore(ctx, store, input.Key, env)
	if err != nil {
		return nil, err
	}
	flag.Enabled = input.Enabled
	if err := s.applyRulesUpdateWithStore(ctx, store, flag, input.Rules); err != nil {
		return nil, err
	}
	if err := store.Update(ctx, flag); err != nil {
		return nil, &OperationError{Op: opServiceUpdateFlagStoreUpdate, Key: flag.Key, Environment: string(flag.Environment), FlagID: flag.ID, Cause: err}
	}
	return flag, nil
}

func (s *Service) getFlagOrErrWithStore(ctx context.Context, store Store, key string, env DeploymentStage) (*Flag, error) {
	flag, err := store.GetByKeyAndEnvironment(ctx, key, env)
	if err != nil {
		return nil, &OperationError{Op: opServiceGetFlagOrErrStoreGetByKeyAndEnvironment, Key: key, Environment: string(env), Cause: err}
	}
	if flag == nil {
		return nil, &NotFoundError{Key: key, Environment: string(env)}
	}
	return flag, nil
}

func (s *Service) applyRulesUpdateWithStore(ctx context.Context, store Store, flag *Flag, rules []*model.RuleInput) error {
	if rules == nil {
		return nil
	}
	if len(rules) == 0 {
		flag.RolloutStrategy = RolloutStrategyNone
		if err := store.ReplaceRulesByFlagID(ctx, flag.ID, nil); err != nil {
			return &OperationError{Op: opServiceUpdateFlagReplaceRulesByFlagIDClear, FlagID: flag.ID, Key: flag.Key, Environment: string(flag.Environment), Cause: err}
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
	if err := store.ReplaceRulesByFlagID(ctx, flag.ID, ruleInputsToRules(flag.ID, rules)); err != nil {
		return &OperationError{Op: opServiceUpdateFlagReplaceRulesByFlagID, FlagID: flag.ID, Key: flag.Key, Environment: string(flag.Environment), Cause: err}
	}
	return nil
}

// EvaluateFlag returns whether the flag is enabled for the given evaluation context.
func (s *Service) EvaluateFlag(ctx context.Context, key string, evalCtx model.EvaluationContextInput) (bool, error) {
	return s.EvaluateFlagInEnvironment(ctx, key, defaultEnvironment, evalCtx)
}

// EvaluateFlagInEnvironment returns whether the flag is enabled for the given evaluation context in the target environment.
func (s *Service) EvaluateFlagInEnvironment(ctx context.Context, key string, env DeploymentStage, evalCtx model.EvaluationContextInput) (bool, error) {
	if evalCtx.UserID == "" {
		return false, &InvalidUserIDError{UserID: evalCtx.UserID}
	}

	flag, err := s.store.GetByKeyAndEnvironment(ctx, key, env)
	if err != nil {
		return false, &OperationError{Op: opServiceEvaluateFlagStoreGetByKeyAndEnvironment, Key: key, Environment: string(env), Cause: err}
	}
	if flag == nil || !flag.Enabled {
		return false, nil
	}

	rules, err := s.store.GetRulesByFlagID(ctx, flag.ID)
	if err != nil {
		return false, &OperationError{Op: opServiceEvaluateFlagStoreGetRulesByFlagID, FlagID: flag.ID, Key: key, Environment: string(env), Cause: err}
	}
	if len(rules) == 0 {
		return true, nil
	}

	return evaluateRulesByStrategy(flag.RolloutStrategy, evalCtx.UserID, evalCtx.Email, rules)
}

// DeleteFlag removes a flag by key and deployment stage. Rules are removed by DB CASCADE.
func (s *Service) DeleteFlag(ctx context.Context, key string, env DeploymentStage) (bool, error) {
	if s.audit == nil {
		ok, _, err := s.deleteFlagWithStoreAndID(ctx, s.store, key, env)
		return ok, err
	}

	auditCtx, err := s.prepareAuditTx(ctx)
	if err != nil {
		return false, err
	}
	committed := false
	defer func() {
		if !committed {
			_ = auditCtx.tx.Rollback()
		}
	}()

	deleted, entityID, err := s.deleteFlagWithStoreAndID(ctx, auditCtx.store, key, env)
	if err != nil {
		return false, err
	}
	if err := auditCtx.audit.Create(ctx, &audit.Entry{
		Entity:   audit.EntityFeatureFlag,
		EntityID: entityID,
		Action:   audit.ActionDelete,
		ActorID:  auditCtx.actorID,
	}); err != nil {
		return false, fmt.Errorf("audit create entry: %w", err)
	}
	if err := auditCtx.tx.Commit(); err != nil {
		return false, err
	}
	committed = true
	return deleted, nil
}

func (s *Service) deleteFlagWithStoreAndID(ctx context.Context, store Store, key string, env DeploymentStage) (bool, string, error) {
	flag, err := s.getFlagOrErrWithStore(ctx, store, key, env)
	if err != nil {
		return false, "", err
	}
	if err := store.Delete(ctx, flag.ID); err != nil {
		return false, "", &OperationError{Op: opServiceDeleteFlagStoreDelete, FlagID: flag.ID, Key: key, Environment: string(env), Cause: err}
	}
	return true, flag.ID, nil
}

func (s *Service) prepareAuditTx(ctx context.Context) (*auditTxContext, error) {
	storeTxAware, ok := s.store.(TxAwareStore)
	if !ok {
		return nil, errors.New("audit: flags store is not tx-aware")
	}

	actorID, tx, auditTxStore, err := audit.PrepareWriteTx(ctx, s.audit)
	if err != nil {
		return nil, err
	}
	return &auditTxContext{
		actorID: actorID,
		tx:      tx,
		store:   storeTxAware.WithTx(tx),
		audit:   auditTxStore,
	}, nil
}

func (s *Service) ensureUniqueFlagWithStore(ctx context.Context, store Store, key string, env DeploymentStage) error {
	existing, err := store.GetByKeyAndEnvironment(ctx, key, env)
	if err != nil {
		return &OperationError{Op: opServiceEnsureUniqueFlagStoreGetByKeyAndEnv, Key: key, Environment: string(env), Cause: err}
	}
	if existing != nil {
		return &DuplicateKeyError{Key: key, Environment: string(env)}
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
			return "", &RulesStrategyMismatchError{RuleTypes: ruleTypes}
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
	return &RulesStrategyMismatchError{CurrentStrategy: string(current), Message: msg}
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
		return false, &InvalidRuleError{Value: value, Reason: "not a number"}
	}
	if percentage < 0 || percentage > 100 {
		return false, &InvalidRuleError{Value: value, Reason: "must be 0-100"}
	}

	bucket := hashToBucket(userID)
	return bucket < percentage, nil
}

func hashToBucket(userID string) int {
	h := fnv.New32a()
	_, _ = h.Write([]byte(userID))
	return int(h.Sum32() % 100)
}
