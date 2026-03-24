package users_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/havlinj/featureflag-api/graph/model"
	"github.com/havlinj/featureflag-api/internal/audit"
	"github.com/havlinj/featureflag-api/internal/auth"
	"github.com/havlinj/featureflag-api/internal/testutil/auditmock"
	"github.com/havlinj/featureflag-api/internal/users"
	"github.com/havlinj/featureflag-api/internal/users/mock"
)

// --- CreateUser ---

func TestService_CreateUser_happy_path(t *testing.T) {
	ctx := context.Background()
	store := &mock.Store{}
	store.GetByEmailReturns = []mock.GetByEmailResult{
		{User: nil, Err: nil},
	}
	created := &users.User{ID: "u1", Email: "a@b.com", Role: users.RoleAdmin}
	store.CreateReturns = []mock.CreateResult{
		{User: created, Err: nil},
	}
	svc := users.NewService(store)
	input := model.CreateUserInput{Email: "a@b.com", Role: model.RoleAdmin}

	got, err := svc.CreateUser(ctx, input)

	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	if got == nil || got.Email != "a@b.com" || got.Role != model.RoleAdmin {
		t.Errorf("got %+v", got)
	}
	if len(store.CreateCalls) != 1 {
		t.Errorf("Create calls: want 1, got %d", len(store.CreateCalls))
	}
}

func TestService_CreateUser_duplicate_email_returns_ErrDuplicateEmail(t *testing.T) {
	ctx := context.Background()
	store := &mock.Store{}
	store.GetByEmailReturns = []mock.GetByEmailResult{
		{User: &users.User{ID: "u1", Email: "a@b.com"}, Err: nil},
	}
	svc := users.NewService(store)
	input := model.CreateUserInput{Email: "a@b.com", Role: model.RoleViewer}

	_, err := svc.CreateUser(ctx, input)

	var e *users.DuplicateEmailError
	if !errors.As(err, &e) {
		t.Errorf("expected *DuplicateEmailError, got %v", err)
	}
	if e.Email != "a@b.com" {
		t.Errorf("expected Email=a@b.com, got %q", e.Email)
	}
	if len(store.CreateCalls) != 0 {
		t.Errorf("Create should not be called, got %d calls", len(store.CreateCalls))
	}
}

func TestService_CreateUser_uniqueness_lookup_store_error_returns_wrapped(t *testing.T) {
	ctx := context.Background()
	store := &mock.Store{}
	wantErr := errors.New("GetByEmail failed")
	store.GetByEmailReturns = []mock.GetByEmailResult{{User: nil, Err: wantErr}}
	svc := users.NewService(store)
	input := model.CreateUserInput{Email: "new@x.com", Role: model.RoleDeveloper}

	_, err := svc.CreateUser(ctx, input)

	if !errors.Is(err, wantErr) {
		t.Fatalf("expected wrapped %v, got %v", wantErr, err)
	}
	var opErr *users.OperationError
	if !errors.As(err, &opErr) {
		t.Fatalf("expected *users.OperationError, got %T", err)
	}
	if opErr.Op != "users.service.ensure_unique_email.store_get_by_email" {
		t.Fatalf("unexpected op %q", opErr.Op)
	}
	if opErr.Email != "new@x.com" {
		t.Fatalf("unexpected context fields: %+v", opErr)
	}
	if len(store.CreateCalls) != 0 {
		t.Fatalf("Create must not run after failed email uniqueness lookup, got %d calls", len(store.CreateCalls))
	}
}

func TestService_CreateUser_withPassword_passesHashToStore(t *testing.T) {
	ctx := context.Background()
	store := &mock.Store{}
	store.GetByEmailReturns = []mock.GetByEmailResult{{User: nil, Err: nil}}
	store.CreateReturns = []mock.CreateResult{
		{User: &users.User{ID: "u1", Email: "a@b.com", Role: users.RoleDeveloper}, Err: nil},
	}
	svc := users.NewService(store)
	pass := "secret123"
	input := model.CreateUserInput{Email: "a@b.com", Role: model.RoleDeveloper, Password: &pass}

	_, err := svc.CreateUser(ctx, input)

	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	if len(store.CreateCalls) != 1 {
		t.Fatalf("Create calls: want 1, got %d", len(store.CreateCalls))
	}
	if store.CreateCalls[0].User.PasswordHash == nil {
		t.Error("expected PasswordHash to be set when password provided")
	}
}

