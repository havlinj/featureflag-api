//go:build integration

package integration

import (
	"context"
	"log"
	"net/http"
	"testing"
	"time"

	"github.com/jan-havlin-dev/featureflag-api/internal/app"
	"github.com/jan-havlin-dev/featureflag-api/internal/flags"
	"github.com/jan-havlin-dev/featureflag-api/internal/testutil"
	"github.com/jan-havlin-dev/featureflag-api/internal/users"
)

func TestFlagsAPI_GraphQLOverHTTPS(t *testing.T) {
	database, cleanup := testutil.PostgresForIntegration(t)
	defer cleanup()
	testutil.TruncateAll(t, database)

	flagsStore := flags.NewPostgresStore(database.Conn())
	usersStore := users.NewPostgresStore(database.Conn())

	addr := testutil.MakeFreeSocketAddr()
	tlsConfig, err := testutil.NewTLSConfigForServer()
	if err != nil {
		t.Fatalf("create TLS config: %v", err)
	}

	jwtSecret := []byte("test-jwt-secret")
	a := app.NewApp(tlsConfig, flagsStore, usersStore, jwtSecret)
	go func() {
		if err := a.Run(addr); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := a.Shutdown(ctx); err != nil {
			t.Fatalf("shutdown: %v", err)
		}
	}()

	time.Sleep(100 * time.Millisecond)

	client := testutil.NewClientForIntegration("https://" + addr)

	// 0) Seed admin and login to get token (createFlag requires admin or developer)
	adminToken := testutil.SeedAdminAndLogin(t, database, client, "admin@test.com", "adminpass")
	client.SetToken(adminToken)

	// 1) createFlag mutation
	createResp, err := client.DoRequest(`
		mutation CreateFlag($input: CreateFlagInput!) {
			createFlag(input: $input) {
				id
				key
				enabled
				environment
			}
		}
	`, map[string]interface{}{
		"input": map[string]interface{}{
			"key":         "test-flag",
			"description": "Integration test flag",
			"environment": "dev",
		},
	})
	if err != nil {
		t.Fatalf("createFlag request: %v", err)
	}
	if createResp.Data == nil || (createResp.Errors != nil && len(createResp.Errors) > 0) {
		t.Fatalf("createFlag: expected data, got data=%v errors=%v", createResp.Data, createResp.Errors)
	}
	createData := createResp.Data
	createFlag, _ := createData["createFlag"].(map[string]interface{})
	if createFlag["key"] != "test-flag" || createFlag["enabled"] != false {
		t.Errorf("createFlag: got %+v", createFlag)
	}

	// 2) updateFlag mutation
	updateResp, err := client.DoRequest(`
		mutation UpdateFlag($input: UpdateFlagInput!) {
			updateFlag(input: $input) {
				id
				key
				enabled
			}
		}
	`, map[string]interface{}{
		"input": map[string]interface{}{
			"key":     "test-flag",
			"enabled": true,
		},
	})
	if err != nil {
		t.Fatalf("updateFlag request: %v", err)
	}
	if updateResp.Data == nil || (updateResp.Errors != nil && len(updateResp.Errors) > 0) {
		t.Fatalf("updateFlag: expected data, got data=%v errors=%v", updateResp.Data, updateResp.Errors)
	}
	updateData := updateResp.Data
	updatedFlag, _ := updateData["updateFlag"].(map[string]interface{})
	if updatedFlag["enabled"] != true {
		t.Errorf("updateFlag: expected enabled true, got %+v", updatedFlag)
	}

	// 3) evaluateFlag query – flag is enabled, no rules → true
	evalResp, err := client.DoRequest(`
		query EvaluateFlag($key: String!, $userId: ID!) {
			evaluateFlag(key: $key, userId: $userId)
		}
	`, map[string]interface{}{
		"key":    "test-flag",
		"userId": "user-1",
	})
	if err != nil {
		t.Fatalf("evaluateFlag request: %v", err)
	}
	if evalResp.Data == nil || (evalResp.Errors != nil && len(evalResp.Errors) > 0) {
		t.Fatalf("evaluateFlag: expected data, got data=%v errors=%v", evalResp.Data, evalResp.Errors)
	}
	evalData := evalResp.Data
	if got, _ := evalData["evaluateFlag"].(bool); !got {
		t.Errorf("evaluateFlag: expected true (flag enabled, no rules), got false")
	}
}

