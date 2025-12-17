package handlers

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/tranmh/gassigeher/internal/config"
	"github.com/tranmh/gassigeher/internal/middleware"
	"github.com/tranmh/gassigeher/internal/models"
	"github.com/tranmh/gassigeher/internal/repository"
)

// CentralAdminHandler handles platform-wide administration
type CentralAdminHandler struct {
	db         *sql.DB
	cfg        *config.Config
	tenantRepo *repository.TenantRepository
	userRepo   *repository.UserRepository
}

// NewCentralAdminHandler creates a new central admin handler
func NewCentralAdminHandler(db *sql.DB, cfg *config.Config) *CentralAdminHandler {
	return &CentralAdminHandler{
		db:         db,
		cfg:        cfg,
		tenantRepo: repository.NewTenantRepository(db),
		userRepo:   repository.NewUserRepository(db),
	}
}

// PlatformStats represents platform-wide statistics
type PlatformStats struct {
	TotalTenants       int       `json:"total_tenants"`
	ActiveTenants      int       `json:"active_tenants"`
	TotalUsers         int       `json:"total_users"`
	TotalDogs          int       `json:"total_dogs"`
	TotalBookings      int       `json:"total_bookings"`
	BookingsThisMonth  int       `json:"bookings_this_month"`
	NewTenantsThisWeek int       `json:"new_tenants_this_week"`
	GeneratedAt        time.Time `json:"generated_at"`
}

// TenantListItem represents a tenant in the admin list
type TenantListItem struct {
	ID           int        `json:"id"`
	Slug         string     `json:"slug"`
	Name         string     `json:"name"`
	ContactEmail string     `json:"contact_email"`
	Status       string     `json:"status"`
	UserCount    int        `json:"user_count"`
	DogCount     int        `json:"dog_count"`
	CreatedAt    time.Time  `json:"created_at"`
	LastLoginAt  *time.Time `json:"last_login_at,omitempty"`
}

// GetPlatformStats returns platform-wide statistics
// GET /api/central-admin/stats
func (h *CentralAdminHandler) GetPlatformStats(w http.ResponseWriter, r *http.Request) {
	stats := PlatformStats{
		GeneratedAt: time.Now(),
	}

	// Total tenants
	err := h.db.QueryRow(`SELECT COUNT(*) FROM tenants`).Scan(&stats.TotalTenants)
	if err != nil {
		log.Printf("Error getting total tenants: %v", err)
	}

	// Active tenants
	err = h.db.QueryRow(`SELECT COUNT(*) FROM tenants WHERE status = 'active'`).Scan(&stats.ActiveTenants)
	if err != nil {
		log.Printf("Error getting active tenants: %v", err)
	}

	// Total users
	err = h.db.QueryRow(`SELECT COUNT(*) FROM users WHERE is_deleted = 0`).Scan(&stats.TotalUsers)
	if err != nil {
		log.Printf("Error getting total users: %v", err)
	}

	// Total dogs
	err = h.db.QueryRow(`SELECT COUNT(*) FROM dogs`).Scan(&stats.TotalDogs)
	if err != nil {
		log.Printf("Error getting total dogs: %v", err)
	}

	// Total bookings
	err = h.db.QueryRow(`SELECT COUNT(*) FROM bookings`).Scan(&stats.TotalBookings)
	if err != nil {
		log.Printf("Error getting total bookings: %v", err)
	}

	// Bookings this month
	startOfMonth := time.Now().Format("2006-01") + "-01"
	err = h.db.QueryRow(`SELECT COUNT(*) FROM bookings WHERE date >= ?`, startOfMonth).Scan(&stats.BookingsThisMonth)
	if err != nil {
		log.Printf("Error getting bookings this month: %v", err)
	}

	// New tenants this week
	oneWeekAgo := time.Now().AddDate(0, 0, -7).Format("2006-01-02")
	err = h.db.QueryRow(`SELECT COUNT(*) FROM tenants WHERE created_at >= ?`, oneWeekAgo).Scan(&stats.NewTenantsThisWeek)
	if err != nil {
		log.Printf("Error getting new tenants: %v", err)
	}

	respondJSON(w, http.StatusOK, stats)
}

