package main

import (
	"crypto/subtle"
	"net/http"
	"os"
)

// ProxySecretMiddleware validates the X-RapidAPI-Proxy-Secret header against the
// RAPIDAPI_PROXY_SECRET environment variable. Requests that do not carry a
// matching secret are rejected with 401 before reaching the handler.
//
// When RAPIDAPI_PROXY_SECRET is empty the middleware is a pass-through
// (development mode), allowing local testing without the RapidAPI gateway.
// The secret is read once at middleware construction time so per-request cost
// is a single header lookup plus a constant-time comparison.
//
// Mount this middleware before the rate limiter so unauthenticated requests
// never consume rate-limit quota.
func ProxySecretMiddleware() func(http.Handler) http.Handler {
	secret := os.Getenv("RAPIDAPI_PROXY_SECRET")
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Development mode: no secret configured, skip validation.
			if secret == "" {
				next.ServeHTTP(w, r)
				return
			}
			provided := r.Header.Get("X-RapidAPI-Proxy-Secret")
			if provided == "" ||
				subtle.ConstantTimeCompare([]byte(provided), []byte(secret)) != 1 {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = w.Write([]byte(`{"error":{"code":401,"message":"unauthorized: invalid or missing proxy secret","type":"unauthorized"}}`))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
