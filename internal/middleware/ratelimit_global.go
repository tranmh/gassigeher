package middleware

import (
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// GlobalRateLimiter implements per-IP rate limiting for all endpoints
type GlobalRateLimiter struct {
	limiters map[string]*rateLimiterEntry
	mu       sync.RWMutex
	rps      rate.Limit // requests per second
	burst    int
}

type rateLimiterEntry struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// NewGlobalRateLimiter creates a new global rate limiter
func NewGlobalRateLimiter(rps float64, burst int) *GlobalRateLimiter {
	grl := &GlobalRateLimiter{
		limiters: make(map[string]*rateLimiterEntry),
		rps:      rate.Limit(rps),
		burst:    burst,
	}

	// Start cleanup goroutine to remove stale entries
	go grl.cleanupStaleEntries()

	return grl
}

// GetLimiter returns the rate limiter for a given IP
func (g *GlobalRateLimiter) GetLimiter(ip string) *rate.Limiter {
	g.mu.Lock()
	defer g.mu.Unlock()

	entry, exists := g.limiters[ip]
	if !exists {
		entry = &rateLimiterEntry{
			limiter:  rate.NewLimiter(g.rps, g.burst),
			lastSeen: time.Now(),
		}
		g.limiters[ip] = entry
	} else {
		entry.lastSeen = time.Now()
	}

	return entry.limiter
}

// cleanupStaleEntries removes rate limiter entries not seen in the last hour
func (g *GlobalRateLimiter) cleanupStaleEntries() {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		g.mu.Lock()
		cutoff := time.Now().Add(-1 * time.Hour)
		for ip, entry := range g.limiters {
			if entry.lastSeen.Before(cutoff) {
				delete(g.limiters, ip)
			}
		}
		g.mu.Unlock()
	}
}

// GlobalRateLimit creates a middleware that limits requests per IP
func GlobalRateLimit(rps float64, burst int) func(http.Handler) http.Handler {
	limiter := NewGlobalRateLimiter(rps, burst)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Use existing getClientIP function with empty trusted proxies map
			ip := getClientIP(r, nil)

			if !limiter.GetLimiter(ip).Allow() {
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("Retry-After", "60")
				w.WriteHeader(http.StatusTooManyRequests)
				w.Write([]byte(`{"error":"Zu viele Anfragen. Bitte warten Sie einen Moment."}`))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
