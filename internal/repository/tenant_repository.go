package repository

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/tranmh/gassigeher/internal/models"
)

// TenantRepository handles tenant database operations
type TenantRepository struct {
	db *sql.DB
}

// NewTenantRepository creates a new tenant repository
func NewTenantRepository(db *sql.DB) *TenantRepository {
	return &TenantRepository{db: db}
}

// Create creates a new tenant
func (r *TenantRepository) Create(tenant *models.Tenant) error {
	query := `
		INSERT INTO tenants (
			slug, name, status, contact_email, contact_phone,
			address, city, postal_code, federal_state, is_demo
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	result, err := r.db.Exec(
		query,
		tenant.Slug,
		tenant.Name,
		tenant.Status,
		tenant.ContactEmail,
		tenant.ContactPhone,
		tenant.Address,
		tenant.City,
		tenant.PostalCode,
		tenant.FederalState,
		tenant.IsDemo,
	)
	if err != nil {
		return fmt.Errorf("failed to create tenant: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get tenant ID: %w", err)
	}

	tenant.ID = int(id)
	tenant.CreatedAt = time.Now()
	tenant.UpdatedAt = time.Now()
	return nil
}

// CreateTx creates a new tenant within a transaction
func (r *TenantRepository) CreateTx(tx *sql.Tx, tenant *models.Tenant) (int, error) {
	query := `
		INSERT INTO tenants (
			slug, name, status, contact_email, contact_phone,
			address, city, postal_code, federal_state, is_demo
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	result, err := tx.Exec(
		query,
		tenant.Slug,
		tenant.Name,
		tenant.Status,
		tenant.ContactEmail,
		tenant.ContactPhone,
		tenant.Address,
		tenant.City,
		tenant.PostalCode,
		tenant.FederalState,
		tenant.IsDemo,
	)
	if err != nil {
		return 0, fmt.Errorf("failed to create tenant: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("failed to get tenant ID: %w", err)
	}

	return int(id), nil
}

// FindByID finds a tenant by ID
func (r *TenantRepository) FindByID(id int) (*models.Tenant, error) {
	query := `
		SELECT id, slug, name, status, contact_email, contact_phone,
		       address, city, postal_code, federal_state, is_demo,
		       created_at, updated_at, suspended_at, suspended_reason, deleted_at
		FROM tenants
		WHERE id = ?
	`

	tenant := &models.Tenant{}
	err := r.db.QueryRow(query, id).Scan(
		&tenant.ID,
		&tenant.Slug,
		&tenant.Name,
		&tenant.Status,
		&tenant.ContactEmail,
		&tenant.ContactPhone,
		&tenant.Address,
		&tenant.City,
		&tenant.PostalCode,
		&tenant.FederalState,
		&tenant.IsDemo,
		&tenant.CreatedAt,
		&tenant.UpdatedAt,
		&tenant.SuspendedAt,
		&tenant.SuspendedReason,
		&tenant.DeletedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find tenant: %w", err)
	}

	return tenant, nil
}

// FindBySlug finds a tenant by slug (subdomain)
func (r *TenantRepository) FindBySlug(slug string) (*models.Tenant, error) {
	query := `
		SELECT id, slug, name, status, contact_email, contact_phone,
		       address, city, postal_code, federal_state, is_demo,
		       created_at, updated_at, suspended_at, suspended_reason, deleted_at
		FROM tenants
		WHERE slug = ?
	`

	tenant := &models.Tenant{}
	err := r.db.QueryRow(query, slug).Scan(
		&tenant.ID,
		&tenant.Slug,
		&tenant.Name,
		&tenant.Status,
		&tenant.ContactEmail,
		&tenant.ContactPhone,
		&tenant.Address,
		&tenant.City,
		&tenant.PostalCode,
		&tenant.FederalState,
		&tenant.IsDemo,
		&tenant.CreatedAt,
		&tenant.UpdatedAt,
		&tenant.SuspendedAt,
		&tenant.SuspendedReason,
		&tenant.DeletedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find tenant: %w", err)
	}

	return tenant, nil
}

// FindAll returns all tenants with optional status filter
func (r *TenantRepository) FindAll(status string) ([]*models.Tenant, error) {
	var query string
	var args []interface{}

	if status != "" && status != "all" {
		query = `
			SELECT id, slug, name, status, contact_email, contact_phone,
			       address, city, postal_code, federal_state, is_demo,
			       created_at, updated_at, suspended_at, suspended_reason, deleted_at
			FROM tenants
			WHERE status = ?
			ORDER BY name ASC
		`
		args = append(args, status)
	} else {
		query = `
			SELECT id, slug, name, status, contact_email, contact_phone,
			       address, city, postal_code, federal_state, is_demo,
			       created_at, updated_at, suspended_at, suspended_reason, deleted_at
			FROM tenants
			ORDER BY name ASC
		`
	}

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to find tenants: %w", err)
	}
	defer rows.Close()

	var tenants []*models.Tenant
	for rows.Next() {
		tenant := &models.Tenant{}
		err := rows.Scan(
			&tenant.ID,
			&tenant.Slug,
			&tenant.Name,
			&tenant.Status,
			&tenant.ContactEmail,
			&tenant.ContactPhone,
			&tenant.Address,
			&tenant.City,
			&tenant.PostalCode,
			&tenant.FederalState,
			&tenant.IsDemo,
			&tenant.CreatedAt,
			&tenant.UpdatedAt,
			&tenant.SuspendedAt,
			&tenant.SuspendedReason,
			&tenant.DeletedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan tenant: %w", err)
		}
		tenants = append(tenants, tenant)
	}

	return tenants, nil
}

// Update updates a tenant
func (r *TenantRepository) Update(tenant *models.Tenant) error {
	query := `
		UPDATE tenants SET
			name = ?,
			contact_email = ?,
			contact_phone = ?,
			address = ?,
			city = ?,
			postal_code = ?,
			federal_state = ?,
			updated_at = ?
		WHERE id = ?
	`

	now := time.Now()
	_, err := r.db.Exec(
		query,
		tenant.Name,
		tenant.ContactEmail,
		tenant.ContactPhone,
		tenant.Address,
		tenant.City,
		tenant.PostalCode,
		tenant.FederalState,
		now,
		tenant.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update tenant: %w", err)
	}

	tenant.UpdatedAt = now
	return nil
}

// Suspend suspends a tenant
func (r *TenantRepository) Suspend(id int, reason string) error {
	query := `
		UPDATE tenants SET
			status = ?,
			suspended_at = ?,
			suspended_reason = ?,
			updated_at = ?
		WHERE id = ?
	`

	now := time.Now()
	_, err := r.db.Exec(query, models.TenantStatusSuspended, now, reason, now, id)
	if err != nil {
		return fmt.Errorf("failed to suspend tenant: %w", err)
	}

	return nil
}

// Activate activates a suspended tenant
func (r *TenantRepository) Activate(id int) error {
	query := `
		UPDATE tenants SET
			status = ?,
			suspended_at = NULL,
			suspended_reason = NULL,
			updated_at = ?
		WHERE id = ?
	`

	now := time.Now()
	_, err := r.db.Exec(query, models.TenantStatusActive, now, id)
	if err != nil {
		return fmt.Errorf("failed to activate tenant: %w", err)
	}

	return nil
}

// SoftDelete marks a tenant as deleted
func (r *TenantRepository) SoftDelete(id int) error {
	query := `
		UPDATE tenants SET
			status = ?,
			deleted_at = ?,
			updated_at = ?
		WHERE id = ?
	`

	now := time.Now()
	_, err := r.db.Exec(query, models.TenantStatusDeleted, now, now, id)
	if err != nil {
		return fmt.Errorf("failed to delete tenant: %w", err)
	}

	return nil
}

// SlugExists checks if a slug is already taken
func (r *TenantRepository) SlugExists(slug string) (bool, error) {
	query := `SELECT COUNT(*) FROM tenants WHERE slug = ?`
	var count int
	err := r.db.QueryRow(query, slug).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check slug: %w", err)
	}
	return count > 0, nil
}

