/*//go:build integration*/

package integration

import (
	"context"
	"log"
	"net/http"
	"testing"
	"time"

	"github.com/jan-havlin-dev/featureflag-api/internal/app"
	"github.com/jan-havlin-dev/featureflag-api/internal/testutil"
)

func TestAppAPI(t *testing.T) {
	addr := testutil.MakeFreeSocketAddr()
	app := app.NewApp()

	go func() {
		if err := app.Run(addr); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := app.Shutdown(ctx); err != nil {
			t.Fatalf("failed to shutdown app: %v", err)
		}
	}()

	test_client := testutil.NewClient()

	/*    resp, err := http.Get("http://localhost:8080/health")
	if err != nil {
	    t.Fatal(err)
	}

	if resp.StatusCode != http.StatusOK {
	    t.Fatalf("expected 200, got %d", resp.StatusCode)
	} */
}
