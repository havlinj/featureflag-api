package users_test

import (
	"context"
	"errors"
	"testing"

	"github.com/jan-havlin-dev/featureflag-api/graph/model"
	"github.com/jan-havlin-dev/featureflag-api/internal/users"
	"github.com/jan-havlin-dev/featureflag-api/internal/users/mock"
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
	input := model.CreateUserInput{Email: "a@b.com", Role: "admin"}

	got, err := svc.CreateUser(ctx, input)

	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	if got == nil || got.Email != "a@b.com" || got.Role != "admin" {
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
	input := model.CreateUserInput{Email: "a@b.com", Role: "viewer"}

	_, err := svc.CreateUser(ctx, input)

	if !errors.Is(err, users.ErrDuplicateEmail) {
		t.Errorf("expected ErrDuplicateEmail, got %v", err)
	}
	if len(store.CreateCalls) != 0 {
		t.Errorf("Create should not be called, got %d calls", len(store.CreateCalls))
	}
}

func TestService_CreateUser_invalid_role_returns_error(t *testing.T) {
	ctx := context.Background()
	store := &mock.Store{}
	store.GetByEmailReturns = []mock.GetByEmailResult{
		{User: nil, Err: nil},
	}
	svc := users.NewService(store)
	input := model.CreateUserInput{Email: "a@b.com", Role: "invalid"}

	_, err := svc.CreateUser(ctx, input)

	if err == nil {
		t.Fatal("expected error for invalid role")
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

	if !errors.Is(err, users.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

// --- DeleteUser ---

func TestService_DeleteUser_not_found_returns_false_nil(t *testing.T) {
	ctx := context.Background()
	store := &mock.Store{}
	store.DeleteReturns = []error{users.ErrNotFound}
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