// --- GetUser ---

func TestService_GetUser_not_found_returns_nil_nil(t *testing.T) {
	ctx := context.Background()
	store := &mock.Store{}
	store.GetByIDReturns = []mock.GetByIDResult{
		{User: nil, Err: nil},
	}
	svc := users.NewService(store)

	got, err := svc.GetUser(ctx, "missing")

	if err != nil {
		t.Fatalf("GetUser: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil, got %+v", got)
	}
}

func TestService_GetUser_found_returns_model_with_created_at(t *testing.T) {
	ctx := context.Background()
	store := &mock.Store{}
	createdAt := time.Date(2024, 6, 15, 9, 30, 0, 0, time.UTC)
	u := &users.User{ID: "u-found", Email: "found@example.com", Role: users.RoleDeveloper, CreatedAt: createdAt}
	store.GetByIDReturns = []mock.GetByIDResult{{User: u, Err: nil}}
	svc := users.NewService(store)

	got, err := svc.GetUser(ctx, "u-found")

	if err != nil {
		t.Fatalf("GetUser: %v", err)
	}
	if got == nil || got.Email != "found@example.com" || got.Role != model.RoleDeveloper {
		t.Fatalf("got %+v", got)
	}
	if got.CreatedAt != "2024-06-15T09:30:00Z" {
		t.Errorf("CreatedAt: want 2024-06-15T09:30:00Z, got %q", got.CreatedAt)
	}
}

func TestService_GetUser_store_error_returns_wrapped_error(t *testing.T) {
	ctx := context.Background()
	store := &mock.Store{}
	wantErr := errors.New("GetByID failed")
	store.GetByIDReturns = []mock.GetByIDResult{
		{User: nil, Err: wantErr},
	}
	svc := users.NewService(store)

	got, err := svc.GetUser(ctx, "some-id")

	if got != nil {
		t.Errorf("expected nil, got %+v", got)
	}
	if !errors.Is(err, wantErr) {
		t.Errorf("expected wrapped %v, got %v", wantErr, err)
	}
	var opErr *users.OperationError
	if !errors.As(err, &opErr) {
		t.Fatalf("expected *users.OperationError, got %T", err)
	}
	if opErr.Op != "users.service.get_user.store_get_by_id" {
		t.Fatalf("unexpected op %q", opErr.Op)
	}
	if opErr.ID != "some-id" {
		t.Fatalf("unexpected context fields: %+v", opErr)
	}
}

// --- GetUserByEmail ---

func TestService_GetUserByEmail_found_returns_user(t *testing.T) {
	ctx := context.Background()
	store := &mock.Store{}
	u := &users.User{ID: "u1", Email: "x@y.com", Role: users.RoleDeveloper}
	store.GetByEmailReturns = []mock.GetByEmailResult{
		{User: u, Err: nil},
	}
	svc := users.NewService(store)

	got, err := svc.GetUserByEmail(ctx, "x@y.com")

	if err != nil {
		t.Fatalf("GetUserByEmail: %v", err)
	}
	if got == nil || got.Email != "x@y.com" {
		t.Errorf("got %+v", got)
	}
}

func TestService_GetUserByEmail_store_error_returns_wrapped_error(t *testing.T) {
	ctx := context.Background()
	store := &mock.Store{}
	wantErr := errors.New("GetByEmail failed")
	store.GetByEmailReturns = []mock.GetByEmailResult{
		{User: nil, Err: wantErr},
	}
	svc := users.NewService(store)

	got, err := svc.GetUserByEmail(ctx, "x@y.com")

	if got != nil {
		t.Errorf("expected nil, got %+v", got)
	}
	if !errors.Is(err, wantErr) {
		t.Errorf("expected wrapped %v, got %v", wantErr, err)
	}
	var opErr *users.OperationError
	if !errors.As(err, &opErr) {
		t.Fatalf("expected *users.OperationError, got %T", err)
	}
	if opErr.Op != "users.service.get_user_by_email.store_get_by_email" {
		t.Fatalf("unexpected op %q", opErr.Op)
	}
	if opErr.Email != "x@y.com" {
		t.Fatalf("unexpected context fields: %+v", opErr)
	}
}

// --- UpdateUser ---