// ListTenants returns all tenants with statistics
// GET /api/central-admin/tenants
func (h *CentralAdminHandler) ListTenants(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	activeOnly := r.URL.Query().Get("active_only") == "true"
	searchTerm := r.URL.Query().Get("search")

	query := `
		SELECT
			t.id, t.slug, t.name, t.contact_email, t.status, t.created_at,
			(SELECT COUNT(*) FROM users u WHERE u.tenant_id = t.id AND u.is_deleted = 0) as user_count,
			(SELECT COUNT(*) FROM dogs d WHERE d.tenant_id = t.id) as dog_count,
			(SELECT MAX(u.last_activity_at) FROM users u WHERE u.tenant_id = t.id) as last_login_at
		FROM tenants t
		WHERE 1=1
	`
	args := []interface{}{}

	if activeOnly {
		query += ` AND t.status = 'active'`
	}

	if searchTerm != "" {
		query += ` AND (t.name LIKE ? OR t.slug LIKE ? OR t.contact_email LIKE ?)`
		searchPattern := "%" + searchTerm + "%"
		args = append(args, searchPattern, searchPattern, searchPattern)
	}

	query += ` ORDER BY t.created_at DESC`

	rows, err := h.db.Query(query, args...)
	if err != nil {
		log.Printf("Error listing tenants: %v", err)
		respondError(w, http.StatusInternalServerError, "Fehler beim Laden der Tierheime")
		return
	}
	defer rows.Close()

	tenants := []TenantListItem{}
	for rows.Next() {
		var t TenantListItem
		err := rows.Scan(
			&t.ID, &t.Slug, &t.Name, &t.ContactEmail, &t.Status, &t.CreatedAt,
			&t.UserCount, &t.DogCount, &t.LastLoginAt,
		)
		if err != nil {
			log.Printf("Error scanning tenant: %v", err)
			continue
		}
		tenants = append(tenants, t)
	}

	respondJSON(w, http.StatusOK, tenants)
}

// GetTenant returns detailed tenant information
// GET /api/central-admin/tenants/{id}
func (h *CentralAdminHandler) GetTenant(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	tenantID, err := strconv.Atoi(vars["id"])
	if err != nil {
		respondError(w, http.StatusBadRequest, "Ungültige Tierheim-ID")
		return
	}

	tenant, err := h.tenantRepo.FindByID(tenantID)
	if err != nil {
		respondError(w, http.StatusNotFound, "Tierheim nicht gefunden")
		return
	}

	// Get additional stats
	var userCount, dogCount, bookingCount int
	h.db.QueryRow(`SELECT COUNT(*) FROM users WHERE tenant_id = ? AND is_deleted = 0`, tenantID).Scan(&userCount)
	h.db.QueryRow(`SELECT COUNT(*) FROM dogs WHERE tenant_id = ?`, tenantID).Scan(&dogCount)
	h.db.QueryRow(`SELECT COUNT(*) FROM bookings WHERE tenant_id = ?`, tenantID).Scan(&bookingCount)

	response := map[string]interface{}{
		"tenant":        tenant,
		"user_count":    userCount,
		"dog_count":     dogCount,
		"booking_count": bookingCount,
	}

	respondJSON(w, http.StatusOK, response)
}

// CentralAdminUpdateTenantRequest represents tenant update payload from central admin
type CentralAdminUpdateTenantRequest struct {
	Name         *string `json:"name,omitempty"`
	ContactEmail *string `json:"contact_email,omitempty"`
	ContactPhone *string `json:"contact_phone,omitempty"`
	Status       *string `json:"status,omitempty"` // active, suspended, deleted
}

