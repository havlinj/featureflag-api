package middleware

import (
	"net/http"
	"strings"

	"github.com/havlinj/featureflag-api/internal/auth"
)

// Auth returns middleware that reads Authorization: Bearer <token>, validates the JWT,
// and sets claims in the request context. If the header is present but the token
// is invalid, it responds with 401 and does not call the next handler.
// If the header is missing, the request continues with no user in context.
func Auth(secret []byte) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			if header == "" {
				next.ServeHTTP(w, r)
				return
			}
			prefix := "Bearer "
			if !strings.HasPrefix(header, prefix) {
				writeUnauthorized(w)
				return
			}
			tokenString := strings.TrimPrefix(header, prefix)
			claims, err := auth.ParseAndValidate(tokenString, secret)
			if err != nil {
				writeUnauthorized(w)
				return
			}
			ctx := auth.WithClaims(r.Context(), claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func writeUnauthorized(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	_, _ = w.Write([]byte(`{"error":"unauthorized"}`))
}
