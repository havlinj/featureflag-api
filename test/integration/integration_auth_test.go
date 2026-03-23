//go:build integration

package integration

import (
	"strings"
	"testing"

	"github.com/havlinj/featureflag-api/internal/testutil"
)

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
	requireDataAndNoErrors(t, createResp)
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
	requireGraphQLErrors(t, createResp)
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
	requireDataAndNoErrors(t, createUserResp)

	loginResp, err := client.DoRequest(`
		mutation Login($input: LoginInput!) {
			login(input: $input) { token }
		}
	`, map[string]interface{}{"input": map[string]interface{}{"email": "dev@test.com", "password": devPass}})
	if err != nil {
		t.Fatalf("login as dev: %v", err)
	}
	requireDataAndNoErrors(t, loginResp)
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
	requireDataAndNoErrors(t, createFlagResp)

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
	requireDataAndNoErrors(t, createViewerResp)

	viewerLoginResp, err := client.DoRequest(`
		mutation Login($input: LoginInput!) {
			login(input: $input) { token }
		}
	`, map[string]interface{}{"input": map[string]interface{}{"email": "viewer@test.com", "password": viewerPass}})
	if err != nil {
		t.Fatalf("login as viewer: %v", err)
	}
	requireDataAndNoErrors(t, viewerLoginResp)
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
	requireGraphQLErrors(t, viewerCreateFlagResp)
}

func TestLogin_invalidCredentials_errorIsSanitized(t *testing.T) {
	database, cleanup := testutil.PostgresForIntegration(t)
	defer cleanup()
	testutil.TruncateAll(t, database)
	_, client, shutdown := startAppWithDB(t, database)
	defer shutdown()

	loginResp, err := client.DoRequest(`
		mutation Login($input: LoginInput!) {
			login(input: $input) { token }
		}
	`, map[string]interface{}{
		"input": map[string]interface{}{
			"email":    "missing-user@test.com",
			"password": "wrong-password",
		},
	})
	if err != nil {
		t.Fatalf("login request: %v", err)
	}
	requireGraphQLErrors(t, loginResp)
	errText := graphqlErrorMessages(loginResp)
	if !strings.Contains(errText, "invalid credentials") {
		t.Fatalf("expected sanitized invalid credentials message, got: %s", errText)
	}
	if strings.Contains(errText, "missing-user@test.com") {
		t.Fatalf("error message leaks email context: %s", errText)
	}
}

func TestCreateFlag_forbiddenError_isSanitized(t *testing.T) {
	database, cleanup := testutil.PostgresForIntegration(t)
	defer cleanup()
	testutil.TruncateAll(t, database)
	_, client, shutdown := startAppWithDB(t, database)
	defer shutdown()

	adminToken := testutil.SeedAdminAndLogin(t, database, client, "admin@test.com", "adminpass")
	client.SetToken(adminToken)
	viewerPass := "viewer-pass"
	createViewerResp, err := client.DoRequest(`
		mutation CreateUser($input: CreateUserInput!) {
			createUser(input: $input) { id email role }
		}
	`, map[string]interface{}{
		"input": map[string]interface{}{
			"email":    "viewer-sanitized@test.com",
			"role":     "viewer",
			"password": viewerPass,
		},
	})
	if err != nil {
		t.Fatalf("create viewer: %v", err)
	}
	requireDataAndNoErrors(t, createViewerResp)

	viewerLoginResp, err := client.DoRequest(`
		mutation Login($input: LoginInput!) {
			login(input: $input) { token }
		}
	`, map[string]interface{}{
		"input": map[string]interface{}{
			"email":    "viewer-sanitized@test.com",
			"password": viewerPass,
		},
	})
	if err != nil {
		t.Fatalf("viewer login request: %v", err)
	}
	requireDataAndNoErrors(t, viewerLoginResp)
	loginData, _ := viewerLoginResp.Data["login"].(map[string]interface{})
	viewerToken, _ := loginData["token"].(string)
	client.SetToken(viewerToken)

	createFlagResp, err := client.DoRequest(`
		mutation CreateFlag($input: CreateFlagInput!) {
			createFlag(input: $input) { id key }
		}
	`, map[string]interface{}{
		"input": map[string]interface{}{
			"key":         "viewer-forbidden-flag",
			"description": "forbidden",
			"environment": "dev",
		},
	})
	if err != nil {
		t.Fatalf("viewer create flag request: %v", err)
	}
	requireGraphQLErrors(t, createFlagResp)
	errText := graphqlErrorMessages(createFlagResp)
	if !strings.Contains(errText, "forbidden") {
		t.Fatalf("expected sanitized forbidden message, got: %s", errText)
	}
	if strings.Contains(errText, "allowed") || strings.Contains(errText, "admin") || strings.Contains(errText, "developer") {
		t.Fatalf("error message leaks role policy details: %s", errText)
	}
}
