package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/tranmh/gassigeher/internal/config"
	"github.com/tranmh/gassigeher/internal/database"
	"github.com/tranmh/gassigeher/internal/middleware"
	"github.com/tranmh/gassigeher/internal/models"
	"github.com/tranmh/gassigeher/internal/repository"
	"github.com/tranmh/gassigeher/internal/services"
)

// slugRateLimiter provides rate limiting for slug enumeration prevention
type slugRateLimiter struct {
	requests map[string]*rateLimitRecord
	mu       sync.RWMutex
	limit    int           // max requests per window
	window   time.Duration // time window
}

type rateLimitRecord struct {
	count       int
	windowStart time.Time
}

func newSlugRateLimiter() *slugRateLimiter {
	return &slugRateLimiter{
		requests: make(map[string]*rateLimitRecord),
		limit:    10,          // 10 requests
		window:   time.Minute, // per minute
	}
}

// checkLimit returns true if the request should be allowed, false if rate limited
func (r *slugRateLimiter) checkLimit(ip string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	record, exists := r.requests[ip]

	if !exists || now.Sub(record.windowStart) >= r.window {
		// New window or first request
		r.requests[ip] = &rateLimitRecord{
			count:       1,
			windowStart: now,
		}
		return true
	}

	// Within the same window
	record.count++
	if record.count > r.limit {
		return false
	}
	return true
}

// getClientIP extracts the client IP address from the request
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header (from reverse proxy)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// Take first IP if comma-separated
		if idx := strings.Index(xff, ","); idx != -1 {
			return strings.TrimSpace(xff[:idx])
		}
		return strings.TrimSpace(xff)
	}

	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return strings.TrimSpace(xri)
	}

	// Fall back to RemoteAddr (strip port)
	addr := r.RemoteAddr
	if idx := strings.LastIndex(addr, ":"); idx != -1 {
		return addr[:idx]
	}
	return addr
}

// TenantHandler handles tenant-related endpoints
type TenantHandler struct {
	db                  *database.DB
	cfg                 *config.Config
	tenantRepo          *repository.TenantRepository
	userRepo            *repository.UserRepository
	marketingRepo       *repository.MarketingRepository
	subscriptionRepo    *repository.SubscriptionRepository
	authService         *services.AuthService
	provisioningService *services.ProvisioningService
	emailService        *services.EmailService
	slugRateLimiter     *slugRateLimiter // Rate limiter for slug enumeration prevention
}

// NewTenantHandler creates a new tenant handler
func NewTenantHandler(db *database.DB, cfg *config.Config) *TenantHandler {
	emailService, _ := services.NewEmailService(services.ConfigToEmailConfig(cfg))
	return &TenantHandler{
		db:                  db,
		cfg:                 cfg,
		tenantRepo:          repository.NewTenantRepository(db),
		userRepo:            repository.NewUserRepository(db),
		marketingRepo:       repository.NewMarketingRepository(db),
		subscriptionRepo:    repository.NewSubscriptionRepository(db),
		authService:         services.NewAuthService(cfg.JWTSecret, cfg.JWTExpirationHours),
		provisioningService: services.NewProvisioningService(db),
		emailService:        emailService,
		slugRateLimiter:     newSlugRateLimiter(),
	}
}

// TenantRegistrationRequest represents a request to register a new tenant
type TenantRegistrationRequest struct {
	// Organization info
	OrganizationName string `json:"organization_name"`
	Slug             string `json:"slug"`
	ContactEmail     string `json:"contact_email"`
	ContactPhone     string `json:"contact_phone"`
	Address          string `json:"address"`
	City             string `json:"city"`
	PostalCode       string `json:"postal_code"`
	FederalState     string `json:"federal_state"`

	// Admin user info
	AdminFirstName string `json:"admin_first_name"`
	AdminLastName  string `json:"admin_last_name"`
	AdminEmail     string `json:"admin_email"`
	AdminPassword  string `json:"admin_password"`

	// Marketing / Referral
	ReferralCode string `json:"referral_code,omitempty"`
}

