package users

import (
	"context"
	"fmt"

	"github.com/jan-havlin-dev/featureflag-api/graph/model"
)

// Service holds business logic for users. It depends on Store so it can be mocked in tests.
type Service struct {
	Store Store
}

// NewService returns a users service that uses the given store.
func NewService(store Store) *Service {
	return &Service{Store: store}
}

// CreateUser creates a new user. Returns ErrDuplicateEmail if email already exists.
func (s *Service) CreateUser(ctx context.Context, input model.CreateUserInput) (*model.User, error) {
	if err := s.ensureUniqueEmail(ctx, input.Email); err != nil {
		return nil, err
	}
	role, err := parseRole(input.Role)
	if err != nil {
		return nil, err
	}
	user := &User{Email: input.Email, Role: role}
	created, err := s.Store.Create(ctx, user)
	if err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}
	return userToModel(created), nil
}

// GetUser returns the user by ID, or nil if not found.
func (s *Service) GetUser(ctx context.Context, id string) (*model.User, error) {
	u, err := s.Store.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}
	if u == nil {
		return nil, nil
	}
	return userToModel(u), nil
}

// GetUserByEmail returns the user by email, or nil if not found.
func (s *Service) GetUserByEmail(ctx context.Context, email string) (*model.User, error) {
	u, err := s.Store.GetByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("get user by email: %w", err)
	}
	if u == nil {
		return nil, nil
	}
	return userToModel(u), nil
}

// UpdateUser updates an existing user. Returns ErrNotFound if user does not exist.
func (s *Service) UpdateUser(ctx context.Context, input model.UpdateUserInput) (*model.User, error) {
	u, err := s.Store.GetByID(ctx, input.ID)
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}
	if u == nil {
		return nil, ErrNotFound
	}
	if input.Email != nil {
		u.Email = *input.Email
	}
	if input.Role != nil {
		role, err := parseRole(*input.Role)
		if err != nil {
			return nil, err
		}
		u.Role = role
	}
	if err := s.Store.Update(ctx, u); err != nil {
		return nil, fmt.Errorf("update user: %w", err)
	}
	return userToModel(u), nil
}

// DeleteUser removes a user by ID. Returns ErrNotFound if user does not exist.
func (s *Service) DeleteUser(ctx context.Context, id string) (bool, error) {
	if err := s.Store.Delete(ctx, id); err != nil {
		if err == ErrNotFound {
			return false, nil
		}
		return false, fmt.Errorf("delete user: %w", err)
	}
	return true, nil
}

func (s *Service) ensureUniqueEmail(ctx context.Context, email string) error {
	existing, err := s.Store.GetByEmail(ctx, email)
	if err != nil {
		return fmt.Errorf("check existing email: %w", err)
	}
	if existing != nil {
		return ErrDuplicateEmail
	}
	return nil
}

func parseRole(s string) (Role, error) {
	switch s {
	case "admin":
		return RoleAdmin, nil
	case "developer":
		return RoleDeveloper, nil
	case "viewer":
		return RoleViewer, nil
	default:
		return "", fmt.Errorf("users: invalid role %q", s)
	}
}

func userToModel(u *User) *model.User {
	if u == nil {
		return nil
	}
	return &model.User{
		ID:        u.ID,
		Email:     u.Email,
		Role:      string(u.Role),
		CreatedAt: u.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}
