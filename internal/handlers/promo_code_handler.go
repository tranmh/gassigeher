package handlers

import (
	"encoding/json"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/tranmh/gassigeher/internal/config"
	"github.com/tranmh/gassigeher/internal/database"
	"github.com/tranmh/gassigeher/internal/models"
	"github.com/tranmh/gassigeher/internal/repository"
)

// PromoCodeHandler handles promo code related HTTP requests
type PromoCodeHandler struct {
	db            *database.DB
	cfg           *config.Config
	promoCodeRepo *repository.PromoCodeRepository
}

// NewPromoCodeHandler creates a new promo code handler
func NewPromoCodeHandler(db *database.DB, cfg *config.Config) *PromoCodeHandler {
	return &PromoCodeHandler{
		db:            db,
		cfg:           cfg,
		promoCodeRepo: repository.NewPromoCodeRepository(db),
	}
}

// promoCodePattern allows alphanumeric characters, hyphens, and underscores
var promoCodePattern = regexp.MustCompile(`^[A-Z0-9_-]+$`)

// CreatePromoCodeRequest represents a request to create a promo code
type CreatePromoCodeRequest struct {
	Code          string  `json:"code"`
	Description   string  `json:"description,omitempty"`
	DiscountType  string  `json:"discount_type"`  // percentage, fixed, free_months
	DiscountValue int     `json:"discount_value"` // percentage (1-100), cents, or months (1-24)
	MaxUses       *int    `json:"max_uses,omitempty"`
	ValidForPlans string  `json:"valid_for_plans,omitempty"` // JSON array: ["pro"]
	IsActive      bool    `json:"is_active"`
	ExpiresAt     *string `json:"expires_at,omitempty"` // ISO 8601 format
}

// CreatePromoCode creates a new promo code (Central Admin only)
// POST /api/v1/central-admin/promo-codes
func (h *PromoCodeHandler) CreatePromoCode(w http.ResponseWriter, r *http.Request) {
	var req CreatePromoCodeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Ungültige Anfrage")
		return
	}

	// Sanitize and validate code
	code := strings.ToUpper(strings.TrimSpace(req.Code))
	if code == "" {
		respondError(w, http.StatusBadRequest, "Code ist erforderlich")
		return
	}
	if len(code) < 3 || len(code) > 50 {
		respondError(w, http.StatusBadRequest, "Code muss zwischen 3 und 50 Zeichen lang sein")
		return
	}
	if !promoCodePattern.MatchString(code) {
		respondError(w, http.StatusBadRequest, "Code darf nur Buchstaben, Zahlen, Bindestriche und Unterstriche enthalten")
		return
	}

	// Check if code already exists
	existing, err := h.promoCodeRepo.GetByCode(code)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Fehler beim Prüfen des Codes")
		return
	}
	if existing != nil {
		respondError(w, http.StatusConflict, "Code existiert bereits")
		return
	}

	// Validate discount type and value
	switch req.DiscountType {
	case models.DiscountTypePercentage:
		if req.DiscountValue < 1 || req.DiscountValue > models.MaxPromoDiscountPercent {
			respondError(w, http.StatusBadRequest, "Prozent muss zwischen 1 und 100 liegen")
			return
		}
	case models.DiscountTypeFixed:
		if req.DiscountValue < 1 {
			respondError(w, http.StatusBadRequest, "Rabattbetrag muss positiv sein")
			return
		}
	case models.DiscountTypeFreeMonths:
		if req.DiscountValue < 1 || req.DiscountValue > models.MaxPromoDiscountMonths {
			respondError(w, http.StatusBadRequest, "Gratismonate müssen zwischen 1 und 24 liegen")
			return
		}
	default:
		respondError(w, http.StatusBadRequest, "Ungültiger Rabatttyp (percentage, fixed, free_months)")
		return
	}

	promoCode := &models.PromoCode{
		Code:          code,
		Description:   req.Description,
		DiscountType:  req.DiscountType,
		DiscountValue: req.DiscountValue,
		MaxUses:       req.MaxUses,
		ValidForPlans: req.ValidForPlans,
		IsActive:      req.IsActive,
	}

	// Parse expires_at if provided (accepts RFC3339 or YYYY-MM-DD format)
	if req.ExpiresAt != nil && *req.ExpiresAt != "" {
		var parsedTime time.Time
		var parseErr error
		// Try RFC3339 first (ISO 8601)
		parsedTime, parseErr = time.Parse(time.RFC3339, *req.ExpiresAt)
		if parseErr != nil {
			// Try simple date format
			parsedTime, parseErr = time.Parse("2006-01-02", *req.ExpiresAt)
		}
		if parseErr != nil {
			respondError(w, http.StatusBadRequest, "Ungültiges Ablaufdatum (erwartet: YYYY-MM-DD oder ISO 8601)")
			return
		}
		promoCode.ExpiresAt = &parsedTime
	}

	if err := h.promoCodeRepo.Create(promoCode); err != nil {
		respondError(w, http.StatusInternalServerError, "Fehler beim Erstellen des Codes")
		return
	}

	respondJSON(w, http.StatusCreated, promoCode)
}