// GetStats returns statistics for a tenant
func (r *TenantRepository) GetStats(tenantID int) (*models.TenantStats, error) {
	stats := &models.TenantStats{TenantID: tenantID}

	// Total users
	err := r.db.QueryRow(`SELECT COUNT(*) FROM users WHERE tenant_id = ? AND is_deleted = 0`, tenantID).Scan(&stats.TotalUsers)
	if err != nil {
		return nil, fmt.Errorf("failed to count users: %w", err)
	}

	// Active users
	err = r.db.QueryRow(`SELECT COUNT(*) FROM users WHERE tenant_id = ? AND is_deleted = 0 AND is_active = 1`, tenantID).Scan(&stats.ActiveUsers)
	if err != nil {
		return nil, fmt.Errorf("failed to count active users: %w", err)
	}

	// Total dogs
	err = r.db.QueryRow(`SELECT COUNT(*) FROM dogs WHERE tenant_id = ?`, tenantID).Scan(&stats.TotalDogs)
	if err != nil {
		return nil, fmt.Errorf("failed to count dogs: %w", err)
	}

	// Available dogs
	err = r.db.QueryRow(`SELECT COUNT(*) FROM dogs WHERE tenant_id = ? AND is_available = 1`, tenantID).Scan(&stats.AvailableDogs)
	if err != nil {
		return nil, fmt.Errorf("failed to count available dogs: %w", err)
	}

	// Total bookings
	err = r.db.QueryRow(`SELECT COUNT(*) FROM bookings WHERE tenant_id = ?`, tenantID).Scan(&stats.TotalBookings)
	if err != nil {
		return nil, fmt.Errorf("failed to count bookings: %w", err)
	}

	// Bookings this month
	firstOfMonth := time.Now().Format("2006-01-01")
	err = r.db.QueryRow(`SELECT COUNT(*) FROM bookings WHERE tenant_id = ? AND date >= ?`, tenantID, firstOfMonth).Scan(&stats.BookingsThisMonth)
	if err != nil {
		return nil, fmt.Errorf("failed to count monthly bookings: %w", err)
	}

	return stats, nil
}