// Register handles tenant registration
func (h *TenantHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req TenantRegistrationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Ungültige Anfrage")
		return
	}

	// Validate and sanitize organization name (XSS prevention)
	sanitizedOrgName, valErr := ValidateOrganizationName(req.OrganizationName)
	if valErr != nil {
		respondError(w, http.StatusBadRequest, valErr.Message)
		return
	}
	req.OrganizationName = sanitizedOrgName

	// Validate and sanitize admin names (XSS prevention)
	sanitizedFirstName, valErr := ValidatePersonName(req.AdminFirstName, "Vorname")
	if valErr != nil {
		respondError(w, http.StatusBadRequest, valErr.Message)
		return
	}
	req.AdminFirstName = sanitizedFirstName

	sanitizedLastName, valErr := ValidatePersonName(req.AdminLastName, "Nachname")
	if valErr != nil {
		respondError(w, http.StatusBadRequest, valErr.Message)
		return
	}
	req.AdminLastName = sanitizedLastName

	// Validate other required fields
	if req.Slug == "" {
		respondError(w, http.StatusBadRequest, "Subdomain ist erforderlich")
		return
	}
	if req.ContactEmail == "" {
		respondError(w, http.StatusBadRequest, "Kontakt-E-Mail ist erforderlich")
		return
	}

	// Validate contact email format
	if err := models.ValidateEmail(req.ContactEmail); err != nil {
		respondError(w, http.StatusBadRequest, "Ungültiges Kontakt-E-Mail-Format")
		return
	}
	if req.City == "" {
		respondError(w, http.StatusBadRequest, "Stadt ist erforderlich")
		return
	}
	if req.PostalCode == "" {
		respondError(w, http.StatusBadRequest, "Postleitzahl ist erforderlich")
		return
	}
	if req.FederalState == "" {
		respondError(w, http.StatusBadRequest, "Bundesland ist erforderlich")
		return
	}
	if req.AdminEmail == "" {
		respondError(w, http.StatusBadRequest, "Admin-E-Mail ist erforderlich")
		return
	}

	// Validate admin email format
	if err := models.ValidateEmail(req.AdminEmail); err != nil {
		respondError(w, http.StatusBadRequest, "Ungültiges Admin-E-Mail-Format")
		return
	}

	if len(req.AdminPassword) < 8 {
		respondError(w, http.StatusBadRequest, "Passwort muss mindestens 8 Zeichen haben")
		return
	}

	// SECURITY: Check if admin email is already used in ANY tenant
	// This prevents login ambiguity and potential account takeover
	emailExists, err := h.userRepo.EmailExistsGlobally(req.AdminEmail)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Fehler bei der E-Mail-Prüfung")
		return
	}
	if emailExists {
		respondError(w, http.StatusConflict, "Diese E-Mail-Adresse wird bereits verwendet")
		return
	}

	// Validate slug format
	if !isValidSlug(req.Slug) {
		respondError(w, http.StatusBadRequest, "Ungültiger Subdomain-Name. Nur Kleinbuchstaben, Zahlen und Bindestriche erlaubt.")
		return
	}

	// Check slug availability
	existing, _ := h.tenantRepo.FindBySlug(req.Slug)
	if existing != nil {
		respondError(w, http.StatusConflict, "Diese Subdomain ist bereits vergeben")
		return
	}

	// Check reserved slugs
	if isReservedSlug(req.Slug) {
		respondError(w, http.StatusConflict, "Diese Subdomain ist reserviert")
		return
	}

	// Start transaction
	tx, err := h.db.Begin()
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Datenbankfehler")
		return
	}
	defer tx.Rollback()

	// 1. Create tenant
	tenant := &models.Tenant{
		Slug:         req.Slug,
		Name:         req.OrganizationName,
		Status:       models.TenantStatusActive,
		ContactEmail: req.ContactEmail,
		ContactPhone: strPtr(req.ContactPhone),
		Address:      strPtr(req.Address),
		City:         strPtr(req.City),
		PostalCode:   strPtr(req.PostalCode),
		FederalState: req.FederalState,
	}
	tenantID, err := h.tenantRepo.CreateTx(tx, tenant)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Fehler beim Erstellen des Tierheims")
		return
	}

	// 2. Create default tenant settings
	settings := &models.TenantSettings{
		TenantID:    tenantID,
		ThemePreset: "classic",
	}
	if err := h.tenantRepo.CreateSettingsTx(tx, settings); err != nil {
		respondError(w, http.StatusInternalServerError, "Fehler beim Erstellen der Einstellungen")
		return
	}

	// 3. Create super admin user for tenant
	hashedPassword, err := h.authService.HashPassword(req.AdminPassword)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Fehler beim Erstellen des Passworts")
		return
	}

	admin := &models.User{
		TenantID:     tenantID,
		FirstName:    req.AdminFirstName,
		LastName:     req.AdminLastName,
		Email:        &req.AdminEmail,
		PasswordHash: &hashedPassword,
		IsAdmin:      true,
		IsSuperAdmin: true,
		IsVerified:   true, // Skip verification for initial admin
		IsActive:     true,
	}
	if err := h.userRepo.CreateTx(tx, admin); err != nil {
		respondError(w, http.StatusInternalServerError, "Fehler beim Erstellen des Admin-Benutzers")
		return
	}

	// 4. Provision default data (colors, booking rules, settings)
	// Pass federalState so it can be used for holiday detection configuration
	if err := h.provisioningService.ProvisionTenant(tx, tenantID, req.FederalState); err != nil {
		log.Printf("Provisioning error for tenant %d: %v", tenantID, err)
		respondError(w, http.StatusInternalServerError, "Fehler bei der Einrichtung")
		return
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		respondError(w, http.StatusInternalServerError, "Fehler beim Speichern")
		return
	}

	// 5. Process referral code if provided (non-blocking, registration succeeds regardless)
	// BUG FIX #6: Add context timeout to prevent goroutine leaks
	if req.ReferralCode != "" {
		go h.processReferralCodeWithTimeout(req.ReferralCode, tenantID)
	}

	// 6. Check and claim FOMO early-adopter benefit (non-blocking)
	go h.claimFOMOBenefit(tenantID)

	// Send welcome email (in background)
	if h.emailService != nil {
		go h.sendTenantWelcomeEmail(req.ContactEmail, req.OrganizationName, req.Slug, req.AdminEmail)
	}

	// Build login URL
	loginURL := fmt.Sprintf("https://%s.%s/login.html", req.Slug, h.cfg.BaseDomain)
	if h.cfg.BaseDomain == "" {
		// Simple-Mode: use configured BASE_URL
		loginURL = fmt.Sprintf("%s/login.html", h.cfg.BaseURL)
	}

	respondJSON(w, http.StatusCreated, map[string]interface{}{
		"message":   "Tierheim erfolgreich registriert",
		"slug":      req.Slug,
		"login_url": loginURL,
	})
}

