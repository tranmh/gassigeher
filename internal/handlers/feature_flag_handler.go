package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/tranmh/gassigeher/internal/middleware"
	"github.com/tranmh/gassigeher/internal/models"
	"github.com/tranmh/gassigeher/internal/repository"
	"github.com/tranmh/gassigeher/internal/services"
)

// FeatureFlagHandler handles feature flag endpoints
type FeatureFlagHandler struct {
	service *services.FeatureFlagService
}

// NewFeatureFlagHandler creates a new feature flag handler
func NewFeatureFlagHandler(db *sql.DB) *FeatureFlagHandler {
	repo := repository.NewFeatureFlagRepository(db)
	service := services.NewFeatureFlagService(repo)
	return &FeatureFlagHandler{
		service: service,
	}
}

// GetService returns the underlying service for direct access
func (h *FeatureFlagHandler) GetService() *services.FeatureFlagService {
	return h.service
}

// ListFlags returns all feature flags (central admin only)
// GET /api/v1/central-admin/feature-flags
func (h *FeatureFlagHandler) ListFlags(w http.ResponseWriter, r *http.Request) {
	flags, err := h.service.GetAllFlags()
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Fehler beim Laden der Feature Flags")
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"flags": flags,
	})
}

// GetFlag returns a single feature flag
// GET /api/v1/central-admin/feature-flags/{id}
func (h *FeatureFlagHandler) GetFlag(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		respondError(w, http.StatusBadRequest, "Ungültige Flag-ID")
		return
	}

	flags, err := h.service.GetAllFlags()
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Fehler beim Laden des Feature Flags")
		return
	}

	// Find flag by ID
	var flag *models.FeatureFlag
	for _, f := range flags {
		if f.ID == id {
			flag = f
			break
		}
	}

	if flag == nil {
		respondError(w, http.StatusNotFound, "Feature Flag nicht gefunden")
		return
	}

	respondJSON(w, http.StatusOK, flag)
}

// CreateFlag creates a new feature flag (central admin only)
// POST /api/v1/central-admin/feature-flags
func (h *FeatureFlagHandler) CreateFlag(w http.ResponseWriter, r *http.Request) {
	var req models.CreateFeatureFlagRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Ungültige Anfrage")
		return
	}

	// HIGH-6 fix: Validate request
	if err := req.Validate(); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	flag, err := h.service.CreateFlag(&req)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	respondJSON(w, http.StatusCreated, flag)
}

// UpdateFlag updates a feature flag (central admin only)
// PUT /api/v1/central-admin/feature-flags/{id}
func (h *FeatureFlagHandler) UpdateFlag(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		respondError(w, http.StatusBadRequest, "Ungültige Flag-ID")
		return
	}

	var req models.UpdateFeatureFlagRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Ungültige Anfrage")
		return
	}

	// HIGH-6 fix: Validate request
	if err := req.Validate(); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	flag, err := h.service.UpdateFlag(id, &req)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, flag)
}

// DeleteFlag deletes a feature flag (central admin only)
// DELETE /api/v1/central-admin/feature-flags/{id}
func (h *FeatureFlagHandler) DeleteFlag(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		respondError(w, http.StatusBadRequest, "Ungültige Flag-ID")
		return
	}

	if err := h.service.DeleteFlag(id); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{
		"message": "Feature Flag gelöscht",
	})
}

// GetTenantFlags returns flags with status for current tenant
// GET /api/v1/admin/feature-flags
func (h *FeatureFlagHandler) GetTenantFlags(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := r.Context().Value(middleware.TenantIDKey).(int)
	if !ok {
		respondError(w, http.StatusInternalServerError, "Request validation failed")
		return
	}

	flags, err := h.service.GetFlagsForTenant(tenantID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Fehler beim Laden der Feature Flags")
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"flags": flags,
	})
}

// SetTenantFlag enables/disables a flag for the current tenant
// PUT /api/v1/admin/feature-flags/{id}
func (h *FeatureFlagHandler) SetTenantFlag(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := r.Context().Value(middleware.TenantIDKey).(int)
	if !ok {
		respondError(w, http.StatusInternalServerError, "Request validation failed")
		return
	}

	vars := mux.Vars(r)
	flagID, err := strconv.Atoi(vars["id"])
	if err != nil {
		respondError(w, http.StatusBadRequest, "Ungültige Flag-ID")
		return
	}

	var req models.SetTenantFeatureFlagRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Ungültige Anfrage")
		return
	}

	userID, _ := r.Context().Value(middleware.UserIDKey).(int)

	if err := h.service.SetTenantFlag(tenantID, flagID, req.IsEnabled, &userID); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{
		"message": "Feature Flag aktualisiert",
	})
}

// ResetTenantFlag removes a tenant-specific override (falls back to global)
// DELETE /api/v1/admin/feature-flags/{id}
func (h *FeatureFlagHandler) ResetTenantFlag(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := r.Context().Value(middleware.TenantIDKey).(int)
	if !ok {
		respondError(w, http.StatusInternalServerError, "Request validation failed")
		return
	}

	vars := mux.Vars(r)
	flagID, err := strconv.Atoi(vars["id"])
	if err != nil {
		respondError(w, http.StatusBadRequest, "Ungültige Flag-ID")
		return
	}

	if err := h.service.RemoveTenantFlag(tenantID, flagID); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{
		"message": "Tenant-Override entfernt, globale Einstellung gilt",
	})
}

// CheckFlag checks if a specific flag is enabled for the current tenant
// GET /api/v1/feature-flags/{key}/check
func (h *FeatureFlagHandler) CheckFlag(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.GetTenantID(r)
	vars := mux.Vars(r)
	key := vars["key"]

	enabled := h.service.IsEnabled(key, tenantID)

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"key":     key,
		"enabled": enabled,
	})
}

// CheckMultipleFlags checks multiple flags at once
// POST /api/v1/feature-flags/check
func (h *FeatureFlagHandler) CheckMultipleFlags(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.GetTenantID(r)

	var req struct {
		Keys []string `json:"keys"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Ungültige Anfrage")
		return
	}

	result := make(map[string]bool)
	for _, key := range req.Keys {
		result[key] = h.service.IsEnabled(key, tenantID)
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"flags": result,
	})
}
