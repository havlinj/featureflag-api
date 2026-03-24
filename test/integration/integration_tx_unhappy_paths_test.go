//go:build integration

package integration

import (
	"testing"

	"github.com/havlinj/featureflag-api/internal/audit"
	"github.com/havlinj/featureflag-api/internal/testutil"
)

func TestTxUnhappy_UpdateFlag_audit_failure_rolls_back_flag_change(t *testing.T) {
	database, cleanup := testutil.PostgresForIntegration(t)
	defer cleanup()
	testutil.TruncateAll(t, database)

	baseAudit := audit.NewPostgresStore(database.Conn())
	auditStore := NewFaultInjectingAuditStore(baseAudit, audit.EntityFeatureFlag, audit.ActionUpdate)
	_, client, shutdown := startAppWithCustomAuditStore(t, database, auditStore)
	defer shutdown()

	adminToken := testutil.SeedAdminAndLogin(t, database, client, "admin@test.com", "adminpass")
	client.SetToken(adminToken)

	createResp, err := client.DoRequest(`
		mutation CreateFlag($input: CreateFlagInput!) {
			createFlag(input: $input) { id key enabled }
		}
	`, map[string]interface{}{
		"input": map[string]interface{}{
			"key":         "tx-unhappy-flag",
			"environment": "dev",
		},
	})
	if err != nil {
		t.Fatalf("createFlag request: %v", err)
	}
	requireDataAndNoErrors(t, createResp)

	updateResp, err := client.DoRequest(`
		mutation UpdateFlag($input: UpdateFlagInput!) {
			updateFlag(input: $input) { id key enabled }
		}
	`, map[string]interface{}{
		"input": map[string]interface{}{
			"key":     "tx-unhappy-flag",
			"enabled": true,
		},
	})
	if err != nil {
		t.Fatalf("updateFlag request: %v", err)
	}
	requireGraphQLErrors(t, updateResp)

	evalResp, err := client.DoRequest(`
		query EvaluateFlag($key: String!, $evaluationContext: EvaluationContextInput!) {
			evaluateFlag(key: $key, evaluationContext: $evaluationContext)
		}
	`, map[string]interface{}{
		"key":               "tx-unhappy-flag",
		"evaluationContext": map[string]interface{}{"userId": "user-1"},
	})
	if err != nil {
		t.Fatalf("evaluateFlag request: %v", err)
	}
	requireDataAndNoErrors(t, evalResp)
	if got, _ := evalResp.Data["evaluateFlag"].(bool); got {
		t.Fatal("expected evaluateFlag=false; failed audited update must be rolled back")
	}
}

func TestTxUnhappy_CreateExperiment_audit_failure_rolls_back_experiment_create(t *testing.T) {
	database, cleanup := testutil.PostgresForIntegration(t)
	defer cleanup()
	testutil.TruncateAll(t, database)

	baseAudit := audit.NewPostgresStore(database.Conn())
	auditStore := NewFaultInjectingAuditStore(baseAudit, audit.EntityExperiment, audit.ActionCreate)
	_, client, shutdown := startAppWithCustomAuditStore(t, database, auditStore)
	defer shutdown()

	adminToken := testutil.SeedAdminAndLogin(t, database, client, "admin@test.com", "adminpass")
	client.SetToken(adminToken)

	createResp, err := client.DoRequest(`
		mutation CreateExperiment($input: CreateExperimentInput!) {
			createExperiment(input: $input) { id key environment }
		}
	`, map[string]interface{}{
		"input": map[string]interface{}{
			"key":         "tx-unhappy-exp",
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
	requireGraphQLErrors(t, createResp)

	getResp, err := client.DoRequest(`
		query Experiment($key: String!, $environment: String!) {
			experiment(key: $key, environment: $environment) { id key environment }
		}
	`, map[string]interface{}{
		"key":         "tx-unhappy-exp",
		"environment": "dev",
	})
	if err != nil {
		t.Fatalf("experiment query: %v", err)
	}
	requireDataAndNoErrors(t, getResp)
	if getResp.Data["experiment"] != nil {
		t.Fatalf("expected null experiment after failed audited create, got %v", getResp.Data["experiment"])
	}
}

func TestTxUnhappy_DeleteUser_not_found_does_not_write_audit_entry(t *testing.T) {
	database, cleanup := testutil.PostgresForIntegration(t)
	defer cleanup()
	testutil.TruncateAll(t, database)
	_, client, shutdown := startAppWithDB(t, database)
	defer shutdown()

	adminToken := testutil.SeedAdminAndLogin(t, database, client, "admin@test.com", "adminpass")
	client.SetToken(adminToken)

	delResp, err := client.DoRequest(`
		mutation DeleteUser($id: ID!) {
			deleteUser(id: $id)
		}
	`, map[string]interface{}{"id": "00000000-0000-0000-0000-000000000000"})
	if err != nil {
		t.Fatalf("deleteUser request: %v", err)
	}
	requireDataAndNoErrors(t, delResp)
	if got, _ := delResp.Data["deleteUser"].(bool); got {
		t.Fatalf("expected deleteUser=false for missing user, got %v", delResp.Data["deleteUser"])
	}

	queryResp, err := client.DoRequest(`
		query AuditLogs($filter: AuditLogsFilterInput, $limit: Int, $offset: Int) {
			auditLogs(filter: $filter, limit: $limit, offset: $offset) {
				entity
				action
			}
		}
	`, map[string]interface{}{
		"filter": map[string]interface{}{
			"entity": "user",
			"action": "delete",
		},
		"limit":  10,
		"offset": 0,
	})
	if err != nil {
		t.Fatalf("auditLogs request: %v", err)
	}
	requireDataAndNoErrors(t, queryResp)
	list, _ := queryResp.Data["auditLogs"].([]interface{})
	if len(list) != 0 {
		t.Fatalf("expected no user/delete audit entry for not-found delete, got %d entries", len(list))
	}
}