// UpdateTenant updates tenant information
// PUT /api/central-admin/tenants/{id}
func (h *CentralAdminHandler) UpdateTenant(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	tenantID, err := strconv.Atoi(vars["id"])
	if err != nil {
		respondError(w, http.StatusBadRequest, "Ungültige Tierheim-ID")
		return
	}

	tenant, err := h.tenantRepo.FindByID(tenantID)
	if err != nil {
		respondError(w, http.StatusNotFound, "Tierheim nicht gefunden")
		return
	}

	var req CentralAdminUpdateTenantRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Ungültige Anfrage")
		return
	}

	// Apply updates
	if req.Name != nil {
		tenant.Name = *req.Name
	}
	if req.ContactEmail != nil {
		tenant.ContactEmail = *req.ContactEmail
	}
	if req.ContactPhone != nil {
		tenant.ContactPhone = req.ContactPhone
	}
	if req.Status != nil {
		tenant.Status = *req.Status
	}

	if err := h.tenantRepo.Update(tenant); err != nil {
		log.Printf("Error updating tenant: %v", err)
		respondError(w, http.StatusInternalServerError, "Fehler beim Aktualisieren")
		return
	}

	// Audit log
	adminID, _ := r.Context().Value(middleware.UserIDKey).(int)
	log.Printf("AUDIT: Central admin %d updated tenant %d", adminID, tenantID)

	respondJSON(w, http.StatusOK, tenant)
}

// ActivateTenant activates a tenant
// POST /api/central-admin/tenants/{id}/activate
func (h *CentralAdminHandler) ActivateTenant(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	tenantID, err := strconv.Atoi(vars["id"])
	if err != nil {
		respondError(w, http.StatusBadRequest, "Ungültige Tierheim-ID")
		return
	}

	tenant, err := h.tenantRepo.FindByID(tenantID)
	if err != nil {
		respondError(w, http.StatusNotFound, "Tierheim nicht gefunden")
		return
	}

	if err := h.tenantRepo.Activate(tenantID); err != nil {
		log.Printf("Error activating tenant: %v", err)
		respondError(w, http.StatusInternalServerError, "Fehler beim Aktivieren")
		return
	}

	// Audit log
	adminID, _ := r.Context().Value(middleware.UserIDKey).(int)
	log.Printf("AUDIT: Central admin %d activated tenant %d (%s)", adminID, tenantID, tenant.Slug)

	respondJSON(w, http.StatusOK, map[string]string{"message": "Tierheim aktiviert"})
}

// DeactivateTenant deactivates (suspends) a tenant
// POST /api/central-admin/tenants/{id}/deactivate
func (h *CentralAdminHandler) DeactivateTenant(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	tenantID, err := strconv.Atoi(vars["id"])
	if err != nil {
		respondError(w, http.StatusBadRequest, "Ungültige Tierheim-ID")
		return
	}

	tenant, err := h.tenantRepo.FindByID(tenantID)
	if err != nil {
		respondError(w, http.StatusNotFound, "Tierheim nicht gefunden")
		return
	}

	if err := h.tenantRepo.Suspend(tenantID, "Deaktiviert durch Central Admin"); err != nil {
		log.Printf("Error deactivating tenant: %v", err)
		respondError(w, http.StatusInternalServerError, "Fehler beim Deaktivieren")
		return
	}

	// Audit log
	adminID, _ := r.Context().Value(middleware.UserIDKey).(int)
	log.Printf("AUDIT: Central admin %d deactivated tenant %d (%s)", adminID, tenantID, tenant.Slug)

	respondJSON(w, http.StatusOK, map[string]string{"message": "Tierheim deaktiviert"})
}

// ListCentralAdmins returns all central admins
// GET /api/central-admin/admins
func (h *CentralAdminHandler) ListCentralAdmins(w http.ResponseWriter, r *http.Request) {
	rows, err := h.db.Query(`
		SELECT id, first_name, last_name, email, is_active, created_at, last_activity_at
		FROM users
		WHERE is_central_admin = 1 AND is_deleted = 0
		ORDER BY created_at ASC
	`)
	if err != nil {
		log.Printf("Error listing central admins: %v", err)
		respondError(w, http.StatusInternalServerError, "Fehler beim Laden der Administratoren")
		return
	}
	defer rows.Close()

	type CentralAdmin struct {
		ID             int       `json:"id"`
		FirstName      string    `json:"first_name"`
		LastName       string    `json:"last_name"`
		Email          *string   `json:"email"`
		IsActive       bool      `json:"is_active"`
		CreatedAt      time.Time `json:"created_at"`
		LastActivityAt time.Time `json:"last_activity_at"`
	}

	admins := []CentralAdmin{}
	for rows.Next() {
		var a CentralAdmin
		if err := rows.Scan(&a.ID, &a.FirstName, &a.LastName, &a.Email, &a.IsActive, &a.CreatedAt, &a.LastActivityAt); err != nil {
			log.Printf("Error scanning central admin: %v", err)
			continue
		}
		admins = append(admins, a)
	}

	respondJSON(w, http.StatusOK, admins)
}

