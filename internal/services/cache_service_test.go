package services

import (
	"testing"
	"time"
)

// ============================================================================
// CacheConfig Tests
// ============================================================================

func TestDefaultCacheConfig(t *testing.T) {
	cfg := DefaultCacheConfig()

	if cfg.DefaultTTL != 5*time.Minute {
		t.Errorf("DefaultTTL = %v, want 5m", cfg.DefaultTTL)
	}

	if cfg.MaxEntries != 10000 {
		t.Errorf("MaxEntries = %d, want 10000", cfg.MaxEntries)
	}

	if cfg.CleanupInterval != 1*time.Minute {
		t.Errorf("CleanupInterval = %v, want 1m", cfg.CleanupInterval)
	}
}

// ============================================================================
// CacheService Constructor Tests
// ============================================================================

func TestNewCacheService(t *testing.T) {
	cfg := CacheConfig{
		DefaultTTL:      10 * time.Second,
		MaxEntries:      100,
		CleanupInterval: 0, // Disable cleanup for tests
	}

	cache := NewCacheService(cfg)
	defer cache.Close()

	if cache == nil {
		t.Fatal("NewCacheService returned nil")
	}

	if cache.defaultTTL != 10*time.Second {
		t.Errorf("defaultTTL = %v, want 10s", cache.defaultTTL)
	}

	if cache.maxEntries != 100 {
		t.Errorf("maxEntries = %d, want 100", cache.maxEntries)
	}
}

func TestNewDefaultCacheService(t *testing.T) {
	cache := NewDefaultCacheService()
	defer cache.Close()

	if cache == nil {
		t.Fatal("NewDefaultCacheService returned nil")
	}
}

// ============================================================================
// Basic Operations Tests
// ============================================================================

func TestCacheService_SetAndGet(t *testing.T) {
	cache := NewCacheService(CacheConfig{
		DefaultTTL:      1 * time.Minute,
		CleanupInterval: 0,
	})
	defer cache.Close()

	cache.Set("key1", "value1")

	result := cache.Get("key1")
	if result != "value1" {
		t.Errorf("Get(key1) = %v, want value1", result)
	}
}

func TestCacheService_GetNonExistent(t *testing.T) {
	cache := NewCacheService(CacheConfig{
		DefaultTTL:      1 * time.Minute,
		CleanupInterval: 0,
	})
	defer cache.Close()

	result := cache.Get("nonexistent")
	if result != nil {
		t.Errorf("Get(nonexistent) = %v, want nil", result)
	}
}

func TestCacheService_SetWithTTL(t *testing.T) {
	cache := NewCacheService(CacheConfig{
		DefaultTTL:      1 * time.Minute,
		CleanupInterval: 0,
	})
	defer cache.Close()

	cache.SetWithTTL("key", "value", 100*time.Millisecond)

	// Should exist immediately
	result := cache.Get("key")
	if result != "value" {
		t.Errorf("Get(key) immediately = %v, want value", result)
	}

	// Should expire after TTL
	time.Sleep(150 * time.Millisecond)
	result = cache.Get("key")
	if result != nil {
		t.Errorf("Get(key) after expiration = %v, want nil", result)
	}
}

func TestCacheService_Delete(t *testing.T) {
	cache := NewCacheService(CacheConfig{
		DefaultTTL:      1 * time.Minute,
		CleanupInterval: 0,
	})
	defer cache.Close()

	cache.Set("key", "value")
	cache.Delete("key")

	result := cache.Get("key")
	if result != nil {
		t.Errorf("Get(key) after delete = %v, want nil", result)
	}
}

func TestCacheService_DeleteNonExistent(t *testing.T) {
	cache := NewCacheService(CacheConfig{
		DefaultTTL:      1 * time.Minute,
		CleanupInterval: 0,
	})
	defer cache.Close()

	// Should not panic
	cache.Delete("nonexistent")
}

func TestCacheService_DeletePrefix(t *testing.T) {
	cache := NewCacheService(CacheConfig{
		DefaultTTL:      1 * time.Minute,
		CleanupInterval: 0,
	})
	defer cache.Close()

	cache.Set("user:1", "alice")
	cache.Set("user:2", "bob")
	cache.Set("dog:1", "max")

	deleted := cache.DeletePrefix("user:")
	if deleted != 2 {
		t.Errorf("DeletePrefix(user:) deleted %d, want 2", deleted)
	}

	if cache.Get("user:1") != nil {
		t.Error("user:1 should be deleted")
	}
	if cache.Get("user:2") != nil {
		t.Error("user:2 should be deleted")
	}
	if cache.Get("dog:1") == nil {
		t.Error("dog:1 should not be deleted")
	}
}