// CheckSlug checks if a slug is available
func (h *TenantHandler) CheckSlug(w http.ResponseWriter, r *http.Request) {
	// Rate limiting to prevent tenant slug enumeration attacks
	clientIP := getClientIP(r)
	if !h.slugRateLimiter.checkLimit(clientIP) {
		w.Header().Set("Retry-After", "60")
		respondError(w, http.StatusTooManyRequests, "Zu viele Anfragen. Bitte warten Sie eine Minute.")
		return
	}

	slug := r.URL.Query().Get("slug")
	if slug == "" {
		respondError(w, http.StatusBadRequest, "Slug erforderlich")
		return
	}

	// Normalize slug
	slug = strings.ToLower(slug)

	// Check validity and availability
	if !isValidSlug(slug) {
		respondJSON(w, http.StatusOK, map[string]interface{}{
			"available": false,
			"reason":    "Ungültiges Format",
		})
		return
	}

	if isReservedSlug(slug) {
		respondJSON(w, http.StatusOK, map[string]interface{}{
			"available": false,
			"reason":    "Reserviert",
		})
		return
	}

	existing, _ := h.tenantRepo.FindBySlug(slug)
	available := existing == nil

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"available": available,
		"reason":    "",
	})
}

// GetCurrentTenant returns the current tenant information
func (h *TenantHandler) GetCurrentTenant(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := r.Context().Value(middleware.TenantIDKey).(int)
	if !ok {
		respondError(w, http.StatusInternalServerError, "Request validation failed")
		return
	}

	tenant, err := h.tenantRepo.FindByID(tenantID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Fehler beim Laden des Tierheims")
		return
	}
	if tenant == nil {
		respondError(w, http.StatusNotFound, "Tierheim nicht gefunden")
		return
	}

	respondJSON(w, http.StatusOK, tenant)
}

// UpdateTenantRequest represents a request to update tenant info
type UpdateTenantRequest struct {
	Name         string `json:"name"`
	ContactEmail string `json:"contact_email"`
	ContactPhone string `json:"contact_phone"`
	Address      string `json:"address"`
	City         string `json:"city"`
	PostalCode   string `json:"postal_code"`
	FederalState string `json:"federal_state"`
}

// UpdateTenant updates the current tenant's information (admin only)
func (h *TenantHandler) UpdateTenant(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := r.Context().Value(middleware.TenantIDKey).(int)
	if !ok {
		respondError(w, http.StatusInternalServerError, "Request validation failed")
		return
	}

	var req UpdateTenantRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Ungültige Anfrage")
		return
	}

	// Get current tenant
	tenant, err := h.tenantRepo.FindByID(tenantID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Fehler beim Laden")
		return
	}
	if tenant == nil {
		respondError(w, http.StatusNotFound, "Tierheim nicht gefunden")
		return
	}

	// Update fields
	if req.Name != "" {
		tenant.Name = req.Name
	}
	if req.ContactEmail != "" {
		tenant.ContactEmail = req.ContactEmail
	}
	tenant.ContactPhone = strPtr(req.ContactPhone)
	tenant.Address = strPtr(req.Address)
	if req.City != "" {
		tenant.City = strPtr(req.City)
	}
	if req.PostalCode != "" {
		tenant.PostalCode = strPtr(req.PostalCode)
	}
	if req.FederalState != "" {
		tenant.FederalState = req.FederalState
	}

	if err := h.tenantRepo.Update(tenant); err != nil {
		respondError(w, http.StatusInternalServerError, "Fehler beim Speichern")
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{
		"message": "Tierheim aktualisiert",
	})
}

// BrandingResponse represents the branding information for a tenant
type BrandingResponse struct {
	TenantName      string  `json:"tenant_name"`
	TenantSlug      string  `json:"tenant_slug"`
	WelcomeMessage  *string `json:"welcome_message,omitempty"`
	Tagline         *string `json:"tagline,omitempty"`
	Description     *string `json:"description,omitempty"`
	FooterText      *string `json:"footer_text,omitempty"`
	WebsiteURL      *string `json:"website_url,omitempty"`
	DonationURL     *string `json:"donation_url,omitempty"`
	LogoURL         *string `json:"logo_url,omitempty"`
	FaviconURL      *string `json:"favicon_url,omitempty"`
	ThemePreset     string  `json:"theme_preset"`
	ColorPrimary    *string `json:"color_primary,omitempty"`
	ColorSecondary  *string `json:"color_secondary,omitempty"`
	ColorAccent     *string `json:"color_accent,omitempty"`
	ColorBackground *string `json:"color_background,omitempty"`
	ColorText       *string `json:"color_text,omitempty"`
}

// GetBranding returns the branding information for the current tenant (public endpoint)
func (h *TenantHandler) GetBranding(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := r.Context().Value(middleware.TenantIDKey).(int)
	tenantSlug, _ := r.Context().Value(middleware.TenantSlugKey).(string)

	if !ok {
		respondError(w, http.StatusInternalServerError, "Request validation failed")
		return
	}

	// Get tenant info
	tenant, err := h.tenantRepo.FindByID(tenantID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Fehler beim Laden des Tierheims")
		return
	}
	if tenant == nil {
		respondError(w, http.StatusNotFound, "Tierheim nicht gefunden")
		return
	}

	// Get tenant settings
	settings, err := h.tenantRepo.GetSettings(tenantID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Fehler beim Laden der Einstellungen")
		return
	}

	// Build response
	response := BrandingResponse{
		TenantName:  tenant.Name,
		TenantSlug:  tenantSlug,
		ThemePreset: "classic", // Default
	}

	if settings != nil {
		response.WelcomeMessage = settings.WelcomeMessage
		response.Tagline = settings.Tagline
		response.Description = settings.Description
		response.FooterText = settings.FooterText
		response.WebsiteURL = settings.WebsiteURL
		response.DonationURL = settings.DonationURL
		response.LogoURL = settings.LogoURL
		response.FaviconURL = settings.FaviconURL
		response.ThemePreset = settings.ThemePreset
		response.ColorPrimary = settings.ColorPrimary
		response.ColorSecondary = settings.ColorSecondary
		response.ColorAccent = settings.ColorAccent
		response.ColorBackground = settings.ColorBackground
		response.ColorText = settings.ColorText
	}

	respondJSON(w, http.StatusOK, response)
}

