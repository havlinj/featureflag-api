package middleware

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRecovery_whenHandlerPanics_returns500(t *testing.T) {
	handler := Recovery(slog.Default())(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic("boom")
	}))
	req := httptest.NewRequest(http.MethodGet, "/graphql", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}
}