func TestCacheService_Clear(t *testing.T) {
	cache := NewCacheService(CacheConfig{
		DefaultTTL:      1 * time.Minute,
		CleanupInterval: 0,
	})
	defer cache.Close()

	cache.Set("key1", "value1")
	cache.Set("key2", "value2")
	cache.Clear()

	if cache.Get("key1") != nil {
		t.Error("key1 should be cleared")
	}
	if cache.Get("key2") != nil {
		t.Error("key2 should be cleared")
	}

	stats := cache.GetStats()
	if stats.EntryCount != 0 {
		t.Errorf("EntryCount after clear = %d, want 0", stats.EntryCount)
	}
}

func TestCacheService_Exists(t *testing.T) {
	cache := NewCacheService(CacheConfig{
		DefaultTTL:      1 * time.Minute,
		CleanupInterval: 0,
	})
	defer cache.Close()

	cache.Set("exists", "value")

	if !cache.Exists("exists") {
		t.Error("Exists(exists) should be true")
	}
	if cache.Exists("nonexistent") {
		t.Error("Exists(nonexistent) should be false")
	}
}

// ============================================================================
// JSON Operations Tests
// ============================================================================

func TestCacheService_SetJSONAndGetJSON(t *testing.T) {
	cache := NewCacheService(CacheConfig{
		DefaultTTL:      1 * time.Minute,
		CleanupInterval: 0,
	})
	defer cache.Close()

	type TestStruct struct {
		Name  string `json:"name"`
		Value int    `json:"value"`
	}

	original := TestStruct{Name: "test", Value: 42}
	err := cache.SetJSON("jsonkey", original)
	if err != nil {
		t.Fatalf("SetJSON error: %v", err)
	}

	var retrieved TestStruct
	found := cache.GetJSON("jsonkey", &retrieved)
	if !found {
		t.Error("GetJSON should return true for existing key")
	}
	if retrieved.Name != "test" || retrieved.Value != 42 {
		t.Errorf("GetJSON returned %+v, want %+v", retrieved, original)
	}
}

func TestCacheService_SetJSONWithTTL(t *testing.T) {
	cache := NewCacheService(CacheConfig{
		DefaultTTL:      1 * time.Minute,
		CleanupInterval: 0,
	})
	defer cache.Close()

	err := cache.SetJSONWithTTL("key", map[string]string{"foo": "bar"}, 100*time.Millisecond)
	if err != nil {
		t.Fatalf("SetJSONWithTTL error: %v", err)
	}

	// Should exist immediately
	var result map[string]string
	if !cache.GetJSON("key", &result) {
		t.Error("GetJSON should return true immediately after set")
	}

	// Should expire
	time.Sleep(150 * time.Millisecond)
	if cache.GetJSON("key", &result) {
		t.Error("GetJSON should return false after expiration")
	}
}

func TestCacheService_GetJSON_NonExistent(t *testing.T) {
	cache := NewCacheService(CacheConfig{
		DefaultTTL:      1 * time.Minute,
		CleanupInterval: 0,
	})
	defer cache.Close()

	var result map[string]string
	if cache.GetJSON("nonexistent", &result) {
		t.Error("GetJSON should return false for nonexistent key")
	}
}

// ============================================================================
// Stats Tests
// ============================================================================

func TestCacheService_GetStats(t *testing.T) {
	cache := NewCacheService(CacheConfig{
		DefaultTTL:      1 * time.Minute,
		CleanupInterval: 0,
	})
	defer cache.Close()

	cache.Set("key1", "value1")
	cache.Set("key2", "value2")
	cache.Get("key1") // hit
	cache.Get("nonexistent") // miss
	cache.Delete("key2")

	stats := cache.GetStats()

	if stats.Sets != 2 {
		t.Errorf("Stats.Sets = %d, want 2", stats.Sets)
	}
	if stats.Hits != 1 {
		t.Errorf("Stats.Hits = %d, want 1", stats.Hits)
	}
	if stats.Misses != 1 {
		t.Errorf("Stats.Misses = %d, want 1", stats.Misses)
	}
	if stats.Deletes != 1 {
		t.Errorf("Stats.Deletes = %d, want 1", stats.Deletes)
	}
}

func TestCacheService_GetHitRate(t *testing.T) {
	cache := NewCacheService(CacheConfig{
		DefaultTTL:      1 * time.Minute,
		CleanupInterval: 0,
	})
	defer cache.Close()

	// No operations - hit rate should be 0
	if rate := cache.GetHitRate(); rate != 0 {
		t.Errorf("GetHitRate with no ops = %f, want 0", rate)
	}

	cache.Set("key", "value")
	cache.Get("key") // hit
	cache.Get("key") // hit
	cache.Get("miss") // miss

	// 2 hits, 1 miss = 66.67%
	rate := cache.GetHitRate()
	if rate < 66 || rate > 67 {
		t.Errorf("GetHitRate = %f, want ~66.67", rate)
	}
}

// ============================================================================
// Eviction Tests
// ============================================================================

