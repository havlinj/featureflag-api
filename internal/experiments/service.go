package experiments

import (
	"context"
	"fmt"
	"hash/fnv"

	"github.com/havlinj/featureflag-api/graph/model"
)

const weightSum = 100

// Service holds core business logic for experiments. It depends on Store for
// persistence so that the latter can be mocked in unit tests.
type Service struct {
	Store Store
}

// NewService returns an experiments service that uses the given store.
func NewService(store Store) *Service {
	return &Service{Store: store}
}

// CreateExperiment creates a new experiment with the given variants.
// Variant weights must sum to 100; otherwise returns *InvalidWeightsError.
func (s *Service) CreateExperiment(ctx context.Context, input model.CreateExperimentInput) (*model.Experiment, error) {
	if err := validateVariantWeights(input.Variants); err != nil {
		return nil, err
	}
	if err := s.ensureUniqueExperiment(ctx, input.Key, input.Environment); err != nil {
		return nil, err
	}
	exp := &Experiment{Key: input.Key, Environment: input.Environment}
	created, err := s.Store.CreateExperiment(ctx, exp)
	if err != nil {
		return nil, fmt.Errorf("create experiment: %w", err)
	}
	if err := s.persistVariants(ctx, created.ID, input.Variants); err != nil {
		return nil, err
	}
	return experimentToModel(created), nil
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
	existing, err := s.Store.GetExperimentByKeyAndEnvironment(ctx, key, environment)
	if err != nil {
		return fmt.Errorf("check existing experiment: %w", err)
	}
	if existing != nil {
		return &DuplicateExperimentError{Key: key, Environment: environment}
	}
	return nil
}

func (s *Service) persistVariants(ctx context.Context, experimentID string, inputs []*model.ExperimentVariantInput) error {
	for _, in := range inputs {
		v := &Variant{ExperimentID: experimentID, Name: in.Name, Weight: in.Weight}
		_, err := s.Store.CreateVariant(ctx, v)
		if err != nil {
			return fmt.Errorf("create variant %q: %w", in.Name, err)
		}
	}
	return nil
}

// GetExperiment returns the experiment for the given key and environment, or *ExperimentNotFoundError.
func (s *Service) GetExperiment(ctx context.Context, key, environment string) (*model.Experiment, error) {
	exp, err := s.Store.GetExperimentByKeyAndEnvironment(ctx, key, environment)
	if err != nil {
		return nil, fmt.Errorf("get experiment: %w", err)
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
	variants, err := s.Store.GetVariantsByExperimentID(ctx, exp.ID)
	if err != nil {
		return nil, fmt.Errorf("get variants: %w", err)
	}
	if len(variants) == 0 {
		return nil, &VariantNotFoundError{ExperimentKey: experimentKey, Environment: environment}
	}
	existing, err := s.Store.GetAssignment(ctx, userID, exp.ID)
	if err != nil {
		return nil, fmt.Errorf("get assignment: %w", err)
	}
	if existing != nil {
		return s.variantByID(ctx, existing.VariantID, variants)
	}
	assigned := assignVariantByWeight(userID, exp.ID, variants)
	if err := s.Store.UpsertAssignment(ctx, &Assignment{UserID: userID, ExperimentID: exp.ID, VariantID: assigned.ID}); err != nil {
		return nil, fmt.Errorf("upsert assignment: %w", err)
	}
	return variantToModel(assigned), nil
}

func (s *Service) getExperimentOrErr(ctx context.Context, key, environment string) (*Experiment, error) {
	exp, err := s.Store.GetExperimentByKeyAndEnvironment(ctx, key, environment)
	if err != nil {
		return nil, fmt.Errorf("get experiment: %w", err)
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