// GetAllPromoCodes returns all promo codes (Central Admin only)
// GET /api/v1/central-admin/promo-codes
func (h *PromoCodeHandler) GetAllPromoCodes(w http.ResponseWriter, r *http.Request) {
	activeOnly := r.URL.Query().Get("active_only") == "true"

	codes, err := h.promoCodeRepo.GetAll(activeOnly)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Fehler beim Laden der Codes")
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"promo_codes": codes,
		"count":       len(codes),
	})
}

// GetPromoCode returns a single promo code (Central Admin only)
// GET /api/v1/central-admin/promo-codes/{id}
func (h *PromoCodeHandler) GetPromoCode(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		respondError(w, http.StatusBadRequest, "Ungültige Code-ID")
		return
	}

	code, err := h.promoCodeRepo.GetByID(id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Fehler beim Laden des Codes")
		return
	}
	if code == nil {
		respondError(w, http.StatusNotFound, "Code nicht gefunden")
		return
	}

	// Get usage history
	uses, err := h.promoCodeRepo.GetCodeUses(id)
	if err != nil {
		uses = []*models.PromoCodeUse{} // Don't fail, just return empty list
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"promo_code": code,
		"uses":       uses,
	})
}

// UpdatePromoCode updates a promo code (Central Admin only)
// PUT /api/v1/central-admin/promo-codes/{id}
func (h *PromoCodeHandler) UpdatePromoCode(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		respondError(w, http.StatusBadRequest, "Ungültige Code-ID")
		return
	}

	code, err := h.promoCodeRepo.GetByID(id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Fehler beim Laden des Codes")
		return
	}
	if code == nil {
		respondError(w, http.StatusNotFound, "Code nicht gefunden")
		return
	}

	var req CreatePromoCodeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Ungültige Anfrage")
		return
	}

	// Validate code format
	newCode := strings.ToUpper(strings.TrimSpace(req.Code))
	if newCode == "" {
		respondError(w, http.StatusBadRequest, "Code ist erforderlich")
		return
	}
	if !promoCodePattern.MatchString(newCode) {
		respondError(w, http.StatusBadRequest, "Code darf nur Buchstaben, Zahlen, Bindestriche und Unterstriche enthalten")
		return
	}

	// Check if new code conflicts with another code
	if newCode != code.Code {
		existing, err := h.promoCodeRepo.GetByCode(newCode)
		if err != nil {
			respondError(w, http.StatusInternalServerError, "Fehler beim Prüfen des Codes")
			return
		}
		if existing != nil {
			respondError(w, http.StatusConflict, "Code existiert bereits")
			return
		}
	}

	// Validate discount type and value
	switch req.DiscountType {
	case models.DiscountTypePercentage:
		if req.DiscountValue < 1 || req.DiscountValue > models.MaxPromoDiscountPercent {
			respondError(w, http.StatusBadRequest, "Prozent muss zwischen 1 und 100 liegen")
			return
		}
	case models.DiscountTypeFixed:
		if req.DiscountValue < 1 {
			respondError(w, http.StatusBadRequest, "Rabattbetrag muss positiv sein")
			return
		}
	case models.DiscountTypeFreeMonths:
		if req.DiscountValue < 1 || req.DiscountValue > models.MaxPromoDiscountMonths {
			respondError(w, http.StatusBadRequest, "Gratismonate müssen zwischen 1 und 24 liegen")
			return
		}
	default:
		respondError(w, http.StatusBadRequest, "Ungültiger Rabatttyp")
		return
	}

	// Update fields
	code.Code = newCode
	code.Description = req.Description
	code.DiscountType = req.DiscountType
	code.DiscountValue = req.DiscountValue
	code.MaxUses = req.MaxUses
	code.ValidForPlans = req.ValidForPlans
	code.IsActive = req.IsActive

	// Parse expires_at (accepts RFC3339 or YYYY-MM-DD format)
	if req.ExpiresAt != nil && *req.ExpiresAt != "" {
		var parsedTime time.Time
		var parseErr error
		// Try RFC3339 first (ISO 8601)
		parsedTime, parseErr = time.Parse(time.RFC3339, *req.ExpiresAt)
		if parseErr != nil {
			// Try simple date format
			parsedTime, parseErr = time.Parse("2006-01-02", *req.ExpiresAt)
		}
		if parseErr != nil {
			respondError(w, http.StatusBadRequest, "Ungültiges Ablaufdatum (erwartet: YYYY-MM-DD oder ISO 8601)")
			return
		}
		code.ExpiresAt = &parsedTime
	} else {
		code.ExpiresAt = nil
	}

	if err := h.promoCodeRepo.Update(code); err != nil {
		respondError(w, http.StatusInternalServerError, "Fehler beim Aktualisieren des Codes")
		return
	}

	respondJSON(w, http.StatusOK, code)
}