func TestCacheService_EvictsOldestWhenFull(t *testing.T) {
	cache := NewCacheService(CacheConfig{
		DefaultTTL:      1 * time.Minute,
		MaxEntries:      3,
		CleanupInterval: 0,
	})
	defer cache.Close()

	cache.Set("key1", "value1")
	time.Sleep(10 * time.Millisecond)
	cache.Set("key2", "value2")
	time.Sleep(10 * time.Millisecond)
	cache.Set("key3", "value3")
	time.Sleep(10 * time.Millisecond)

	// This should evict key1 (oldest)
	cache.Set("key4", "value4")

	if cache.Get("key1") != nil {
		t.Error("key1 should have been evicted")
	}
	if cache.Get("key2") == nil {
		t.Error("key2 should still exist")
	}
	if cache.Get("key3") == nil {
		t.Error("key3 should still exist")
	}
	if cache.Get("key4") == nil {
		t.Error("key4 should exist")
	}
}

// TestCacheService_MaxEntriesStrictlyEnforced tests that cache never exceeds maxEntries
// BUG FIX: evictOldest() only removed 1 entry, but we need to ensure strict limit
func TestCacheService_MaxEntriesStrictlyEnforced(t *testing.T) {
	maxEntries := 5
	cache := NewCacheService(CacheConfig{
		DefaultTTL:      1 * time.Minute,
		MaxEntries:      maxEntries,
		CleanupInterval: 0,
	})
	defer cache.Close()

	// Add more entries than maxEntries allows
	for i := 0; i < 20; i++ {
		key := "key" + itoa(i)
		cache.Set(key, "value")
	}

	// Cache size should never exceed maxEntries
	stats := cache.GetStats()
	if stats.EntryCount > maxEntries {
		t.Errorf("Cache exceeded maxEntries: got %d, max %d", stats.EntryCount, maxEntries)
	}
}

// TestCacheService_MaxEntriesWithUpdates tests that updating existing keys doesn't break limits
func TestCacheService_MaxEntriesWithUpdates(t *testing.T) {
	maxEntries := 3
	cache := NewCacheService(CacheConfig{
		DefaultTTL:      1 * time.Minute,
		MaxEntries:      maxEntries,
		CleanupInterval: 0,
	})
	defer cache.Close()

	// Fill cache to max
	cache.Set("key1", "value1")
	cache.Set("key2", "value2")
	cache.Set("key3", "value3")

	// Update existing key (should NOT trigger eviction)
	cache.Set("key2", "updated_value2")

	// All keys should still exist
	if cache.Get("key1") == nil {
		t.Error("key1 should still exist after update")
	}
	if cache.Get("key2") != "updated_value2" {
		t.Error("key2 should have updated value")
	}
	if cache.Get("key3") == nil {
		t.Error("key3 should still exist after update")
	}

	// Entry count should still be at max
	stats := cache.GetStats()
	if stats.EntryCount != maxEntries {
		t.Errorf("EntryCount = %d, want %d", stats.EntryCount, maxEntries)
	}
}

// ============================================================================
// Close Tests
// ============================================================================

func TestCacheService_Close(t *testing.T) {
	cache := NewCacheService(CacheConfig{
		DefaultTTL:      1 * time.Minute,
		CleanupInterval: 100 * time.Millisecond,
	})

	// Should not panic on first close
	cache.Close()

	// Should not panic on second close
	cache.Close()
}

// ============================================================================
// Cache Key Generator Tests
// ============================================================================

func TestCacheKeyGenerators(t *testing.T) {
	tests := []struct {
		name     string
		function func() string
		want     string
	}{
		{"CacheKeyDog", func() string { return CacheKeyDog(1, 2) }, "dog:1:2"},
		{"CacheKeyDogs", func() string { return CacheKeyDogs(1) }, "dogs:1"},
		{"CacheKeyUser", func() string { return CacheKeyUser(1, 2) }, "user:1:2"},
		{"CacheKeySettings", func() string { return CacheKeySettings(1) }, "settings:1"},
		{"CacheKeyBookingSlots", func() string { return CacheKeyBookingSlots(1, "2025-01-15") }, "slots:1:2025-01-15"},
		{"CacheKeyHolidays", func() string { return CacheKeyHolidays(2025, "BW") }, "holidays:2025:BW"},
		{"CacheKeyColors", func() string { return CacheKeyColors(1) }, "colors:1"},
		{"CacheKeyTheme", func() string { return CacheKeyTheme(1) }, "theme:1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.function()
			if got != tt.want {
				t.Errorf("%s = %s, want %s", tt.name, got, tt.want)
			}
		})
	}
}

// ============================================================================
// itoa Helper Tests
// ============================================================================

func TestItoa(t *testing.T) {
	tests := []struct {
		input int
		want  string
	}{
		{0, "0"},
		{1, "1"},
		{10, "10"},
		{123, "123"},
		{-1, "-1"},
		{-123, "-123"},
		{1000000, "1000000"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := itoa(tt.input)
			if got != tt.want {
				t.Errorf("itoa(%d) = %s, want %s", tt.input, got, tt.want)
			}
		})
	}
}
