package middleware

import (
	"database/sql"
	"net/http"
	"strconv"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"
	"golang.org/x/time/rate"

	"github.com/tranmh/gassigeher/internal/repository"
)

// Per-tenant rate limiting with subscription tier-based limits
// Implements dual-layer limiting: tenant-wide ceiling + per-IP within tenant

// TenantRateLimitConfig defines rate limits per subscription tier
type TenantRateLimitConfig struct {
	// Tenant-wide ceiling (shared across all users in tenant)
	TenantRPS   float64
	TenantBurst int

	// Per-IP limit within tenant
	PerIPRPS   float64
	PerIPBurst int
}

// TenantRateLimitConfigs defines rate limits per subscription tier
var TenantRateLimitConfigs = map[string]TenantRateLimitConfig{
	"free": {
		TenantRPS:   30,  // 30 req/s tenant-wide
		TenantBurst: 60,  // Allow burst of 60
		PerIPRPS:    20,  // 20 req/s per IP within tenant
		PerIPBurst:  40,  // Allow burst of 40
	},
	"pro": {
		TenantRPS:   100, // 100 req/s tenant-wide
		TenantBurst: 200, // Allow burst of 200
		PerIPRPS:    50,  // 50 req/s per IP within tenant
		PerIPBurst:  100, // Allow burst of 100
	},
}

// DefaultTenantRateLimitConfig is used when tier cannot be determined
var DefaultTenantRateLimitConfig = TenantRateLimitConfigs["free"]

// tenantIPKey combines tenant ID and IP for per-IP limiting within tenant
type tenantIPKey struct {
	tenantID int
	ip       string
}

// TenantRateLimiter manages rate limits per tenant and per-IP within tenant
type TenantRateLimiter struct {
	// Tenant-wide limiters (keyed by tenant ID)
	tenantLimiters map[int]*tenantLimiterEntry
	tenantMu       sync.RWMutex

	// Per-IP within tenant limiters (keyed by tenantID:IP)
	ipLimiters map[tenantIPKey]*ipLimiterEntry
	ipMu       sync.RWMutex

	// Dependencies
	subscriptionRepo *repository.SubscriptionRepository

	// Cleanup control
	stopChan chan struct{}

	// Singleflight group to prevent duplicate DB queries during cache miss
	tierGroup singleflight.Group
}

type tenantLimiterEntry struct {
	limiter   *rate.Limiter
	tier      string
	lastSeen  time.Time
	cacheTime time.Time // When tier was last fetched from DB
}

type ipLimiterEntry struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// NewTenantRateLimiter creates a new tenant-aware rate limiter
func NewTenantRateLimiter(db *sql.DB) *TenantRateLimiter {
	trl := &TenantRateLimiter{
		tenantLimiters:   make(map[int]*tenantLimiterEntry),
		ipLimiters:       make(map[tenantIPKey]*ipLimiterEntry),
		subscriptionRepo: repository.NewSubscriptionRepository(db),
		stopChan:         make(chan struct{}),
	}

	// Start cleanup goroutine to remove stale entries
	go trl.cleanupStaleEntries()

	return trl
}

// Close stops the cleanup goroutine and releases resources
func (t *TenantRateLimiter) Close() {
	close(t.stopChan)
}

// getTenantTier fetches the subscription tier for a tenant
// Caches result for 5 minutes to avoid DB hits on every request
// Uses singleflight to prevent duplicate DB queries during concurrent cache misses
func (t *TenantRateLimiter) getTenantTier(tenantID int) string {
	// Check cache first
	t.tenantMu.RLock()
	entry, exists := t.tenantLimiters[tenantID]
	if exists && time.Since(entry.cacheTime) < 5*time.Minute {
		tier := entry.tier
		t.tenantMu.RUnlock()
		return tier
	}
	t.tenantMu.RUnlock()

	// Use singleflight to deduplicate concurrent DB queries for the same tenant
	// This prevents the "thundering herd" problem during cache miss
	key := "tier:" + strconv.Itoa(tenantID)
	result, _, _ := t.tierGroup.Do(key, func() (interface{}, error) {
		// Fetch from database
		subscription, err := t.subscriptionRepo.GetSubscriptionByTenant(tenantID)
		if err != nil || subscription == nil || subscription.Plan == nil {
			return "free", nil // Default to free tier on error
		}
		return subscription.Plan.Slug, nil
	})

	tier, ok := result.(string)
	if !ok {
		return "free"
	}
	return tier
}

// getTenantLimiter returns the tenant-wide limiter, creating if needed
func (t *TenantRateLimiter) getTenantLimiter(tenantID int) *rate.Limiter {
	tier := t.getTenantTier(tenantID)
	config, ok := TenantRateLimitConfigs[tier]
	if !ok {
		config = DefaultTenantRateLimitConfig
	}

	t.tenantMu.Lock()
	defer t.tenantMu.Unlock()

	entry, exists := t.tenantLimiters[tenantID]
	if !exists {
		entry = &tenantLimiterEntry{
			limiter:   rate.NewLimiter(rate.Limit(config.TenantRPS), config.TenantBurst),
			tier:      tier,
			lastSeen:  time.Now(),
			cacheTime: time.Now(),
		}
		t.tenantLimiters[tenantID] = entry
	} else {
		entry.lastSeen = time.Now()
		// Refresh limiter if tier changed (upgrade/downgrade)
		if entry.tier != tier {
			entry.limiter = rate.NewLimiter(rate.Limit(config.TenantRPS), config.TenantBurst)
			entry.tier = tier
			entry.cacheTime = time.Now()
		}
	}

	return entry.limiter
}

