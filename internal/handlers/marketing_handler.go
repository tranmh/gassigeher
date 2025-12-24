package handlers

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/tranmh/gassigeher/internal/models"
	"github.com/tranmh/gassigeher/internal/repository"
)

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
		respondError(w, http.StatusInternalServerError, "Failed to fetch campaigns")
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
	id, _ := strconv.Atoi(mux.Vars(r)["id"])
	campaign, err := h.marketingRepo.GetCampaign(id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to fetch campaign")
		return
	}
	if campaign == nil {
		respondError(w, http.StatusNotFound, "Campaign not found")
		return
	}
	respondJSON(w, http.StatusOK, campaign)
}

// CreateCampaign creates a new campaign (Central Admin only)
// POST /api/v1/central-admin/marketing/campaigns
func (h *MarketingHandler) CreateCampaign(w http.ResponseWriter, r *http.Request) {
	var campaign models.MarketingCampaign
	if err := json.NewDecoder(r.Body).Decode(&campaign); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate type
	validTypes := map[string]bool{"fomo_countdown": true, "referral": true, "reference_page": true, "custom": true}
	if !validTypes[campaign.Type] {
		respondError(w, http.StatusBadRequest, "Invalid campaign type")
		return
	}

	if err := h.marketingRepo.CreateCampaign(&campaign); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to create campaign")
		return
	}
	respondJSON(w, http.StatusCreated, campaign)
}

// UpdateCampaign updates a campaign (Central Admin only)
// PUT /api/v1/central-admin/marketing/campaigns/:id
func (h *MarketingHandler) UpdateCampaign(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(mux.Vars(r)["id"])
	campaign, err := h.marketingRepo.GetCampaign(id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to fetch campaign")
		return
	}
	if campaign == nil {
		respondError(w, http.StatusNotFound, "Campaign not found")
		return
	}

	var req models.UpdateCampaignRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
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
		respondError(w, http.StatusInternalServerError, "Failed to update campaign")
		return
	}
	respondJSON(w, http.StatusOK, campaign)
}

// DeleteCampaign deletes a campaign (Central Admin only)
// DELETE /api/v1/central-admin/marketing/campaigns/:id
func (h *MarketingHandler) DeleteCampaign(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(mux.Vars(r)["id"])
	if err := h.marketingRepo.DeleteCampaign(id); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to delete campaign")
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{"message": "Campaign deleted"})
}

// GetActiveFOMO returns the active FOMO countdown (Public)
// GET /api/v1/marketing/fomo
func (h *MarketingHandler) GetActiveFOMO(w http.ResponseWriter, r *http.Request) {
	campaign, err := h.marketingRepo.GetActiveCampaignByType("fomo_countdown")
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to fetch FOMO campaign")
		return
	}
	if campaign == nil {
		respondJSON(w, http.StatusOK, map[string]interface{}{"active": false})
		return
	}

	config, _ := campaign.GetFOMOConfig()
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"active":  true,
		"name":    campaign.Name,
		"config":  config,
		"ends_at": campaign.EndDate,
	})
}

// ========== Referral Codes ==========

// ListReferralCodes returns all referral codes (Central Admin only)
// GET /api/v1/central-admin/marketing/referral-codes
func (h *MarketingHandler) ListReferralCodes(w http.ResponseWriter, r *http.Request) {
	codes, err := h.marketingRepo.ListReferralCodes()
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to fetch referral codes")
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
	id, _ := strconv.Atoi(mux.Vars(r)["id"])
	code, err := h.marketingRepo.GetReferralCode(id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to fetch referral code")
		return
	}
	if code == nil {
		respondError(w, http.StatusNotFound, "Referral code not found")
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
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	code := &models.ReferralCode{
		Code:                   strings.ToUpper(strings.TrimSpace(req.Code)),
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

	// Parse expiry date
	if req.ExpiresAt != nil {
		t, _ := time.Parse("2006-01-02", *req.ExpiresAt)
		code.ExpiresAt = &t
	}

	// Check for duplicate
	existing, _ := h.marketingRepo.GetReferralCodeByCode(code.Code)
	if existing != nil {
		respondError(w, http.StatusConflict, "Referral code already exists")
		return
	}

	if err := h.marketingRepo.CreateReferralCode(code); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to create referral code")
		return
	}
	respondJSON(w, http.StatusCreated, code)
}

// UpdateReferralCode updates a referral code (Central Admin only)
// PUT /api/v1/central-admin/marketing/referral-codes/:id
func (h *MarketingHandler) UpdateReferralCode(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(mux.Vars(r)["id"])
	code, err := h.marketingRepo.GetReferralCode(id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to fetch referral code")
		return
	}
	if code == nil {
		respondError(w, http.StatusNotFound, "Referral code not found")
		return
	}

	var req models.CreateReferralCodeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Code != "" {
		code.Code = strings.ToUpper(strings.TrimSpace(req.Code))
	}
	code.ReferrerEmail = req.ReferrerEmail
	code.DiscountMonthsReferrer = req.DiscountMonthsReferrer
	code.DiscountMonthsReferee = req.DiscountMonthsReferee
	code.MaxUses = req.MaxUses

	if req.ExpiresAt != nil {
		t, _ := time.Parse("2006-01-02", *req.ExpiresAt)
		code.ExpiresAt = &t
	}

	if err := h.marketingRepo.UpdateReferralCode(code); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to update referral code")
		return
	}
	respondJSON(w, http.StatusOK, code)
}

