package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/gorilla/mux"
	"github.com/tranmh/gassigeher/internal/config"
	"github.com/tranmh/gassigeher/internal/database"
	"github.com/tranmh/gassigeher/internal/middleware"
	"github.com/tranmh/gassigeher/internal/models"
	"github.com/tranmh/gassigeher/internal/repository"
	"github.com/tranmh/gassigeher/internal/services"
)

// SettingsHandler handles system settings-related HTTP requests
type SettingsHandler struct {
	db           *database.DB
	cfg          *config.Config
	settingsRepo *repository.SettingsRepository
	imageService *services.ImageService
}

// NewSettingsHandler creates a new settings handler
func NewSettingsHandler(db *database.DB, cfg *config.Config) *SettingsHandler {
	return &SettingsHandler{
		db:           db,
		cfg:          cfg,
		settingsRepo: repository.NewSettingsRepository(db),
		imageService: services.NewImageService(cfg.UploadDir),
	}
}

// GetAllSettings gets all system settings (admin only)
func (h *SettingsHandler) GetAllSettings(w http.ResponseWriter, r *http.Request) {
	// SaaS: Extract tenant ID from context
	tenantID, _ := r.Context().Value(middleware.TenantIDKey).(int)

	settings, err := h.settingsRepo.GetAll(tenantID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to get settings")
		return
	}

	respondJSON(w, http.StatusOK, settings)
}

// UpdateSetting updates a system setting (admin only)
func (h *SettingsHandler) UpdateSetting(w http.ResponseWriter, r *http.Request) {
	// SaaS: Extract tenant ID from context
	tenantID, _ := r.Context().Value(middleware.TenantIDKey).(int)

	// Get key from URL
	vars := mux.Vars(r)
	key := vars["key"]

	// Parse request
	var req models.UpdateSettingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// HIGH-7: Use centralized validation with type-specific and range validation
	// This replaces the previous scattered validation logic with a single call
	// to models.ValidateSetting which validates:
	// - Integer settings with min/max ranges (booking_advance_days: 1-365, etc.)
	// - Boolean settings (whatsapp_group_enabled)
	// - Special formats (registration_password, whatsapp_group_link)
	if err := models.ValidateSetting(key, req.Value); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Update setting
	if err := h.settingsRepo.Update(tenantID, key, req.Value); err != nil {
		if err.Error() == "setting not found" {
			respondError(w, http.StatusNotFound, err.Error())
			return
		}
		respondError(w, http.StatusInternalServerError, "Failed to update setting")
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "Setting updated successfully"})
}

// Default logo URL (Tierheim Goeppingen) - only for Simple-Mode
const defaultLogoURL = "https://www.tierheim-goeppingen.de/wp-content/uploads/2017/04/Logo_4c_homepagebanner3.png"

// Placeholder logo URL for SaaS-Mode tenants without custom logo
const placeholderLogoURL = "/assets/images/placeholders/logo-placeholder.svg"

// GetLogo returns the current logo URL (public endpoint, no auth required)
func (h *SettingsHandler) GetLogo(w http.ResponseWriter, r *http.Request) {
	// SaaS: Extract tenant ID from context
	tenantID, _ := r.Context().Value(middleware.TenantIDKey).(int)

	setting, err := h.settingsRepo.Get(tenantID, "site_logo")
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to get logo setting")
		return
	}

	// Determine default logo based on mode
	// SaaS-Mode (tenantID > 0): use placeholder
	// Simple-Mode (tenantID == 0): use Tierheim Goeppingen logo
	var logoURL string
	if setting != nil && setting.Value != "" {
		logoURL = setting.Value
	} else if tenantID > 0 {
		// SaaS-Mode: new tenants get neutral placeholder
		logoURL = placeholderLogoURL
	} else {
		// Simple-Mode: use original default
		logoURL = defaultLogoURL
	}

	respondJSON(w, http.StatusOK, map[string]string{"logo_url": logoURL})
}

// UploadLogo handles uploading a custom site logo (admin only)
func (h *SettingsHandler) UploadLogo(w http.ResponseWriter, r *http.Request) {
	// SaaS: Extract tenant ID from context
	tenantID, _ := r.Context().Value(middleware.TenantIDKey).(int)

	// SECURITY: Limit request body size to prevent DoS attacks
	maxSizeMB := h.cfg.MaxUploadSizeMB
	if maxSizeMB <= 0 {
		maxSizeMB = 10 // Default 10MB if not configured
	}
	maxSize := int64(maxSizeMB) << 20
	r.Body = http.MaxBytesReader(w, r.Body, maxSize)

	// Parse multipart form with max size limit
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
	if _, err := file.Seek(0, 0); err != nil {
		log.Printf("Error seeking file: %v", err)
		respondError(w, http.StatusInternalServerError, "Fehler beim Verarbeiten der Datei")
		return
	}

	// Process and save logo
	logoPath, err := h.imageService.ProcessLogo(file)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to process logo")
		return
	}

	// Upsert setting with local path (prefixed with /uploads/)
	// Use Upsert because site_logo setting may not exist yet
	localURL := "/uploads/" + logoPath
	if err := h.settingsRepo.Upsert(tenantID, "site_logo", localURL); err != nil {
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
	// SaaS: Extract tenant ID from context
	tenantID, _ := r.Context().Value(middleware.TenantIDKey).(int)

	enabledSetting, err := h.settingsRepo.Get(tenantID, "whatsapp_group_enabled")
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to get WhatsApp enabled setting")
		return
	}

	linkSetting, err := h.settingsRepo.Get(tenantID, "whatsapp_group_link")
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
	// SaaS: Extract tenant ID from context
	tenantID, _ := r.Context().Value(middleware.TenantIDKey).(int)

	// Delete custom logo file (if exists)
	h.imageService.DeleteLogo()

	// Determine default logo based on mode
	var resetLogoURL string
	if tenantID > 0 {
		// SaaS-Mode: reset to placeholder
		resetLogoURL = placeholderLogoURL
	} else {
		// Simple-Mode: reset to Tierheim Goeppingen logo
		resetLogoURL = defaultLogoURL
	}

	// Reset setting to appropriate default URL
	// Use Upsert because site_logo setting may not exist yet
	if err := h.settingsRepo.Upsert(tenantID, "site_logo", resetLogoURL); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to reset logo setting")
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{
		"message":  "Logo reset to default",
		"logo_url": resetLogoURL,
	})
}
