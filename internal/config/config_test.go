package config

import (
	"crypto/tls"
	"errors"
	"testing"
)

func TestGetDSN_returnsDATABASE_DSNWhenSet(t *testing.T) {
	getenv := func(key string) string {
		if key == "DATABASE_DSN" {
			return "postgres://custom/db"
		}
		return ""
	}

	result := GetDSN(getenv)

	if result != "postgres://custom/db" {
		t.Errorf("expected postgres://custom/db, got %q", result)
	}
}

func TestGetDSN_buildsFromEnvWhenDATABASE_DSNEmpty(t *testing.T) {
	getenv := func(key string) string {
		m := map[string]string{
			"PGHOST": "dbhost", "PGPORT": "5433", "PGUSER": "u", "PGPASSWORD": "p", "PGDATABASE": "mydb",
		}
		return m[key]
	}

	result := GetDSN(getenv)

	expected := "postgres://u:p@dbhost:5433/mydb?sslmode=disable"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestGetDSN_usesDefaultHostWhenPGHOSTEmpty(t *testing.T) {
	getenv := func(key string) string {
		if key == "PGPORT" {
			return "5432"
		}
		if key == "PGDATABASE" {
			return "d"
		}
		return ""
	}

	result := GetDSN(getenv)

	if result != "postgres://postgres:@localhost:5432/d?sslmode=disable" {
		t.Errorf("expected localhost in DSN, got %q", result)
	}
}

func TestGetDSN_usesDefaultPortWhenPGPORTEmpty(t *testing.T) {
	getenv := func(key string) string {
		if key == "PGHOST" {
			return "h"
		}
		if key == "PGDATABASE" {
			return "d"
		}
		return ""
	}

	result := GetDSN(getenv)

	if result != "postgres://postgres:@h:5432/d?sslmode=disable" {
		t.Errorf("expected port 5432 in DSN, got %q", result)
	}
}

func TestGetDSN_usesDefaultUserWhenPGUSEREmpty(t *testing.T) {
	getenv := func(key string) string {
		if key == "PGHOST" {
			return "h"
		}
		if key == "PGPORT" {
			return "5432"
		}
		if key == "PGDATABASE" {
			return "d"
		}
		return ""
	}

	result := GetDSN(getenv)

	if result != "postgres://postgres:@h:5432/d?sslmode=disable" {
		t.Errorf("expected user postgres in DSN, got %q", result)
	}
}

func TestGetDSN_usesDefaultDatabaseWhenPGDATABASEEmpty(t *testing.T) {
	getenv := func(key string) string {
		if key == "PGHOST" {
			return "h"
		}
		if key == "PGPORT" {
			return "5432"
		}
		return ""
	}

	result := GetDSN(getenv)

	if result != "postgres://postgres:@h:5432/featureflag?sslmode=disable" {
		t.Errorf("expected database featureflag in DSN, got %q", result)
	}
}

func TestGetListenAddr_returnsDefaultWhenEmpty(t *testing.T) {
	getenv := func(string) string { return "" }

	result := GetListenAddr(getenv)

	if result != ":8080" {
		t.Errorf("expected :8080, got %q", result)
	}
}

func TestGetListenAddr_returnsValueWhenSet(t *testing.T) {
	getenv := func(key string) string {
		if key == "LISTEN_ADDR" {
			return ":9443"
		}
		return ""
	}

	result := GetListenAddr(getenv)

	if result != ":9443" {
		t.Errorf("expected :9443, got %q", result)
	}
}

func TestGetJWTSecret_returnsErrorWhenEmpty(t *testing.T) {
	getenv := func(string) string { return "" }

	_, err := GetJWTSecret(getenv)

	if err != ErrMissingJWTSecret {
		t.Errorf("expected ErrMissingJWTSecret, got %v", err)
	}
}

func TestGetJWTSecret_returnsSecretWhenSet(t *testing.T) {
	getenv := func(key string) string {
		if key == "JWT_SECRET" {
			return "my-secret"
		}
		return ""
	}

	result, err := GetJWTSecret(getenv)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "my-secret" {
		t.Errorf("expected my-secret, got %q", result)
	}
}

func TestLoadTLSConfig_returnsNilWhenCertFileEmpty(t *testing.T) {
	getenv := func(key string) string {
		if key == "TLS_KEY_FILE" {
			return "/key"
		}
		return ""
	}
	loader := func(_, _ string) (*tls.Config, error) { return &tls.Config{}, nil }

	cfg, err := LoadTLSConfig(getenv, loader)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg != nil {
		t.Errorf("expected nil config when cert file empty, got %v", cfg)
	}
}

func TestLoadTLSConfig_returnsNilWhenKeyFileEmpty(t *testing.T) {
	getenv := func(key string) string {
		if key == "TLS_CERT_FILE" {
			return "/cert"
		}
		return ""
	}
	loader := func(_, _ string) (*tls.Config, error) { return &tls.Config{}, nil }

	cfg, err := LoadTLSConfig(getenv, loader)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg != nil {
		t.Errorf("expected nil config when key file empty, got %v", cfg)
	}
}

func TestLoadTLSConfig_returnsNilWhenBothEmpty(t *testing.T) {
	getenv := func(string) string { return "" }
	loader := func(_, _ string) (*tls.Config, error) { return &tls.Config{}, nil }

	cfg, err := LoadTLSConfig(getenv, loader)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg != nil {
		t.Errorf("expected nil config, got %v", cfg)
	}
}

func TestLoadTLSConfig_returnsErrorWhenLoaderFails(t *testing.T) {
	getenv := func(key string) string {
		if key == "TLS_CERT_FILE" {
			return "/cert"
		}
		if key == "TLS_KEY_FILE" {
			return "/key"
		}
		return ""
	}
	loadErr := errors.New("file not found")
	loader := func(_, _ string) (*tls.Config, error) { return nil, loadErr }

	cfg, err := LoadTLSConfig(getenv, loader)

	if cfg != nil {
		t.Errorf("expected nil config on loader error, got %v", cfg)
	}
	if err == nil {
		t.Fatal("expected error from loader")
	}
	if !errors.Is(err, loadErr) {
		t.Errorf("expected wrapped loadErr, got %v", err)
	}
}

func TestLoadTLSConfig_returnsConfigWhenLoaderSucceeds(t *testing.T) {
	getenv := func(key string) string {
		if key == "TLS_CERT_FILE" {
			return "/cert"
		}
		if key == "TLS_KEY_FILE" {
			return "/key"
		}
		return ""
	}
	want := &tls.Config{}
	loader := func(_, _ string) (*tls.Config, error) { return want, nil }

	cfg, err := LoadTLSConfig(getenv, loader)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg != want {
		t.Errorf("expected loader result, got %v", cfg)
	}
}
