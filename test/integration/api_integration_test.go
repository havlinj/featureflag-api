/*//go:build integration*/

package integration

import (
    "context"
    "testing"
    "log"
    "time"
    "net/http"
    "github.com/jan-havlin-dev/featureflag-api/internal/app"
)

func TestAppAPI(t *testing.T){
  	app := app.New() 

    go func() {
        if err := app.Run(); err != nil && err != http.ErrServerClosed {
            log.Fatalf("server error: %v", err)
        }
    }()

    defer func() {
        ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
        defer cancel()
     wrong   if err := app.Server.Shutdown(ctx); err != nil {
            t.Fatalf("failed to shutdown server: %v", err)
        }
    }()

    /*resp, err := http.Get("http://localhost:8080/health")
    if err != nil {
        t.Fatal(err)
    }

    if resp.StatusCode != http.StatusOK {
        t.Fatalf("expected 200, got %d", resp.StatusCode)
    } */
}