func TestService_UpdateUser_not_found_returns_ErrNotFound(t *testing.T) {
	ctx := context.Background()
	store := &mock.Store{}
	store.GetByIDReturns = []mock.GetByIDResult{
		{User: nil, Err: nil},
	}
	svc := users.NewService(store)
	email := "new@x.com"
	input := model.UpdateUserInput{ID: "missing", Email: &email}

	_, err := svc.UpdateUser(ctx, input)

	var e *users.NotFoundError
	if !errors.As(err, &e) {
		t.Errorf("expected *NotFoundError, got %v", err)
	}
	if e.ID != "missing" {
		t.Errorf("expected ID=missing, got %q", e.ID)
	}
}

func TestService_UpdateUser_get_error_includes_context(t *testing.T) {
	ctx := context.Background()
	store := &mock.Store{}
	wantErr := errors.New("GetByID failed")
	store.GetByIDReturns = []mock.GetByIDResult{
		{User: nil, Err: wantErr},
	}
	svc := users.NewService(store)
	input := model.UpdateUserInput{ID: "user-ctx-id"}

	_, err := svc.UpdateUser(ctx, input)

	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected wrapped %v, got %v", wantErr, err)
	}
	var opErr *users.OperationError
	if !errors.As(err, &opErr) {
		t.Fatalf("expected *users.OperationError, got %T", err)
	}
	if opErr.Op != "users.service.update_user.store_get_by_id" {
		t.Fatalf("unexpected op %q", opErr.Op)
	}
	if opErr.ID != "user-ctx-id" {
		t.Fatalf("unexpected context fields: %+v", opErr)
	}
}

func TestService_UpdateUser_withPassword_updatesHash(t *testing.T) {
	ctx := context.Background()
	store := &mock.Store{}
	u := &users.User{ID: "u1", Email: "old@x.com", Role: users.RoleViewer}
	store.GetByIDReturns = []mock.GetByIDResult{{User: u, Err: nil}}
	store.UpdateReturns = []error{nil}
	svc := users.NewService(store)
	pass := "newpass"
	input := model.UpdateUserInput{ID: "u1", Password: &pass}

	_, err := svc.UpdateUser(ctx, input)

	if err != nil {
		t.Fatalf("UpdateUser: %v", err)
	}
	if len(store.UpdateCalls) != 1 {
		t.Fatalf("Update calls: want 1, got %d", len(store.UpdateCalls))
	}
	if store.UpdateCalls[0].User.PasswordHash == nil {
		t.Error("expected PasswordHash to be set when password provided")
	}
}

// --- DeleteUser ---

func TestService_DeleteUser_not_found_returns_false_nil(t *testing.T) {
	ctx := context.Background()
	store := &mock.Store{}
	store.DeleteReturns = []error{&users.NotFoundError{ID: "missing"}}
	svc := users.NewService(store)

	got, err := svc.DeleteUser(ctx, "missing")

	if err != nil {
		t.Fatalf("DeleteUser: %v", err)
	}
	if got {
		t.Error("expected false when user not found")
	}
}

func TestService_DeleteUser_deleted_returns_true_nil(t *testing.T) {
	ctx := context.Background()
	store := &mock.Store{}
	store.DeleteReturns = []error{nil}
	svc := users.NewService(store)

	got, err := svc.DeleteUser(ctx, "some-id")

	if err != nil {
		t.Fatalf("DeleteUser: %v", err)
	}
	if !got {
		t.Error("expected true when user deleted")
	}
}

func TestService_DeleteUser_store_error_returns_wrapped_error(t *testing.T) {
	ctx := context.Background()
	store := &mock.Store{}
	wantErr := errors.New("Store.Delete failed")
	store.DeleteReturns = []error{wantErr}
	svc := users.NewService(store)

	got, err := svc.DeleteUser(ctx, "some-id")

	if got {
		t.Error("expected false when Delete fails")
	}
	if !errors.Is(err, wantErr) {
		t.Errorf("expected wrapped %v, got %v", wantErr, err)
	}
	var opErr *users.OperationError
	if !errors.As(err, &opErr) {
		t.Fatalf("expected *users.OperationError, got %T", err)
	}
	if opErr.Op != "users.service.delete_user.store_delete" {
		t.Fatalf("unexpected op %q", opErr.Op)
	}
	if opErr.ID != "some-id" {
		t.Fatalf("unexpected context fields: %+v", opErr)
	}
}

// --- Login ---

