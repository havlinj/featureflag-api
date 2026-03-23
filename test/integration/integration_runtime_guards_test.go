//go:build integration

package integration

import (
	"bytes"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/havlinj/featureflag-api/internal/testutil"
)

func TestGraphQLRequestBodyLimit_rejectsLargePayload(t *testing.T) {
	database, cleanup := testutil.PostgresForIntegration(t)
	defer cleanup()
	testutil.TruncateAll(t, database)
	_, client, shutdown := startAppWithDB(t, database)
	defer shutdown()

	largePayload := bytes.Repeat([]byte("x"), 2<<20)
	req, err := http.NewRequest(http.MethodPost, client.URL, bytes.NewReader(largePayload))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Client.Do(req)
	if err != nil {
		t.Fatalf("send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected status 413, got %d", resp.StatusCode)
	}
}

func TestGraphQLRequestBodyLimit_allowsSmallPayload(t *testing.T) {
	database, cleanup := testutil.PostgresForIntegration(t)
	defer cleanup()
	testutil.TruncateAll(t, database)
	_, client, shutdown := startAppWithDB(t, database)
	defer shutdown()

	reqBody := `{"query":"{ __typename }"}`
	req, err := http.NewRequest(http.MethodPost, client.URL, strings.NewReader(reqBody))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Client.Do(req)
	if err != nil {
		t.Fatalf("send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected status 200, got %d body=%s", resp.StatusCode, string(body))
	}
}
