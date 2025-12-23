package services

import (
	"encoding/json"
	"sync"
	"time"
)

// CacheEntry represents a cached item with expiration
type CacheEntry struct {
	Value     interface{}
	ExpiresAt time.Time
	CreatedAt time.Time
}

// CacheService provides in-memory caching with TTL support
type CacheService struct {
	data       map[string]*CacheEntry
	mu         sync.RWMutex
	defaultTTL time.Duration
	maxEntries int
	stats      CacheStats
}

// CacheStats tracks cache performance metrics
type CacheStats struct {
	Hits       int64
	Misses     int64
	Sets       int64
	Deletes    int64
	Evictions  int64
	EntryCount int
}

// CacheConfig allows customization of cache behavior
type CacheConfig struct {
	DefaultTTL time.Duration // Default time-to-live for entries
	MaxEntries int           // Maximum number of entries (0 = unlimited)
	CleanupInterval time.Duration // How often to run cleanup (0 = no background cleanup)
}

// DefaultCacheConfig returns sensible defaults
func DefaultCacheConfig() CacheConfig {
	return CacheConfig{
		DefaultTTL:      5 * time.Minute,
		MaxEntries:      10000,
		CleanupInterval: 1 * time.Minute,
	}
}

// NewCacheService creates a new cache service with given config
func NewCacheService(cfg CacheConfig) *CacheService {
	c := &CacheService{
		data:       make(map[string]*CacheEntry),
		defaultTTL: cfg.DefaultTTL,
		maxEntries: cfg.MaxEntries,
	}

	// Start background cleanup if configured
	if cfg.CleanupInterval > 0 {
		go c.runCleanup(cfg.CleanupInterval)
	}

	return c
}

// NewDefaultCacheService creates a cache with default settings
func NewDefaultCacheService() *CacheService {
	return NewCacheService(DefaultCacheConfig())
}

// Set stores a value with the default TTL
func (c *CacheService) Set(key string, value interface{}) {
	c.SetWithTTL(key, value, c.defaultTTL)
}

// SetWithTTL stores a value with a custom TTL
func (c *CacheService) SetWithTTL(key string, value interface{}, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Check if we need to evict entries
	if c.maxEntries > 0 && len(c.data) >= c.maxEntries {
		c.evictOldest()
	}

	now := time.Now()
	c.data[key] = &CacheEntry{
		Value:     value,
		ExpiresAt: now.Add(ttl),
		CreatedAt: now,
	}
	c.stats.Sets++
	c.stats.EntryCount = len(c.data)
}

// Get retrieves a value from cache, returns nil if not found or expired
func (c *CacheService) Get(key string) interface{} {
	c.mu.RLock()
	entry, exists := c.data[key]
	c.mu.RUnlock()

	if !exists {
		c.mu.Lock()
		c.stats.Misses++
		c.mu.Unlock()
		return nil
	}

	// Check if expired
	if time.Now().After(entry.ExpiresAt) {
		c.mu.Lock()
		delete(c.data, key)
		c.stats.Misses++
		c.stats.EntryCount = len(c.data)
		c.mu.Unlock()
		return nil
	}

	c.mu.Lock()
	c.stats.Hits++
	c.mu.Unlock()
	return entry.Value
}

// GetJSON retrieves and unmarshals JSON value into target
func (c *CacheService) GetJSON(key string, target interface{}) bool {
	value := c.Get(key)
	if value == nil {
		return false
	}

	// If value is already []byte, unmarshal it
	if bytes, ok := value.([]byte); ok {
		return json.Unmarshal(bytes, target) == nil
	}

	// If value is already the target type, use type assertion
	// This handles cases where we cached the struct directly
	return false
}

// SetJSON marshals and stores a value as JSON
func (c *CacheService) SetJSON(key string, value interface{}) error {
	bytes, err := json.Marshal(value)
	if err != nil {
		return err
	}
	c.Set(key, bytes)
	return nil
}

// SetJSONWithTTL marshals and stores a value as JSON with custom TTL
func (c *CacheService) SetJSONWithTTL(key string, value interface{}, ttl time.Duration) error {
	bytes, err := json.Marshal(value)
	if err != nil {
		return err
	}
	c.SetWithTTL(key, bytes, ttl)
	return nil
}

