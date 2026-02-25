package graphql

import (
	"context"
	"crypto/tls"
	"net/http"
)

// Server runs the GraphQL HTTP(S) server.
type Server struct {
	srv *http.Server
}

// NewServer builds a new server that serves the given handler over TLS when tlsConfig is non-nil.
func NewServer(handler http.Handler, tlsConfig *tls.Config) *Server {
	return &Server{
		srv: &http.Server{
			Handler:   handler,
			TLSConfig: tlsConfig,
		},
	}
}

// Run starts the server on addr. Uses TLS when TLSConfig was set.
func (s *Server) Run(addr string) error {
	s.srv.Addr = addr
	if s.srv.TLSConfig != nil && len(s.srv.TLSConfig.Certificates) > 0 {
		return s.srv.ListenAndServeTLS("", "")
	}
	return s.srv.ListenAndServe()
}

// Shutdown gracefully stops the server.
func (s *Server) Shutdown(ctx context.Context) error {
	return s.srv.Shutdown(ctx)
}