// --- Tenant Settings ---

// CreateSettings creates settings for a tenant
func (r *TenantRepository) CreateSettings(settings *models.TenantSettings) error {
	query := `
		INSERT INTO tenant_settings (
			tenant_id, theme_preset, color_primary, color_secondary,
			color_accent, color_background, color_text,
			logo_url, favicon_url, welcome_message, footer_text,
			website_url, donation_url
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	result, err := r.db.Exec(
		query,
		settings.TenantID,
		settings.ThemePreset,
		settings.ColorPrimary,
		settings.ColorSecondary,
		settings.ColorAccent,
		settings.ColorBackground,
		settings.ColorText,
		settings.LogoURL,
		settings.FaviconURL,
		settings.WelcomeMessage,
		settings.FooterText,
		settings.WebsiteURL,
		settings.DonationURL,
	)
	if err != nil {
		return fmt.Errorf("failed to create tenant settings: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get settings ID: %w", err)
	}

	settings.ID = int(id)
	settings.CreatedAt = time.Now()
	settings.UpdatedAt = time.Now()
	return nil
}

// CreateSettingsTx creates settings for a tenant within a transaction
func (r *TenantRepository) CreateSettingsTx(tx *sql.Tx, settings *models.TenantSettings) error {
	query := `
		INSERT INTO tenant_settings (
			tenant_id, theme_preset, color_primary, color_secondary,
			color_accent, color_background, color_text,
			logo_url, favicon_url, welcome_message, footer_text,
			website_url, donation_url
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := tx.Exec(
		query,
		settings.TenantID,
		settings.ThemePreset,
		settings.ColorPrimary,
		settings.ColorSecondary,
		settings.ColorAccent,
		settings.ColorBackground,
		settings.ColorText,
		settings.LogoURL,
		settings.FaviconURL,
		settings.WelcomeMessage,
		settings.FooterText,
		settings.WebsiteURL,
		settings.DonationURL,
	)
	if err != nil {
		return fmt.Errorf("failed to create tenant settings: %w", err)
	}

	return nil
}

// GetSettings retrieves settings for a tenant
func (r *TenantRepository) GetSettings(tenantID int) (*models.TenantSettings, error) {
	query := `
		SELECT id, tenant_id, theme_preset, color_primary, color_secondary,
		       color_accent, color_background, color_text,
		       logo_url, favicon_url, welcome_message, footer_text,
		       website_url, donation_url, created_at, updated_at
		FROM tenant_settings
		WHERE tenant_id = ?
	`

	settings := &models.TenantSettings{}
	err := r.db.QueryRow(query, tenantID).Scan(
		&settings.ID,
		&settings.TenantID,
		&settings.ThemePreset,
		&settings.ColorPrimary,
		&settings.ColorSecondary,
		&settings.ColorAccent,
		&settings.ColorBackground,
		&settings.ColorText,
		&settings.LogoURL,
		&settings.FaviconURL,
		&settings.WelcomeMessage,
		&settings.FooterText,
		&settings.WebsiteURL,
		&settings.DonationURL,
		&settings.CreatedAt,
		&settings.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get tenant settings: %w", err)
	}

	return settings, nil
}

// UpdateSettings updates settings for a tenant
func (r *TenantRepository) UpdateSettings(settings *models.TenantSettings) error {
	query := `
		UPDATE tenant_settings SET
			theme_preset = ?,
			color_primary = ?,
			color_secondary = ?,
			color_accent = ?,
			color_background = ?,
			color_text = ?,
			logo_url = ?,
			favicon_url = ?,
			welcome_message = ?,
			footer_text = ?,
			website_url = ?,
			donation_url = ?,
			updated_at = ?
		WHERE tenant_id = ?
	`

	now := time.Now()
	_, err := r.db.Exec(
		query,
		settings.ThemePreset,
		settings.ColorPrimary,
		settings.ColorSecondary,
		settings.ColorAccent,
		settings.ColorBackground,
		settings.ColorText,
		settings.LogoURL,
		settings.FaviconURL,
		settings.WelcomeMessage,
		settings.FooterText,
		settings.WebsiteURL,
		settings.DonationURL,
		now,
		settings.TenantID,
	)
	if err != nil {
		return fmt.Errorf("failed to update tenant settings: %w", err)
	}

	settings.UpdatedAt = now
	return nil
}

// UpdateLogo updates the logo URL for a tenant
func (r *TenantRepository) UpdateLogo(tenantID int, logoURL string) error {
	query := `UPDATE tenant_settings SET logo_url = ?, updated_at = ? WHERE tenant_id = ?`
	_, err := r.db.Exec(query, logoURL, time.Now(), tenantID)
	if err != nil {
		return fmt.Errorf("failed to update logo: %w", err)
	}
	return nil
}

// UpdateFavicon updates the favicon URL for a tenant
func (r *TenantRepository) UpdateFavicon(tenantID int, faviconURL string) error {
	query := `UPDATE tenant_settings SET favicon_url = ?, updated_at = ? WHERE tenant_id = ?`
	_, err := r.db.Exec(query, faviconURL, time.Now(), tenantID)
	if err != nil {
		return fmt.Errorf("failed to update favicon: %w", err)
	}
	return nil
}

// --- Demo Tenant Methods ---

// IsDemoTenant checks if a tenant is a demo tenant
func (r *TenantRepository) IsDemoTenant(tenantID int) (bool, error) {
	query := `SELECT is_demo FROM tenants WHERE id = ?`
	var isDemo bool
	err := r.db.QueryRow(query, tenantID).Scan(&isDemo)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("failed to check demo tenant: %w", err)
	}
	return isDemo, nil
}

// GetDemoTenant returns the demo tenant if it exists
func (r *TenantRepository) GetDemoTenant() (*models.Tenant, error) {
	// Use is_demo = 1 which works across SQLite, MySQL, and PostgreSQL
	// (PostgreSQL treats 1 as truthy for boolean columns)
	query := `
		SELECT id, slug, name, status, contact_email, contact_phone,
		       address, city, postal_code, federal_state, is_demo,
		       created_at, updated_at, suspended_at, suspended_reason, deleted_at
		FROM tenants
		WHERE is_demo = 1
		LIMIT 1
	`

	tenant := &models.Tenant{}
	err := r.db.QueryRow(query).Scan(
		&tenant.ID,
		&tenant.Slug,
		&tenant.Name,
		&tenant.Status,
		&tenant.ContactEmail,
		&tenant.ContactPhone,
		&tenant.Address,
		&tenant.City,
		&tenant.PostalCode,
		&tenant.FederalState,
		&tenant.IsDemo,
		&tenant.CreatedAt,
		&tenant.UpdatedAt,
		&tenant.SuspendedAt,
		&tenant.SuspendedReason,
		&tenant.DeletedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get demo tenant: %w", err)
	}

	return tenant, nil
}
