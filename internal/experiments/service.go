package experiments

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"hash/fnv"

	"github.com/havlinj/featureflag-api/graph/model"
	"github.com/havlinj/featureflag-api/internal/audit"
)

const weightSum = 100

// Service holds core business logic for experiments. It depends on Store for
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

// NewService returns an experiments service that uses the given store.
func NewService(store Store) *Service {
	return &Service{store: store}
}

// NewServiceWithAudit returns an experiments service that writes audit logs for critical mutations.
func NewServiceWithAudit(store Store, auditStore audit.Store) *Service {
	return &Service{store: store, audit: auditStore}
}

// CreateExperiment creates a new experiment with the given variants.
// Variant weights must sum to 100; otherwise returns *InvalidWeightsError.
func (s *Service) CreateExperiment(ctx context.Context, input model.CreateExperimentInput) (*model.Experiment, error) {
	if s.audit != nil {
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

		created, err := s.createExperimentWithStore(ctx, auditCtx.store, input)
		if err != nil {
			return nil, err
		}
		if err := auditCtx.audit.Create(ctx, &audit.Entry{
			Entity:   audit.EntityExperiment,
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
		return experimentToModel(created), nil
	}

	storeTxAware, canUseTx := s.store.(TxAwareStore)
	storeTxStarter, canStartTx := s.store.(TxStarter)
	if canUseTx && canStartTx {
		tx, err := storeTxStarter.BeginTx(ctx)
		if err != nil {
			return nil, err
		}
		committed := false
		defer func() {
			if !committed {
				_ = tx.Rollback()
			}
		}()

		created, err := s.createExperimentWithStore(ctx, storeTxAware.WithTx(tx), input)
		if err != nil {
			return nil, err
		}
		if err := tx.Commit(); err != nil {
			return nil, err
		}
		committed = true
		return experimentToModel(created), nil
	}

	created, err := s.createExperimentWithStore(ctx, s.store, input)
	if err != nil {
		return nil, err
	}
	return experimentToModel(created), nil
}

func (s *Service) createExperimentWithStore(ctx context.Context, store Store, input model.CreateExperimentInput) (*Experiment, error) {
	if err := validateVariantWeights(input.Variants); err != nil {
		return nil, err
	}
	if err := s.ensureUniqueExperimentWithStore(ctx, store, input.Key, input.Environment); err != nil {
		return nil, err
	}
	exp := &Experiment{Key: input.Key, Environment: input.Environment}
	created, err := store.CreateExperiment(ctx, exp)
	if err != nil {
		return nil, &OperationError{Op: opServiceCreateExperimentStoreCreateExperiment, Key: input.Key, Environment: input.Environment, Cause: err}
	}
	if err := s.persistVariantsWithStore(ctx, store, created.ID, input.Variants); err != nil {
		return nil, err
	}
	return created, nil
}

func validateVariantWeights(variants []*model.ExperimentVariantInput) error {
	if len(variants) == 0 {
		return &InvalidWeightsError{Sum: 0, Weights: nil, Reason: "no variants provided"}
	}
	sum := 0
	weights := make([]int, 0, len(variants))
	for _, v := range variants {
		if v.Weight < 0 {
			weights = append(weights, v.Weight)
			return &InvalidWeightsError{Sum: sum, Weights: weights, Reason: "negative weight"}
		}
		sum += v.Weight
		weights = append(weights, v.Weight)
	}
	if sum != weightSum {
		return &InvalidWeightsError{Sum: sum, Weights: weights}
	}
	return nil
}

func (s *Service) ensureUniqueExperiment(ctx context.Context, key, environment string) error {
	return s.ensureUniqueExperimentWithStore(ctx, s.store, key, environment)
}

func (s *Service) prepareAuditTx(ctx context.Context) (*auditTxContext, error) {
	storeTxAware, ok := s.store.(TxAwareStore)
	if !ok {
		return nil, errors.New("audit: experiments store is not tx-aware")
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

func (s *Service) ensureUniqueExperimentWithStore(ctx context.Context, store Store, key, environment string) error {
	existing, err := store.GetExperimentByKeyAndEnvironment(ctx, key, environment)
	if err != nil {
		return &OperationError{Op: opServiceEnsureUniqueExperimentStoreGetByKeyAndEnvironment, Key: key, Environment: environment, Cause: err}
	}
	if existing != nil {
		return &DuplicateExperimentError{Key: key, Environment: environment}
	}
	return nil
}

func (s *Service) persistVariants(ctx context.Context, experimentID string, inputs []*model.ExperimentVariantInput) error {
	return s.persistVariantsWithStore(ctx, s.store, experimentID, inputs)
}

func (s *Service) persistVariantsWithStore(ctx context.Context, store Store, experimentID string, inputs []*model.ExperimentVariantInput) error {
	for _, in := range inputs {
		v := &Variant{ExperimentID: experimentID, Name: in.Name, Weight: in.Weight}
		_, err := store.CreateVariant(ctx, v)
		if err != nil {
			return &OperationError{
				Op:            opServiceCreateExperimentStoreCreateVariant,
				ExperimentID:  experimentID,
				VariantName:   in.Name,
				VariantWeight: in.Weight,
				Cause:         err,
			}
		}
	}
	return nil
}

// GetExperiment returns the experiment for the given key and environment, or *ExperimentNotFoundError.
func (s *Service) GetExperiment(ctx context.Context, key, environment string) (*model.Experiment, error) {
	exp, err := s.store.GetExperimentByKeyAndEnvironment(ctx, key, environment)
	if err != nil {
		return nil, &OperationError{Op: opServiceGetExperimentStoreGetByKeyAndEnvironment, Key: key, Environment: environment, Cause: err}
	}
	if exp == nil {
		return nil, &ExperimentNotFoundError{Key: key, Environment: environment}
	}
	return experimentToModel(exp), nil
}

// GetAssignment returns the variant assigned to the user for the experiment (deterministic).
// If the user has no assignment yet, one is computed from hash(userID+experimentID) and persisted.
func (s *Service) GetAssignment(ctx context.Context, userID, experimentKey, environment string) (*model.ExperimentVariant, error) {
	if userID == "" {
		return nil, &InvalidUserIDError{UserID: userID}
	}
	exp, err := s.getExperimentOrErr(ctx, experimentKey, environment)
	if err != nil {
		return nil, err
	}
	variants, err := s.store.GetVariantsByExperimentID(ctx, exp.ID)
	if err != nil {
		return nil, &OperationError{Op: opServiceGetAssignmentStoreGetVariantsByExperimentID, Key: experimentKey, Environment: environment, ExperimentID: exp.ID, UserID: userID, Cause: err}
	}
	if len(variants) == 0 {
		return nil, &VariantNotFoundError{ExperimentKey: experimentKey, Environment: environment}
	}
	existing, err := s.store.GetAssignment(ctx, userID, exp.ID)
	if err != nil {
		return nil, &OperationError{Op: opServiceGetAssignmentStoreGetAssignment, Key: experimentKey, Environment: environment, ExperimentID: exp.ID, UserID: userID, Cause: err}
	}
	if existing != nil {
		return s.variantByID(ctx, existing.VariantID, variants)
	}
	assigned := assignVariantByWeight(userID, exp.ID, variants)
	if err := s.store.UpsertAssignment(ctx, &Assignment{UserID: userID, ExperimentID: exp.ID, VariantID: assigned.ID}); err != nil {
		return nil, &OperationError{
			Op:           opServiceGetAssignmentStoreUpsertAssignment,
			Key:          experimentKey,
			Environment:  environment,
			ExperimentID: exp.ID,
			UserID:       userID,
			VariantID:    assigned.ID,
			Cause:        err,
		}
	}
	return variantToModel(assigned), nil
}

func (s *Service) getExperimentOrErr(ctx context.Context, key, environment string) (*Experiment, error) {
	exp, err := s.store.GetExperimentByKeyAndEnvironment(ctx, key, environment)
	if err != nil {
		return nil, &OperationError{Op: opServiceGetExperimentOrErrStoreGetByKeyAndEnvironment, Key: key, Environment: environment, Cause: err}
	}
	if exp == nil {
		return nil, &ExperimentNotFoundError{Key: key, Environment: environment}
	}
	return exp, nil
}

func (s *Service) variantByID(ctx context.Context, variantID string, variants []*Variant) (*model.ExperimentVariant, error) {
	for _, v := range variants {
		if v.ID == variantID {
			return variantToModel(v), nil
		}
	}
	return nil, &VariantNotFoundError{VariantID: variantID}
}

// assignVariantByWeight picks a variant deterministically from hash(userID+experimentID).
// Weights must sum to 100; bucket is 0..99, mapped to cumulative weight ranges.
func assignVariantByWeight(userID, experimentID string, variants []*Variant) *Variant {
	bucket := hashToBucket(userID + experimentID)
	cum := 0
	for _, v := range variants {
		cum += v.Weight
		if bucket < cum {
			return v
		}
	}
	return variants[len(variants)-1]
}

func hashToBucket(seed string) int {
	h := fnv.New32a()
	_, _ = h.Write([]byte(seed))
	return int(h.Sum32() % 100)
}

func experimentToModel(exp *Experiment) *model.Experiment {
	if exp == nil {
		return nil
	}
	return &model.Experiment{
		ID:          exp.ID,
		Key:         exp.Key,
		Environment: exp.Environment,
	}
}

func variantToModel(v *Variant) *model.ExperimentVariant {
	if v == nil {
		return nil
	}
	return &model.ExperimentVariant{
		ID:           v.ID,
		ExperimentID: v.ExperimentID,
		Name:         v.Name,
		Weight:       v.Weight,
	}
}
