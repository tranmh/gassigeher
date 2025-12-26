package handlers

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/tranmh/gassigeher/internal/models"
	"github.com/tranmh/gassigeher/internal/repository"
)

// MaxDiscountMonths is the maximum allowed discount months for referral codes
const MaxDiscountMonths = 24

// referralCodePattern validates referral codes - only alphanumeric and hyphens allowed
var referralCodePattern = regexp.MustCompile(`^[A-Z0-9-]+$`)

// MarketingHandler handles marketing-related requests
type MarketingHandler struct {
	db            *sql.DB
	marketingRepo *repository.MarketingRepository
}

// NewMarketingHandler creates a new marketing handler
func NewMarketingHandler(db *sql.DB) *MarketingHandler {
	return &MarketingHandler{
		db:            db,
		marketingRepo: repository.NewMarketingRepository(db),
	}
}

// ========== Campaigns ==========

// ListCampaigns returns all marketing campaigns (Central Admin only)
// GET /api/v1/central-admin/marketing/campaigns
func (h *MarketingHandler) ListCampaigns(w http.ResponseWriter, r *http.Request) {
	campaigns, err := h.marketingRepo.ListCampaigns()
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Fehler beim Laden der Kampagnen")
		return
	}
	if campaigns == nil {
		campaigns = []*models.MarketingCampaign{}
	}
	respondJSON(w, http.StatusOK, campaigns)
}

// GetCampaign returns a campaign by ID (Central Admin only)
// GET /api/v1/central-admin/marketing/campaigns/:id
func (h *MarketingHandler) GetCampaign(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		respondError(w, http.StatusBadRequest, "Ungültige Kampagnen-ID")
		return
	}
	campaign, err := h.marketingRepo.GetCampaign(id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Fehler beim Laden der Kampagne")
		return
	}
	if campaign == nil {
		respondError(w, http.StatusNotFound, "Kampagne nicht gefunden")
		return
	}
	respondJSON(w, http.StatusOK, campaign)
}

// CreateCampaign creates a new campaign (Central Admin only)
// POST /api/v1/central-admin/marketing/campaigns
func (h *MarketingHandler) CreateCampaign(w http.ResponseWriter, r *http.Request) {
	var campaign models.MarketingCampaign
	if err := json.NewDecoder(r.Body).Decode(&campaign); err != nil {
		respondError(w, http.StatusBadRequest, "Ungültige Anfrage")
		return
	}

	// Validate name
	campaign.Name = strings.TrimSpace(campaign.Name)
	if campaign.Name == "" {
		respondError(w, http.StatusBadRequest, "Name ist erforderlich")
		return
	}
	if len(campaign.Name) > 255 {
		respondError(w, http.StatusBadRequest, "Name darf maximal 255 Zeichen lang sein")
		return
	}

	// Validate type
	validTypes := map[string]bool{"fomo_countdown": true, "referral": true, "reference_page": true, "custom": true}
	if !validTypes[campaign.Type] {
		respondError(w, http.StatusBadRequest, "Ungültiger Kampagnentyp")
		return
	}

	// Validate date range if both are provided
	if campaign.StartDate != nil && campaign.EndDate != nil {
		if campaign.EndDate.Before(*campaign.StartDate) {
			respondError(w, http.StatusBadRequest, "Enddatum muss nach dem Startdatum liegen")
			return
		}
	}

	// Validate config JSON if provided
	if campaign.Config != nil && *campaign.Config != "" {
		var configTest interface{}
		if err := json.Unmarshal([]byte(*campaign.Config), &configTest); err != nil {
			respondError(w, http.StatusBadRequest, "Ungültiges JSON-Format in config")
			return
		}
	}

	if err := h.marketingRepo.CreateCampaign(&campaign); err != nil {
		respondError(w, http.StatusInternalServerError, "Fehler beim Erstellen der Kampagne")
		return
	}
	respondJSON(w, http.StatusCreated, campaign)
}

