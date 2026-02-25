//go:build integration

package integration

import (
	"context"
	"log"
	"net/http"
	"testing"
	"time"

	"github.com/jan-havlin-dev/featureflag-api/internal/app"
	"github.com/jan-havlin-dev/featureflag-api/internal/testutil"
)

func TestAppAPI_GraphQLOverHTTPS(t *testing.T) {
	addr := testutil.MakeFreeSocketAddr()
	tlsConfig, err := testutil.NewTLSConfigForServer()
	if err != nil {
		t.Fatalf("create TLS config: %v", err)
	}

	a := app.NewApp(tlsConfig, &testutil.MockFlagsStore{})
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

	// Give server time to listen
	time.Sleep(100 * time.Millisecond)

	client := testutil.NewClientForIntegration("https://" + addr)

	// 1) createFlag mutation – ensure app understands the command over HTTPS
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
	// GraphQL returns 200 with either data or errors; both mean the app understood the request
	if createResp.Data == nil && (createResp.Errors == nil || len(createResp.Errors) == 0) {
		t.Error("createFlag: expected response to have data or errors")
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
	if updateResp.Data == nil && (updateResp.Errors == nil || len(updateResp.Errors) == 0) {
		t.Error("updateFlag: expected response to have data or errors")
	}

	// 3) evaluateFlag query
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
	if evalResp.Data == nil && (evalResp.Errors == nil || len(evalResp.Errors) == 0) {
		t.Error("evaluateFlag: expected response to have data or errors")
	}
}
