package middleware

import (
	"crypto/subtle"
	"net/http"
)

// MetricsBasicAuth creates a middleware that protects the metrics endpoint with Basic Auth
// This is the standard way to secure Prometheus /metrics endpoints
func MetricsBasicAuth(username, password string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// If no password is configured, allow access (for development)
			if password == "" {
				next.ServeHTTP(w, r)
				return
			}

			// Get credentials from request
			user, pass, ok := r.BasicAuth()
			if !ok {
				// No credentials provided - request authentication
				w.Header().Set("WWW-Authenticate", `Basic realm="Prometheus Metrics"`)
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			// Constant-time comparison to prevent timing attacks
			userMatch := subtle.ConstantTimeCompare([]byte(user), []byte(username)) == 1
			passMatch := subtle.ConstantTimeCompare([]byte(pass), []byte(password)) == 1

			if !userMatch || !passMatch {
				// Invalid credentials
				w.Header().Set("WWW-Authenticate", `Basic realm="Prometheus Metrics"`)
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			// Credentials valid - proceed
			next.ServeHTTP(w, r)
		})
	}
}

// WrapWithMetricsAuth wraps a handler with Basic Auth protection for metrics
// This is a convenience function for protecting individual handlers
func WrapWithMetricsAuth(handler http.HandlerFunc, username, password string) http.HandlerFunc {
	if password == "" {
		// No password configured - return handler as-is (for development)
		return handler
	}

	return func(w http.ResponseWriter, r *http.Request) {
		// Get credentials from request
		user, pass, ok := r.BasicAuth()
		if !ok {
			w.Header().Set("WWW-Authenticate", `Basic realm="Prometheus Metrics"`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Constant-time comparison to prevent timing attacks
		userMatch := subtle.ConstantTimeCompare([]byte(user), []byte(username)) == 1
		passMatch := subtle.ConstantTimeCompare([]byte(pass), []byte(password)) == 1

		if !userMatch || !passMatch {
			w.Header().Set("WWW-Authenticate", `Basic realm="Prometheus Metrics"`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		handler(w, r)
	}
}