// UpdateCampaign updates a campaign (Central Admin only)
// PUT /api/v1/central-admin/marketing/campaigns/:id
func (h *MarketingHandler) UpdateCampaign(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		respondError(w, http.StatusBadRequest, "Ungültige Kampagnen-ID")
		return
	}
	campaign, err := h.marketingRepo.GetCampaign(id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Fehler beim Laden der Kampagne")
		return
	}
	if campaign == nil {
		respondError(w, http.StatusNotFound, "Kampagne nicht gefunden")
		return
	}

	var req models.UpdateCampaignRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Ungültige Anfrage")
		return
	}

	if req.Name != nil {
		campaign.Name = *req.Name
	}
	if req.Description != nil {
		campaign.Description = req.Description
	}
	if req.Config != nil {
		campaign.Config = req.Config
	}
	if req.IsActive != nil {
		campaign.IsActive = *req.IsActive
	}
	if req.StartDate != nil {
		t, _ := time.Parse("2006-01-02", *req.StartDate)
		campaign.StartDate = &t
	}
	if req.EndDate != nil {
		t, _ := time.Parse("2006-01-02", *req.EndDate)
		campaign.EndDate = &t
	}

	if err := h.marketingRepo.UpdateCampaign(campaign); err != nil {
		respondError(w, http.StatusInternalServerError, "Fehler beim Aktualisieren der Kampagne")
		return
	}
	respondJSON(w, http.StatusOK, campaign)
}

// DeleteCampaign deletes a campaign (Central Admin only)
// DELETE /api/v1/central-admin/marketing/campaigns/:id
func (h *MarketingHandler) DeleteCampaign(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		respondError(w, http.StatusBadRequest, "Ungültige Kampagnen-ID")
		return
	}
	if err := h.marketingRepo.DeleteCampaign(id); err != nil {
		respondError(w, http.StatusInternalServerError, "Fehler beim Löschen der Kampagne")
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{"message": "Kampagne gelöscht"})
}

// GetActiveFOMO returns the active FOMO countdown (Public)
// GET /api/v1/marketing/fomo
func (h *MarketingHandler) GetActiveFOMO(w http.ResponseWriter, r *http.Request) {
	campaign, err := h.marketingRepo.GetActiveCampaignByType("fomo_countdown")
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Fehler beim Laden der FOMO-Kampagne")
		return
	}
	if campaign == nil {
		respondJSON(w, http.StatusOK, map[string]interface{}{"active": false})
		return
	}

	config, _ := campaign.GetFOMOConfig()
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"active": true,
		"campaign": map[string]interface{}{
			"id":          campaign.ID,
			"type":        campaign.Type,
			"name":        campaign.Name,
			"description": campaign.Description,
			"config":      config,
			"start_date":  campaign.StartDate,
			"end_date":    campaign.EndDate,
		},
	})
}

// ========== Referral Codes ==========

// ListReferralCodes returns all referral codes (Central Admin only)
// GET /api/v1/central-admin/marketing/referral-codes
func (h *MarketingHandler) ListReferralCodes(w http.ResponseWriter, r *http.Request) {
	codes, err := h.marketingRepo.ListReferralCodes()
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Fehler beim Laden der Empfehlungscodes")
		return
	}
	if codes == nil {
		codes = []*models.ReferralCode{}
	}
	respondJSON(w, http.StatusOK, codes)
}

// GetReferralCode returns a referral code by ID (Central Admin only)
// GET /api/v1/central-admin/marketing/referral-codes/:id
func (h *MarketingHandler) GetReferralCode(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		respondError(w, http.StatusBadRequest, "Ungültige Empfehlungscode-ID")
		return
	}
	code, err := h.marketingRepo.GetReferralCode(id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Fehler beim Laden des Empfehlungscodes")
		return
	}
	if code == nil {
		respondError(w, http.StatusNotFound, "Empfehlungscode nicht gefunden")
		return
	}

	// Get uses
	uses, _ := h.marketingRepo.GetReferralUses(id)
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"code": code,
		"uses": uses,
	})
}

