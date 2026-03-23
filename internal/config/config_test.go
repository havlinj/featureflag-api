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

	expected := "postgres://u:p@dbhost:5433/mydb?sslmode=require"
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

	if result != "postgres://postgres:@localhost:5432/d?sslmode=require" {
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

	if result != "postgres://postgres:@h:5432/d?sslmode=require" {
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

	if result != "postgres://postgres:@h:5432/d?sslmode=require" {
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

	if result != "postgres://postgres:@h:5432/featureflag?sslmode=require" {
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

	var e *MissingJWTSecretError
	if !errors.As(err, &e) {
		t.Errorf("expected *MissingJWTSecretError, got %v", err)
	}
	if e.EnvVar != "JWT_SECRET" {
		t.Errorf("expected EnvVar=JWT_SECRET, got %q", e.EnvVar)
	}
}

func TestMissingJWTSecretError_Error_full_message(t *testing.T) {
	e := &MissingJWTSecretError{EnvVar: "JWT_SECRET"}
	got := e.Error()
	want := "config: JWT_SECRET must be set (empty or unset)"
	if got != want {
		t.Errorf("Error() = %q; want %q", got, want)
	}
}

func TestGetJWTSecret_returnsSecretWhenSet(t *testing.T) {
	getenv := func(key string) string {
		if key == "JWT_SECRET" {
			return "my-secret-at-least-32-bytes-long"
		}
		return ""
	}

	result, err := GetJWTSecret(getenv)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "my-secret-at-least-32-bytes-long" {
		t.Errorf("expected long secret, got %q", result)
	}
}

func TestGetJWTSecret_returnsErrorWhenTooShort(t *testing.T) {
	getenv := func(key string) string {
		if key == "JWT_SECRET" {
			return "short-secret"
		}
		return ""
	}

	_, err := GetJWTSecret(getenv)

	var e *WeakJWTSecretError
	if !errors.As(err, &e) {
		t.Errorf("expected *WeakJWTSecretError, got %v", err)
	}
	if e.MinLength != minJWTSecretLength {
		t.Errorf("expected MinLength=%d, got %d", minJWTSecretLength, e.MinLength)
	}
}

func TestWeakJWTSecretError_Error_exact_string(t *testing.T) {
	e := &WeakJWTSecretError{MinLength: 32, ActualLength: 8}

	got := e.Error()

	const want = "config: JWT_SECRET too short (min=32 actual=8)"
	if got != want {
		t.Errorf("Error() = %q, want %q", got, want)
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
