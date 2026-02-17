/*//go:build integration*/

package integration

import (
    "context"
    "testing"
    "net/http"
    "github.com/jan-havlin-dev/featureflag-api/internal/app"
)

func TestAppAPI(t *testing.T){
  	a := app.New() 
wrire correctly
    go a.Server.ListenAndServe()
    defer a.Server.Shutdown(context.Background())

    resp, err := http.Get("http://localhost:8080/health")
    if err != nil {
        t.Fatal(err)
    }

    if resp.StatusCode != http.StatusOK {
        t.Fatalf("expected 200, got %d", resp.StatusCode)
    } */
}