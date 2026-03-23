package users

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/havlinj/featureflag-api/graph/model"
	"github.com/havlinj/featureflag-api/internal/audit"
	"github.com/havlinj/featureflag-api/internal/auth"
)

// Service holds business logic for users. It depends on Store so it can be mocked in tests.
type Service struct {
	Store Store
	Audit audit.Store
}

type auditTxContext struct {
	actorID string
	tx      *sql.Tx
	store   Store
	audit   audit.Store
}

// NewService returns a users service that uses the given store.
func NewService(store Store) *Service {
	return &Service{Store: store}
}

// NewServiceWithAudit returns a users service that writes audit logs for critical mutations.
func NewServiceWithAudit(store Store, auditStore audit.Store) *Service {
	return &Service{Store: store, Audit: auditStore}
}

// CreateUser creates a new user. Returns ErrDuplicateEmail if email already exists.
func (s *Service) CreateUser(ctx context.Context, input model.CreateUserInput) (*model.User, error) {
	if s.Audit == nil {
		created, err := s.createUserWithStore(ctx, s.Store, input)
		if err != nil {
			return nil, err
		}
		return userToModel(created), nil
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

	created, err := s.createUserWithStore(ctx, auditCtx.store, input)
	if err != nil {
		return nil, err
	}
	if err := auditCtx.audit.Create(ctx, &audit.Entry{
		Entity:   audit.EntityUser,
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

	return userToModel(created), nil
}

func (s *Service) createUserWithStore(ctx context.Context, store Store, input model.CreateUserInput) (*User, error) {
	if err := s.ensureUniqueEmailWithStore(ctx, store, input.Email); err != nil {
		return nil, err
	}
	user := &User{Email: input.Email, Role: roleFromModel(input.Role)}
	if err := setPasswordIfProvided(user, input.Password); err != nil {
		return nil, err
	}
	created, err := store.Create(ctx, user)
	if err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}
	return created, nil
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
	if s.Audit == nil {
		updated, err := s.updateUserWithStore(ctx, s.Store, input)
		if err != nil {
			return nil, err
		}
		return userToModel(updated), nil
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

	updated, err := s.updateUserWithStore(ctx, auditCtx.store, input)
	if err != nil {
		return nil, err
	}
	if err := auditCtx.audit.Create(ctx, &audit.Entry{
		Entity:   audit.EntityUser,
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

	return userToModel(updated), nil
}

func (s *Service) updateUserWithStore(ctx context.Context, store Store, input model.UpdateUserInput) (*User, error) {
	u, err := store.GetByID(ctx, input.ID)
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}
	if u == nil {
		return nil, &NotFoundError{ID: input.ID}
	}
	if err := applyUpdateFieldsToUser(u, input); err != nil {
		return nil, err
	}
	if err := store.Update(ctx, u); err != nil {
		return nil, fmt.Errorf("update user: %w", err)
	}
	return u, nil
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
	if s.Audit == nil {
		return s.deleteUserWithStore(ctx, s.Store, id)
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

	deleted, err := s.deleteUserWithStore(ctx, auditCtx.store, id)
	if err != nil {
		return false, err
	}
	if !deleted {
		// Nothing was changed; no audit entry should be written.
		return false, nil
	}
	if err := auditCtx.audit.Create(ctx, &audit.Entry{
		Entity:   audit.EntityUser,
		EntityID: id,
		Action:   audit.ActionDelete,
		ActorID:  auditCtx.actorID,
	}); err != nil {
		return false, fmt.Errorf("audit create entry: %w", err)
	}
	if err := auditCtx.tx.Commit(); err != nil {
		return false, err
	}
	committed = true

	return true, nil
}

func (s *Service) deleteUserWithStore(ctx context.Context, store Store, id string) (bool, error) {
	if err := store.Delete(ctx, id); err != nil {
		var e *NotFoundError
		if errors.As(err, &e) {
			return false, nil
		}
		return false, fmt.Errorf("delete user: %w", err)
	}
	return true, nil
}

func (s *Service) ensureUniqueEmail(ctx context.Context, email string) error {
	return s.ensureUniqueEmailWithStore(ctx, s.Store, email)
}

func (s *Service) prepareAuditTx(ctx context.Context) (*auditTxContext, error) {
	storeTxAware, ok := s.Store.(TxAwareStore)
	if !ok {
		return nil, errors.New("audit: users store is not tx-aware")
	}

	actorID, tx, auditTxStore, err := audit.PrepareWriteTx(ctx, s.Audit)
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

func (s *Service) ensureUniqueEmailWithStore(ctx context.Context, store Store, email string) error {
	existing, err := store.GetByEmail(ctx, email)
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