// DeletePromoCode deletes a promo code (Central Admin only)
// DELETE /api/v1/central-admin/promo-codes/{id}
func (h *PromoCodeHandler) DeletePromoCode(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		respondError(w, http.StatusBadRequest, "Ungültige Code-ID")
		return
	}

	code, err := h.promoCodeRepo.GetByID(id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Fehler beim Laden des Codes")
		return
	}
	if code == nil {
		respondError(w, http.StatusNotFound, "Code nicht gefunden")
		return
	}

	if err := h.promoCodeRepo.Delete(id); err != nil {
		respondError(w, http.StatusInternalServerError, "Fehler beim Löschen des Codes")
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "Code erfolgreich gelöscht"})
}

// ValidatePromoCode validates a promo code (Public endpoint)
// GET /api/v1/promo-codes/validate/{code}
func (h *PromoCodeHandler) ValidatePromoCode(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	codeStr := strings.ToUpper(strings.TrimSpace(vars["code"]))

	if codeStr == "" {
		respondJSON(w, http.StatusOK, map[string]interface{}{
			"valid":   false,
			"message": "Code ist erforderlich",
		})
		return
	}

	code, err := h.promoCodeRepo.GetByCode(codeStr)
	if err != nil {
		respondJSON(w, http.StatusOK, map[string]interface{}{
			"valid":   false,
			"message": "Fehler beim Prüfen des Codes",
		})
		return
	}

	if code == nil || !code.IsValid() {
		respondJSON(w, http.StatusOK, map[string]interface{}{
			"valid":   false,
			"message": "Ungültiger oder abgelaufener Code",
		})
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"valid":          true,
		"discount_type":  code.DiscountType,
		"discount_value": code.DiscountValue,
		"description":    code.GetDiscountDescription(),
		"message":        code.GetDiscountDescription(),
	})
}