func TestService_Login_happy_path_returns_userID_and_role(t *testing.T) {
	ctx := context.Background()
	hash := mustHash(t, "correct")
	u := &users.User{ID: "u1", Email: "a@b.com", Role: users.RoleAdmin, PasswordHash: &hash}
	store := &mock.Store{}
	store.GetByEmailReturns = []mock.GetByEmailResult{{User: u, Err: nil}}
	svc := users.NewService(store)

	userID, role, err := svc.Login(ctx, "a@b.com", "correct")

	if err != nil {
		t.Fatalf("Login: %v", err)
	}
	if userID != "u1" || role != "admin" {
		t.Errorf("expected userID=u1 role=admin, got userID=%q role=%q", userID, role)
	}
}

func TestService_Login_user_not_found_returns_ErrInvalidCredentials(t *testing.T) {
	ctx := context.Background()
	store := &mock.Store{}
	store.GetByEmailReturns = []mock.GetByEmailResult{{User: nil, Err: nil}}
	svc := users.NewService(store)

	_, _, err := svc.Login(ctx, "missing@test.com", "any")

	var e *users.InvalidCredentialsError
	if !errors.As(err, &e) {
		t.Errorf("expected *InvalidCredentialsError, got %v", err)
	}
	if e.Email != "missing@test.com" {
		t.Errorf("expected Email=missing@test.com, got %q", e.Email)
	}
}

func TestService_Login_nil_password_hash_returns_ErrInvalidCredentials(t *testing.T) {
	ctx := context.Background()
	u := &users.User{ID: "u1", Email: "a@b.com", Role: users.RoleViewer, PasswordHash: nil}
	store := &mock.Store{}
	store.GetByEmailReturns = []mock.GetByEmailResult{{User: u, Err: nil}}
	svc := users.NewService(store)

	_, _, err := svc.Login(ctx, "a@b.com", "any")

	var e *users.InvalidCredentialsError
	if !errors.As(err, &e) {
		t.Errorf("expected *InvalidCredentialsError, got %v", err)
	}
	if e.Email != "a@b.com" {
		t.Errorf("expected Email=a@b.com, got %q", e.Email)
	}
}

func TestService_Login_wrong_password_returns_ErrInvalidCredentials(t *testing.T) {
	ctx := context.Background()
	hash := mustHash(t, "correct")
	u := &users.User{ID: "u1", Email: "a@b.com", Role: users.RoleAdmin, PasswordHash: &hash}
	store := &mock.Store{}
	store.GetByEmailReturns = []mock.GetByEmailResult{{User: u, Err: nil}}
	svc := users.NewService(store)

	_, _, err := svc.Login(ctx, "a@b.com", "wrong")

	var e *users.InvalidCredentialsError
	if !errors.As(err, &e) {
		t.Errorf("expected *InvalidCredentialsError, got %v", err)
	}
	if e.Email != "a@b.com" {
		t.Errorf("expected Email=a@b.com, got %q", e.Email)
	}
}

func TestService_Login_store_error_returns_wrapped_error(t *testing.T) {
	ctx := context.Background()
	store := &mock.Store{}
	wantErr := errors.New("GetByEmail failed")
	store.GetByEmailReturns = []mock.GetByEmailResult{{User: nil, Err: wantErr}}
	svc := users.NewService(store)

	_, _, err := svc.Login(ctx, "a@b.com", "any")

	if !errors.Is(err, wantErr) {
		t.Fatalf("expected wrapped %v, got %v", wantErr, err)
	}
	var opErr *users.OperationError
	if !errors.As(err, &opErr) {
		t.Fatalf("expected *users.OperationError, got %T", err)
	}
	if opErr.Op != "users.service.login.store_get_by_email" {
		t.Fatalf("unexpected op %q", opErr.Op)
	}
	if opErr.Email != "a@b.com" {
		t.Fatalf("unexpected context fields: %+v", opErr)
	}
}

func TestService_CreateUser_without_password_keeps_password_hash_nil(t *testing.T) {
	ctx := context.Background()
	store := &mock.Store{}
	store.GetByEmailReturns = []mock.GetByEmailResult{{User: nil, Err: nil}}
	store.CreateReturns = []mock.CreateResult{
		{User: &users.User{ID: "u1", Email: "a@b.com", Role: users.RoleDeveloper}, Err: nil},
	}
	svc := users.NewService(store)
	input := model.CreateUserInput{Email: "a@b.com", Role: model.RoleDeveloper}

	_, err := svc.CreateUser(ctx, input)

	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	if len(store.CreateCalls) != 1 {
		t.Fatalf("Create calls: want 1, got %d", len(store.CreateCalls))
	}
	if store.CreateCalls[0].User.PasswordHash != nil {
		t.Fatal("expected PasswordHash to stay nil when password is not provided")
	}
}