// PromoteToCentralAdmin promotes a user to central admin
// POST /api/central-admin/admins/{id}/promote
func (h *CentralAdminHandler) PromoteToCentralAdmin(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID, err := strconv.Atoi(vars["id"])
	if err != nil {
		respondError(w, http.StatusBadRequest, "Ungültige Benutzer-ID")
		return
	}

	user, err := h.userRepo.FindByID(userID)
	if err != nil {
		respondError(w, http.StatusNotFound, "Benutzer nicht gefunden")
		return
	}

	if user.IsCentralAdmin {
		respondError(w, http.StatusBadRequest, "Benutzer ist bereits Central Admin")
		return
	}

	// Update user
	_, err = h.db.Exec(`UPDATE users SET is_central_admin = 1 WHERE id = ?`, userID)
	if err != nil {
		log.Printf("Error promoting to central admin: %v", err)
		respondError(w, http.StatusInternalServerError, "Fehler beim Befördern")
		return
	}

	// Audit log
	adminID, _ := r.Context().Value(middleware.UserIDKey).(int)
	log.Printf("AUDIT: Central admin %d promoted user %d to central admin", adminID, userID)

	respondJSON(w, http.StatusOK, map[string]string{"message": "Benutzer zu Central Admin befördert"})
}

// DemoteFromCentralAdmin removes central admin privileges
// POST /api/central-admin/admins/{id}/demote
func (h *CentralAdminHandler) DemoteFromCentralAdmin(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID, err := strconv.Atoi(vars["id"])
	if err != nil {
		respondError(w, http.StatusBadRequest, "Ungültige Benutzer-ID")
		return
	}

	// Prevent self-demotion
	currentUserID, _ := r.Context().Value(middleware.UserIDKey).(int)
	if currentUserID == userID {
		respondError(w, http.StatusBadRequest, "Sie können sich nicht selbst degradieren")
		return
	}

	user, err := h.userRepo.FindByID(userID)
	if err != nil {
		respondError(w, http.StatusNotFound, "Benutzer nicht gefunden")
		return
	}

	if !user.IsCentralAdmin {
		respondError(w, http.StatusBadRequest, "Benutzer ist kein Central Admin")
		return
	}

	// Update user
	_, err = h.db.Exec(`UPDATE users SET is_central_admin = 0 WHERE id = ?`, userID)
	if err != nil {
		log.Printf("Error demoting from central admin: %v", err)
		respondError(w, http.StatusInternalServerError, "Fehler beim Degradieren")
		return
	}

	// Audit log
	log.Printf("AUDIT: Central admin %d demoted user %d from central admin", currentUserID, userID)

	respondJSON(w, http.StatusOK, map[string]string{"message": "Central Admin Rechte entfernt"})
}

// GetTenantUsers returns users for a specific tenant
// GET /api/central-admin/tenants/{id}/users
func (h *CentralAdminHandler) GetTenantUsers(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	tenantID, err := strconv.Atoi(vars["id"])
	if err != nil {
		respondError(w, http.StatusBadRequest, "Ungültige Tierheim-ID")
		return
	}

	// Verify tenant exists
	_, err = h.tenantRepo.FindByID(tenantID)
	if err != nil {
		respondError(w, http.StatusNotFound, "Tierheim nicht gefunden")
		return
	}

	users, err := h.userRepo.FindAll(nil, tenantID)
	if err != nil {
		log.Printf("Error getting tenant users: %v", err)
		respondError(w, http.StatusInternalServerError, "Fehler beim Laden der Benutzer")
		return
	}

	// Remove sensitive data
	for i := range users {
		users[i].PasswordHash = nil
		users[i].VerificationToken = nil
		users[i].PasswordResetToken = nil
	}

	respondJSON(w, http.StatusOK, users)
}

