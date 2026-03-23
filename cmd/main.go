package main

import (
	"context"
	"crypto/tls"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/havlinj/featureflag-api/internal/app"
	"github.com/havlinj/featureflag-api/internal/audit"
	"github.com/havlinj/featureflag-api/internal/config"
	"github.com/havlinj/featureflag-api/internal/db"
	"github.com/havlinj/featureflag-api/internal/experiments"
	"github.com/havlinj/featureflag-api/internal/flags"
	"github.com/havlinj/featureflag-api/internal/users"
)

func main() {
	dsn := config.GetDSN(os.Getenv)
	jwtSecret, err := config.GetJWTSecret(os.Getenv)
	if err != nil {
		log.Fatal(err)
	}
	listenAddr := config.GetListenAddr(os.Getenv)

	tlsConfig, err := config.LoadTLSConfig(os.Getenv, loadTLSKeyPair)
	if err != nil {
		log.Fatalf("load TLS: %v", err)
	}

	ctx := context.Background()
	database, err := db.Open(ctx, dsn)
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	defer database.Close()

	if err := database.EnsureSchema(ctx); err != nil {
		log.Fatalf("ensure schema: %v", err)
	}

	flagsStore := flags.NewPostgresStore(database.Conn())
	usersStore := users.NewPostgresStore(database.Conn())
	experimentsStore := experiments.NewPostgresStore(database.Conn())
	auditStore := audit.NewPostgresStore(database.Conn())
	a := app.NewApp(tlsConfig, flagsStore, usersStore, experimentsStore, auditStore, []byte(jwtSecret))

	go func() {
		if err := a.Run(listenAddr); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := a.Shutdown(shutdownCtx); err != nil {
		log.Printf("shutdown: %v", err)
	}
}

func loadTLSKeyPair(certFile, keyFile string) (*tls.Config, error) {
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, err
	}
	return &tls.Config{Certificates: []tls.Certificate{cert}}, nil
}