// CreateReferralCode creates a new referral code (Central Admin only)
// POST /api/v1/central-admin/marketing/referral-codes
func (h *MarketingHandler) CreateReferralCode(w http.ResponseWriter, r *http.Request) {
	var req models.CreateReferralCodeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Ungültige Anfrage")
		return
	}

	// Validate discount months - must be non-negative
	if req.DiscountMonthsReferrer < 0 {
		respondError(w, http.StatusBadRequest, "Rabattmonate für Empfehler darf nicht negativ sein")
		return
	}
	if req.DiscountMonthsReferee < 0 {
		respondError(w, http.StatusBadRequest, "Rabattmonate für Empfohlene darf nicht negativ sein")
		return
	}

	// Validate discount months - max limit (prevent abuse)
	if req.DiscountMonthsReferrer > MaxDiscountMonths {
		respondError(w, http.StatusBadRequest, "Rabattmonate für Empfehler darf maximal 24 sein")
		return
	}
	if req.DiscountMonthsReferee > MaxDiscountMonths {
		respondError(w, http.StatusBadRequest, "Rabattmonate für Empfohlene darf maximal 24 sein")
		return
	}

	// Sanitize and validate code - only alphanumeric and hyphens allowed
	sanitizedCode := strings.ToUpper(strings.TrimSpace(req.Code))
	if sanitizedCode != "" {
		// Remove any HTML tags and dangerous characters
		sanitizedCode = sanitizeReferralCode(sanitizedCode)
		if sanitizedCode == "" || !referralCodePattern.MatchString(sanitizedCode) {
			respondError(w, http.StatusBadRequest, "Ungültiger Empfehlungscode. Nur Buchstaben, Zahlen und Bindestriche erlaubt.")
			return
		}
	}

	code := &models.ReferralCode{
		Code:                   sanitizedCode,
		ReferrerEmail:          req.ReferrerEmail,
		DiscountMonthsReferrer: req.DiscountMonthsReferrer,
		DiscountMonthsReferee:  req.DiscountMonthsReferee,
		MaxUses:                req.MaxUses,
		IsActive:               true,
	}

	// Generate code if not provided
	if code.Code == "" {
		code.Code = generateReferralCode()
	}

	// Parse expiry date - try multiple formats
	if req.ExpiresAt != nil && *req.ExpiresAt != "" {
		var parsedTime time.Time
		var err error

		// Try RFC3339 first (e.g., "2025-12-31T23:59:59Z")
		parsedTime, err = time.Parse(time.RFC3339, *req.ExpiresAt)
		if err != nil {
			// Try date-only format (e.g., "2025-12-31")
			parsedTime, err = time.Parse("2006-01-02", *req.ExpiresAt)
		}
		if err != nil {
			respondError(w, http.StatusBadRequest, "Ungültiges Datumsformat für expires_at (erwartet: YYYY-MM-DD oder ISO 8601)")
			return
		}

		// Validate expiry date is not in the past
		if parsedTime.Before(time.Now()) {
			respondError(w, http.StatusBadRequest, "Ablaufdatum darf nicht in der Vergangenheit liegen")
			return
		}

		code.ExpiresAt = &parsedTime
	}

	// Check for duplicate
	existing, _ := h.marketingRepo.GetReferralCodeByCode(code.Code)
	if existing != nil {
		respondError(w, http.StatusConflict, "Empfehlungscode existiert bereits")
		return
	}

	if err := h.marketingRepo.CreateReferralCode(code); err != nil {
		respondError(w, http.StatusInternalServerError, "Fehler beim Erstellen des Empfehlungscodes")
		return
	}
	respondJSON(w, http.StatusCreated, code)
}

