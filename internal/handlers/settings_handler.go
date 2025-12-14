package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
	"github.com/tranmh/gassigeher/internal/config"
	"github.com/tranmh/gassigeher/internal/models"
	"github.com/tranmh/gassigeher/internal/repository"
	"github.com/tranmh/gassigeher/internal/services"
)

// SettingsHandler handles system settings-related HTTP requests
type SettingsHandler struct {
	db           *sql.DB
	cfg          *config.Config
	settingsRepo *repository.SettingsRepository
	imageService *services.ImageService
}

// NewSettingsHandler creates a new settings handler
func NewSettingsHandler(db *sql.DB, cfg *config.Config) *SettingsHandler {
	return &SettingsHandler{
		db:           db,
		cfg:          cfg,
		settingsRepo: repository.NewSettingsRepository(db),
		imageService: services.NewImageService(cfg.UploadDir),
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

	// Settings that allow empty values
	allowEmptySettings := map[string]bool{
		"whatsapp_group_link": true,
	}

	// Validate request (skip for settings that allow empty values)
	if !allowEmptySettings[key] {
		if err := req.Validate(); err != nil {
			respondError(w, http.StatusBadRequest, err.Error())
			return
		}
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

	// Validate registration password format (8 alphanumeric characters)
	if key == "registration_password" {
		if !regexp.MustCompile(`^[a-zA-Z0-9]{8}$`).MatchString(req.Value) {
			respondError(w, http.StatusBadRequest, "Registration password must be exactly 8 alphanumeric characters")
			return
		}
	}

	// Validate WhatsApp group enabled (boolean as string)
	if key == "whatsapp_group_enabled" {
		if req.Value != "true" && req.Value != "false" {
			respondError(w, http.StatusBadRequest, "WhatsApp group enabled must be 'true' or 'false'")
			return
		}
	}

	// Validate WhatsApp group link (must be valid WhatsApp URL or empty)
	if key == "whatsapp_group_link" {
		if req.Value != "" && !strings.HasPrefix(req.Value, "https://chat.whatsapp.com/") {
			respondError(w, http.StatusBadRequest, "WhatsApp group link must start with https://chat.whatsapp.com/")
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

// Default logo URL (Tierheim Goeppingen)
const defaultLogoURL = "https://www.tierheim-goeppingen.de/wp-content/uploads/2017/04/Logo_4c_homepagebanner3.png"

// GetLogo returns the current logo URL (public endpoint, no auth required)
func (h *SettingsHandler) GetLogo(w http.ResponseWriter, r *http.Request) {
	setting, err := h.settingsRepo.Get("site_logo")
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to get logo setting")
		return
	}

	logoURL := defaultLogoURL
	if setting != nil && setting.Value != "" {
		logoURL = setting.Value
	}

	respondJSON(w, http.StatusOK, map[string]string{"logo_url": logoURL})
}

// UploadLogo handles uploading a custom site logo (admin only)
func (h *SettingsHandler) UploadLogo(w http.ResponseWriter, r *http.Request) {
	// Parse multipart form with max size limit
	maxSize := int64(h.cfg.MaxUploadSizeMB) << 20
	if err := r.ParseMultipartForm(maxSize); err != nil {
		respondError(w, http.StatusBadRequest, "File too large or invalid form data")
		return
	}

	// Get file from form
	file, header, err := r.FormFile("logo")
	if err != nil {
		respondError(w, http.StatusBadRequest, "No file uploaded")
		return
	}
	defer file.Close()

	// Validate file extension
	ext := strings.ToLower(filepath.Ext(header.Filename))
	if ext != ".jpg" && ext != ".jpeg" && ext != ".png" {
		respondError(w, http.StatusBadRequest, "Only JPEG and PNG files are allowed")
		return
	}

	// Validate MIME type (magic bytes) to prevent file type spoofing
	if errMsg, valid := ValidateImageMIMEType(file); !valid {
		respondError(w, http.StatusBadRequest, errMsg)
		return
	}
	// Reset file reader position after MIME check
	file.Seek(0, 0)

	// Process and save logo
	logoPath, err := h.imageService.ProcessLogo(file)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to process logo")
		return
	}

	// Update setting with local path (prefixed with /uploads/)
	localURL := "/uploads/" + logoPath
	if err := h.settingsRepo.Update("site_logo", localURL); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to update logo setting")
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{
		"message":  "Logo uploaded successfully",
		"logo_url": localURL,
	})
}

// GetWhatsAppSettings returns the WhatsApp group settings (public endpoint, no auth required)
func (h *SettingsHandler) GetWhatsAppSettings(w http.ResponseWriter, r *http.Request) {
	enabledSetting, err := h.settingsRepo.Get("whatsapp_group_enabled")
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to get WhatsApp enabled setting")
		return
	}

	linkSetting, err := h.settingsRepo.Get("whatsapp_group_link")
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to get WhatsApp link setting")
		return
	}

	enabled := false
	if enabledSetting != nil && enabledSetting.Value == "true" {
		enabled = true
	}

	link := ""
	if linkSetting != nil {
		link = linkSetting.Value
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"enabled": enabled,
		"link":    link,
	})
}

// ResetLogo resets the site logo to the default (admin only)
func (h *SettingsHandler) ResetLogo(w http.ResponseWriter, r *http.Request) {
	// Delete custom logo file (if exists)
	h.imageService.DeleteLogo()

	// Reset setting to default URL
	if err := h.settingsRepo.Update("site_logo", defaultLogoURL); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to reset logo setting")
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{
		"message":  "Logo reset to default",
		"logo_url": defaultLogoURL,
	})
}