func TestUsersAPI_GraphQLOverHTTPS(t *testing.T) {
	database, cleanup := testutil.PostgresForIntegration(t)
	defer cleanup()
	testutil.TruncateAll(t, database)

	flagsStore := flags.NewPostgresStore(database.Conn())
	usersStore := users.NewPostgresStore(database.Conn())

	addr := testutil.MakeFreeSocketAddr()
	tlsConfig, err := testutil.NewTLSConfigForServer()
	if err != nil {
		t.Fatalf("create TLS config: %v", err)
	}

	jwtSecret := []byte("test-jwt-secret")
	a := app.NewApp(tlsConfig, flagsStore, usersStore, jwtSecret)
	go func() {
		if err := a.Run(addr); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := a.Shutdown(ctx); err != nil {
			t.Fatalf("shutdown: %v", err)
		}
	}()

	time.Sleep(100 * time.Millisecond)

	client := testutil.NewClientForIntegration("https://" + addr)

	// 0) Seed admin and login to get token (required for RBAC)
	adminToken := testutil.SeedAdminAndLogin(t, database, client, "admin@test.com", "adminpass")
	client.SetToken(adminToken)

	// 1) createUser
	createResp, err := client.DoRequest(`
		mutation CreateUser($input: CreateUserInput!) {
			createUser(input: $input) {
				id
				email
				role
				createdAt
			}
		}
	`, map[string]interface{}{
		"input": map[string]interface{}{
			"email": "admin@example.com",
			"role":  "admin",
		},
	})
	if err != nil {
		t.Fatalf("createUser request: %v", err)
	}
	if createResp.Data == nil || (createResp.Errors != nil && len(createResp.Errors) > 0) {
		t.Fatalf("createUser: expected data, got data=%v errors=%v", createResp.Data, createResp.Errors)
	}
	createData := createResp.Data
	createUser, _ := createData["createUser"].(map[string]interface{})
	userID, _ := createUser["id"].(string)
	if userID == "" {
		t.Fatal("createUser: expected id in response")
	}

	// 2) user(id)
	getResp, err := client.DoRequest(`
		query User($id: ID!) {
			user(id: $id) {
				id
				email
				role
			}
		}
	`, map[string]interface{}{"id": userID})
	if err != nil {
		t.Fatalf("user query: %v", err)
	}
	if getResp.Data == nil {
		t.Fatalf("user: expected data, got errors=%v", getResp.Errors)
	}
	getData := getResp.Data
	gotUser, _ := getData["user"].(map[string]interface{})
	if gotUser["email"] != "admin@example.com" || gotUser["role"] != "admin" {
		t.Errorf("user: got %+v", gotUser)
	}

	// 3) updateUser
	_, err = client.DoRequest(`
		mutation UpdateUser($input: UpdateUserInput!) {
			updateUser(input: $input) {
				id
				email
				role
			}
		}
	`, map[string]interface{}{
		"input": map[string]interface{}{
			"id":    userID,
			"email": "updated@example.com",
			"role":  "developer",
		},
	})
	if err != nil {
		t.Fatalf("updateUser request: %v", err)
	}

	// 4) user(id) again
	getResp2, err := client.DoRequest(`
		query User($id: ID!) {
			user(id: $id) {
				id
				email
				role
			}
		}
	`, map[string]interface{}{"id": userID})
	if err != nil {
		t.Fatalf("user query after update: %v", err)
	}
	getData2 := getResp2.Data
	gotUser2, _ := getData2["user"].(map[string]interface{})
	if gotUser2["email"] != "updated@example.com" || gotUser2["role"] != "developer" {
		t.Errorf("user after update: got %+v", gotUser2)
	}

	// 5) deleteUser
	delResp, err := client.DoRequest(`
		mutation DeleteUser($id: ID!) {
			deleteUser(id: $id)
		}
	`, map[string]interface{}{"id": userID})
	if err != nil {
		t.Fatalf("deleteUser request: %v", err)
	}
	if delResp.Data == nil {
		t.Fatalf("deleteUser: expected data, got errors=%v", delResp.Errors)
	}
	delData := delResp.Data
	if delData["deleteUser"] != true {
		t.Errorf("deleteUser: expected true, got %v", delData["deleteUser"])
	}

	// 6) user(id) after delete → null
	getResp3, err := client.DoRequest(`
		query User($id: ID!) {
			user(id: $id) {
				id
			}
		}
	`, map[string]interface{}{"id": userID})
	if err != nil {
		t.Fatalf("user query after delete: %v", err)
	}
	getData3 := getResp3.Data
	if getData3["user"] != nil {
		t.Errorf("user after delete: expected null, got %v", getData3["user"])
	}
}
