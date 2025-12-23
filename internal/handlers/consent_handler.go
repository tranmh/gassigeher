package handlers

import (
	"database/sql"
	"net/http"
	"strings"

	"github.com/tranmh/gassigeher/internal/middleware"
	"github.com/tranmh/gassigeher/internal/models"
	"github.com/tranmh/gassigeher/internal/repository"
)

// ConsentHandler handles consent-related endpoints
type ConsentHandler struct {
	consentRepo *repository.ConsentRepository
}

// NewConsentHandler creates a new consent handler
func NewConsentHandler(db *sql.DB) *ConsentHandler {
	return &ConsentHandler{
		consentRepo: repository.NewConsentRepository(db),
	}
}

// GetConsentStatus returns the current user's consent status
// GET /api/users/me/consent
func (h *ConsentHandler) GetConsentStatus(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(int)
	if !ok {
		respondError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	tenantID := middleware.GetTenantID(r)

	status, err := h.consentRepo.GetConsentStatus(userID, tenantID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Fehler beim Laden des Consent-Status")
		return
	}

	respondJSON(w, http.StatusOK, status)
}

// GetConsentHistory returns the current user's consent history
// GET /api/users/me/consent/history
func (h *ConsentHandler) GetConsentHistory(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(int)
	if !ok {
		respondError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	tenantID := middleware.GetTenantID(r)

	consents, err := h.consentRepo.FindByUserID(userID, tenantID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Fehler beim Laden der Consent-Historie")
		return
	}

	respondJSON(w, http.StatusOK, consents)
}

// UpdateConsent records new consent acceptance
// POST /api/users/me/consent
func (h *ConsentHandler) UpdateConsent(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(int)
	if !ok {
		respondError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	tenantID := middleware.GetTenantID(r)
	ipAddress := getConsentClientIP(r)
	userAgent := r.UserAgent()

	err := h.consentRepo.RecordConsent(userID, tenantID, ipAddress, userAgent)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Fehler beim Speichern der Einwilligung")
		return
	}

	// Get updated status
	status, _ := h.consentRepo.GetConsentStatus(userID, tenantID)

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"message": "Einwilligung erfolgreich gespeichert",
		"status":  status,
	})
}

// GetCurrentConsentVersions returns the current versions of terms/privacy
// GET /api/consent/versions
func (h *ConsentHandler) GetCurrentConsentVersions(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, models.CurrentConsentVersions)
}

func getConsentClientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		if idx := strings.Index(xff, ","); idx != -1 {
			return strings.TrimSpace(xff[:idx])
		}
		return strings.TrimSpace(xff)
	}
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return strings.TrimSpace(xri)
	}
	if r.RemoteAddr != "" {
		if idx := strings.LastIndex(r.RemoteAddr, ":"); idx != -1 {
			return r.RemoteAddr[:idx]
		}
		return r.RemoteAddr
	}
	return ""
}
