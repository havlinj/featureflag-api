package users

import (
	"context"
	"errors"
	"fmt"

	"github.com/havlinj/featureflag-api/graph/model"
	"github.com/havlinj/featureflag-api/internal/auth"
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
	user := &User{Email: input.Email, Role: roleFromModel(input.Role)}
	if err := setPasswordIfProvided(user, input.Password); err != nil {
		return nil, err
	}
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

// Login verifies email and password and returns the user ID and role for token issuance.
// Returns ErrNotFound if no user, ErrInvalidCredentials if password does not match.
func (s *Service) Login(ctx context.Context, email, password string) (userID, role string, err error) {
	u, err := s.Store.GetByEmail(ctx, email)
	if err != nil {
		return "", "", fmt.Errorf("login: %w", err)
	}
	if u == nil {
		return "", "", &NotFoundError{Email: email}
	}
	if u.PasswordHash == nil || !auth.PasswordMatches(*u.PasswordHash, password) {
		return "", "", &InvalidCredentialsError{Email: email}
	}
	return u.ID, string(u.Role), nil
}

// UpdateUser updates an existing user. Returns ErrNotFound if user does not exist.
func (s *Service) UpdateUser(ctx context.Context, input model.UpdateUserInput) (*model.User, error) {
	u, err := s.Store.GetByID(ctx, input.ID)
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}
	if u == nil {
		return nil, &NotFoundError{ID: input.ID}
	}
	if err := applyUpdateFieldsToUser(u, input); err != nil {
		return nil, err
	}
	if err := s.Store.Update(ctx, u); err != nil {
		return nil, fmt.Errorf("update user: %w", err)
	}
	return userToModel(u), nil
}

func applyUpdateFieldsToUser(u *User, input model.UpdateUserInput) error {
	if input.Email != nil {
		u.Email = *input.Email
	}
	if input.Role != nil {
		u.Role = roleFromModel(*input.Role)
	}
	if err := setPasswordIfProvided(u, input.Password); err != nil {
		return err
	}
	return nil
}

// setPasswordIfProvided hashes password and sets u.PasswordHash when password is non-nil and non-empty.
func setPasswordIfProvided(u *User, password *string) error {
	if password == nil || *password == "" {
		return nil
	}
	hash, err := auth.HashPassword(*password)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}
	u.PasswordHash = &hash
	return nil
}

// DeleteUser removes a user by ID. Returns ErrNotFound if user does not exist.
func (s *Service) DeleteUser(ctx context.Context, id string) (bool, error) {
	if err := s.Store.Delete(ctx, id); err != nil {
		var e *NotFoundError
		if errors.As(err, &e) {
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
		return &DuplicateEmailError{Email: email}
	}
	return nil
}

// roleFromModel maps GraphQL enum to domain Role. Values match (admin, developer, viewer).
func roleFromModel(m model.Role) Role {
	return Role(m)
}

// roleToModel maps domain Role to GraphQL enum for API responses.
func roleToModel(r Role) model.Role {
	return model.Role(r)
}

func userToModel(u *User) *model.User {
	if u == nil {
		return nil
	}
	return &model.User{
		ID:        u.ID,
		Email:     u.Email,
		Role:      roleToModel(u.Role),
		CreatedAt: u.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}
