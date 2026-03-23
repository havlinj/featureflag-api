//go:build integration

package integration

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/havlinj/featureflag-api/internal/app"
	"github.com/havlinj/featureflag-api/internal/audit"
	"github.com/havlinj/featureflag-api/internal/db"
	"github.com/havlinj/featureflag-api/internal/experiments"
	"github.com/havlinj/featureflag-api/internal/flags"
	"github.com/havlinj/featureflag-api/internal/testutil"
	"github.com/havlinj/featureflag-api/internal/users"
	"github.com/havlinj/featureflag-api/transport/graphql"
)

// storesFromDB creates all domain stores from the same DB connection (mirrors production wiring).
func storesFromDB(database *db.DB) (flags.Store, users.Store, experiments.Store, audit.Store) {
	conn := database.Conn()
	return flags.NewPostgresStore(conn), users.NewPostgresStore(conn), experiments.NewPostgresStore(conn), audit.NewPostgresStore(conn)
}

// startAppWithDB starts the app with the given database, runs the server in a goroutine,
// and returns the app, a GraphQL client, and a shutdown function. Caller must call defer shutdown().
func startAppWithDB(t *testing.T, database *db.DB) (*app.App, *testutil.GraphQLClient, func()) {
	t.Helper()
	flagsStore, usersStore, experimentsStore, auditStore := storesFromDB(database)
	addr := testutil.MakeFreeSocketAddr()
	tlsConfig, err := testutil.NewTLSConfigForServer()
	if err != nil {
		t.Fatalf("create TLS config: %v", err)
	}
	jwtSecret := []byte("test-jwt-secret")
	a := app.NewApp(tlsConfig, flagsStore, usersStore, experimentsStore, auditStore, jwtSecret)
	go func() {
		if err := a.Run(addr); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()
	client := testutil.NewClientForIntegration("https://" + addr)
	waitForGraphQLReady(t, client)
	shutdown := func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = a.Shutdown(ctx)
	}
	return a, client, shutdown
}

func waitForGraphQLReady(t *testing.T, client *testutil.GraphQLClient) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := client.DoRequest(`query Ready { __typename }`, nil)
		if err == nil && resp != nil && len(resp.Errors) == 0 {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("server did not become ready before timeout: %s", time.Now().Format(time.RFC3339))
}

// requireDataAndNoErrors fails the test if the GraphQL response has no data or has errors.
func requireDataAndNoErrors(t *testing.T, resp *graphql.GraphQLResponse) {
	t.Helper()
	if resp.Data == nil || (resp.Errors != nil && len(resp.Errors) > 0) {
		t.Fatalf("expected data and no errors, got data=%v errors=%v", resp.Data, resp.Errors)
	}
}

// requireGraphQLErrors fails the test if the GraphQL response has no errors.
func requireGraphQLErrors(t *testing.T, resp *graphql.GraphQLResponse) {
	t.Helper()
	if resp.Errors == nil || len(resp.Errors) == 0 {
		t.Fatal("expected GraphQL errors, got none")
	}
}

func graphqlErrorMessages(resp *graphql.GraphQLResponse) string {
	if resp == nil || len(resp.Errors) == 0 {
		return ""
	}
	parts := make([]string, 0, len(resp.Errors))
	for _, e := range resp.Errors {
		parts = append(parts, fmt.Sprintf("%v", e))
	}
	return strings.Join(parts, " | ")
}
