package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/tranmh/gassigeher/internal/config"
	"github.com/tranmh/gassigeher/internal/models"
	"github.com/tranmh/gassigeher/internal/repository"
)

// SettingsHandler handles system settings-related HTTP requests
type SettingsHandler struct {
	db           *sql.DB
	cfg          *config.Config
	settingsRepo *repository.SettingsRepository
}

// NewSettingsHandler creates a new settings handler
func NewSettingsHandler(db *sql.DB, cfg *config.Config) *SettingsHandler {
	return &SettingsHandler{
		db:           db,
		cfg:          cfg,
		settingsRepo: repository.NewSettingsRepository(db),
	}
}

// GetAllSettings gets all system settings (admin only)
func (h *SettingsHandler) GetAllSettings(w http.ResponseWriter, r *http.Request) {
	settings, err := h.settingsRepo.GetAll()
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to get settings")
		return
	}

	respondJSON(w, http.StatusOK, settings)
}

// UpdateSetting updates a system setting (admin only)
func (h *SettingsHandler) UpdateSetting(w http.ResponseWriter, r *http.Request) {
	// Get key from URL
	vars := mux.Vars(r)
	key := vars["key"]

	// Parse request
	var req models.UpdateSettingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate request
	if err := req.Validate(); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	// BUGFIX #3: Validate numeric settings to prevent silent failures
	// These settings must be valid positive integers
	numericSettings := map[string]bool{
		"booking_advance_days":      true,
		"cancellation_notice_hours": true,
		"auto_deactivation_days":    true,
	}

	if numericSettings[key] {
		if val, err := strconv.Atoi(req.Value); err != nil || val <= 0 {
			respondError(w, http.StatusBadRequest, "Value must be a positive integer")
			return
		}
	}

	// Update setting
	if err := h.settingsRepo.Update(key, req.Value); err != nil {
		if err.Error() == "setting not found" {
			respondError(w, http.StatusNotFound, err.Error())
			return
		}
		respondError(w, http.StatusInternalServerError, "Failed to update setting")
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "Setting updated successfully"})
}
