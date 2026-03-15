//go:build integration

package integration

import (
	"testing"

	"github.com/havlinj/featureflag-api/internal/testutil"
)

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
	requireDataAndNoErrors(t, createResp)
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
	requireDataAndNoErrors(t, updateResp)
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
	requireDataAndNoErrors(t, evalResp)
	evalData := evalResp.Data
	if got, _ := evalData["evaluateFlag"].(bool); !got {
		t.Errorf("evaluateFlag: expected true (flag enabled, no rules), got false")
	}

	// 4) createFlag with rules (percentage), updateFlag to enable it, then evaluateFlag
	createPctResp, err := client.DoRequest(`
		mutation CreateFlag($input: CreateFlagInput!) {
			createFlag(input: $input) {
				id
				key
				rolloutStrategy
			}
		}
	`, map[string]interface{}{
		"input": map[string]interface{}{
			"key":         "pct-flag",
			"environment": "dev",
			"rules":       []map[string]interface{}{{"type": "PERCENTAGE", "value": "100"}},
		},
	})
	if err != nil {
		t.Fatalf("createFlag with rules: %v", err)
	}
	requireDataAndNoErrors(t, createPctResp)
	updatePctResp, err := client.DoRequest(`
		mutation UpdateFlag($input: UpdateFlagInput!) {
			updateFlag(input: $input) { id key enabled }
		}
	`, map[string]interface{}{
		"input": map[string]interface{}{
			"key":     "pct-flag",
			"enabled": true,
		},
	})
	if err != nil {
		t.Fatalf("updateFlag pct-flag: %v", err)
	}
	requireDataAndNoErrors(t, updatePctResp)
	evalPctResp, err := client.DoRequest(`
		query EvaluateFlag($key: String!, $evaluationContext: EvaluationContextInput!) {
			evaluateFlag(key: $key, evaluationContext: $evaluationContext)
		}
	`, map[string]interface{}{
		"key":               "pct-flag",
		"evaluationContext": map[string]interface{}{"userId": "user-1"},
	})
	if err != nil {
		t.Fatalf("evaluateFlag pct: %v", err)
	}
	requireDataAndNoErrors(t, evalPctResp)
	// 100% rollout → bucket (0–99) < 100 is always true
	if got, _ := evalPctResp.Data["evaluateFlag"].(bool); !got {
		t.Errorf("evaluateFlag with percentage 100: expected true, got false")
	}

	// 5) deleteFlag
	delResp, err := client.DoRequest(`
		mutation DeleteFlag($key: String!, $environment: String!) {
			deleteFlag(key: $key, environment: $environment)
		}
	`, map[string]interface{}{"key": "test-flag", "environment": "dev"})
	if err != nil {
		t.Fatalf("deleteFlag: %v", err)
	}
	requireDataAndNoErrors(t, delResp)
	if delResp.Data["deleteFlag"] != true {
		t.Errorf("deleteFlag: expected true, got %v", delResp.Data["deleteFlag"])
	}

	// 6) evaluateFlag after delete → false (flag no longer exists)
	evalAfterDelResp, err := client.DoRequest(`
		query EvaluateFlag($key: String!, $evaluationContext: EvaluationContextInput!) {
			evaluateFlag(key: $key, evaluationContext: $evaluationContext)
		}
	`, map[string]interface{}{
		"key":               "test-flag",
		"evaluationContext": map[string]interface{}{"userId": "user-1"},
	})
	if err != nil {
		t.Fatalf("evaluateFlag after delete: %v", err)
	}
	requireDataAndNoErrors(t, evalAfterDelResp)
	if got, _ := evalAfterDelResp.Data["evaluateFlag"].(bool); got {
		t.Errorf("evaluateFlag after delete: expected false, got true")
	}
}
