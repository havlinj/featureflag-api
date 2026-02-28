package middleware

import "net/http"

// Chain builds a single handler by wrapping inner with each middleware in order.
// First middleware in the slice is the outermost (runs first on request).
func Chain(inner http.Handler, mws ...func(http.Handler) http.Handler) http.Handler {
	h := inner
	for i := len(mws) - 1; i >= 0; i-- {
		h = mws[i](h)
	}
	return h
}
