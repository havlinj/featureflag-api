package app

import (
	"context"
	"crypto/tls"

	"github.com/jan-havlin-dev/featureflag-api/graph"
	"github.com/jan-havlin-dev/featureflag-api/internal/flags"
	"github.com/jan-havlin-dev/featureflag-api/transport/graphql"
	"github.com/99designs/gqlgen/graphql/handler"
)

type App struct {
	Server Server
}

// NewApp builds the application. Pass non-nil tlsConfig to serve over HTTPS.
// flagsStore is the persistence for flags (e.g. flags.NewPostgresStore(db.Conn()) after db.Open and db.EnsureSchema).
func NewApp(tlsConfig *tls.Config, flagsStore flags.Store) *App {
	resolver := &graphql.Resolver{Flags: flags.NewService(flagsStore)}
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
