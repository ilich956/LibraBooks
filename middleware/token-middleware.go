package middleware

import (
	"context"
	"net/http"
)

// contextKey is a custom type for context keys
type contextKey string

// tokenKey is the context key for the token value
const tokenKey contextKey = "token"

// AttachTokenToRequest is a middleware that attaches token to request context
func AttachTokenToRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Retrieve token from request cookies
		cookie, err := r.Cookie("token")
		if err == nil {
			// Token found, attach it to the request context
			ctx := context.WithValue(r.Context(), tokenKey, cookie.Value)
			r = r.WithContext(ctx)
		}

		// Call the next handler
		next.ServeHTTP(w, r)
	})
}
