package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/tranmh/gassigeher/internal/config"
	"github.com/tranmh/gassigeher/internal/middleware"
	"github.com/tranmh/gassigeher/internal/models"
	"github.com/tranmh/gassigeher/internal/repository"
	"github.com/tranmh/gassigeher/internal/services"
)

// TenantHandler handles tenant-related endpoints
type TenantHandler struct {
	db                  *sql.DB
	cfg                 *config.Config
	tenantRepo          *repository.TenantRepository
	userRepo            *repository.UserRepository
	authService         *services.AuthService
	provisioningService *services.ProvisioningService
	emailService        *services.EmailService
}

// NewTenantHandler creates a new tenant handler
func NewTenantHandler(db *sql.DB, cfg *config.Config) *TenantHandler {
	emailService, _ := services.NewEmailService(services.ConfigToEmailConfig(cfg))
	return &TenantHandler{
		db:                  db,
		cfg:                 cfg,
		tenantRepo:          repository.NewTenantRepository(db),
		userRepo:            repository.NewUserRepository(db),
		authService:         services.NewAuthService(cfg.JWTSecret, cfg.JWTExpirationHours),
		provisioningService: services.NewProvisioningService(db),
		emailService:        emailService,
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
}

// Register handles tenant registration
func (h *TenantHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req TenantRegistrationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Ungültige Anfrage")
		return
	}

	// Validate required fields
	if req.OrganizationName == "" {
		respondError(w, http.StatusBadRequest, "Organisationsname ist erforderlich")
		return
	}
	if req.Slug == "" {
		respondError(w, http.StatusBadRequest, "Subdomain ist erforderlich")
		return
	}
	if req.ContactEmail == "" {
		respondError(w, http.StatusBadRequest, "Kontakt-E-Mail ist erforderlich")
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
	if req.AdminFirstName == "" || req.AdminLastName == "" {
		respondError(w, http.StatusBadRequest, "Admin-Name ist erforderlich")
		return
	}
	if req.AdminEmail == "" {
		respondError(w, http.StatusBadRequest, "Admin-E-Mail ist erforderlich")
		return
	}
	if len(req.AdminPassword) < 8 {
		respondError(w, http.StatusBadRequest, "Passwort muss mindestens 8 Zeichen haben")
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
	if err := h.provisioningService.ProvisionTenant(tx, tenantID); err != nil {
		fmt.Printf("Provisioning error for tenant %d: %v\n", tenantID, err)
		respondError(w, http.StatusInternalServerError, "Fehler bei der Einrichtung")
		return
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		respondError(w, http.StatusInternalServerError, "Fehler beim Speichern")
		return
	}

	// Send welcome email (in background)
	if h.emailService != nil {
		go h.sendTenantWelcomeEmail(req.ContactEmail, req.OrganizationName, req.Slug, req.AdminEmail)
	}

	// Build login URL
	loginURL := fmt.Sprintf("https://%s.%s/login", req.Slug, h.cfg.BaseDomain)
	if h.cfg.BaseDomain == "" {
		loginURL = fmt.Sprintf("http://localhost:%s/login", h.cfg.Port)
	}

	respondJSON(w, http.StatusCreated, map[string]interface{}{
		"message":   "Tierheim erfolgreich registriert",
		"slug":      req.Slug,
		"login_url": loginURL,
	})
}

// CheckSlug checks if a slug is available
func (h *TenantHandler) CheckSlug(w http.ResponseWriter, r *http.Request) {
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
	tenantID, _ := r.Context().Value(middleware.TenantIDKey).(int)
	if tenantID == 0 {
		respondError(w, http.StatusBadRequest, "Kein Tenant gefunden")
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
	tenantID, _ := r.Context().Value(middleware.TenantIDKey).(int)
	if tenantID == 0 {
		respondError(w, http.StatusBadRequest, "Kein Tenant gefunden")
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

// GetTenantStats returns statistics for the current tenant
func (h *TenantHandler) GetTenantStats(w http.ResponseWriter, r *http.Request) {
	tenantID, _ := r.Context().Value(middleware.TenantIDKey).(int)
	if tenantID == 0 {
		respondError(w, http.StatusBadRequest, "Kein Tenant gefunden")
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
		"www", "api", "admin", "app", "mail", "email", "smtp", "ftp",
		"support", "help", "billing", "status", "dev", "staging", "test",
		"demo", "blog", "news", "docs", "static", "assets", "cdn", "media",
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

	loginURL := fmt.Sprintf("https://%s.%s/login", slug, h.cfg.BaseDomain)
	if h.cfg.BaseDomain == "" {
		loginURL = fmt.Sprintf("http://localhost:%s/login", h.cfg.Port)
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
