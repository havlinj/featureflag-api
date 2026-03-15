//go:build integration

package integration

import (
	"context"
	"log"
	"net/http"
	"testing"
	"time"

	"github.com/havlinj/featureflag-api/internal/app"
	"github.com/havlinj/featureflag-api/internal/db"
	"github.com/havlinj/featureflag-api/internal/experiments"
	"github.com/havlinj/featureflag-api/internal/flags"
	"github.com/havlinj/featureflag-api/internal/testutil"
	"github.com/havlinj/featureflag-api/internal/users"
	"github.com/havlinj/featureflag-api/transport/graphql"
)

// storesFromDB creates all domain stores from the same DB connection (mirrors production wiring).
func storesFromDB(database *db.DB) (flags.Store, users.Store, experiments.Store) {
	conn := database.Conn()
	return flags.NewPostgresStore(conn), users.NewPostgresStore(conn), experiments.NewPostgresStore(conn)
}

// startAppWithDB starts the app with the given database, runs the server in a goroutine,
// and returns the app, a GraphQL client, and a shutdown function. Caller must call defer shutdown().
func startAppWithDB(t *testing.T, database *db.DB) (*app.App, *testutil.GraphQLClient, func()) {
	t.Helper()
	flagsStore, usersStore, experimentsStore := storesFromDB(database)
	addr := testutil.MakeFreeSocketAddr()
	tlsConfig, err := testutil.NewTLSConfigForServer()
	if err != nil {
		t.Fatalf("create TLS config: %v", err)
	}
	jwtSecret := []byte("test-jwt-secret")
	a := app.NewApp(tlsConfig, flagsStore, usersStore, experimentsStore, jwtSecret)
	go func() {
		if err := a.Run(addr); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()
	time.Sleep(100 * time.Millisecond)
	client := testutil.NewClientForIntegration("https://" + addr)
	shutdown := func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = a.Shutdown(ctx)
	}
	return a, client, shutdown
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
