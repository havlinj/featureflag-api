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
