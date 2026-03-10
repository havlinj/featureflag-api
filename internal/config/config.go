package config

import (
	"crypto/tls"
	"errors"
	"fmt"
)

// GetDSN returns the database DSN. If getenv("DATABASE_DSN") is non-empty, it returns that.
// Otherwise it builds a DSN from PGHOST, PGPORT, PGUSER, PGPASSWORD, PGDATABASE with defaults.
func GetDSN(getenv func(string) string) string {
	if s := getenv("DATABASE_DSN"); s != "" {
		return s
	}
	return buildDSNFromEnv(getenv)
}

func buildDSNFromEnv(getenv func(string) string) string {
	host := getenv("PGHOST")
	if host == "" {
		host = "localhost"
	}
	port := getenv("PGPORT")
	if port == "" {
		port = "5432"
	}
	user := getenv("PGUSER")
	if user == "" {
		user = "postgres"
	}
	password := getenv("PGPASSWORD")
	dbname := getenv("PGDATABASE")
	if dbname == "" {
		dbname = "featureflag"
	}
	return "postgres://" + user + ":" + password + "@" + host + ":" + port + "/" + dbname + "?sslmode=disable"
}

// GetListenAddr returns the listen address. Defaults to ":8080" if getenv("LISTEN_ADDR") is empty.
func GetListenAddr(getenv func(string) string) string {
	addr := getenv("LISTEN_ADDR")
	if addr == "" {
		return ":8080"
	}
	return addr
}

// ErrMissingJWTSecret is returned when JWT_SECRET is not set.
var ErrMissingJWTSecret = errors.New("JWT_SECRET must be set")

// GetJWTSecret returns the JWT secret from getenv("JWT_SECRET"). Returns ErrMissingJWTSecret if empty.
func GetJWTSecret(getenv func(string) string) (string, error) {
	s := getenv("JWT_SECRET")
	if s == "" {
		return "", ErrMissingJWTSecret
	}
	return s, nil
}

// TLSConfigLoader loads a tls.Config from cert and key file paths.
// Used so callers can inject a loader (e.g. for tests).
type TLSConfigLoader func(certFile, keyFile string) (*tls.Config, error)

// LoadTLSConfig returns a TLS config when getenv("TLS_CERT_FILE") and getenv("TLS_KEY_FILE") are both set;
// otherwise returns nil, nil. Uses loader to load the key pair. If loader returns an error, it is wrapped.
func LoadTLSConfig(getenv func(string) string, loader TLSConfigLoader) (*tls.Config, error) {
	certFile := getenv("TLS_CERT_FILE")
	keyFile := getenv("TLS_KEY_FILE")
	if certFile == "" || keyFile == "" {
		return nil, nil
	}
	cfg, err := loader(certFile, keyFile)
	if err != nil {
		return nil, fmt.Errorf("load TLS key pair: %w", err)
	}
	return cfg, nil
}
