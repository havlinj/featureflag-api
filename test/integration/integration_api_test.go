//go:build integration

package integration

import (
	"context"
	"log"
	"net/http"
	"testing"
	"time"

	"github.com/jan-havlin-dev/featureflag-api/internal/app"
	"github.com/jan-havlin-dev/featureflag-api/internal/db"
	"github.com/jan-havlin-dev/featureflag-api/internal/flags"
	"github.com/jan-havlin-dev/featureflag-api/internal/testutil"
	"github.com/jan-havlin-dev/featureflag-api/internal/users"
)

// startAppWithDB starts the app with the given database, runs the server in a goroutine,
// and returns the app, a GraphQL client, and a shutdown function. Caller must call defer shutdown().
func startAppWithDB(t *testing.T, database *db.DB) (*app.App, *testutil.GraphQLClient, func()) {
	t.Helper()
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
	time.Sleep(100 * time.Millisecond)
	client := testutil.NewClientForIntegration("https://" + addr)
	shutdown := func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = a.Shutdown(ctx)
	}
	return a, client, shutdown
}

func TestFlagsAPI_GraphQLOverHTTPS(t *testing.T) {
	database, cleanup := testutil.PostgresForIntegration(t)
	defer cleanup()
	testutil.TruncateAll(t, database)
	_, client, shutdown := startAppWithDB(t, database)
	defer shutdown()

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
		query EvaluateFlag($key: String!, $evaluationContext: EvaluationContextInput!) {
			evaluateFlag(key: $key, evaluationContext: $evaluationContext)
		}
	`, map[string]interface{}{
		"key": "test-flag",
		"evaluationContext": map[string]interface{}{
			"userId": "user-1",
		},
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
	_, client, shutdown := startAppWithDB(t, database)
	defer shutdown()

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

func TestLogin_returnsToken_and_tokenWorksForProtectedMutation(t *testing.T) {
	database, cleanup := testutil.PostgresForIntegration(t)
	defer cleanup()
	testutil.TruncateAll(t, database)
	_, client, shutdown := startAppWithDB(t, database)
	defer shutdown()

	token := testutil.SeedAdminAndLogin(t, database, client, "admin@test.com", "adminpass")

	if token == "" {
		t.Fatal("login should return non-empty token")
	}

	client.SetToken(token)
	createResp, err := client.DoRequest(`
		mutation CreateFlag($input: CreateFlagInput!) {
			createFlag(input: $input) { id key enabled environment }
		}
	`, map[string]interface{}{
		"input": map[string]interface{}{
			"key":         "post-login-flag",
			"description": "Created after login",
			"environment": "dev",
		},
	})
	if err != nil {
		t.Fatalf("createFlag after login: %v", err)
	}
	if createResp.Data == nil || (createResp.Errors != nil && len(createResp.Errors) > 0) {
		t.Fatalf("createFlag with valid token: expected data, got data=%v errors=%v", createResp.Data, createResp.Errors)
	}
}

func TestProtectedMutation_withoutAuth_returnsError(t *testing.T) {
	database, cleanup := testutil.PostgresForIntegration(t)
	defer cleanup()
	testutil.TruncateAll(t, database)
	_, client, shutdown := startAppWithDB(t, database)
	defer shutdown()

	createResp, err := client.DoRequest(`
		mutation CreateFlag($input: CreateFlagInput!) {
			createFlag(input: $input) { id key }
		}
	`, map[string]interface{}{
		"input": map[string]interface{}{
			"key":         "no-auth-flag",
			"description": "Should fail",
			"environment": "dev",
		},
	})
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	if createResp.Errors == nil || len(createResp.Errors) == 0 {
		t.Fatal("createFlag without token should return GraphQL errors (unauthorized)")
	}
}

func TestAdminCreatedUser_canLogin_and_roleEnforced(t *testing.T) {
	database, cleanup := testutil.PostgresForIntegration(t)
	defer cleanup()
	testutil.TruncateAll(t, database)
	_, client, shutdown := startAppWithDB(t, database)
	defer shutdown()

	adminToken := testutil.SeedAdminAndLogin(t, database, client, "admin@test.com", "adminpass")
	client.SetToken(adminToken)

	devPass := "devpass"
	createUserResp, err := client.DoRequest(`
		mutation CreateUser($input: CreateUserInput!) {
			createUser(input: $input) { id email role }
		}
	`, map[string]interface{}{
		"input": map[string]interface{}{
			"email":    "dev@test.com",
			"role":     "developer",
			"password": devPass,
		},
	})
	if err != nil {
		t.Fatalf("createUser: %v", err)
	}
	if createUserResp.Data == nil || (createUserResp.Errors != nil && len(createUserResp.Errors) > 0) {
		t.Fatalf("createUser: expected data, got data=%v errors=%v", createUserResp.Data, createUserResp.Errors)
	}

	loginResp, err := client.DoRequest(`
		mutation Login($input: LoginInput!) {
			login(input: $input) { token }
		}
	`, map[string]interface{}{
		"input": map[string]interface{}{"email": "dev@test.com", "password": devPass},
	})
	if err != nil {
		t.Fatalf("login as dev: %v", err)
	}
	if loginResp.Data == nil || (loginResp.Errors != nil && len(loginResp.Errors) > 0) {
		t.Fatalf("login as dev: expected data, got data=%v errors=%v", loginResp.Data, loginResp.Errors)
	}
	loginData, _ := loginResp.Data["login"].(map[string]interface{})
	devToken, _ := loginData["token"].(string)
	if devToken == "" {
		t.Fatal("dev login should return non-empty token")
	}

	client.SetToken(devToken)
	createFlagResp, err := client.DoRequest(`
		mutation CreateFlag($input: CreateFlagInput!) {
			createFlag(input: $input) { id key }
		}
	`, map[string]interface{}{
		"input": map[string]interface{}{
			"key":         "dev-created-flag",
			"description": "By developer",
			"environment": "dev",
		},
	})
	if err != nil {
		t.Fatalf("createFlag as dev: %v", err)
	}
	if createFlagResp.Data == nil || (createFlagResp.Errors != nil && len(createFlagResp.Errors) > 0) {
		t.Fatalf("developer should be allowed to createFlag: data=%v errors=%v", createFlagResp.Data, createFlagResp.Errors)
	}

	viewerPass := "viewerpass"
	client.SetToken(adminToken)
	createViewerResp, err := client.DoRequest(`
		mutation CreateUser($input: CreateUserInput!) {
			createUser(input: $input) { id email role }
		}
	`, map[string]interface{}{
		"input": map[string]interface{}{
			"email":    "viewer@test.com",
			"role":     "viewer",
			"password": viewerPass,
		},
	})
	if err != nil {
		t.Fatalf("createUser viewer: %v", err)
	}
	if createViewerResp.Data == nil || (createViewerResp.Errors != nil && len(createViewerResp.Errors) > 0) {
		t.Fatalf("createUser viewer: expected data, got data=%v errors=%v", createViewerResp.Data, createViewerResp.Errors)
	}

	viewerLoginResp, err := client.DoRequest(`
		mutation Login($input: LoginInput!) {
			login(input: $input) { token }
		}
	`, map[string]interface{}{
		"input": map[string]interface{}{"email": "viewer@test.com", "password": viewerPass},
	})
	if err != nil {
		t.Fatalf("login as viewer: %v", err)
	}
	if viewerLoginResp.Data == nil || (viewerLoginResp.Errors != nil && len(viewerLoginResp.Errors) > 0) {
		t.Fatalf("login as viewer: expected data, got data=%v errors=%v", viewerLoginResp.Data, viewerLoginResp.Errors)
	}
	viewerLoginData, _ := viewerLoginResp.Data["login"].(map[string]interface{})
	viewerToken, _ := viewerLoginData["token"].(string)
	client.SetToken(viewerToken)

	viewerCreateFlagResp, err := client.DoRequest(`
		mutation CreateFlag($input: CreateFlagInput!) {
			createFlag(input: $input) { id key }
		}
	`, map[string]interface{}{
		"input": map[string]interface{}{
			"key":         "viewer-cannot-create",
			"description": "Should fail",
			"environment": "dev",
		},
	})
	if err != nil {
		t.Fatalf("createFlag as viewer request: %v", err)
	}
	if viewerCreateFlagResp.Errors == nil || len(viewerCreateFlagResp.Errors) == 0 {
		t.Fatal("viewer must not be allowed to createFlag (expect GraphQL errors)")
	}
}