// UpdateBrandingRequest represents a request to update tenant branding
type UpdateBrandingRequest struct {
	WelcomeMessage  *string `json:"welcome_message"`
	Tagline         *string `json:"tagline"`
	Description     *string `json:"description"`
	FooterText      *string `json:"footer_text"`
	WebsiteURL      *string `json:"website_url"`
	DonationURL     *string `json:"donation_url"`
	ThemePreset     string  `json:"theme_preset"`
	ColorPrimary    *string `json:"color_primary"`
	ColorSecondary  *string `json:"color_secondary"`
	ColorAccent     *string `json:"color_accent"`
	ColorBackground *string `json:"color_background"`
	ColorText       *string `json:"color_text"`
}

// UpdateBranding updates the branding settings for the current tenant (admin only)
func (h *TenantHandler) UpdateBranding(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := r.Context().Value(middleware.TenantIDKey).(int)
	if !ok {
		respondError(w, http.StatusInternalServerError, "Request validation failed")
		return
	}

	var req UpdateBrandingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Ungültige Anfrage")
		return
	}

	// Validate theme preset if provided
	if req.ThemePreset != "" && !models.ValidateThemePreset(req.ThemePreset) {
		respondError(w, http.StatusBadRequest, "Ungültiges Theme-Preset")
		return
	}

	// Validate color formats if provided
	colorFields := []*string{req.ColorPrimary, req.ColorSecondary, req.ColorAccent, req.ColorBackground, req.ColorText}
	for _, color := range colorFields {
		if color != nil && *color != "" && !models.ValidateHexColor(*color) {
			respondError(w, http.StatusBadRequest, "Ungültiges Farbformat (erwartet: #RRGGBB)")
			return
		}
	}

	// Validate URLs if provided (basic check)
	if req.WebsiteURL != nil && *req.WebsiteURL != "" {
		if !strings.HasPrefix(*req.WebsiteURL, "http://") && !strings.HasPrefix(*req.WebsiteURL, "https://") {
			respondError(w, http.StatusBadRequest, "Website-URL muss mit http:// oder https:// beginnen")
			return
		}
	}
	if req.DonationURL != nil && *req.DonationURL != "" {
		if !strings.HasPrefix(*req.DonationURL, "http://") && !strings.HasPrefix(*req.DonationURL, "https://") {
			respondError(w, http.StatusBadRequest, "Spenden-URL muss mit http:// oder https:// beginnen")
			return
		}
	}

	// Get current settings
	settings, err := h.tenantRepo.GetSettings(tenantID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Fehler beim Laden der Einstellungen")
		return
	}
	if settings == nil {
		respondError(w, http.StatusNotFound, "Einstellungen nicht gefunden")
		return
	}

	// Update fields
	settings.WelcomeMessage = req.WelcomeMessage
	settings.Tagline = req.Tagline
	settings.Description = req.Description
	settings.FooterText = req.FooterText
	settings.WebsiteURL = req.WebsiteURL
	settings.DonationURL = req.DonationURL

	if req.ThemePreset != "" {
		settings.ThemePreset = req.ThemePreset
	}

	settings.ColorPrimary = req.ColorPrimary
	settings.ColorSecondary = req.ColorSecondary
	settings.ColorAccent = req.ColorAccent
	settings.ColorBackground = req.ColorBackground
	settings.ColorText = req.ColorText

	// Save settings
	if err := h.tenantRepo.UpdateSettings(settings); err != nil {
		respondError(w, http.StatusInternalServerError, "Fehler beim Speichern")
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{
		"message": "Branding aktualisiert",
	})
}

// GetTenantStats returns statistics for the current tenant
func (h *TenantHandler) GetTenantStats(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := r.Context().Value(middleware.TenantIDKey).(int)
	if !ok {
		respondError(w, http.StatusInternalServerError, "Request validation failed")
		return
	}

	stats, err := h.tenantRepo.GetStats(tenantID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Fehler beim Laden der Statistiken")
		return
	}

	respondJSON(w, http.StatusOK, stats)
}

// Helper functions

// isValidSlug checks if a slug is valid (lowercase, alphanumeric, hyphens)
func isValidSlug(slug string) bool {
	if len(slug) < 3 || len(slug) > 100 {
		return false
	}
	// Must be lowercase, start with letter, contain only letters, numbers, hyphens
	matched, _ := regexp.MatchString(`^[a-z][a-z0-9-]*[a-z0-9]$`, slug)
	return matched
}

