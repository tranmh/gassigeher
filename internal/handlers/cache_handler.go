package handlers

import (
	"net/http"

	"github.com/tranmh/gassigeher/internal/services"
)

// CacheHandler handles cache-related endpoints
type CacheHandler struct {
	cache *services.CacheService
}

// NewCacheHandler creates a new cache handler
func NewCacheHandler(cache *services.CacheService) *CacheHandler {
	return &CacheHandler{
		cache: cache,
	}
}

// GetStats returns cache statistics (admin only)
// GET /api/v1/admin/cache/stats
func (h *CacheHandler) GetStats(w http.ResponseWriter, r *http.Request) {
	stats := h.cache.GetStats()
	hitRate := h.cache.GetHitRate()

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"stats":    stats,
		"hit_rate": hitRate,
	})
}

// Clear clears the entire cache (admin only)
// DELETE /api/v1/admin/cache
func (h *CacheHandler) Clear(w http.ResponseWriter, r *http.Request) {
	h.cache.Clear()

	respondJSON(w, http.StatusOK, map[string]string{
		"message": "Cache geleert",
	})
}

// ClearPrefix clears cache entries with a specific prefix (admin only)
// DELETE /api/v1/admin/cache/prefix/{prefix}
func (h *CacheHandler) ClearPrefix(w http.ResponseWriter, r *http.Request) {
	prefix := r.URL.Query().Get("prefix")
	if prefix == "" {
		respondError(w, http.StatusBadRequest, "Prefix ist erforderlich")
		return
	}

	count := h.cache.DeletePrefix(prefix)

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"message":       "Cache-Einträge gelöscht",
		"deleted_count": count,
	})
}
