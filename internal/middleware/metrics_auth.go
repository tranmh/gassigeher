package middleware

import (
	"crypto/subtle"
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// metricsRateLimiter provides per-IP rate limiting for metrics endpoint
// Prevents brute force attacks on Basic Auth credentials
var (
	metricsLimiters   = make(map[string]*metricsLimiterEntry)
	metricsLimitersMu sync.RWMutex
)

type metricsLimiterEntry struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// getMetricsLimiter returns a rate limiter for the given IP
// Limits to 10 requests per minute with burst of 5 (prevents brute force)
func getMetricsLimiter(ip string) *rate.Limiter {
	metricsLimitersMu.Lock()
	defer metricsLimitersMu.Unlock()

	entry, exists := metricsLimiters[ip]
	if !exists {
		// 10 requests per minute = 0.167 per second, burst of 5
		entry = &metricsLimiterEntry{
			limiter:  rate.NewLimiter(rate.Every(6*time.Second), 5),
			lastSeen: time.Now(),
		}
		metricsLimiters[ip] = entry
	} else {
		entry.lastSeen = time.Now()
	}

	return entry.limiter
}

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
// Includes rate limiting to prevent brute force attacks (10 req/min per IP)
func WrapWithMetricsAuth(handler http.HandlerFunc, username, password string) http.HandlerFunc {
	if password == "" {
		// No password configured - return handler as-is (for development)
		return handler
	}

	return func(w http.ResponseWriter, r *http.Request) {
		// Rate limiting to prevent brute force attacks
		ip := getClientIP(r, nil)
		if !getMetricsLimiter(ip).Allow() {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Retry-After", "60")
			w.WriteHeader(http.StatusTooManyRequests)
			w.Write([]byte(`{"error":"Too many requests to metrics endpoint"}`))
			return
		}

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