func TestService_UpdateUser_with_empty_password_does_not_change_password_hash(t *testing.T) {
	ctx := context.Background()
	existingHash := mustHash(t, "keep-me")
	u := &users.User{ID: "u1", Email: "old@x.com", Role: users.RoleViewer, PasswordHash: &existingHash}
	store := &mock.Store{}
	store.GetByIDReturns = []mock.GetByIDResult{{User: u, Err: nil}}
	store.UpdateReturns = []error{nil}
	svc := users.NewService(store)
	empty := ""
	input := model.UpdateUserInput{ID: "u1", Password: &empty}

	_, err := svc.UpdateUser(ctx, input)

	if err != nil {
		t.Fatalf("UpdateUser: %v", err)
	}
	if len(store.UpdateCalls) != 1 {
		t.Fatalf("Update calls: want 1, got %d", len(store.UpdateCalls))
	}
	if store.UpdateCalls[0].User.PasswordHash == nil {
		t.Fatal("expected PasswordHash to remain set")
	}
	if *store.UpdateCalls[0].User.PasswordHash != existingHash {
		t.Fatal("expected empty password to keep existing hash unchanged")
	}
}

func mustHash(t *testing.T, password string) string {
	t.Helper()
	hash, err := auth.HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	return hash
}

type txAwareUsersStoreMock struct {
	inner users.Store
}

func (s *txAwareUsersStoreMock) Create(ctx context.Context, user *users.User) (*users.User, error) {
	return s.inner.Create(ctx, user)
}

func (s *txAwareUsersStoreMock) GetByID(ctx context.Context, id string) (*users.User, error) {
	return s.inner.GetByID(ctx, id)
}

func (s *txAwareUsersStoreMock) GetByEmail(ctx context.Context, email string) (*users.User, error) {
	return s.inner.GetByEmail(ctx, email)
}

func (s *txAwareUsersStoreMock) Update(ctx context.Context, user *users.User) error {
	return s.inner.Update(ctx, user)
}

func (s *txAwareUsersStoreMock) Delete(ctx context.Context, id string) error {
	return s.inner.Delete(ctx, id)
}

func (s *txAwareUsersStoreMock) WithTx(tx *sql.Tx) users.Store {
	return s
}

func TestService_CreateUser_withAudit_missingActor_returns_error(t *testing.T) {
	store := &txAwareUsersStoreMock{inner: &mock.Store{}}
	svc := users.NewServiceWithAudit(store, &auditmock.TxAware{})
	input := model.CreateUserInput{Email: "a@b.com", Role: model.RoleAdmin}

	_, err := svc.CreateUser(context.Background(), input)

	var e *audit.MissingActorIDError
	if !errors.As(err, &e) {
		t.Fatalf("expected *audit.MissingActorIDError, got %T (%v)", err, err)
	}
}

func TestService_UpdateUser_withAudit_notTxAwareAuditStore_returns_error(t *testing.T) {
	store := &txAwareUsersStoreMock{inner: &mock.Store{}}
	svc := users.NewServiceWithAudit(store, &auditmock.TxStarter{})
	input := model.UpdateUserInput{ID: "u1"}
	ctx := auth.WithActorID(context.Background(), "u1")

	_, err := svc.UpdateUser(ctx, input)

	var e *audit.TxAwareRequiredError
	if !errors.As(err, &e) {
		t.Fatalf("expected *audit.TxAwareRequiredError, got %T (%v)", err, err)
	}
}

func TestService_DeleteUser_withAudit_beginTx_error_is_returned(t *testing.T) {
	store := &txAwareUsersStoreMock{inner: &mock.Store{}}
	wantErr := errors.New("begin tx failed")
	svc := users.NewServiceWithAudit(store, &auditmock.TxAware{
		TxStarter: auditmock.TxStarter{BeginErr: wantErr},
	})
	ctx := auth.WithActorID(context.Background(), "u1")

	_, err := svc.DeleteUser(ctx, "u1")

	if !errors.Is(err, wantErr) {
		t.Fatalf("expected %v, got %v", wantErr, err)
	}
}
