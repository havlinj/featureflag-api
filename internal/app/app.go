package app

import (
	"context"
	"crypto/tls"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/jan-havlin-dev/featureflag-api/graph"
	"github.com/jan-havlin-dev/featureflag-api/internal/flags"
	"github.com/jan-havlin-dev/featureflag-api/internal/users"
	"github.com/jan-havlin-dev/featureflag-api/transport/graphql"
)

type App struct {
	Server Server
}

// NewApp builds the application. Pass non-nil tlsConfig to serve over HTTPS.
// flagsStore and usersStore are persistence layers (e.g. PostgresStore; use mocks in tests).
func NewApp(tlsConfig *tls.Config, flagsStore flags.Store, usersStore users.Store) *App {
	resolver := &graphql.Resolver{
		Flags: flags.NewService(flagsStore),
		Users: users.NewService(usersStore),
	}
	schema := graph.NewExecutableSchema(graph.Config{Resolvers: resolver})
	h := handler.NewDefaultServer(schema)
	srv := graphql.NewServer(h, tlsConfig)
	return &App{Server: Server{GraphQLServer: srv}}
}

func (a *App) Run(addr string) error {
	return a.Server.GraphQLServer.Run(addr)
}

func (a *App) Shutdown(ctx context.Context) error {
	return a.Server.GraphQLServer.Shutdown(ctx)
}
