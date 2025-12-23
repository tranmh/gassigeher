package middleware

import (
	"net/http"
	"sync"
	"time"
)

// Auth endpoint rate limiting for registration and password reset
// Prevents abuse of sensitive auth endpoints (account enumeration, spam registration)

// authRateLimiter limits sensitive auth endpoints (register, password reset)
// Uses sliding window algorithm matching the existing login limiter pattern
type authRateLimiter struct {
	requests       map[string][]time.Time
	mu             sync.Mutex
	limit          int
	window         time.Duration
	trustedProxies map[string]bool
}

// Shared limiter for all auth endpoints (register, forgot-password, reset-password)
// Conservative limit: 3 requests per minute per IP
var authEndpointLimiter = &authRateLimiter{
	requests:       make(map[string][]time.Time),
	limit:          3,               // 3 attempts
	window:         1 * time.Minute, // per minute
	trustedProxies: make(map[string]bool),
}

// SetAuthRateLimiterTrustedProxies configures trusted proxies for auth limiter
// Only call during initialization or with proper synchronization
func SetAuthRateLimiterTrustedProxies(proxies []string) {
	authEndpointLimiter.mu.Lock()
	defer authEndpointLimiter.mu.Unlock()

	authEndpointLimiter.trustedProxies = make(map[string]bool)
	for _, proxy := range proxies {
		authEndpointLimiter.trustedProxies[proxy] = true
	}
}

// ResetAuthRateLimiter clears all rate limit state (for testing)
func ResetAuthRateLimiter() {
	authEndpointLimiter.mu.Lock()
	defer authEndpointLimiter.mu.Unlock()
	authEndpointLimiter.requests = make(map[string][]time.Time)
}

// RateLimitAuthEndpoint limits auth endpoints to 3 requests per minute per IP
// Apply to: /api/auth/register, /api/auth/forgot-password, /api/auth/reset-password
// Note: All three endpoints share the same rate limit bucket
func RateLimitAuthEndpoint(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authEndpointLimiter.mu.Lock()

		// Get client IP safely (prevents IP spoofing)
		ip := getClientIP(r, authEndpointLimiter.trustedProxies)

		now := time.Now()

		// Clean old requests outside window
		if requests, exists := authEndpointLimiter.requests[ip]; exists {
			validRequests := []time.Time{}
			for _, reqTime := range requests {
				if now.Sub(reqTime) < authEndpointLimiter.window {
					validRequests = append(validRequests, reqTime)
				}
			}
			authEndpointLimiter.requests[ip] = validRequests
		}

		// Check if limit exceeded
		if len(authEndpointLimiter.requests[ip]) >= authEndpointLimiter.limit {
			authEndpointLimiter.mu.Unlock()
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Retry-After", "60")
			w.WriteHeader(http.StatusTooManyRequests)
			w.Write([]byte(`{"error":"Zu viele Anfragen. Bitte warten Sie eine Minute."}`))
			return
		}

		// Add current request
		authEndpointLimiter.requests[ip] = append(authEndpointLimiter.requests[ip], now)
		authEndpointLimiter.mu.Unlock()

		// Call next handler without holding the lock
		next.ServeHTTP(w, r)
	})
}
