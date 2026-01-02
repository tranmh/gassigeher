package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/tranmh/gassigeher/internal/services"
)

// TestCacheHandler_GetStats tests getting cache statistics
func TestCacheHandler_GetStats(t *testing.T) {
	cfg := services.CacheConfig{
		DefaultTTL:      5 * time.Minute,
		MaxEntries:      1000,
		CleanupInterval: 0, // No cleanup for tests
	}
	cache := services.NewCacheService(cfg)
	defer cache.Close()
	handler := NewCacheHandler(cache)

	t.Run("returns stats with empty cache", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/admin/cache/stats", nil)

		rec := httptest.NewRecorder()
		handler.GetStats(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rec.Code)
		}

		var response map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &response)

		if response["stats"] == nil {
			t.Error("Expected stats field in response")
		}
		if response["hit_rate"] == nil {
			t.Error("Expected hit_rate field in response")
		}
	})

	t.Run("returns stats after adding entries", func(t *testing.T) {
		// Add some cache entries
		cache.Set("test:key1", "value1")
		cache.Set("test:key2", "value2")
		cache.Get("test:key1") // Hit
		cache.Get("test:miss") // Miss

		req := httptest.NewRequest("GET", "/api/v1/admin/cache/stats", nil)

		rec := httptest.NewRecorder()
		handler.GetStats(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rec.Code)
		}

		var response map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &response)

		stats := response["stats"].(map[string]interface{})
		if stats["EntryCount"].(float64) < 2 {
			t.Errorf("Expected at least 2 cache entries, got %v", stats["EntryCount"])
		}
	})
}

// TestCacheHandler_Clear tests clearing the entire cache
func TestCacheHandler_Clear(t *testing.T) {
	cfg := services.CacheConfig{
		DefaultTTL:      5 * time.Minute,
		MaxEntries:      1000,
		CleanupInterval: 0,
	}
	cache := services.NewCacheService(cfg)
	defer cache.Close()
	handler := NewCacheHandler(cache)

	t.Run("clears cache successfully", func(t *testing.T) {
		// Add some cache entries
		cache.Set("test:key1", "value1")
		cache.Set("test:key2", "value2")

		req := httptest.NewRequest("DELETE", "/api/v1/admin/cache", nil)

		rec := httptest.NewRecorder()
		handler.Clear(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rec.Code)
		}

		var response map[string]string
		json.Unmarshal(rec.Body.Bytes(), &response)

		if response["message"] != "Cache geleert" {
			t.Errorf("Expected success message, got %s", response["message"])
		}

		// Verify cache is empty
		value := cache.Get("test:key1")
		if value != nil {
			t.Error("Cache should be empty after clear")
		}
	})
}

// TestCacheHandler_ClearPrefix tests clearing cache by prefix
func TestCacheHandler_ClearPrefix(t *testing.T) {
	cfg := services.CacheConfig{
		DefaultTTL:      5 * time.Minute,
		MaxEntries:      1000,
		CleanupInterval: 0,
	}
	cache := services.NewCacheService(cfg)
	defer cache.Close()
	handler := NewCacheHandler(cache)

	t.Run("clears entries with matching prefix", func(t *testing.T) {
		// Add cache entries with different prefixes
		cache.Set("users:1", "user1")
		cache.Set("users:2", "user2")
		cache.Set("dogs:1", "dog1")

		req := httptest.NewRequest("DELETE", "/api/v1/admin/cache/prefix?prefix=users:", nil)

		rec := httptest.NewRecorder()
		handler.ClearPrefix(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rec.Code)
		}

		var response map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &response)

		deletedCount := int(response["deleted_count"].(float64))
		if deletedCount != 2 {
			t.Errorf("Expected 2 deleted entries, got %d", deletedCount)
		}

		// Verify users entries are deleted
		value := cache.Get("users:1")
		if value != nil {
			t.Error("users:1 should be deleted")
		}

		// Verify dogs entries remain
		value = cache.Get("dogs:1")
		if value == nil {
			t.Error("dogs:1 should still exist")
		}
	})

	t.Run("returns error when prefix is missing", func(t *testing.T) {
		req := httptest.NewRequest("DELETE", "/api/v1/admin/cache/prefix", nil)

		rec := httptest.NewRecorder()
		handler.ClearPrefix(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", rec.Code)
		}
	})

	t.Run("handles empty prefix gracefully", func(t *testing.T) {
		req := httptest.NewRequest("DELETE", "/api/v1/admin/cache/prefix?prefix=", nil)

		rec := httptest.NewRecorder()
		handler.ClearPrefix(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", rec.Code)
		}
	})
}