// sanitizeReferralCode removes HTML tags and dangerous characters from referral code
func sanitizeReferralCode(code string) string {
	// Remove HTML tags
	tagPattern := regexp.MustCompile(`<[^>]*>`)
	code = tagPattern.ReplaceAllString(code, "")

	// Remove any remaining special characters except alphanumeric and hyphen
	cleanPattern := regexp.MustCompile(`[^A-Z0-9-]`)
	code = cleanPattern.ReplaceAllString(code, "")

	return code
}

// UpdateReferralCode updates a referral code (Central Admin only)
// PUT /api/v1/central-admin/marketing/referral-codes/:id
func (h *MarketingHandler) UpdateReferralCode(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		respondError(w, http.StatusBadRequest, "Ungültige Empfehlungscode-ID")
		return
	}
	code, err := h.marketingRepo.GetReferralCode(id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Fehler beim Laden des Empfehlungscodes")
		return
	}
	if code == nil {
		respondError(w, http.StatusNotFound, "Empfehlungscode nicht gefunden")
		return
	}

	var req models.CreateReferralCodeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Ungültige Anfrage")
		return
	}

	if req.Code != "" {
		code.Code = strings.ToUpper(strings.TrimSpace(req.Code))
	}
	code.ReferrerEmail = req.ReferrerEmail
	code.DiscountMonthsReferrer = req.DiscountMonthsReferrer
	code.DiscountMonthsReferee = req.DiscountMonthsReferee
	code.MaxUses = req.MaxUses

	// Parse expiry date - try multiple formats
	if req.ExpiresAt != nil && *req.ExpiresAt != "" {
		var parsedTime time.Time
		var parseErr error

		// Try RFC3339 first (e.g., "2025-12-31T23:59:59Z")
		parsedTime, parseErr = time.Parse(time.RFC3339, *req.ExpiresAt)
		if parseErr != nil {
			// Try date-only format (e.g., "2025-12-31")
			parsedTime, parseErr = time.Parse("2006-01-02", *req.ExpiresAt)
		}
		if parseErr != nil {
			respondError(w, http.StatusBadRequest, "Ungültiges Datumsformat für expires_at (erwartet: YYYY-MM-DD oder ISO 8601)")
			return
		}
		code.ExpiresAt = &parsedTime
	}

	if err := h.marketingRepo.UpdateReferralCode(code); err != nil {
		respondError(w, http.StatusInternalServerError, "Fehler beim Aktualisieren des Empfehlungscodes")
		return
	}
	respondJSON(w, http.StatusOK, code)
}

// ToggleReferralCode toggles the active status of a referral code (Central Admin only)
// PUT /api/v1/central-admin/marketing/referral-codes/:id/toggle
func (h *MarketingHandler) ToggleReferralCode(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		respondError(w, http.StatusBadRequest, "Ungültige Empfehlungscode-ID")
		return
	}
	code, err := h.marketingRepo.GetReferralCode(id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Fehler beim Laden des Empfehlungscodes")
		return
	}
	if code == nil {
		respondError(w, http.StatusNotFound, "Empfehlungscode nicht gefunden")
		return
	}

	code.IsActive = !code.IsActive
	if err := h.marketingRepo.UpdateReferralCode(code); err != nil {
		respondError(w, http.StatusInternalServerError, "Fehler beim Aktualisieren des Empfehlungscodes")
		return
	}
	respondJSON(w, http.StatusOK, code)
}

// DeleteReferralCode deletes a referral code (Central Admin only)
// DELETE /api/v1/central-admin/marketing/referral-codes/:id
func (h *MarketingHandler) DeleteReferralCode(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		respondError(w, http.StatusBadRequest, "Ungültige Empfehlungscode-ID")
		return
	}
	if err := h.marketingRepo.DeleteReferralCode(id); err != nil {
		respondError(w, http.StatusInternalServerError, "Fehler beim Löschen des Empfehlungscodes")
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{"message": "Empfehlungscode gelöscht"})
}

