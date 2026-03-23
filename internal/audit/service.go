package audit

import "context"

// Service is a thin layer over the audit store for GraphQL resolvers.
// Domain services use Store directly to keep audit write close to business logic.
type Service struct {
	store Store
}

func NewService(store Store) *Service {
	return &Service{store: store}
}

func (s *Service) GetByID(ctx context.Context, id string) (*Entry, error) {
	return s.store.GetByID(ctx, id)
}

func (s *Service) List(ctx context.Context, filter ListFilter, limit, offset int) ([]*Entry, error) {
	return s.store.List(ctx, filter, limit, offset)
}
