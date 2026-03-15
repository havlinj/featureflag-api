//go:build integration

package integration

import (
	"testing"

	"github.com/havlinj/featureflag-api/internal/testutil"
)

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
	requireDataAndNoErrors(t, createResp)
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