// ValidateReferralCode validates a referral code (Public - during registration)
// GET /api/v1/marketing/referral/:code
func (h *MarketingHandler) ValidateReferralCode(w http.ResponseWriter, r *http.Request) {
	codeStr := mux.Vars(r)["code"]
	code, err := h.marketingRepo.GetReferralCodeByCode(strings.ToUpper(codeStr))
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Fehler beim Validieren des Codes")
		return
	}
	if code == nil || !code.IsValid() {
		respondJSON(w, http.StatusOK, map[string]interface{}{
			"valid":   false,
			"message": "Ungültiger oder abgelaufener Empfehlungscode",
		})
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"valid":          true,
		"discount_months": code.DiscountMonthsReferee,
		"message":        "Code gültig! Sie erhalten " + strconv.Itoa(code.DiscountMonthsReferee) + " Monat(e) kostenlos.",
	})
}

// ========== Reference Page ==========

// ListReferenceEntries returns reference entries (Public: approved only, Central Admin: all)
// GET /api/v1/marketing/references
// GET /api/v1/central-admin/marketing/references
func (h *MarketingHandler) ListReferenceEntries(w http.ResponseWriter, r *http.Request) {
	// Check if central admin request
	approvedOnly := !strings.Contains(r.URL.Path, "central-admin")

	entries, err := h.marketingRepo.ListReferenceEntries(approvedOnly)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Fehler beim Laden der Referenzen")
		return
	}
	if entries == nil {
		entries = []*models.ReferenceEntry{}
	}
	respondJSON(w, http.StatusOK, entries)
}

// GetReferenceEntry returns a reference entry by ID (Central Admin only)
// GET /api/v1/central-admin/marketing/references/:id
func (h *MarketingHandler) GetReferenceEntry(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		respondError(w, http.StatusBadRequest, "Ungültige Referenz-ID")
		return
	}
	entry, err := h.marketingRepo.GetReferenceEntry(id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Fehler beim Laden der Referenz")
		return
	}
	if entry == nil {
		respondError(w, http.StatusNotFound, "Referenz nicht gefunden")
		return
	}
	respondJSON(w, http.StatusOK, entry)
}

// ApproveReferenceEntry approves a reference entry (Central Admin only)
// PUT /api/v1/central-admin/marketing/references/:id/approve
func (h *MarketingHandler) ApproveReferenceEntry(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		respondError(w, http.StatusBadRequest, "Ungültige Referenz-ID")
		return
	}
	if err := h.marketingRepo.ApproveReferenceEntry(id); err != nil {
		respondError(w, http.StatusInternalServerError, "Fehler beim Genehmigen der Referenz")
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{"message": "Referenz genehmigt"})
}

// DeleteReferenceEntry deletes a reference entry (Central Admin only)
// DELETE /api/v1/central-admin/marketing/references/:id
func (h *MarketingHandler) DeleteReferenceEntry(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		respondError(w, http.StatusBadRequest, "Ungültige Referenz-ID")
		return
	}
	if err := h.marketingRepo.DeleteReferenceEntry(id); err != nil {
		respondError(w, http.StatusInternalServerError, "Fehler beim Löschen der Referenz")
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{"message": "Referenz gelöscht"})
}

// ========== Stats ==========

// GetMarketingStats returns marketing statistics (Central Admin only)
// GET /api/v1/central-admin/marketing/stats
func (h *MarketingHandler) GetMarketingStats(w http.ResponseWriter, r *http.Request) {
	stats, err := h.marketingRepo.GetMarketingStats()
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Fehler beim Laden der Statistiken")
		return
	}
	respondJSON(w, http.StatusOK, stats)
}

// Helper to generate a random referral code
func generateReferralCode() string {
	bytes := make([]byte, 4)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback to timestamp-based code if random fails
		return "GH-" + strings.ToUpper(hex.EncodeToString([]byte(strconv.FormatInt(time.Now().UnixNano(), 16))[:8]))
	}
	return "GH-" + strings.ToUpper(hex.EncodeToString(bytes))
}
