package middleware

import (
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// GlobalRateLimiter implements per-IP rate limiting for all endpoints
type GlobalRateLimiter struct {
	limiters  map[string]*rateLimiterEntry
	mu        sync.RWMutex
	rps       rate.Limit // requests per second
	burst     int
	stopChan  chan struct{} // Channel to signal cleanup goroutine to stop
	closeOnce sync.Once     // BUG FIX: Prevent double-close panic
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
		stopChan: make(chan struct{}),
	}

	// Start cleanup goroutine to remove stale entries
	go grl.cleanupStaleEntries()

	return grl
}

// Close stops the cleanup goroutine and releases resources (safe to call multiple times)
func (g *GlobalRateLimiter) Close() {
	g.closeOnce.Do(func() {
		close(g.stopChan)
	})
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
// BUG FIX: Uses shorter lock durations to avoid blocking requests during cleanup
func (g *GlobalRateLimiter) cleanupStaleEntries() {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-g.stopChan:
			return
		case <-ticker.C:
			cutoff := time.Now().Add(-1 * time.Hour)

			// Phase 1: Identify stale entries (read lock)
			g.mu.RLock()
			staleIPs := make([]string, 0)
			for ip, entry := range g.limiters {
				if entry.lastSeen.Before(cutoff) {
					staleIPs = append(staleIPs, ip)
				}
			}
			g.mu.RUnlock()

			// Phase 2: Delete stale entries in batches (short write locks)
			if len(staleIPs) > 0 {
				const batchSize = 100
				for i := 0; i < len(staleIPs); i += batchSize {
					end := i + batchSize
					if end > len(staleIPs) {
						end = len(staleIPs)
					}
					g.mu.Lock()
					for _, ip := range staleIPs[i:end] {
						delete(g.limiters, ip)
					}
					g.mu.Unlock()
				}
			}
		}
	}
}

// GlobalRateLimitWithCleanup creates a middleware that limits requests per IP
// Returns the middleware function AND the limiter (for cleanup on shutdown)
func GlobalRateLimitWithCleanup(rps float64, burst int) (func(http.Handler) http.Handler, *GlobalRateLimiter) {
	limiter := NewGlobalRateLimiter(rps, burst)

	middleware := func(next http.Handler) http.Handler {
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

	return middleware, limiter
}

// GlobalRateLimit creates a middleware that limits requests per IP
// DEPRECATED: Use GlobalRateLimitWithCleanup for proper resource cleanup
func GlobalRateLimit(rps float64, burst int) func(http.Handler) http.Handler {
	middleware, _ := GlobalRateLimitWithCleanup(rps, burst)
	return middleware
}
