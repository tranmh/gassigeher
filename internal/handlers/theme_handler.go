package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/tranmh/gassigeher/internal/middleware"
	"github.com/tranmh/gassigeher/internal/models"
	"github.com/tranmh/gassigeher/internal/repository"
)

// ThemeHandler handles theme-related endpoints
type ThemeHandler struct {
	tenantRepo *repository.TenantRepository
}

// safeDeref safely dereferences a string pointer, returning empty string if nil
func safeDeref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// NewThemeHandler creates a new theme handler
func NewThemeHandler(db *sql.DB) *ThemeHandler {
	return &ThemeHandler{
		tenantRepo: repository.NewTenantRepository(db),
	}
}

// GetCSS returns dynamic CSS variables for the current tenant's theme
func (h *ThemeHandler) GetCSS(w http.ResponseWriter, r *http.Request) {
	tenantID, _ := r.Context().Value(middleware.TenantIDKey).(int)

	// Get tenant settings
	settings, err := h.tenantRepo.GetSettings(tenantID)
	if err != nil {
		http.Error(w, "Fehler beim Laden des Themes", http.StatusInternalServerError)
		return
	}

	// Determine colors (custom colors or preset)
	var colors models.ThemeColors
	if settings != nil && settings.ColorPrimary != nil && *settings.ColorPrimary != "" {
		// Use custom colors
		colors = models.ThemeColors{
			Primary:    safeDeref(settings.ColorPrimary),
			Secondary:  safeDeref(settings.ColorSecondary),
			Accent:     safeDeref(settings.ColorAccent),
			Background: safeDeref(settings.ColorBackground),
			Text:       safeDeref(settings.ColorText),
		}
	} else if settings != nil && settings.ThemePreset != "" {
		// Use preset
		colors = models.GetThemePreset(settings.ThemePreset)
	} else {
		// Default to classic
		colors = models.GetThemePreset("classic")
	}

	// Generate CSS
	css := fmt.Sprintf(`:root {
    --color-primary: %s;
    --color-secondary: %s;
    --color-accent: %s;
    --color-background: %s;
    --color-text: %s;
}`, colors.Primary, colors.Secondary, colors.Accent, colors.Background, colors.Text)

	w.Header().Set("Content-Type", "text/css; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=60") // Cache for 1 minute (theme-loader.js busts cache on branding changes)
	w.Write([]byte(css))
}

// GetPresets returns all available theme presets
func (h *ThemeHandler) GetPresets(w http.ResponseWriter, r *http.Request) {
	presets := models.GetAllPresetInfo()
	respondJSON(w, http.StatusOK, presets)
}

// GetCurrentTheme returns the current tenant's theme settings
func (h *ThemeHandler) GetCurrentTheme(w http.ResponseWriter, r *http.Request) {
	tenantID, _ := r.Context().Value(middleware.TenantIDKey).(int)

	settings, err := h.tenantRepo.GetSettings(tenantID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Fehler beim Laden des Themes")
		return
	}

	if settings == nil {
		// Return default theme
		respondJSON(w, http.StatusOK, map[string]interface{}{
			"preset": "classic",
			"colors": models.GetThemePreset("classic"),
		})
		return
	}

	// Determine if using custom colors or preset
	if settings.ColorPrimary != nil && *settings.ColorPrimary != "" {
		respondJSON(w, http.StatusOK, map[string]interface{}{
			"preset": "custom",
			"colors": models.ThemeColors{
				Primary:    safeDeref(settings.ColorPrimary),
				Secondary:  safeDeref(settings.ColorSecondary),
				Accent:     safeDeref(settings.ColorAccent),
				Background: safeDeref(settings.ColorBackground),
				Text:       safeDeref(settings.ColorText),
			},
		})
	} else {
		presetName := settings.ThemePreset
		if presetName == "" {
			presetName = "classic"
		}
		respondJSON(w, http.StatusOK, map[string]interface{}{
			"preset": presetName,
			"colors": models.GetThemePreset(presetName),
		})
	}
}

// UpdateThemeRequest represents a request to update the theme
type UpdateThemeRequest struct {
	Preset     string `json:"preset"`     // Preset name or "custom"
	Primary    string `json:"primary"`    // Custom primary color (if preset = "custom")
	Secondary  string `json:"secondary"`  // Custom secondary color
	Accent     string `json:"accent"`     // Custom accent color
	Background string `json:"background"` // Custom background color
	Text       string `json:"text"`       // Custom text color
}

// UpdateTheme updates the tenant's theme settings (admin only)
func (h *ThemeHandler) UpdateTheme(w http.ResponseWriter, r *http.Request) {
	tenantID, _ := r.Context().Value(middleware.TenantIDKey).(int)

	var req UpdateThemeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Ungültige Anfrage")
		return
	}

	// Get current settings
	settings, err := h.tenantRepo.GetSettings(tenantID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Datenbankfehler")
		return
	}

	if settings == nil {
		// Create new settings if they don't exist
		settings = &models.TenantSettings{
			TenantID: tenantID,
		}
		if err := h.tenantRepo.CreateSettings(settings); err != nil {
			respondError(w, http.StatusInternalServerError, "Fehler beim Erstellen der Einstellungen")
			return
		}
	}

	// Update based on preset or custom colors
	if req.Preset == "custom" {
		// Validate custom colors
		if req.Primary == "" || req.Secondary == "" || req.Accent == "" || req.Background == "" || req.Text == "" {
			respondError(w, http.StatusBadRequest, "Alle benutzerdefinierten Farben sind erforderlich")
			return
		}
		settings.ThemePreset = ""
		settings.ColorPrimary = &req.Primary
		settings.ColorSecondary = &req.Secondary
		settings.ColorAccent = &req.Accent
		settings.ColorBackground = &req.Background
		settings.ColorText = &req.Text
	} else {
		// Validate preset name
		if !models.IsValidPreset(req.Preset) {
			respondError(w, http.StatusBadRequest, "Ungültiges Theme-Preset")
			return
		}
		settings.ThemePreset = req.Preset
		// Clear custom colors when using preset
		settings.ColorPrimary = nil
		settings.ColorSecondary = nil
		settings.ColorAccent = nil
		settings.ColorBackground = nil
		settings.ColorText = nil
	}

	if err := h.tenantRepo.UpdateSettings(settings); err != nil {
		respondError(w, http.StatusInternalServerError, "Fehler beim Speichern des Themes")
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{
		"message": "Theme erfolgreich aktualisiert",
	})
}
