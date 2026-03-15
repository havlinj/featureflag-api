//go:build integration

package integration

import (
	"testing"

	"github.com/havlinj/featureflag-api/internal/testutil"
)

func TestExperimentsAPI_GraphQLOverHTTPS(t *testing.T) {
	database, cleanup := testutil.PostgresForIntegration(t)
	defer cleanup()
	testutil.TruncateAll(t, database)
	_, client, shutdown := startAppWithDB(t, database)
	defer shutdown()

	adminToken := testutil.SeedAdminAndLogin(t, database, client, "admin@test.com", "adminpass")
	client.SetToken(adminToken)

	// 1) createExperiment mutation
	createResp, err := client.DoRequest(`
		mutation CreateExperiment($input: CreateExperimentInput!) {
			createExperiment(input: $input) {
				id
				key
				environment
			}
		}
	`, map[string]interface{}{
		"input": map[string]interface{}{
			"key":         "ab-test",
			"environment": "dev",
			"variants": []map[string]interface{}{
				{"name": "control", "weight": 50},
				{"name": "treatment", "weight": 50},
			},
		},
	})
	if err != nil {
		t.Fatalf("createExperiment request: %v", err)
	}
	requireDataAndNoErrors(t, createResp)
	createData := createResp.Data
	exp, _ := createData["createExperiment"].(map[string]interface{})
	if exp["key"] != "ab-test" || exp["environment"] != "dev" {
		t.Errorf("createExperiment: got %+v", exp)
	}
	if exp["id"] == nil || exp["id"] == "" {
		t.Error("createExperiment: expected non-empty id")
	}

	// 2) experiment(key, environment) query – get created experiment
	getResp, err := client.DoRequest(`
		query Experiment($key: String!, $environment: String!) {
			experiment(key: $key, environment: $environment) {
				id
				key
				environment
			}
		}
	`, map[string]interface{}{"key": "ab-test", "environment": "dev"})
	if err != nil {
		t.Fatalf("experiment query: %v", err)
	}
	requireDataAndNoErrors(t, getResp)
	got, _ := getResp.Data["experiment"].(map[string]interface{})
	if got["key"] != "ab-test" || got["environment"] != "dev" {
		t.Errorf("experiment: got %+v", got)
	}

	// 3) experiment(key, environment) not found → null
	getMissingResp, err := client.DoRequest(`
		query Experiment($key: String!, $environment: String!) {
			experiment(key: $key, environment: $environment) {
				id
				key
			}
		}
	`, map[string]interface{}{"key": "nonexistent", "environment": "prod"})
	if err != nil {
		t.Fatalf("experiment query not found: %v", err)
	}
	requireDataAndNoErrors(t, getMissingResp)
	if getMissingResp.Data["experiment"] != nil {
		t.Errorf("experiment not found: expected null, got %v", getMissingResp.Data["experiment"])
	}

	// 4) getAssignment(userId, experimentKey, environment) – deterministic assignment
	userResp, err := client.DoRequest(`
		query UserByEmail($email: String!) {
			userByEmail(email: $email) { id }
		}
	`, map[string]interface{}{"email": "admin@test.com"})
	if err != nil {
		t.Fatalf("userByEmail: %v", err)
	}
	requireDataAndNoErrors(t, userResp)
	u, _ := userResp.Data["userByEmail"].(map[string]interface{})
	userID, _ := u["id"].(string)
	if userID == "" {
		t.Fatal("user id empty")
	}

	assignResp, err := client.DoRequest(`
		query GetAssignment($userId: ID!, $experimentKey: String!, $environment: String!) {
			getAssignment(userId: $userId, experimentKey: $experimentKey, environment: $environment) {
				id
				experimentId
				name
				weight
			}
		}
	`, map[string]interface{}{
		"userId":        userID,
		"experimentKey": "ab-test",
		"environment":   "dev",
	})
	if err != nil {
		t.Fatalf("getAssignment: %v", err)
	}
	requireDataAndNoErrors(t, assignResp)
	variant, _ := assignResp.Data["getAssignment"].(map[string]interface{})
	if variant["name"] != "control" && variant["name"] != "treatment" {
		t.Errorf("getAssignment: expected control or treatment, got name=%v", variant["name"])
	}

	// 5) getAssignment again same user → same variant (determinism)
	assignResp2, err := client.DoRequest(`
		query GetAssignment($userId: ID!, $experimentKey: String!, $environment: String!) {
			getAssignment(userId: $userId, experimentKey: $experimentKey, environment: $environment) {
				id
				name
			}
		}
	`, map[string]interface{}{
		"userId":        userID,
		"experimentKey": "ab-test",
		"environment":   "dev",
	})
	if err != nil {
		t.Fatalf("getAssignment second call: %v", err)
	}
	requireDataAndNoErrors(t, assignResp2)
	variant2, _ := assignResp2.Data["getAssignment"].(map[string]interface{})
	if variant2["name"] != variant["name"] {
		t.Errorf("getAssignment determinism: same user should get same variant, got %q and %q", variant["name"], variant2["name"])
	}
}