// ToggleReferralCode toggles the active status of a referral code (Central Admin only)
// PUT /api/v1/central-admin/marketing/referral-codes/:id/toggle
func (h *MarketingHandler) ToggleReferralCode(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(mux.Vars(r)["id"])
	code, err := h.marketingRepo.GetReferralCode(id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to fetch referral code")
		return
	}
	if code == nil {
		respondError(w, http.StatusNotFound, "Referral code not found")
		return
	}

	code.IsActive = !code.IsActive
	if err := h.marketingRepo.UpdateReferralCode(code); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to update referral code")
		return
	}
	respondJSON(w, http.StatusOK, code)
}

// DeleteReferralCode deletes a referral code (Central Admin only)
// DELETE /api/v1/central-admin/marketing/referral-codes/:id
func (h *MarketingHandler) DeleteReferralCode(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(mux.Vars(r)["id"])
	if err := h.marketingRepo.DeleteReferralCode(id); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to delete referral code")
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{"message": "Referral code deleted"})
}

// ValidateReferralCode validates a referral code (Public - during registration)
// GET /api/v1/marketing/referral/:code
func (h *MarketingHandler) ValidateReferralCode(w http.ResponseWriter, r *http.Request) {
	codeStr := mux.Vars(r)["code"]
	code, err := h.marketingRepo.GetReferralCodeByCode(strings.ToUpper(codeStr))
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to validate code")
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
		respondError(w, http.StatusInternalServerError, "Failed to fetch reference entries")
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
	id, _ := strconv.Atoi(mux.Vars(r)["id"])
	entry, err := h.marketingRepo.GetReferenceEntry(id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to fetch reference entry")
		return
	}
	if entry == nil {
		respondError(w, http.StatusNotFound, "Reference entry not found")
		return
	}
	respondJSON(w, http.StatusOK, entry)
}

// ApproveReferenceEntry approves a reference entry (Central Admin only)
// PUT /api/v1/central-admin/marketing/references/:id/approve
func (h *MarketingHandler) ApproveReferenceEntry(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(mux.Vars(r)["id"])
	if err := h.marketingRepo.ApproveReferenceEntry(id); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to approve entry")
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{"message": "Entry approved"})
}

// DeleteReferenceEntry deletes a reference entry (Central Admin only)
// DELETE /api/v1/central-admin/marketing/references/:id
func (h *MarketingHandler) DeleteReferenceEntry(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(mux.Vars(r)["id"])
	if err := h.marketingRepo.DeleteReferenceEntry(id); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to delete entry")
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{"message": "Entry deleted"})
}

// ========== Stats ==========

// GetMarketingStats returns marketing statistics (Central Admin only)
// GET /api/v1/central-admin/marketing/stats
func (h *MarketingHandler) GetMarketingStats(w http.ResponseWriter, r *http.Request) {
	stats, err := h.marketingRepo.GetMarketingStats()
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to fetch stats")
		return
	}
	respondJSON(w, http.StatusOK, stats)
}

// Helper to generate a random referral code
func generateReferralCode() string {
	bytes := make([]byte, 4)
	rand.Read(bytes)
	return "GH-" + strings.ToUpper(hex.EncodeToString(bytes))
}
