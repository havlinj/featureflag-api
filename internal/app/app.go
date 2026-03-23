package app

import (
	"context"
	"crypto/tls"
	"log/slog"
	"time"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/havlinj/featureflag-api/graph"
	"github.com/havlinj/featureflag-api/internal/audit"
	"github.com/havlinj/featureflag-api/internal/experiments"
	"github.com/havlinj/featureflag-api/internal/flags"
	"github.com/havlinj/featureflag-api/internal/users"
	"github.com/havlinj/featureflag-api/transport/graphql"
	"github.com/havlinj/featureflag-api/transport/graphql/middleware"
)

type App struct {
	server Server
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
	logger := slog.Default()
	resolver := graphql.NewResolver(
		flags.NewServiceWithAudit(flagsStore, auditStore),
		users.NewServiceWithAudit(usersStore, auditStore),
		experiments.NewServiceWithAudit(experimentsStore, auditStore),
		audit.NewService(auditStore),
		jwtSecret,
		24*time.Hour,
	)
	schema := graph.NewExecutableSchema(graph.Config{Resolvers: resolver})
	gqlHandler := handler.NewDefaultServer(schema)
	gqlHandler.Use(extension.FixedComplexityLimit(200))
	chain := middleware.Chain(gqlHandler,
		middleware.Recovery(logger),
		middleware.Logging(logger),
		middleware.Auth(jwtSecret),
		middleware.BodyLimit(1<<20),
	)
	srv := graphql.NewServer(chain, tlsConfig)
	return &App{server: Server{graphQLServer: srv}}
}

func (a *App) Run(addr string) error {
	return a.server.graphQLServer.Run(addr)
}

func (a *App) Shutdown(ctx context.Context) error {
	return a.server.graphQLServer.Shutdown(ctx)
}