// SearchUsers searches for users across all tenants
// GET /api/central-admin/users/search?q=searchterm
func (h *CentralAdminHandler) SearchUsers(w http.ResponseWriter, r *http.Request) {
	searchTerm := r.URL.Query().Get("q")
	if searchTerm == "" {
		respondError(w, http.StatusBadRequest, "Suchbegriff erforderlich")
		return
	}

	searchPattern := "%" + searchTerm + "%"
	rows, err := h.db.Query(`
		SELECT u.id, u.tenant_id, u.first_name, u.last_name, u.email, u.is_admin, u.is_super_admin,
		       u.is_active, u.created_at, t.name as tenant_name
		FROM users u
		LEFT JOIN tenants t ON u.tenant_id = t.id
		WHERE u.is_deleted = 0
		  AND (u.first_name LIKE ? OR u.last_name LIKE ? OR u.email LIKE ?)
		ORDER BY u.last_name, u.first_name
		LIMIT 100
	`, searchPattern, searchPattern, searchPattern)
	if err != nil {
		log.Printf("Error searching users: %v", err)
		respondError(w, http.StatusInternalServerError, "Fehler bei der Suche")
		return
	}
	defer rows.Close()

	type UserSearchResult struct {
		ID           int       `json:"id"`
		TenantID     int       `json:"tenant_id"`
		FirstName    string    `json:"first_name"`
		LastName     string    `json:"last_name"`
		Email        *string   `json:"email"`
		IsAdmin      bool      `json:"is_admin"`
		IsSuperAdmin bool      `json:"is_super_admin"`
		IsActive     bool      `json:"is_active"`
		CreatedAt    time.Time `json:"created_at"`
		TenantName   *string   `json:"tenant_name"`
	}

	results := []UserSearchResult{}
	for rows.Next() {
		var u UserSearchResult
		if err := rows.Scan(&u.ID, &u.TenantID, &u.FirstName, &u.LastName, &u.Email,
			&u.IsAdmin, &u.IsSuperAdmin, &u.IsActive, &u.CreatedAt, &u.TenantName); err != nil {
			log.Printf("Error scanning user search result: %v", err)
			continue
		}
		results = append(results, u)
	}

	respondJSON(w, http.StatusOK, results)
}

// ExportTenantData exports all data for a tenant (GDPR compliance)
// GET /api/central-admin/tenants/{id}/export
func (h *CentralAdminHandler) ExportTenantData(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	tenantID, err := strconv.Atoi(vars["id"])
	if err != nil {
		respondError(w, http.StatusBadRequest, "Ungültige Tierheim-ID")
		return
	}

	tenant, err := h.tenantRepo.FindByID(tenantID)
	if err != nil {
		respondError(w, http.StatusNotFound, "Tierheim nicht gefunden")
		return
	}

	// Collect all tenant data
	export := map[string]interface{}{
		"tenant":       tenant,
		"exported_at":  time.Now(),
		"exported_by":  r.Context().Value(middleware.UserIDKey),
	}

	// Get users
	users, _ := h.userRepo.FindAll(nil, tenantID)
	for i := range users {
		users[i].PasswordHash = nil
		users[i].VerificationToken = nil
		users[i].PasswordResetToken = nil
	}
	export["users"] = users

	// Get dogs
	var dogs []models.Dog
	rows, err := h.db.Query(`SELECT id, tenant_id, name, breed, size, age,
		category, is_featured, is_available, external_link, photo, photo_thumbnail,
		created_at, updated_at FROM dogs WHERE tenant_id = ?`, tenantID)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var d models.Dog
			rows.Scan(&d.ID, &d.TenantID, &d.Name, &d.Breed, &d.Size, &d.Age,
				&d.Category, &d.IsFeatured, &d.IsAvailable, &d.ExternalLink, &d.Photo, &d.PhotoThumbnail,
				&d.CreatedAt, &d.UpdatedAt)
			dogs = append(dogs, d)
		}
	}
	export["dogs"] = dogs

	// Get bookings count (not full data for performance)
	var bookingCount int
	h.db.QueryRow(`SELECT COUNT(*) FROM bookings WHERE tenant_id = ?`, tenantID).Scan(&bookingCount)
	export["booking_count"] = bookingCount

	// Audit log
	adminID, _ := r.Context().Value(middleware.UserIDKey).(int)
	log.Printf("AUDIT: Central admin %d exported data for tenant %d (%s)", adminID, tenantID, tenant.Slug)

	respondJSON(w, http.StatusOK, export)
}
