package experiments

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/havlinj/featureflag-api/graph/model"
	"github.com/havlinj/featureflag-api/internal/audit"
	"github.com/havlinj/featureflag-api/internal/auth"
)

type minimalExperimentsTxAwareStore struct {
	gotTx       *sql.Tx
	variantCall int
}

func (s *minimalExperimentsTxAwareStore) CreateExperiment(ctx context.Context, exp *Experiment) (*Experiment, error) {
	return exp, nil
}
func (s *minimalExperimentsTxAwareStore) GetExperimentByKeyAndEnvironment(ctx context.Context, key, environment string) (*Experiment, error) {
	return nil, nil
}
func (s *minimalExperimentsTxAwareStore) GetExperimentByID(ctx context.Context, id string) (*Experiment, error) {
	return nil, nil
}
func (s *minimalExperimentsTxAwareStore) CreateVariant(ctx context.Context, v *Variant) (*Variant, error) {
	s.variantCall++
	return v, nil
}
func (s *minimalExperimentsTxAwareStore) GetVariantsByExperimentID(ctx context.Context, experimentID string) ([]*Variant, error) {
	return nil, nil
}
func (s *minimalExperimentsTxAwareStore) GetAssignment(ctx context.Context, userID, experimentID string) (*Assignment, error) {
	return nil, nil
}
func (s *minimalExperimentsTxAwareStore) UpsertAssignment(ctx context.Context, a *Assignment) error {
	return nil
}
func (s *minimalExperimentsTxAwareStore) WithTx(tx *sql.Tx) Store {
	s.gotTx = tx
	return s
}

type minimalExperimentsNonTxStore struct{}

func (s *minimalExperimentsNonTxStore) CreateExperiment(ctx context.Context, exp *Experiment) (*Experiment, error) {
	return exp, nil
}
func (s *minimalExperimentsNonTxStore) GetExperimentByKeyAndEnvironment(ctx context.Context, key, environment string) (*Experiment, error) {
	return nil, nil
}
func (s *minimalExperimentsNonTxStore) GetExperimentByID(ctx context.Context, id string) (*Experiment, error) {
	return nil, nil
}
func (s *minimalExperimentsNonTxStore) CreateVariant(ctx context.Context, v *Variant) (*Variant, error) {
	return v, nil
}
func (s *minimalExperimentsNonTxStore) GetVariantsByExperimentID(ctx context.Context, experimentID string) ([]*Variant, error) {
	return nil, nil
}
func (s *minimalExperimentsNonTxStore) GetAssignment(ctx context.Context, userID, experimentID string) (*Assignment, error) {
	return nil, nil
}
func (s *minimalExperimentsNonTxStore) UpsertAssignment(ctx context.Context, a *Assignment) error {
	return nil
}

type minimalExperimentsAuditStore struct {
	gotTx    *sql.Tx
	beginErr error
}

func (s *minimalExperimentsAuditStore) Create(ctx context.Context, entry *audit.Entry) error {
	return nil
}
func (s *minimalExperimentsAuditStore) GetByID(ctx context.Context, id string) (*audit.Entry, error) {
	return nil, nil
}
func (s *minimalExperimentsAuditStore) List(ctx context.Context, filter audit.ListFilter, limit, offset int) ([]*audit.Entry, error) {
	return nil, nil
}
func (s *minimalExperimentsAuditStore) BeginTx(ctx context.Context) (*sql.Tx, error) {
	if s.beginErr != nil {
		return nil, s.beginErr
	}
	return &sql.Tx{}, nil
}
func (s *minimalExperimentsAuditStore) WithTx(tx *sql.Tx) audit.Store {
	s.gotTx = tx
	return s
}

func TestPrepareAuditTx_Success_ConfiguresTxScopedStores(t *testing.T) {
	expStore := &minimalExperimentsTxAwareStore{}
	auditStore := &minimalExperimentsAuditStore{}
	svc := NewServiceWithAudit(expStore, auditStore)
	ctx := auth.WithActorID(context.Background(), "actor-1")

	out, err := svc.prepareAuditTx(ctx)

	if err != nil {
		t.Fatalf("prepareAuditTx: %v", err)
	}
	if out == nil || out.actorID != "actor-1" || out.tx == nil {
		t.Fatalf("unexpected audit tx context: %+v", out)
	}
	if expStore.gotTx == nil || auditStore.gotTx == nil {
		t.Fatal("expected both stores to receive tx via WithTx")
	}
}

func TestPrepareAuditTx_StoreNotTxAware_ReturnsError(t *testing.T) {
	svc := NewServiceWithAudit(&minimalExperimentsNonTxStore{}, &minimalExperimentsAuditStore{})

	out, err := svc.prepareAuditTx(context.Background())

	if err == nil {
		t.Fatal("expected error for non tx-aware store")
	}
	if out != nil {
		t.Fatalf("expected nil audit tx context, got %+v", out)
	}
}

func TestPrepareAuditTx_BeginTxError_Propagates(t *testing.T) {
	expected := errors.New("begin tx failed")
	expStore := &minimalExperimentsTxAwareStore{}
	auditStore := &minimalExperimentsAuditStore{beginErr: expected}
	svc := NewServiceWithAudit(expStore, auditStore)
	ctx := auth.WithActorID(context.Background(), "actor-1")

	out, err := svc.prepareAuditTx(ctx)

	if out != nil {
		t.Fatalf("expected nil context, got %+v", out)
	}
	if !errors.Is(err, expected) {
		t.Fatalf("expected begin error, got %v", err)
	}
}

func TestPersistVariantsWithStore_CreatesAllVariants(t *testing.T) {
	store := &minimalExperimentsTxAwareStore{}
	svc := NewService(store)
	inputs := []*model.ExperimentVariantInput{
		{Name: "A", Weight: 50},
		{Name: "B", Weight: 50},
	}

	err := svc.persistVariantsWithStore(context.Background(), store, "exp-1", inputs)

	if err != nil {
		t.Fatalf("persistVariantsWithStore: %v", err)
	}
	if store.variantCall != 2 {
		t.Fatalf("expected 2 CreateVariant calls, got %d", store.variantCall)
	}
}

func TestModelMappers_NilAndNonNil(t *testing.T) {
	if got := experimentToModel(nil); got != nil {
		t.Fatalf("expected nil for nil experiment, got %+v", got)
	}
	if got := variantToModel(nil); got != nil {
		t.Fatalf("expected nil for nil variant, got %+v", got)
	}

	exp := &Experiment{ID: "e1", Key: "ab", Environment: "dev"}
	v := &Variant{ID: "v1", ExperimentID: "e1", Name: "A", Weight: 50}

	mExp := experimentToModel(exp)
	mVar := variantToModel(v)
	if mExp == nil || mExp.ID != "e1" || mExp.Key != "ab" || mExp.Environment != "dev" {
		t.Fatalf("unexpected experiment model: %+v", mExp)
	}
	if mVar == nil || mVar.ID != "v1" || mVar.ExperimentID != "e1" || mVar.Name != "A" || mVar.Weight != 50 {
		t.Fatalf("unexpected variant model: %+v", mVar)
	}
}
