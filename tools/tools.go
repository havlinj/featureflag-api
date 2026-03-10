//go:build tools

// This file exists only so that go mod tidy (and the module graph) includes the
// dependencies of code-generation tools (e.g. gqlgen). Our application code never
// imports these packages; they are needed only when running "go run github.com/99designs/gqlgen generate".
// The blank imports below pull those tool dependencies into go.sum so that command
// can run without "missing go.sum entry" errors. The "tools" build tag ensures this
// file is not compiled into the application binary.
package tools

import (
	_ "github.com/99designs/gqlgen/api"
	_ "github.com/99designs/gqlgen/codegen/config"
	_ "github.com/99designs/gqlgen/internal/imports"
)
