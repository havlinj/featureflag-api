//go:build integration

package integration

import (
	"testing"

	"github.com/havlinj/featureflag-api/internal/testutil"
)

func TestAuditLogsAPI_createFlag_writes_and_reads_audit_entry(t *testing.T) {
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
			createFlag(input: $input) { id key }
		}
	`, map[string]interface{}{
		"input": map[string]interface{}{
			"key":         "audit-flag",
			"description": "audit integration",
			"environment": "dev",
		},
	})
	if err != nil {
		t.Fatalf("createFlag request: %v", err)
	}
	requireDataAndNoErrors(t, createResp)
	created, _ := createResp.Data["createFlag"].(map[string]interface{})
	flagID, _ := created["id"].(string)
	if flagID == "" {
		t.Fatal("expected created flag id")
	}

	// 2) query seeded admin ID (for actorId assertion)
	userResp, err := client.DoRequest(`
		query UserByEmail($email: String!) {
			userByEmail(email: $email) { id }
		}
	`, map[string]interface{}{"email": "admin@test.com"})
	if err != nil {
		t.Fatalf("userByEmail request: %v", err)
	}
	requireDataAndNoErrors(t, userResp)
	userMap, _ := userResp.Data["userByEmail"].(map[string]interface{})
	adminID, _ := userMap["id"].(string)
	if adminID == "" {
		t.Fatal("expected admin id")
	}

	// 3) auditLogs query (filter: feature_flag/create)
	queryResp, err := client.DoRequest(`
		query AuditLogs($filter: AuditLogsFilterInput, $limit: Int, $offset: Int) {
			auditLogs(filter: $filter, limit: $limit, offset: $offset) {
				id
				entity
				entityId
				action
				actorId
			}
		}
	`, map[string]interface{}{
		"filter": map[string]interface{}{
			"entity": "feature_flag",
			"action": "create",
		},
		"limit":  10,
		"offset": 0,
	})
	if err != nil {
		t.Fatalf("auditLogs request: %v", err)
	}
	requireDataAndNoErrors(t, queryResp)

	list, _ := queryResp.Data["auditLogs"].([]interface{})
	if len(list) == 0 {
		t.Fatal("expected at least one audit log entry")
	}
	first, _ := list[0].(map[string]interface{})
	if first["entity"] != "feature_flag" || first["action"] != "create" {
		t.Fatalf("unexpected audit entry: %+v", first)
	}
	if first["entityId"] != flagID {
		t.Fatalf("expected entityId=%s, got %v", flagID, first["entityId"])
	}
	if first["actorId"] != adminID {
		t.Fatalf("expected actorId=%s, got %v", adminID, first["actorId"])
	}
}

func integrationAdminActorID(t *testing.T, client *testutil.GraphQLClient) string {
	t.Helper()
	userResp, err := client.DoRequest(`
		query UserByEmail($email: String!) {
			userByEmail(email: $email) { id }
		}
	`, map[string]interface{}{"email": "admin@test.com"})
	if err != nil {
		t.Fatalf("userByEmail request: %v", err)
	}
	requireDataAndNoErrors(t, userResp)
	userMap, _ := userResp.Data["userByEmail"].(map[string]interface{})
	id, _ := userMap["id"].(string)
	if id == "" {
		t.Fatal("expected admin id")
	}
	return id
}

func requireFirstAuditLog(t *testing.T, client *testutil.GraphQLClient, entity, action, wantEntityID, wantActorID string) {
	t.Helper()
	queryResp, err := client.DoRequest(`
		query AuditLogs($filter: AuditLogsFilterInput, $limit: Int, $offset: Int) {
			auditLogs(filter: $filter, limit: $limit, offset: $offset) {
				entity
				entityId
				action
				actorId
			}
		}
	`, map[string]interface{}{
		"filter": map[string]interface{}{
			"entity": entity,
			"action": action,
		},
		"limit":  10,
		"offset": 0,
	})
	if err != nil {
		t.Fatalf("auditLogs request: %v", err)
	}
	requireDataAndNoErrors(t, queryResp)
	list, _ := queryResp.Data["auditLogs"].([]interface{})
	if len(list) == 0 {
		t.Fatalf("expected at least one audit log for entity=%q action=%q", entity, action)
	}
	first, _ := list[0].(map[string]interface{})
	if first["entityId"] != wantEntityID {
		t.Fatalf("entityId: want %q, got %v", wantEntityID, first["entityId"])
	}
	if first["actorId"] != wantActorID {
		t.Fatalf("actorId: want %q, got %v", wantActorID, first["actorId"])
	}
}

func TestAuditLogsAPI_updateFlag_writes_audit_entry(t *testing.T) {
	database, cleanup := testutil.PostgresForIntegration(t)
	defer cleanup()
	testutil.TruncateAll(t, database)
	_, client, shutdown := startAppWithDB(t, database)
	defer shutdown()

	adminToken := testutil.SeedAdminAndLogin(t, database, client, "admin@test.com", "adminpass")
	client.SetToken(adminToken)

	createResp, err := client.DoRequest(`
		mutation CreateFlag($input: CreateFlagInput!) {
			createFlag(input: $input) { id key }
		}
	`, map[string]interface{}{
		"input": map[string]interface{}{
			"key":         "audit-update-flag",
			"environment": "dev",
		},
	})
	if err != nil {
		t.Fatalf("createFlag request: %v", err)
	}
	requireDataAndNoErrors(t, createResp)
	created, _ := createResp.Data["createFlag"].(map[string]interface{})
	flagID, _ := created["id"].(string)
	if flagID == "" {
		t.Fatal("expected created flag id")
	}

	updateResp, err := client.DoRequest(`
		mutation UpdateFlag($input: UpdateFlagInput!) {
			updateFlag(input: $input) { id enabled }
		}
	`, map[string]interface{}{
		"input": map[string]interface{}{
			"key":     "audit-update-flag",
			"enabled": true,
		},
	})
	if err != nil {
		t.Fatalf("updateFlag request: %v", err)
	}
	requireDataAndNoErrors(t, updateResp)
	updated, _ := updateResp.Data["updateFlag"].(map[string]interface{})
	if updated["id"] != flagID {
		t.Fatalf("updateFlag id: want %q, got %v", flagID, updated["id"])
	}

	adminID := integrationAdminActorID(t, client)
	requireFirstAuditLog(t, client, "feature_flag", "update", flagID, adminID)
}

func TestAuditLogsAPI_deleteFlag_writes_audit_entry(t *testing.T) {
	database, cleanup := testutil.PostgresForIntegration(t)
	defer cleanup()
	testutil.TruncateAll(t, database)
	_, client, shutdown := startAppWithDB(t, database)
	defer shutdown()

	adminToken := testutil.SeedAdminAndLogin(t, database, client, "admin@test.com", "adminpass")
	client.SetToken(adminToken)

	createResp, err := client.DoRequest(`
		mutation CreateFlag($input: CreateFlagInput!) {
			createFlag(input: $input) { id key }
		}
	`, map[string]interface{}{
		"input": map[string]interface{}{
			"key":         "audit-delete-flag",
			"environment": "dev",
		},
	})
	if err != nil {
		t.Fatalf("createFlag request: %v", err)
	}
	requireDataAndNoErrors(t, createResp)
	created, _ := createResp.Data["createFlag"].(map[string]interface{})
	flagID, _ := created["id"].(string)
	if flagID == "" {
		t.Fatal("expected created flag id")
	}

	delResp, err := client.DoRequest(`
		mutation DeleteFlag($key: String!, $environment: String!) {
			deleteFlag(key: $key, environment: $environment)
		}
	`, map[string]interface{}{
		"key":         "audit-delete-flag",
		"environment": "dev",
	})
	if err != nil {
		t.Fatalf("deleteFlag request: %v", err)
	}
	requireDataAndNoErrors(t, delResp)
	if delResp.Data["deleteFlag"] != true {
		t.Fatalf("deleteFlag: expected true, got %v", delResp.Data["deleteFlag"])
	}

	adminID := integrationAdminActorID(t, client)
	requireFirstAuditLog(t, client, "feature_flag", "delete", flagID, adminID)
}

func TestAuditLogsAPI_createUser_writes_audit_entry(t *testing.T) {
	database, cleanup := testutil.PostgresForIntegration(t)
	defer cleanup()
	testutil.TruncateAll(t, database)
	_, client, shutdown := startAppWithDB(t, database)
	defer shutdown()

	adminToken := testutil.SeedAdminAndLogin(t, database, client, "admin@test.com", "adminpass")
	client.SetToken(adminToken)

	createResp, err := client.DoRequest(`
		mutation CreateUser($input: CreateUserInput!) {
			createUser(input: $input) { id email }
		}
	`, map[string]interface{}{
		"input": map[string]interface{}{
			"email": "audit-new-user@example.com",
			"role":  "developer",
		},
	})
	if err != nil {
		t.Fatalf("createUser request: %v", err)
	}
	requireDataAndNoErrors(t, createResp)
	u, _ := createResp.Data["createUser"].(map[string]interface{})
	userID, _ := u["id"].(string)
	if userID == "" {
		t.Fatal("expected new user id")
	}

	adminID := integrationAdminActorID(t, client)
	requireFirstAuditLog(t, client, "user", "create", userID, adminID)
}

func TestAuditLogsAPI_createExperiment_writes_audit_entry(t *testing.T) {
	database, cleanup := testutil.PostgresForIntegration(t)
	defer cleanup()
	testutil.TruncateAll(t, database)
	_, client, shutdown := startAppWithDB(t, database)
	defer shutdown()

	adminToken := testutil.SeedAdminAndLogin(t, database, client, "admin@test.com", "adminpass")
	client.SetToken(adminToken)

	createResp, err := client.DoRequest(`
		mutation CreateExperiment($input: CreateExperimentInput!) {
			createExperiment(input: $input) { id key }
		}
	`, map[string]interface{}{
		"input": map[string]interface{}{
			"key":         "audit-exp-key",
			"environment": "dev",
			"variants": []map[string]interface{}{
				{"name": "A", "weight": 50},
				{"name": "B", "weight": 50},
			},
		},
	})
	if err != nil {
		t.Fatalf("createExperiment request: %v", err)
	}
	requireDataAndNoErrors(t, createResp)
	exp, _ := createResp.Data["createExperiment"].(map[string]interface{})
	expID, _ := exp["id"].(string)
	if expID == "" {
		t.Fatal("expected experiment id")
	}

	adminID := integrationAdminActorID(t, client)
	requireFirstAuditLog(t, client, "experiment", "create", expID, adminID)
}

func TestAuditLogsAPI_negative_offset_returns_error(t *testing.T) {
	database, cleanup := testutil.PostgresForIntegration(t)
	defer cleanup()
	testutil.TruncateAll(t, database)
	_, client, shutdown := startAppWithDB(t, database)
	defer shutdown()

	adminToken := testutil.SeedAdminAndLogin(t, database, client, "admin@test.com", "adminpass")
	client.SetToken(adminToken)

	resp, err := client.DoRequest(`
		query AuditLogs($offset: Int) {
			auditLogs(offset: $offset) { id }
		}
	`, map[string]interface{}{"offset": -1})
	if err != nil {
		t.Fatalf("auditLogs request: %v", err)
	}
	requireGraphQLErrors(t, resp)
}
