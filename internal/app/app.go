package app

import (
	"context"
	"crypto/tls"
	"log/slog"
	"time"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/havlinj/featureflag-api/graph"
	"github.com/havlinj/featureflag-api/internal/audit"
	"github.com/havlinj/featureflag-api/internal/experiments"
	"github.com/havlinj/featureflag-api/internal/flags"
	"github.com/havlinj/featureflag-api/internal/users"
	"github.com/havlinj/featureflag-api/transport/graphql"
	"github.com/havlinj/featureflag-api/transport/graphql/middleware"
)

type App struct {
	Server Server
}

// NewApp builds the application. Pass non-nil tlsConfig to serve over HTTPS.
// flagsStore, usersStore, and experimentsStore are persistence layers (e.g. PostgresStore; use mocks in tests).
// jwtSecret is used to sign and verify JWTs; must be non-empty for login and protected routes.
func NewApp(
	tlsConfig *tls.Config,
	flagsStore flags.Store,
	usersStore users.Store,
	experimentsStore experiments.Store,
	auditStore audit.Store,
	jwtSecret []byte,
) *App {
	resolver := &graphql.Resolver{
		Flags:       flags.NewServiceWithAudit(flagsStore, auditStore),
		Users:       users.NewServiceWithAudit(usersStore, auditStore),
		Experiments: experiments.NewServiceWithAudit(experimentsStore, auditStore),
		Audit:       audit.NewService(auditStore),
		JWTSecret:   jwtSecret,
		JWTExpiry:   24 * time.Hour,
	}
	schema := graph.NewExecutableSchema(graph.Config{Resolvers: resolver})
	gqlHandler := handler.NewDefaultServer(schema)
	chain := middleware.Chain(gqlHandler,
		middleware.Logging(slog.Default()),
		middleware.Auth(jwtSecret),
	)
	srv := graphql.NewServer(chain, tlsConfig)
	return &App{Server: Server{GraphQLServer: srv}}
}

func (a *App) Run(addr string) error {
	return a.Server.GraphQLServer.Run(addr)
}

func (a *App) Shutdown(ctx context.Context) error {
	return a.Server.GraphQLServer.Shutdown(ctx)
}
