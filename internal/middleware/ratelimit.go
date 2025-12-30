package middleware

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

// BUG FIX #6: Simple rate limiting for login endpoint
// Prevents brute force attacks

type rateLimiter struct {
	requests       map[string][]time.Time
	mu             sync.Mutex
	limit          int
	window         time.Duration
	trustedProxies map[string]bool // IP addresses of trusted proxies
	stopChan       chan struct{}   // Channel to signal cleanup goroutine to stop
	stopOnce       sync.Once       // Prevent double-close panic
}

var loginLimiter = &rateLimiter{
	requests:       make(map[string][]time.Time),
	limit:          5,               // 5 attempts
	window:         1 * time.Minute, // per minute
	trustedProxies: make(map[string]bool),
	stopChan:       make(chan struct{}),
}

func init() {
	// Start cleanup goroutine for login rate limiter
	go loginLimiter.cleanupStaleEntries()
}

// SetTrustedProxies configures which proxy IPs are trusted for X-Forwarded-For
// Only call during initialization or with proper synchronization
func SetTrustedProxies(proxies []string) {
	loginLimiter.mu.Lock()
	defer loginLimiter.mu.Unlock()

	loginLimiter.trustedProxies = make(map[string]bool)
	for _, proxy := range proxies {
		loginLimiter.trustedProxies[proxy] = true
	}
}

// ResetRateLimiter clears all rate limit state (for testing)
func ResetRateLimiter() {
	loginLimiter.mu.Lock()
	defer loginLimiter.mu.Unlock()
	loginLimiter.requests = make(map[string][]time.Time)
}

// CloseLoginRateLimiter stops the cleanup goroutine (call on server shutdown)
func CloseLoginRateLimiter() {
	loginLimiter.stopOnce.Do(func() {
		close(loginLimiter.stopChan)
	})
}

// cleanupStaleEntries removes rate limiter entries not seen in the last 2x window
// This prevents memory leaks from accumulating IP entries
func (r *rateLimiter) cleanupStaleEntries() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-r.stopChan:
			return
		case <-ticker.C:
			r.mu.Lock()
			now := time.Now()
			// Remove entries where all requests are older than 2x window
			cutoff := now.Add(-2 * r.window)
			for ip, requests := range r.requests {
				// Check if all requests are old
				allOld := true
				for _, reqTime := range requests {
					if reqTime.After(cutoff) {
						allOld = false
						break
					}
				}
				if allOld {
					delete(r.requests, ip)
				}
			}
			r.mu.Unlock()
		}
	}
}

// getClientIP extracts the client IP address safely
// Only trusts X-Forwarded-For when the immediate connection is from a trusted proxy
func getClientIP(r *http.Request, trustedProxies map[string]bool) string {
	// Extract IP from RemoteAddr (format: "IP:port" or just "IP")
	remoteIP := r.RemoteAddr
	if host, _, err := net.SplitHostPort(remoteIP); err == nil {
		remoteIP = host
	}

	// Only trust X-Forwarded-For if the request comes from a trusted proxy
	if len(trustedProxies) > 0 && trustedProxies[remoteIP] {
		if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
			// X-Forwarded-For can be comma-separated list: "client, proxy1, proxy2"
			// The first IP is the original client (if we trust the chain)
			ips := strings.Split(forwarded, ",")
			if len(ips) > 0 {
				clientIP := strings.TrimSpace(ips[0])
				if clientIP != "" {
					return clientIP
				}
			}
		}
	}

	// Default: use RemoteAddr (prevents IP spoofing when not behind trusted proxy)
	return remoteIP
}

// RateLimitLogin limits login attempts per IP address
func RateLimitLogin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		loginLimiter.mu.Lock()

		// Get client IP safely (prevents IP spoofing)
		ip := getClientIP(r, loginLimiter.trustedProxies)

		now := time.Now()

		// Clean old requests outside window
		if requests, exists := loginLimiter.requests[ip]; exists {
			validRequests := []time.Time{}
			for _, reqTime := range requests {
				if now.Sub(reqTime) < loginLimiter.window {
					validRequests = append(validRequests, reqTime)
				}
			}
			loginLimiter.requests[ip] = validRequests
		}

		// Check if limit exceeded
		if len(loginLimiter.requests[ip]) >= loginLimiter.limit {
			loginLimiter.mu.Unlock()
			http.Error(w, `{"error":"Zu viele Anmeldeversuche. Bitte versuchen Sie es in einer Minute erneut."}`, http.StatusTooManyRequests)
			return
		}

		// Add current request
		loginLimiter.requests[ip] = append(loginLimiter.requests[ip], now)
		loginLimiter.mu.Unlock()

		// Call next handler without holding the lock
		next.ServeHTTP(w, r)
	})
}

// DONE: BUG #6 FIXED - Rate limiting with IP spoofing prevention