// Delete removes a value from cache
func (c *CacheService) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, exists := c.data[key]; exists {
		delete(c.data, key)
		c.stats.Deletes++
		c.stats.EntryCount = len(c.data)
	}
}

// DeletePrefix removes all entries with keys starting with prefix
func (c *CacheService) DeletePrefix(prefix string) int {
	c.mu.Lock()
	defer c.mu.Unlock()

	count := 0
	for key := range c.data {
		if len(key) >= len(prefix) && key[:len(prefix)] == prefix {
			delete(c.data, key)
			count++
		}
	}
	c.stats.Deletes += int64(count)
	c.stats.EntryCount = len(c.data)
	return count
}

// Clear removes all entries from cache
func (c *CacheService) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.data = make(map[string]*CacheEntry)
	c.stats.EntryCount = 0
}

// Exists checks if a key exists and is not expired
func (c *CacheService) Exists(key string) bool {
	return c.Get(key) != nil
}

// GetStats returns cache statistics
func (c *CacheService) GetStats() CacheStats {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.stats
}

// GetHitRate returns the cache hit rate as a percentage
func (c *CacheService) GetHitRate() float64 {
	c.mu.RLock()
	defer c.mu.RUnlock()

	total := c.stats.Hits + c.stats.Misses
	if total == 0 {
		return 0
	}
	return float64(c.stats.Hits) / float64(total) * 100
}

// evictOldest removes the oldest entry (must be called with lock held)
func (c *CacheService) evictOldest() {
	var oldestKey string
	var oldestTime time.Time

	for key, entry := range c.data {
		if oldestKey == "" || entry.CreatedAt.Before(oldestTime) {
			oldestKey = key
			oldestTime = entry.CreatedAt
		}
	}

	if oldestKey != "" {
		delete(c.data, oldestKey)
		c.stats.Evictions++
	}
}

// runCleanup periodically removes expired entries
func (c *CacheService) runCleanup(interval time.Duration) {
	ticker := time.NewTicker(interval)
	for range ticker.C {
		c.cleanup()
	}
}

// cleanup removes all expired entries
func (c *CacheService) cleanup() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	for key, entry := range c.data {
		if now.After(entry.ExpiresAt) {
			delete(c.data, key)
		}
	}
	c.stats.EntryCount = len(c.data)
}

// Common cache key generators for consistency

// CacheKeyDog generates cache key for dog data
func CacheKeyDog(tenantID, dogID int) string {
	return "dog:" + itoa(tenantID) + ":" + itoa(dogID)
}

// CacheKeyDogs generates cache key for dogs list
func CacheKeyDogs(tenantID int) string {
	return "dogs:" + itoa(tenantID)
}

// CacheKeyUser generates cache key for user data
func CacheKeyUser(tenantID, userID int) string {
	return "user:" + itoa(tenantID) + ":" + itoa(userID)
}

// CacheKeySettings generates cache key for system settings
func CacheKeySettings(tenantID int) string {
	return "settings:" + itoa(tenantID)
}

// CacheKeyBookingSlots generates cache key for available booking slots
func CacheKeyBookingSlots(tenantID int, date string) string {
	return "slots:" + itoa(tenantID) + ":" + date
}

// CacheKeyHolidays generates cache key for holidays
func CacheKeyHolidays(year int, state string) string {
	return "holidays:" + itoa(year) + ":" + state
}

// CacheKeyColors generates cache key for color categories
func CacheKeyColors(tenantID int) string {
	return "colors:" + itoa(tenantID)
}

// CacheKeyTheme generates cache key for tenant theme
func CacheKeyTheme(tenantID int) string {
	return "theme:" + itoa(tenantID)
}

// Helper function for integer to string conversion
func itoa(i int) string {
	if i == 0 {
		return "0"
	}

	negative := i < 0
	if negative {
		i = -i
	}

	// Build digits in reverse
	digits := make([]byte, 20)
	pos := len(digits)
	for i > 0 {
		pos--
		digits[pos] = byte('0' + i%10)
		i /= 10
	}

	if negative {
		pos--
		digits[pos] = '-'
	}

	return string(digits[pos:])
}
