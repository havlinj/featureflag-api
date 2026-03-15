package users_test

import (
	"context"
	"errors"
	"testing"

	"github.com/havlinj/featureflag-api/graph/model"
	"github.com/havlinj/featureflag-api/internal/auth"
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

func TestService_UpdateUser_get_error_returns_wrapped_error(t *testing.T) {
	ctx := context.Background()
	store := &mock.Store{}
	wantErr := errors.New("GetByID failed")
	store.GetByIDReturns = []mock.GetByIDResult{
		{User: nil, Err: wantErr},
	}
	svc := users.NewService(store)
	input := model.UpdateUserInput{ID: "some-id"}

	got, err := svc.UpdateUser(ctx, input)

	if got != nil {
		t.Errorf("expected nil, got %+v", got)
	}
	if !errors.Is(err, wantErr) {
		t.Errorf("expected wrapped %v, got %v", wantErr, err)
	}
	if len(store.UpdateCalls) != 0 {
		t.Error("Update should not be called when GetByID fails")
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

func TestService_Login_user_not_found_returns_ErrNotFound(t *testing.T) {
	ctx := context.Background()
	store := &mock.Store{}
	store.GetByEmailReturns = []mock.GetByEmailResult{{User: nil, Err: nil}}
	svc := users.NewService(store)

	_, _, err := svc.Login(ctx, "missing@test.com", "any")

	var e *users.NotFoundError
	if !errors.As(err, &e) {
		t.Errorf("expected *NotFoundError, got %v", err)
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

func mustHash(t *testing.T, password string) string {
	t.Helper()
	hash, err := auth.HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	return hash
}
