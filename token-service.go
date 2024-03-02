// middleware.go

package main

import (
	"context"
	"net/http"
)

// TokenMiddleware is a middleware that extracts the token from the request
// and adds it to the request context.
func TokenMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract the token from the request (implement your token extraction logic)
		token := extractToken(r)

		// Add the token to the request context
		ctx := context.WithValue(r.Context(), "token", token)
		r = r.WithContext(ctx)

		// Call the next handler
		next.ServeHTTP(w, r)
	})
}

// extractToken is a placeholder function for extracting the token from the request.
func extractToken(r *http.Request) string {
	// Implement your token extraction logic here (from headers, cookies, etc.)
	// For demonstration purposes, returning a dummy token "abc123"
	return "abc123"
}