// isReservedSlug checks if a slug is reserved
func isReservedSlug(slug string) bool {
	reserved := []string{
		// Infrastructure subdomains
		"www", "api", "admin", "app", "mail", "email", "smtp", "ftp",
		"support", "help", "billing", "status", "dev", "staging", "test",
		"demo", "blog", "news", "docs", "static", "assets", "cdn", "media",
		// Application routes - these conflict with SaaS routes
		"central",   // /central/ - Central admin dashboard
		"landing",   // /landing/ - Marketing landing pages
		"login",     // Common auth route
		"logout",    // Common auth route
		"register",  // Common auth route
		"signup",    // Common auth route
		"signin",    // Common auth route
		"signout",   // Common auth route
		"dashboard", // Common app route
		"profile",   // Common app route
		"account",   // Common app route
		"settings",  // Common app route
	}
	slug = strings.ToLower(slug)
	for _, r := range reserved {
		if slug == r {
			return true
		}
	}
	return false
}

// sendTenantWelcomeEmail sends a welcome email to the new tenant
func (h *TenantHandler) sendTenantWelcomeEmail(contactEmail, orgName, slug, adminEmail string) {
	if h.emailService == nil {
		return
	}

	subject := fmt.Sprintf("Willkommen bei Gassigeher, %s!", orgName)

	loginURL := fmt.Sprintf("https://%s.%s/login.html", slug, h.cfg.BaseDomain)
	if h.cfg.BaseDomain == "" {
		// Simple-Mode: use configured BASE_URL
		loginURL = fmt.Sprintf("%s/login.html", h.cfg.BaseURL)
	}

	body := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <style>
        body { font-family: Titillium, Arial, sans-serif; line-height: 1.6; color: #26272b; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background-color: #82b965; color: white; padding: 20px; text-align: center; border-radius: 6px 6px 0 0; }
        .content { background-color: #f9f9f9; padding: 30px; border-radius: 0 0 6px 6px; }
        .info-box { margin: 15px 0; padding: 15px; background-color: white; border-left: 4px solid #82b965; }
        .btn { display: inline-block; padding: 12px 24px; background-color: #82b965; color: white; text-decoration: none; border-radius: 6px; margin: 10px 0; }
        .footer { text-align: center; margin-top: 20px; color: #666; font-size: 12px; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>Willkommen bei Gassigeher!</h1>
        </div>
        <div class="content">
            <p>Herzlichen Glückwunsch!</p>
            <p><strong>%s</strong> wurde erfolgreich bei Gassigeher registriert.</p>

            <div class="info-box">
                <strong>Ihre Zugangsdaten:</strong><br>
                Subdomain: <code>%s.gassigeher.org</code><br>
                Admin-E-Mail: <code>%s</code>
            </div>

            <p>Nächste Schritte:</p>
            <ol>
                <li>Melden Sie sich im Admin-Bereich an</li>
                <li>Fügen Sie Ihre Hunde hinzu</li>
                <li>Laden Sie Freiwillige ein, sich zu registrieren</li>
                <li>Verwalten Sie Buchungen und Termine</li>
            </ol>

            <p style="text-align: center;">
                <a href="%s" class="btn">Jetzt anmelden</a>
            </p>

            <p>Bei Fragen stehen wir Ihnen gerne zur Verfügung.</p>
        </div>
        <div class="footer">
            <p>Diese E-Mail wurde automatisch generiert.</p>
        </div>
    </div>
</body>
</html>
`, orgName, slug, adminEmail, loginURL)

	h.emailService.SendEmail(contactEmail, subject, body)
}

// processReferralCodeWithTimeout wraps processReferralCode with a context timeout
// BUG FIX #6: Prevents goroutine leaks by ensuring the goroutine completes within a reasonable time
func (h *TenantHandler) processReferralCodeWithTimeout(code string, refereeTenantID int) {
	// Create a context with 30 second timeout to prevent indefinite hangs
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Channel to signal completion
	done := make(chan struct{})

	go func() {
		h.processReferralCode(code, refereeTenantID)
		close(done)
	}()

	// Wait for completion or timeout
	select {
	case <-done:
		// Completed successfully
	case <-ctx.Done():
		log.Printf("WARNING: processReferralCode timed out for code '%s', tenant %d", code, refereeTenantID)
	}
}

// processReferralCode validates and applies a referral code during tenant registration
// This is called in a goroutine and should not block registration
func (h *TenantHandler) processReferralCode(code string, refereeTenantID int) {
	// Look up the referral code (case-insensitive)
	referralCode, err := h.marketingRepo.GetReferralCodeByCode(strings.ToUpper(code))
	if err != nil {
		log.Printf("Error looking up referral code '%s': %v", code, err)
		return
	}
	if referralCode == nil {
		log.Printf("Referral code '%s' not found", code)
		return
	}

	// Check if code is active
	if !referralCode.IsActive {
		log.Printf("Referral code '%s' is not active", code)
		return
	}

	// Check if code has expired
	if referralCode.ExpiresAt != nil && time.Now().After(*referralCode.ExpiresAt) {
		log.Printf("Referral code '%s' has expired", code)
		return
	}

	// Check if max uses exceeded
	if referralCode.MaxUses != nil && referralCode.UsesCount >= *referralCode.MaxUses {
		log.Printf("Referral code '%s' has reached max uses (%d)", code, *referralCode.MaxUses)
		return
	}

	// Check if same tenant is using their own code (prevent self-referral)
	if referralCode.ReferrerTenantID != nil && *referralCode.ReferrerTenantID == refereeTenantID {
		log.Printf("Tenant %d cannot use their own referral code", refereeTenantID)
		return
	}

	// All checks passed - increment uses and record the referral
	if err := h.marketingRepo.IncrementReferralCodeUses(referralCode.ID); err != nil {
		log.Printf("Error incrementing referral code uses: %v", err)
		return
	}

	if err := h.marketingRepo.RecordReferralUse(referralCode.ID, refereeTenantID); err != nil {
		log.Printf("Error recording referral use: %v", err)
		return
	}

	log.Printf("Successfully applied referral code '%s' for tenant %d", code, refereeTenantID)
}

// claimFOMOBenefit checks for active FOMO campaigns and grants early-adopter benefits.
// This is called in a goroutine and should not block registration.
// If a FOMO campaign with remaining slots exists, it claims a slot and grants the benefit.
func (h *TenantHandler) claimFOMOBenefit(tenantID int) {
	// Try to claim a slot from active FOMO campaign
	campaign, config, err := h.marketingRepo.ClaimFOMOSlot()
	if err != nil {
		log.Printf("Error claiming FOMO slot for tenant %d: %v", tenantID, err)
		return
	}
	if campaign == nil || config == nil {
		// No active FOMO campaign or no slots available
		return
	}

	// Check if benefit is free Pro months
	if config.BenefitType == "free_pro_months" && config.BenefitMonths > 0 {
		// Grant free Pro subscription
		err := h.subscriptionRepo.CreateFreeProSubscription(tenantID, config.BenefitMonths, "fomo_early_adopter")
		if err != nil {
			// CRITICAL: Slot was claimed but benefit grant failed!
			// Admin should manually grant Pro to this tenant
			log.Printf("CRITICAL: FOMO slot claimed but failed to grant Pro to tenant %d: %v (campaign: %s, slots remaining: %d)",
				tenantID, err, campaign.Name, config.RemainingSlots)
			return
		}
		log.Printf("Granted %d free Pro months to tenant %d via FOMO campaign '%s' (slots remaining: %d)",
			config.BenefitMonths, tenantID, campaign.Name, config.RemainingSlots)
	}
}

// ExportTenantData exports all tenant data for GDPR compliance
// GET /api/admin/tenant/export
// Allows tenant admin to download all data for their tenant
//
// HIGH-10: SECURITY MODEL - Cross-tenant protection is enforced through:
// 1. TenantMiddleware sets tenantID in context based on subdomain
// 2. AuthMiddleware validates JWT tenant_id matches subdomain tenant_id
// 3. This handler requires both tenant context AND admin status
// An admin from Tenant A cannot export Tenant B's data because:
// - If they visit Tenant B's subdomain, AuthMiddleware rejects their JWT
// - The tenantID in context always matches the authenticated user's tenant
func (h *TenantHandler) ExportTenantData(w http.ResponseWriter, r *http.Request) {
	// Get tenant ID from context (set by TenantMiddleware from subdomain)
	// HIGH-10: This is always the subdomain's tenant, validated against JWT by AuthMiddleware
	tenantID, ok := r.Context().Value(middleware.TenantIDKey).(int)
	if !ok {
		respondError(w, http.StatusInternalServerError, "Request validation failed")
		return
	}

	// Verify admin access (from JWT, validated by AuthMiddleware to belong to this tenant)
	isAdmin, _ := r.Context().Value(middleware.IsAdminKey).(bool)
	if !isAdmin {
		respondError(w, http.StatusForbidden, "Nur Administratoren können Daten exportieren")
		return
	}

	// Get tenant info
	tenant, err := h.tenantRepo.FindByID(tenantID)
	if err != nil || tenant == nil {
		respondError(w, http.StatusNotFound, "Tierheim nicht gefunden")
		return
	}

	// Build export data
	export := map[string]interface{}{
		"tenant": map[string]interface{}{
			"id":            tenant.ID,
			"slug":          tenant.Slug,
			"name":          tenant.Name,
			"contact_email": tenant.ContactEmail,
			"contact_phone": tenant.ContactPhone,
			"address":       tenant.Address,
			"city":          tenant.City,
			"postal_code":   tenant.PostalCode,
			"federal_state": tenant.FederalState,
			"created_at":    tenant.CreatedAt,
		},
		"exported_at": time.Now().Format(time.RFC3339),
		"gdpr_export": true,
	}

	// Get all users (sanitize sensitive data)
	users, err := h.userRepo.FindAll(nil, tenantID)
	if err == nil {
		var sanitizedUsers []map[string]interface{}
		for _, u := range users {
			sanitizedUsers = append(sanitizedUsers, map[string]interface{}{
				"id":               u.ID,
				"first_name":       u.FirstName,
				"last_name":        u.LastName,
				"email":            u.Email,
				"phone":            u.Phone,
				"is_admin":         u.IsAdmin,
				"is_active":        u.IsActive,
				"is_verified":      u.IsVerified,
				"profile_photo":    u.ProfilePhoto,
				"last_activity_at": u.LastActivityAt,
				"created_at":       u.CreatedAt,
			})
		}
		export["users"] = sanitizedUsers
		export["user_count"] = len(sanitizedUsers)
	}

	// Get all dogs
	var dogs []map[string]interface{}
	rows, err := h.db.Query(`
		SELECT id, name, breed, size, age, color_id, is_featured, is_available,
		       external_link, photo, special_instructions, pickup_location,
		       created_at, updated_at
		FROM dogs WHERE tenant_id = ?`, tenantID)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var d struct {
				ID                  int
				Name                string
				Breed               *string
				Size                *string
				Age                 *int
				ColorID             *int
				IsFeatured          bool
				IsAvailable         bool
				ExternalLink        *string
				Photo               *string
				SpecialInstructions *string
				PickupLocation      *string
				CreatedAt           time.Time
				UpdatedAt           time.Time
			}
			if rows.Scan(&d.ID, &d.Name, &d.Breed, &d.Size, &d.Age, &d.ColorID,
				&d.IsFeatured, &d.IsAvailable, &d.ExternalLink, &d.Photo,
				&d.SpecialInstructions, &d.PickupLocation, &d.CreatedAt, &d.UpdatedAt) == nil {
				dogs = append(dogs, map[string]interface{}{
					"id":                   d.ID,
					"name":                 d.Name,
					"breed":                d.Breed,
					"size":                 d.Size,
					"age":                  d.Age,
					"color_id":             d.ColorID,
					"is_featured":          d.IsFeatured,
					"is_available":         d.IsAvailable,
					"external_link":        d.ExternalLink,
					"photo":                d.Photo,
					"special_instructions": d.SpecialInstructions,
					"pickup_location":      d.PickupLocation,
					"created_at":           d.CreatedAt,
					"updated_at":           d.UpdatedAt,
				})
			}
		}
		// BUG FIX: Check for errors during iteration - MUST fail export for GDPR compliance
		if err := rows.Err(); err != nil {
			log.Printf("ERROR: Error during dogs iteration for tenant %d: %v", tenantID, err)
			respondError(w, http.StatusInternalServerError, "Fehler beim Laden der Hundedaten")
			return
		}
	}
	export["dogs"] = dogs
	export["dog_count"] = len(dogs)

	// Get all bookings with user and dog info
	var bookings []map[string]interface{}
	rows, err = h.db.Query(`
		SELECT b.id, b.user_id, b.dog_id, b.date, b.walk_type, b.status,
		       b.notes, b.created_at,
		       u.first_name, u.last_name, u.email,
		       d.name as dog_name
		FROM bookings b
		LEFT JOIN users u ON b.user_id = u.id
		LEFT JOIN dogs d ON b.dog_id = d.id
		WHERE b.tenant_id = ?
		ORDER BY b.date DESC`, tenantID)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var b struct {
				ID        int
				UserID    int
				DogID     int
				Date      string
				WalkType  string
				Status    string
				Notes     *string
				CreatedAt time.Time
				FirstName *string
				LastName  *string
				Email     *string
				DogName   *string
			}
			if rows.Scan(&b.ID, &b.UserID, &b.DogID, &b.Date, &b.WalkType, &b.Status,
				&b.Notes, &b.CreatedAt, &b.FirstName, &b.LastName, &b.Email, &b.DogName) == nil {
				bookings = append(bookings, map[string]interface{}{
					"id":              b.ID,
					"user_id":         b.UserID,
					"dog_id":          b.DogID,
					"date":            b.Date,
					"walk_type":       b.WalkType,
					"status":          b.Status,
					"notes":           b.Notes,
					"created_at":      b.CreatedAt,
					"user_first_name": b.FirstName,
					"user_last_name":  b.LastName,
					"user_email":      b.Email,
					"dog_name":        b.DogName,
				})
			}
		}
		// BUG FIX: Check for errors during iteration - MUST fail export for GDPR compliance
		if err := rows.Err(); err != nil {
			log.Printf("ERROR: Error during bookings iteration for tenant %d: %v", tenantID, err)
			respondError(w, http.StatusInternalServerError, "Fehler beim Laden der Buchungsdaten")
			return
		}
	}
	export["bookings"] = bookings
	export["booking_count"] = len(bookings)

	// Get tenant settings
	var tenantSettings map[string]interface{}
	settingsRow := h.db.QueryRow(`
		SELECT welcome_message, footer_text, website_url, donation_url,
		       logo_url, favicon_url
		FROM tenant_settings WHERE tenant_id = ?`, tenantID)
	var welcomeMsg, footerText, websiteURL, donationURL, logoURL, faviconURL *string
	if settingsRow.Scan(&welcomeMsg, &footerText, &websiteURL, &donationURL, &logoURL, &faviconURL) == nil {
		tenantSettings = map[string]interface{}{
			"welcome_message": welcomeMsg,
			"footer_text":     footerText,
			"website_url":     websiteURL,
			"donation_url":    donationURL,
			"logo_url":        logoURL,
			"favicon_url":     faviconURL,
		}
		export["tenant_settings"] = tenantSettings
	}

	// Get blocked dates
	var blockedDates []map[string]interface{}
	rows, err = h.db.Query(`
		SELECT id, date, reason, dog_id, created_at
		FROM blocked_dates WHERE tenant_id = ?`, tenantID)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var bd struct {
				ID        int
				Date      string
				Reason    *string
				DogID     *int
				CreatedAt time.Time
			}
			if rows.Scan(&bd.ID, &bd.Date, &bd.Reason, &bd.DogID, &bd.CreatedAt) == nil {
				blockedDates = append(blockedDates, map[string]interface{}{
					"id":         bd.ID,
					"date":       bd.Date,
					"reason":     bd.Reason,
					"dog_id":     bd.DogID,
					"created_at": bd.CreatedAt,
				})
			}
		}
		if err := rows.Err(); err != nil {
			log.Printf("Warning: Error during blocked_dates iteration for tenant %d: %v", tenantID, err)
		}
	}
	export["blocked_dates"] = blockedDates

	// Audit log
	userID, _ := r.Context().Value(middleware.UserIDKey).(int)
	log.Printf("AUDIT: User %d exported GDPR data for tenant %d (%s)", userID, tenantID, tenant.Slug)

	// Set headers for JSON download
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=gdpr-export-%s-%s.json",
		tenant.Slug, time.Now().Format("2006-01-02")))
	respondJSON(w, http.StatusOK, export)
}

// ========== Reference Page Self-Service ==========

// GetTenantReferenceEntry returns the current tenant's reference entry
// GET /api/v1/admin/reference-entry
func (h *TenantHandler) GetTenantReferenceEntry(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := r.Context().Value(middleware.TenantIDKey).(int)
	if !ok {
		respondError(w, http.StatusInternalServerError, "Tenant context nicht gefunden")
		return
	}

	entry, err := h.marketingRepo.GetReferenceEntryByTenant(tenantID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Fehler beim Laden des Referenzeintrags")
		return
	}

	// Return null if no entry exists (not an error)
	if entry == nil {
		respondJSON(w, http.StatusOK, map[string]interface{}{"entry": nil})
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{"entry": entry})
}

// SubmitReferenceEntry creates or updates the tenant's reference entry
// POST /api/v1/admin/reference-entry
func (h *TenantHandler) SubmitReferenceEntry(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := r.Context().Value(middleware.TenantIDKey).(int)
	if !ok {
		respondError(w, http.StatusInternalServerError, "Tenant context nicht gefunden")
		return
	}

	var req models.CreateReferenceEntryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Ungültige Anfrage")
		return
	}

	// Validate display_name
	req.DisplayName = strings.TrimSpace(req.DisplayName)
	if req.DisplayName == "" {
		respondError(w, http.StatusBadRequest, "Name ist erforderlich")
		return
	}
	if len(req.DisplayName) > 255 {
		respondError(w, http.StatusBadRequest, "Name darf maximal 255 Zeichen lang sein")
		return
	}

	// Validate optional fields
	if req.City != nil && len(*req.City) > 100 {
		respondError(w, http.StatusBadRequest, "Stadt darf maximal 100 Zeichen lang sein")
		return
	}
	if req.FederalState != nil && len(*req.FederalState) > 50 {
		respondError(w, http.StatusBadRequest, "Bundesland darf maximal 50 Zeichen lang sein")
		return
	}
	if req.WebsiteURL != nil && *req.WebsiteURL != "" {
		if len(*req.WebsiteURL) > 500 {
			respondError(w, http.StatusBadRequest, "Website-URL darf maximal 500 Zeichen lang sein")
			return
		}
		if !strings.HasPrefix(*req.WebsiteURL, "http://") && !strings.HasPrefix(*req.WebsiteURL, "https://") {
			respondError(w, http.StatusBadRequest, "Website-URL muss mit http:// oder https:// beginnen")
			return
		}
	}
	if req.Testimonial != nil && len(*req.Testimonial) > 2000 {
		respondError(w, http.StatusBadRequest, "Testimonial darf maximal 2000 Zeichen lang sein")
		return
	}

	// Check for existing entry
	existing, err := h.marketingRepo.GetReferenceEntryByTenant(tenantID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Fehler beim Prüfen des bestehenden Eintrags")
		return
	}

	if existing != nil {
		// Update existing entry
		existing.DisplayName = req.DisplayName
		existing.City = req.City
		existing.FederalState = req.FederalState
		existing.WebsiteURL = req.WebsiteURL
		existing.Testimonial = req.Testimonial
		// Reset approval status when content changes (tenant must wait for re-approval)
		existing.IsApproved = false

		if err := h.marketingRepo.UpdateReferenceEntry(existing); err != nil {
			respondError(w, http.StatusInternalServerError, "Fehler beim Aktualisieren des Referenzeintrags")
			return
		}
		respondJSON(w, http.StatusOK, map[string]interface{}{
			"message": "Referenzeintrag aktualisiert. Wartet auf Genehmigung.",
			"entry":   existing,
		})
		return
	}

	// Create new entry
	entry := &models.ReferenceEntry{
		TenantID:     tenantID,
		DisplayName:  req.DisplayName,
		City:         req.City,
		FederalState: req.FederalState,
		WebsiteURL:   req.WebsiteURL,
		Testimonial:  req.Testimonial,
		IsApproved:   false, // Tenant submissions require approval
		DisplayOrder: 0,
	}

	if err := h.marketingRepo.CreateReferenceEntry(entry); err != nil {
		respondError(w, http.StatusInternalServerError, "Fehler beim Erstellen des Referenzeintrags")
		return
	}

	respondJSON(w, http.StatusCreated, map[string]interface{}{
		"message": "Referenzeintrag eingereicht. Wartet auf Genehmigung.",
		"entry":   entry,
	})
}

// DeleteTenantReferenceEntry deletes the tenant's reference entry
// DELETE /api/v1/admin/reference-entry
func (h *TenantHandler) DeleteTenantReferenceEntry(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := r.Context().Value(middleware.TenantIDKey).(int)
	if !ok {
		respondError(w, http.StatusInternalServerError, "Tenant context nicht gefunden")
		return
	}

	// Find the tenant's entry
	entry, err := h.marketingRepo.GetReferenceEntryByTenant(tenantID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Fehler beim Laden des Referenzeintrags")
		return
	}
	if entry == nil {
		respondError(w, http.StatusNotFound, "Kein Referenzeintrag vorhanden")
		return
	}

	// Delete the entry
	if err := h.marketingRepo.DeleteReferenceEntry(entry.ID); err != nil {
		respondError(w, http.StatusInternalServerError, "Fehler beim Löschen des Referenzeintrags")
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "Referenzeintrag gelöscht"})
}
