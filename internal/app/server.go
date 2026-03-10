package app

import "github.com/havlinj/featureflag-api/transport/graphql"

type Server struct {
	GraphQLServer *graphql.Server
}