// getIPLimiter returns the per-IP limiter within a tenant, creating if needed
func (t *TenantRateLimiter) getIPLimiter(tenantID int, ip string) *rate.Limiter {
	tier := t.getTenantTier(tenantID)
	config, ok := TenantRateLimitConfigs[tier]
	if !ok {
		config = DefaultTenantRateLimitConfig
	}

	key := tenantIPKey{tenantID: tenantID, ip: ip}

	t.ipMu.Lock()
	defer t.ipMu.Unlock()

	entry, exists := t.ipLimiters[key]
	if !exists {
		entry = &ipLimiterEntry{
			limiter:  rate.NewLimiter(rate.Limit(config.PerIPRPS), config.PerIPBurst),
			lastSeen: time.Now(),
		}
		t.ipLimiters[key] = entry
	} else {
		entry.lastSeen = time.Now()
	}

	return entry.limiter
}

// cleanupStaleEntries removes rate limiter entries not seen in the last hour
func (t *TenantRateLimiter) cleanupStaleEntries() {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-t.stopChan:
			return
		case <-ticker.C:
			cutoff := time.Now().Add(-1 * time.Hour)

			// Cleanup tenant limiters
			t.tenantMu.Lock()
			for id, entry := range t.tenantLimiters {
				if entry.lastSeen.Before(cutoff) {
					delete(t.tenantLimiters, id)
				}
			}
			t.tenantMu.Unlock()

			// Cleanup IP limiters
			t.ipMu.Lock()
			for key, entry := range t.ipLimiters {
				if entry.lastSeen.Before(cutoff) {
					delete(t.ipLimiters, key)
				}
			}
			t.ipMu.Unlock()
		}
	}
}

// Singleton limiter instance (initialized in TenantRateLimit)
var tenantRateLimiterInstance *TenantRateLimiter
var tenantRateLimiterOnce sync.Once

// CloseTenantRateLimiter stops the cleanup goroutine and releases resources
// Should be called on server shutdown to prevent goroutine leaks
func CloseTenantRateLimiter() {
	if tenantRateLimiterInstance != nil {
		tenantRateLimiterInstance.Close()
	}
}

// TenantRateLimit creates middleware that enforces both tenant-wide and per-IP limits
// Must be applied after TenantMiddleware (requires tenant ID in context)
func TenantRateLimit(db *sql.DB) func(http.Handler) http.Handler {
	// Use sync.Once to ensure single instance even if called multiple times
	tenantRateLimiterOnce.Do(func() {
		tenantRateLimiterInstance = NewTenantRateLimiter(db)
	})

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tenantID := GetTenantID(r)

			// Skip rate limiting if no tenant (landing page, central admin, etc.)
			if tenantID == 0 {
				next.ServeHTTP(w, r)
				return
			}

			ip := getClientIP(r, nil)

			// Check tenant-wide limit first
			tenantLimiter := tenantRateLimiterInstance.getTenantLimiter(tenantID)
			if !tenantLimiter.Allow() {
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("Retry-After", "60")
				w.WriteHeader(http.StatusTooManyRequests)
				w.Write([]byte(`{"error":"Zu viele Anfragen für dieses Tierheim. Bitte versuchen Sie es später erneut."}`))
				return
			}

			// Check per-IP limit within tenant
			ipLimiter := tenantRateLimiterInstance.getIPLimiter(tenantID, ip)
			if !ipLimiter.Allow() {
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

// ResetTenantRateLimiter clears all rate limit state (for testing)
func ResetTenantRateLimiter() {
	if tenantRateLimiterInstance != nil {
		tenantRateLimiterInstance.tenantMu.Lock()
		tenantRateLimiterInstance.tenantLimiters = make(map[int]*tenantLimiterEntry)
		tenantRateLimiterInstance.tenantMu.Unlock()

		tenantRateLimiterInstance.ipMu.Lock()
		tenantRateLimiterInstance.ipLimiters = make(map[tenantIPKey]*ipLimiterEntry)
		tenantRateLimiterInstance.ipMu.Unlock()
	}
}

// GetTenantRateLimiterInstance returns the singleton instance (for testing)
func GetTenantRateLimiterInstance() *TenantRateLimiter {
	return tenantRateLimiterInstance
}

// InitTenantRateLimiterForTest initializes the tenant rate limiter for testing
// This allows tests to inject a mock database connection
func InitTenantRateLimiterForTest(db *sql.DB) {
	tenantRateLimiterOnce = sync.Once{} // Reset once for testing
	tenantRateLimiterOnce.Do(func() {
		tenantRateLimiterInstance = NewTenantRateLimiter(db)
	})
}
